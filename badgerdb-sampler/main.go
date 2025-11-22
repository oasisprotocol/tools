package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/gogo/protobuf/proto"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmstore "github.com/tendermint/tendermint/proto/tendermint/store"
)

// BadgerVersion is set by build tags
var BadgerVersion string

// Sample contains both raw and decoded key/value information
type Sample struct {
	RawKey       string `json:"raw_key"`
	RawKeySize   int    `json:"raw_key_size"`
	KeyType      string `json:"key_type"`
	DecodedKey   string `json:"decoded_key"`
	RawValue     string `json:"raw_value"`
	RawValueSize int    `json:"raw_value_size"`
	DecodedValue string `json:"decoded_value"`
	Timestamp    int64  `json:"timestamp,omitempty"`
}

// DBStats contains database statistics and samples
type DBStats struct {
	DatabasePath    string         `json:"database_path"`
	DatabaseType    string         `json:"database_type"`
	BadgerDBVersion string         `json:"badgerdb_version"`
	DatabaseSize    int64          `json:"database_size_bytes,omitempty"`
	SampleCount     int            `json:"sample_count"`
	KeyTypeCounts   map[string]int `json:"key_type_counts,omitempty"`
	Samples         []Sample       `json:"samples"`
}

// Runtime history CBOR types (from oasis-core/go/runtime/history/db.go)

// RuntimeHistoryMetadata represents the metadata stored in runtime history DB
type RuntimeHistoryMetadata struct {
	RuntimeID           []byte `cbor:"runtime_id"`
	Version             uint64 `cbor:"version"`
	LastConsensusHeight int64  `cbor:"last_consensus_height"`
	LastRound           uint64 `cbor:"last_round"`
}

// RuntimeHistoryAnnotatedBlock represents an annotated block in runtime history
type RuntimeHistoryAnnotatedBlock struct {
	Height int64                    `cbor:"consensus_height"`
	Block  *RuntimeHistoryBlock     `cbor:"block"`
}

// RuntimeHistoryBlock represents a runtime block
type RuntimeHistoryBlock struct {
	Header RuntimeHistoryBlockHeader `cbor:"header"`
}

// RuntimeHistoryBlockHeader represents a runtime block header
type RuntimeHistoryBlockHeader struct {
	Version        uint16 `cbor:"version"`
	Namespace      []byte `cbor:"namespace"`
	Round          uint64 `cbor:"round"`
	Timestamp      uint64 `cbor:"timestamp"`
	HeaderType     uint8  `cbor:"header_type"`
	PreviousHash   []byte `cbor:"previous_hash"`
	IORoot         []byte `cbor:"io_root"`
	StateRoot      []byte `cbor:"state_root"`
	MessagesHash   []byte `cbor:"messages_hash"`
	InMessagesHash []byte `cbor:"in_msgs_hash"`
}

// RuntimeHistoryRoundResults represents round results in runtime history
type RuntimeHistoryRoundResults struct {
	Messages            []RuntimeHistoryMessageEvent `cbor:"messages,omitempty"`
	GoodComputeEntities [][]byte                     `cbor:"good_compute_entities,omitempty"`
	BadComputeEntities  [][]byte                     `cbor:"bad_compute_entities,omitempty"`
}

// RuntimeHistoryMessageEvent represents a message event
type RuntimeHistoryMessageEvent struct {
	Module string      `cbor:"module,omitempty"`
	Code   uint32      `cbor:"code,omitempty"`
	Index  uint32      `cbor:"index,omitempty"`
	Result cbor.RawMessage `cbor:"result,omitempty"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <database-type> <path-to-db> [output-json] [max-samples]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Database types: consensus-blockstore, consensus-evidence, consensus-mkvs, consensus-state, runtime-mkvs, runtime-history\n")
		fmt.Fprintf(os.Stderr, "Example: %s consensus-blockstore /path/to/blockstore.badger.db\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s runtime-mkvs /path/to/mkvs_storage.badger.db ./outputs/testnet-20220303/emerald-runtime-mkvs.json 500\n", os.Args[0])
		os.Exit(1)
	}

	// Parse arguments
	dbType := normalizeDBType(os.Args[1])
	dbPath := os.Args[2]
	jsonFile := ""
	maxSamples := 1000 // default
	if len(os.Args) > 3 {
		jsonFile = os.Args[3]
	}
	if len(os.Args) > 4 {
		var err error
		maxSamples, err = strconv.Atoi(os.Args[4])
		if err != nil || maxSamples <= 0 {
			log.Fatalf("Invalid max-samples value: %s (must be positive integer)", os.Args[4])
		}
	}

	// Calculate database directory size
	dbSize, err := getDatabaseSize(dbPath)
	if err != nil {
		log.Printf("Warning: Could not calculate database size: %v", err)
	}

	// Open database with two-mode strategy (corruption handling)
	fmt.Printf("Opening database %s (type: %s, size: %d bytes)...\n", dbPath, dbType, dbSize)
	db, err := openDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	stats := DBStats{
		DatabasePath:    dbPath,
		DatabaseType:    dbType,
		BadgerDBVersion: BadgerVersion,
		DatabaseSize:    dbSize,
		KeyTypeCounts:   make(map[string]int),
		Samples:         []Sample{},
	}

	// Collect samples
	fmt.Printf("Collecting and decoding samples...\n")
	err = collectSamples(db, &stats, maxSamples)
	if err != nil {
		log.Fatalf("Error collecting samples: %v", err)
	}

	// Print results
	fmt.Printf("Results:\n")
	output, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
	}
	fmt.Println(string(output))

	// Export to JSON if requested
	if jsonFile != "" {
		jsonDir := filepath.Dir(jsonFile)
		err = os.MkdirAll(jsonDir, 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nWarning: Failed to create directory %s: %v\n", jsonDir, err)
		} else {
			err = os.WriteFile(jsonFile, output, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nWarning: Failed to write JSON file %s: %v\n", jsonFile, err)
			} else {
				fmt.Fprintf(os.Stderr, "\n✓ Results exported to: %s\n", jsonFile)
			}
		}
	}
}

// normalizeDBType removes version suffix from database type (e.g., "consensus-blockstore-v2" -> "consensus-blockstore")
func normalizeDBType(dbType string) string {
	// Remove -v2, -v3, -v4 suffix if present
	for _, suffix := range []string{"-v2", "-v3", "-v4"} {
		if strings.HasSuffix(dbType, suffix) {
			return strings.TrimSuffix(dbType, suffix)
		}
	}
	return dbType
}

// getDatabaseSize calculates the total size of all files in the database directory
func getDatabaseSize(dbPath string) (int64, error) {
	var totalSize int64
	err := filepath.Walk(dbPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize, err
}

// collectSamples iterates through keys and collects samples with full decoding
func collectSamples(db *DB, stats *DBStats, maxSamples int) error {
	return db.View(func(txn *Txn) error {
		opts := DefaultIteratorOptions
		opts.PrefetchValues = true
		opts.PrefetchSize = 100            // Optimal prefetch size for BadgerDB
		opts.AllVersions = false           // Only latest version of each key
		it := txn.NewIterator(opts)
		defer it.Close()

		sampleCount := 0
		startTime := time.Now()

		fmt.Fprintf(os.Stderr, "Starting iteration...\n")

		for it.Rewind(); it.Valid(); it.Next() {
			if sampleCount >= maxSamples {
				break
			}

			// Progress logging every few samples
			if sampleCount > 0 && sampleCount%1000 == 0 {
				elapsed := time.Since(startTime)
				fmt.Fprintf(os.Stderr, "  Progress: %d/%d samples collected (elapsed: %v)\n", sampleCount, maxSamples, elapsed.Round(time.Millisecond))
			}
			item := it.Item()
			key := item.Key()

			sample := Sample{
				RawKey:     fmt.Sprintf("%x", key),
				RawKeySize: len(key),
				KeyType:    "unknown",
				DecodedKey: "",
			}

			// Analyze key based on database type
			switch stats.DatabaseType {
			case "consensus-blockstore":
				keyType, decodedKey := analyzeKeyConsensusBlockstore(key)
				sample.KeyType = keyType
				sample.DecodedKey = decodedKey
			case "consensus-evidence":
				keyType, decodedKey := analyzeKeyConsensusEvidence(key)
				sample.KeyType = keyType
				sample.DecodedKey = decodedKey
			case "consensus-mkvs":
				keyType, decodedKey := analyzeKeyConsensusMkvs(key)
				sample.KeyType = keyType
				sample.DecodedKey = decodedKey
			case "consensus-state":
				keyType, decodedKey := analyzeKeyConsensusState(key)
				sample.KeyType = keyType
				sample.DecodedKey = decodedKey
			case "runtime-mkvs":
				keyType, decodedKey := analyzeKeyRuntimeMkvs(key)
				sample.KeyType = keyType
				sample.DecodedKey = decodedKey
			case "runtime-history":
				keyType, decodedKey := analyzeKeyRuntimeHistory(key)
				sample.KeyType = keyType
				sample.DecodedKey = decodedKey
			}

			// Fetch and decode value
			err := item.Value(func(val []byte) error {
				sample.RawValueSize = len(val)
				sample.RawValue = truncateHex(val, 100)
				sample.DecodedValue = ""

				// Decode value based on database type
				switch stats.DatabaseType {
				case "consensus-blockstore":
					decoded, ts := decodeValueConsensusBlockstore(sample.KeyType, val)
					sample.DecodedValue = decoded
					sample.Timestamp = ts
				case "consensus-evidence":
					sample.DecodedValue = decodeValueConsensusEvidence(sample.KeyType, val)
				case "consensus-mkvs":
					sample.DecodedValue = decodeValueConsensusMkvs(sample.KeyType, val)
				case "consensus-state":
					sample.DecodedValue = decodeValueConsensusState(sample.KeyType, val)
				case "runtime-mkvs":
					sample.DecodedValue = decodeValueRuntimeMkvs(sample.KeyType, val)
				case "runtime-history":
					sample.DecodedValue = decodeValueRuntimeHistory(sample.KeyType, val)
				}
				return nil
			})
			if err != nil {
				log.Printf("Error reading value for key %s: %v", sample.RawKey, err)
			}

			stats.Samples = append(stats.Samples, sample)
			stats.KeyTypeCounts[sample.KeyType]++
			sampleCount++
		}

		stats.SampleCount = sampleCount
		elapsed := time.Since(startTime)
		fmt.Fprintf(os.Stderr, "Iteration complete: %d samples in %v\n", sampleCount, elapsed.Round(time.Millisecond))
		return nil
	})
}

// analyzeKeyConsensusBlockstore parses consensus-blockstore key and returns key type and decoded representation
func analyzeKeyConsensusBlockstore(key []byte) (string, string) {
	if len(key) < 2 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// All keys start with 0x01 (dbVersion from Oasis BadgerDB wrapper)
	if key[0] != 0x01 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// Parse ASCII key after 0x01 prefix
	asciiKey := string(key[1:])

	// blockStore state key
	if asciiKey == "blockStore" {
		return "blockstore_state", "blockStore"
	}

	// Parse structured keys: "H:{height}", "P:{height}:{part}", "C:{height}", "SC:{height}", "BH:{hash}"
	if strings.HasPrefix(asciiKey, "H:") {
		return "block_meta", asciiKey
	}
	if strings.HasPrefix(asciiKey, "P:") {
		return "block_part", asciiKey
	}
	if strings.HasPrefix(asciiKey, "C:") {
		return "block_commit", asciiKey
	}
	if strings.HasPrefix(asciiKey, "SC:") {
		return "seen_commit", asciiKey
	}
	if strings.HasPrefix(asciiKey, "BH:") {
		return "block_hash", asciiKey
	}

	return "unknown", asciiKey
}

// decodeValueConsensusBlockstore attempts to decode protobuf value based on key type
// Returns: (decoded_string, timestamp_unix)
func decodeValueConsensusBlockstore(keyType string, value []byte) (string, int64) {
	switch keyType {
	case "blockstore_state":
		var state tmstore.BlockStoreState
		if err := proto.Unmarshal(value, &state); err == nil {
			return fmt.Sprintf("BlockStoreState{base: %d, height: %d}", state.Base, state.Height), 0
		}

	case "block_meta":
		var meta tmproto.BlockMeta
		if err := proto.Unmarshal(value, &meta); err == nil {
			// Extract timestamp
			tsUnix := int64(0)
			ts := ""
			if meta.Header.Time.Unix() > 0 {
				tsUnix = meta.Header.Time.Unix()
				ts = meta.Header.Time.Format(time.RFC3339)
			}

			// Format AppHash and ChainID
			appHash := ""
			if len(meta.Header.AppHash) >= 8 {
				appHash = fmt.Sprintf("%x", meta.Header.AppHash[:8])
			} else if len(meta.Header.AppHash) > 0 {
				appHash = fmt.Sprintf("%x", meta.Header.AppHash)
			}

			chainID := meta.Header.ChainID
			if len(chainID) > 20 {
				chainID = chainID[:20] + "..."
			}

			return fmt.Sprintf("BlockMeta{height: %d, time: %s, chain: %s, num_txs: %d, app_hash: %s}",
				meta.Header.Height, ts, chainID, meta.NumTxs, appHash), tsUnix
		}

	case "block_part":
		var part tmproto.Part
		if err := proto.Unmarshal(value, &part); err == nil {
			return fmt.Sprintf("Part{index: %d, bytes_size: %d, proof_total: %d}",
				part.Index, len(part.Bytes), part.Proof.Total), 0
		}

	case "block_commit":
		var commit tmproto.Commit
		if err := proto.Unmarshal(value, &commit); err == nil {
			return fmt.Sprintf("Commit{height: %d, round: %d, signatures: %d}",
				commit.Height, commit.Round, len(commit.Signatures)), 0
		}

	case "seen_commit":
		var commit tmproto.Commit
		if err := proto.Unmarshal(value, &commit); err == nil {
			return fmt.Sprintf("Commit{height: %d, round: %d, signatures: %d}",
				commit.Height, commit.Round, len(commit.Signatures)), 0
		}

	case "block_hash":
		// Block hash values are plain strings containing height numbers
		height, err := strconv.ParseInt(string(value), 10, 64)
		if err == nil {
			return fmt.Sprintf("Hash{height: %d}", height), 0
		}

	default:
		return fmt.Sprintf("Unknown type (size: %d bytes)", len(value)), 0
	}

	return fmt.Sprintf("Failed to decode blockstore (type: %s, size: %d bytes)", keyType, len(value)), 0
}

// analyzeKeyConsensusEvidence parses consensus-evidence key and returns key type and decoded representation
func analyzeKeyConsensusEvidence(key []byte) (string, string) {
	if len(key) < 2 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// All keys start with 0x01 (dbVersion from Oasis BadgerDB wrapper)
	if key[0] != 0x01 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// Evidence DB is typically empty or uses simple key patterns
	return fmt.Sprintf("type_%02x", key[1]), fmt.Sprintf("%x", key[1:])
}

// decodeValueConsensusEvidence attempts to decode evidence value
func decodeValueConsensusEvidence(keyType string, value []byte) string {
	if len(value) == 0 {
		return "Empty"
	}

	// Try to decode as DuplicateVoteEvidence
	var evidence tmproto.DuplicateVoteEvidence
	if err := proto.Unmarshal(value, &evidence); err == nil {
		return fmt.Sprintf("DuplicateVoteEvidence{vote_a_height: %d, vote_b_height: %d}",
			evidence.VoteA.Height, evidence.VoteB.Height)
	}

	return fmt.Sprintf("Evidence (type: %s, size: %d bytes)", keyType, len(value))
}

// analyzeKeyConsensusMkvs parses consensus-mkvs key and returns key type and decoded representation
// FIXED: Handles both old format (no dbVersion prefix) and new format (0x01 or 0x05 dbVersion prefix)
// MKVS keys from oasis-core/go/storage/mkvs/db/badger/badger.go
func analyzeKeyConsensusMkvs(key []byte) (string, string) {
	if len(key) < 1 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// Check if key has dbVersion prefix (0x01 or 0x05)
	// Old format: type byte directly at key[0]
	// New format: 0x01 or 0x05 at key[0], type byte at key[1]
	prefixByte := key[0]
	var data []byte

	if prefixByte == 0x01 || prefixByte == 0x05 {
		// New format with dbVersion prefix
		if len(key) < 2 {
			return "unknown", fmt.Sprintf("%x", key)
		}
		prefixByte = key[1]
		data = key[2:]
	} else {
		// Old format without dbVersion prefix (BadgerDB v2 MKVS)
		data = key[1:]
	}

	switch prefixByte {
	case 0x00:
		// nodeKeyFmt: hash.Hash (32 bytes)
		return "node", fmt.Sprintf("node{hash: %x....}", truncateBytes(data, 8))
	case 0x01:
		// writeLogKeyFmt: uint64(version) + TypedHash(new root) + TypedHash(old root)
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			if len(data) >= 8+33 { // version + first TypedHash (33 bytes in v3+)
				rootType := data[8]
				rootHash := data[9:9+32]
				return "write_log", fmt.Sprintf("write_log{v:%d, new_root_type:%d, hash: %x...}", version, rootType, truncateBytes(rootHash, 4))
			}
			return "write_log", fmt.Sprintf("write_log{v:%d}", version)
		}
		return "write_log", "write_log{<malformed>}"
	case 0x02:
		// rootsMetadataKeyFmt: uint64(version)
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			return "roots_metadata", fmt.Sprintf("roots_metadata{v:%d}", version)
		}
		return "roots_metadata", "roots_metadata{<malformed>}"
	case 0x03:
		// rootUpdatedNodesKeyFmt: uint64(version) + TypedHash(root)
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			if len(data) >= 8+33 { // version + TypedHash
				rootType := data[8]
				rootHash := data[9:9+32]
				return "root_updated_nodes", fmt.Sprintf("root_updated_nodes{v:%d, root_type:%d, hash:%x...}", version, rootType, truncateBytes(rootHash, 4))
			}
			return "root_updated_nodes", fmt.Sprintf("root_updated_nodes{v:%d}", version)
		}
		return "root_updated_nodes", "root_updated_nodes{<malformed>}"
	case 0x04:
		// metadataKeyFmt: no additional data
		return "metadata", "metadata"
	case 0x05:
		// multipartRestoreNodeLogKeyFmt: TypedHash (33 bytes)
		if len(data) >= 33 {
			rootType := data[0]
			rootHash := data[1:33]
			return "multipart_restore_log", fmt.Sprintf("multipart_restore_log{type:%d, hash:%x}", rootType, truncateBytes(rootHash, 8))
		}
		return "multipart_restore_log", fmt.Sprintf("multipart_restore_log{hash:%x}", truncateBytes(data, 8))
	case 0x06:
		// rootNodeKeyFmt: TypedHash (33 bytes)
		if len(data) >= 33 {
			rootType := data[0]
			rootHash := data[1:33]
			return "root_node", fmt.Sprintf("root_node{type:%d, hash:%x}", rootType, truncateBytes(rootHash, 8))
		}
		return "root_node", fmt.Sprintf("root_node{hash:%x}", truncateBytes(data, 8))
	default:
		return fmt.Sprintf("unknown_%02x", prefixByte), fmt.Sprintf("unknown_%02x:%x", prefixByte, truncateBytes(data, 8))
	}
}

// decodeValueConsensusMkvs decodes consensus MKVS value with inline node parsing
func decodeValueConsensusMkvs(keyType string, value []byte) string {
	if len(value) == 0 {
		return "Empty"
	}

	if keyType != "node" {
		return fmt.Sprintf("MKVS %s (size: %d bytes)", keyType, len(value))
	}

	// Parse MKVS node: 0x00=leaf, 0x01=internal, 0x02=nil
	switch value[0] {
	case 0x00: // LeafNode: [2-byte keyLen LE][key][4-byte valueLen LE][value]
		data := value[1:]
		if len(data) < 2 {
			return "LeafNode{<malformed>}"
		}

		keyLen := int(binary.LittleEndian.Uint16(data[0:2]))
		data = data[2:]
		if len(data) < keyLen {
			return fmt.Sprintf("LeafNode{keyLen=%d, <truncated>}", keyLen)
		}

		key := data[:keyLen]
		data = data[keyLen:]

		// Decode consensus module key prefix (roothash, staking, registry, etc.)
		// Consensus modules use single-byte prefixes from oasis-core/go/consensus/tendermint/apps/*/state/state.go
		var module string
		if len(key) == 0 {
			module = "<empty>"
		} else {
			switch key[0] {
			// Roothash module (0x20-0x29)
			case 0x20:
				module = "roothash/runtime_state"
			case 0x21:
				module = "roothash/params"
			case 0x22:
				module = "roothash/round_timeout"
			case 0x24:
				module = "roothash/evidence"
			case 0x25:
				module = "roothash/state_root"
			case 0x26:
				module = "roothash/io_root"
			case 0x27:
				module = "roothash/last_round_results"
			case 0x28:
				module = "roothash/incoming_msg_queue_meta"
			case 0x29:
				module = "roothash/incoming_msg_queue"
			// Staking module (0x30-0x3F)
			case 0x30:
				module = "staking/total_supply"
			case 0x31:
				module = "staking/common_pool"
			case 0x32:
				module = "staking/last_block_fees"
			case 0x33:
				module = "staking/governance_deposits"
			case 0x34:
				module = "staking/accounts"
			case 0x35:
				module = "staking/delegations"
			case 0x36:
				module = "staking/debonding_delegations"
			case 0x37:
				module = "staking/allowances"
			case 0x38:
				module = "staking/params"
			// Registry module (0x40-0x4F)
			case 0x40:
				module = "registry/entities"
			case 0x41:
				module = "registry/nodes"
			case 0x42:
				module = "registry/node_by_consensus"
			case 0x43:
				module = "registry/runtimes"
			case 0x44:
				module = "registry/suspended_runtimes"
			case 0x45:
				module = "registry/params"
			case 0x46:
				module = "registry/node_status"
			// Scheduler module (0x50-0x5F)
			case 0x50:
				module = "scheduler/params"
			case 0x51:
				module = "scheduler/committees"
			case 0x52:
				module = "scheduler/validators"
			// Governance module (0x60-0x6F)
			case 0x60:
				module = "governance/params"
			case 0x61:
				module = "governance/proposals"
			case 0x62:
				module = "governance/active_proposals"
			case 0x63:
				module = "governance/votes"
			case 0x64:
				module = "governance/pending_upgrades"
			// Beacon module (0x70-0x7F)
			case 0x70:
				module = "beacon/params"
			case 0x71:
				module = "beacon/future_epoch"
			case 0x72:
				module = "beacon/epoch"
			case 0x73:
				module = "beacon/pvss_state"
			// Keymanager module (0x80-0x8F)
			case 0x80:
				module = "keymanager/status"
			case 0x81:
				module = "keymanager/params"
			// Consensus parameters
			case 0xF1:
				module = "consensus/params"
			default:
				if key[0] >= 'a' && key[0] <= 'z' {
					module = extractModuleName(key)
				} else {
					module = fmt.Sprintf("0x%02x", key[0])
				}
			}
		}

		if len(data) < 4 {
			return fmt.Sprintf("LeafNode{module=%s, <no value>}", module)
		}

		valueLen := int(binary.LittleEndian.Uint32(data[0:4]))
		data = data[4:]
		if len(data) < valueLen {
			return fmt.Sprintf("LeafNode{module=%s, valueLen=%d, <truncated>}", module, valueLen)
		}

		leafValue := data[:valueLen]

		// Try CBOR decode
		var decoded interface{}
		if err := cbor.Unmarshal(leafValue, &decoded); err == nil {
			return fmt.Sprintf("LeafNode{module=%s, value=%s}", module, formatCBOR(decoded, valueLen))
		}

		return fmt.Sprintf("LeafNode{module=%s, value=binary(%d bytes)}", module, valueLen)

	case 0x01: // InternalNode: [2-byte labelBits LE][label][leaf/nil marker][hashes]
		data := value[1:]
		if len(data) < 2 {
			return "InternalNode{<malformed>}"
		}

		labelBits := binary.LittleEndian.Uint16(data[0:2])
		data = data[2:]

		labelBytes := (int(labelBits) + 7) / 8
		if len(data) < labelBytes+1 {
			return fmt.Sprintf("InternalNode{label=%d bits, <truncated>}", labelBits)
		}

		data = data[labelBytes:] // skip label

		hasLeaf := data[0] == 0x00
		if hasLeaf {
			return fmt.Sprintf("InternalNode{label=%d bits, has_leaf=true}", labelBits)
		}

		data = data[1:] // skip nil marker

		// Extract child hashes
		var left, right string
		if len(data) >= 32 {
			left = truncateHex(data[:32], 16)
			data = data[32:]
		}
		if len(data) >= 32 {
			right = truncateHex(data[:32], 16)
		}

		return fmt.Sprintf("InternalNode{label=%d bits, left=%s, right=%s}", labelBits, left, right)

	case 0x02: // NilNode
		return "NilNode{}"

	default:
		return fmt.Sprintf("Unknown node prefix 0x%02x (size: %d)", value[0], len(value))
	}
}

// analyzeKeyConsensusState parses consensus-state key and returns key type and decoded representation
func analyzeKeyConsensusState(key []byte) (string, string) {
	if len(key) < 2 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// All keys start with 0x01 (dbVersion from Oasis BadgerDB wrapper)
	if key[0] != 0x01 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// Parse ASCII key after 0x01 prefix
	decoded := string(key[1:])

	// Check if it's printable ASCII
	if !isPrintableASCII(decoded) {
		return "binary", fmt.Sprintf("%x", key)
	}

	// Extract key type prefix before colon
	colonIdx := strings.IndexByte(decoded, ':')
	if colonIdx != -1 {
		prefix := decoded[:colonIdx]
		switch prefix {
		case "abciResponsesKey":
			return "abci_responses", decoded
		case "consensusParamsKey":
			return "consensus_params", decoded
		case "validatorsKey":
			return "validators", decoded
		case "stateKey":
			return "state", decoded
		case "genesisDoc":
			return "genesis", decoded
		default:
			return prefix, decoded
		}
	}

	return "text_key", decoded
}

// decodeValueConsensusState decodes state value
func decodeValueConsensusState(keyType string, value []byte) string {
	if len(value) == 0 {
		return "Empty"
	}
	return fmt.Sprintf("Tendermint %s data (size: %d bytes)", keyType, len(value))
}

// analyzeKeyRuntimeMkvs parses runtime-mkvs key
// Key format: [prefix byte][type byte][data...]
func analyzeKeyRuntimeMkvs(key []byte) (string, string) {
	if len(key) < 1 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// Check for dbVersion prefix (0x01 or 0x05)
	prefixByte := key[0]
	var data []byte

	if prefixByte == 0x01 || prefixByte == 0x05 {
		if len(key) < 2 {
			return "unknown", fmt.Sprintf("%x", key)
		}
		prefixByte = key[1]
		data = key[2:]
	} else {
		data = key[1:]
	}

	switch prefixByte {
	case 0x00:
		return "node", fmt.Sprintf("node{hash: %s}", truncateHex(data, 16))
	case 0x01:
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			return "write_log", fmt.Sprintf("write_log{v:%d}", version)
		}
		return "write_log", "write_log{<malformed>}"
	case 0x02:
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			return "roots_metadata", fmt.Sprintf("roots_metadata{v:%d}", version)
		}
		return "roots_metadata", "roots_metadata{<malformed>}"
	case 0x03:
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			return "root_updated_nodes", fmt.Sprintf("root_updated_nodes{v:%d}", version)
		}
		return "root_updated_nodes", "root_updated_nodes{<malformed>}"
	case 0x04:
		return "metadata", "metadata"
	default:
		return fmt.Sprintf("unknown_%02x", prefixByte), fmt.Sprintf("%x", key)
	}
}

// decodeValueRuntimeMkvs decodes runtime MKVS value with inline node parsing
func decodeValueRuntimeMkvs(keyType string, value []byte) string {
	if len(value) == 0 {
		return "Empty"
	}

	if keyType != "node" {
		return fmt.Sprintf("MKVS %s (size: %d bytes)", keyType, len(value))
	}

	// Parse MKVS node: 0x00=leaf, 0x01=internal, 0x02=nil
	switch value[0] {
	case 0x00: // LeafNode: [2-byte keyLen LE][key][4-byte valueLen LE][value]
		data := value[1:]
		if len(data) < 2 {
			return "LeafNode{<malformed>}"
		}

		keyLen := int(binary.LittleEndian.Uint16(data[0:2]))
		data = data[2:]
		if len(data) < keyLen {
			return fmt.Sprintf("LeafNode{keyLen=%d, <truncated>}", keyLen)
		}

		key := data[:keyLen]
		data = data[keyLen:]

		// Extract module name from key (runtime-specific: evm, contracts, accounts, etc.)
		module := extractModuleName(key)

		if len(data) < 4 {
			return fmt.Sprintf("LeafNode{module=%s, <no value>}", module)
		}

		valueLen := int(binary.LittleEndian.Uint32(data[0:4]))
		data = data[4:]
		if len(data) < valueLen {
			return fmt.Sprintf("LeafNode{module=%s, valueLen=%d, <truncated>}", module, valueLen)
		}

		leafValue := data[:valueLen]

		// Try CBOR decode
		var decoded interface{}
		if err := cbor.Unmarshal(leafValue, &decoded); err == nil {
			return fmt.Sprintf("LeafNode{module=%s, value=%s}", module, formatCBOR(decoded, valueLen))
		}

		return fmt.Sprintf("LeafNode{module=%s, value=binary(%d bytes)}", module, valueLen)

	case 0x01: // InternalNode: [2-byte labelBits LE][label][leaf/nil marker][hashes]
		data := value[1:]
		if len(data) < 2 {
			return "InternalNode{<malformed>}"
		}

		labelBits := binary.LittleEndian.Uint16(data[0:2])
		data = data[2:]

		labelBytes := (int(labelBits) + 7) / 8
		if len(data) < labelBytes+1 {
			return fmt.Sprintf("InternalNode{label=%d bits, <truncated>}", labelBits)
		}

		data = data[labelBytes:] // skip label

		hasLeaf := data[0] == 0x00
		if hasLeaf {
			return fmt.Sprintf("InternalNode{label=%d bits, has_leaf=true}", labelBits)
		}

		data = data[1:] // skip nil marker

		// Extract child hashes
		var left, right string
		if len(data) >= 32 {
			left = truncateHex(data[:32], 16)
			data = data[32:]
		}
		if len(data) >= 32 {
			right = truncateHex(data[:32], 16)
		}

		return fmt.Sprintf("InternalNode{label=%d bits, left=%s, right=%s}", labelBits, left, right)

	case 0x02: // NilNode
		return "NilNode{}"

	default:
		return fmt.Sprintf("Unknown node prefix 0x%02x (size: %d)", value[0], len(value))
	}
}

// extractModuleName extracts ASCII module name from MKVS key
func extractModuleName(key []byte) string {
	if len(key) == 0 {
		return "<empty>"
	}

	// Skip leading 0x00 if present
	if key[0] == 0x00 && len(key) > 1 {
		key = key[1:]
	}

	// Find ASCII module name
	end := 0
	for i, b := range key {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' {
			end = i + 1
		} else {
			break
		}
	}

	if end > 0 {
		return string(key[:end])
	}
	return truncateHex(key, 16)
}

// formatCBOR formats decoded CBOR value concisely
func formatCBOR(v interface{}, rawLen int) string {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, fmt.Sprintf("%v", k))
		}
		if len(keys) > 3 {
			return fmt.Sprintf("map{%s, +%d}", strings.Join(keys[:3], ","), len(keys)-3)
		}
		return fmt.Sprintf("map{%s}", strings.Join(keys, ","))
	case []interface{}:
		return fmt.Sprintf("array[%d]", len(val))
	case []byte:
		return fmt.Sprintf("bytes(%d)", len(val))
	case string:
		if len(val) > 20 {
			return fmt.Sprintf("%q...", val[:20])
		}
		return fmt.Sprintf("%q", val)
	case uint64:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case bool:
		return fmt.Sprintf("%v", val)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T(%d bytes)", v, rawLen)
	}
}

// analyzeKeyRuntimeHistory parses runtime-history key using binary prefix format
// Key formats from oasis-core/go/runtime/history/db.go:
//   - 0x01: metadata key (1 byte)
//   - 0x02 + uint64: block key (9 bytes) - round number in big-endian
//   - 0x03 + uint64: round_results key (9 bytes) - round number in big-endian
func analyzeKeyRuntimeHistory(key []byte) (string, string) {
	if len(key) == 0 {
		return "unknown", ""
	}

	switch key[0] {
	case 0x01:
		// Metadata key - just the prefix byte
		if len(key) == 1 {
			return "metadata", "metadata"
		}
		// Unexpected extra data after metadata prefix
		return "metadata", fmt.Sprintf("metadata (extra: %x)", key[1:])

	case 0x02:
		// Block key - prefix + 8-byte round number
		if len(key) == 9 {
			round := binary.BigEndian.Uint64(key[1:9])
			return "block", fmt.Sprintf("round:%d", round)
		}
		return "block", fmt.Sprintf("block (malformed, len=%d)", len(key))

	case 0x03:
		// Round results key - prefix + 8-byte round number
		if len(key) == 9 {
			round := binary.BigEndian.Uint64(key[1:9])
			return "round_results", fmt.Sprintf("round:%d", round)
		}
		return "round_results", fmt.Sprintf("round_results (malformed, len=%d)", len(key))

	default:
		return "unknown", fmt.Sprintf("%x", key)
	}
}

// decodeValueRuntimeHistory decodes CBOR-encoded runtime history value
func decodeValueRuntimeHistory(keyType string, value []byte) string {
	if len(value) == 0 {
		return "Empty"
	}

	switch keyType {
	case "metadata":
		var meta RuntimeHistoryMetadata
		if err := cbor.Unmarshal(value, &meta); err != nil {
			return fmt.Sprintf("Failed to decode metadata: %v (size: %d)", err, len(value))
		}
		runtimeID := truncateHex(meta.RuntimeID, 16)
		return fmt.Sprintf("Metadata{version: %d, runtime_id: %s..., last_round: %d, last_consensus_height: %d}",
			meta.Version, runtimeID, meta.LastRound, meta.LastConsensusHeight)

	case "block":
		var block RuntimeHistoryAnnotatedBlock
		if err := cbor.Unmarshal(value, &block); err != nil {
			return fmt.Sprintf("Failed to decode block: %v (size: %d)", err, len(value))
		}
		if block.Block == nil {
			return fmt.Sprintf("AnnotatedBlock{consensus_height: %d, block: nil}", block.Height)
		}
		h := block.Block.Header
		// Format timestamp as human-readable
		ts := time.Unix(int64(h.Timestamp), 0).UTC().Format(time.RFC3339)
		headerType := headerTypeName(h.HeaderType)
		stateRoot := truncateHex(h.StateRoot, 16)
		return fmt.Sprintf("AnnotatedBlock{consensus_height: %d, round: %d, timestamp: %s, header_type: %s, state_root: %s...}",
			block.Height, h.Round, ts, headerType, stateRoot)

	case "round_results":
		var results RuntimeHistoryRoundResults
		if err := cbor.Unmarshal(value, &results); err != nil {
			return fmt.Sprintf("Failed to decode round_results: %v (size: %d)", err, len(value))
		}
		return fmt.Sprintf("RoundResults{messages: %d, good_entities: %d, bad_entities: %d}",
			len(results.Messages), len(results.GoodComputeEntities), len(results.BadComputeEntities))

	default:
		return fmt.Sprintf("Runtime %s data (size: %d bytes)", keyType, len(value))
	}
}

// headerTypeName converts header type byte to string
func headerTypeName(headerType uint8) string {
	switch headerType {
	case 0:
		return "Invalid"
	case 1:
		return "Normal"
	case 2:
		return "RoundFailed"
	case 3:
		return "EpochTransition"
	case 4:
		return "Suspended"
	default:
		return fmt.Sprintf("Unknown(%d)", headerType)
	}
}

// isPrintableASCII checks if a string contains only printable ASCII characters
func isPrintableASCII(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < 32 || r > 126 {
			return false
		}
	}
	return true
}

// truncateBytes returns first n bytes or all bytes if shorter
func truncateBytes(data []byte, n int) []byte {
	if len(data) <= n {
		return data
	}
	return data[:n]
}

// truncateHex converts bytes to hex and truncates if too long
func truncateHex(data []byte, maxLen int) string {
	hex := fmt.Sprintf("%x", data)
	if len(hex) > maxLen {
		return hex[:maxLen] + "..."
	}
	return hex
}
