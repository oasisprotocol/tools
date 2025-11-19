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
	maxSamples := 200 // default
	if len(os.Args) >= 4 {
		jsonFile = os.Args[3]
	}
	if len(os.Args) >= 5 {
		var err error
		maxSamples, err = strconv.Atoi(os.Args[4])
		if err != nil || maxSamples <= 0 {
			log.Fatalf("Invalid max-samples value: %s (must be positive integer)", os.Args[5])
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
			log.Fprintf(os.Stderr, "\nWarning: Failed to create directory %s: %v", jsonDir, err)
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
		opts.PrefetchSize = maxSamples
		it := txn.NewIterator(opts)
		defer it.Close()

		sampleCount := 0

		for it.Rewind(); it.Valid(); it.Next() {
			if sampleCount >= maxSamples {
				break
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

// decodeValueConsensusMkvs decodes MKVS value
func decodeValueConsensusMkvs(keyType string, value []byte) string {
	if len(value) == 0 {
		return "Empty"
	}

	switch keyType {
	case "node":
		return fmt.Sprintf("Serialized node (size: %d bytes)", len(value))
	case "write_log", "roots_metadata", "root_updated_nodes", "metadata":
		return fmt.Sprintf("CBOR-encoded %s (size: %d bytes)", keyType, len(value))
	default:
		return fmt.Sprintf("MKVS %s data (size: %d bytes)", keyType, len(value))
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

// analyzeKeyRuntimeMkvs parses runtime-mkvs key (same format as consensus-mkvs)
func analyzeKeyRuntimeMkvs(key []byte) (string, string) {
	return analyzeKeyConsensusMkvs(key)
}

// decodeValueRuntimeMkvs decodes runtime MKVS value (same format as consensus MKVS)
func decodeValueRuntimeMkvs(keyType string, value []byte) string {
	return decodeValueConsensusMkvs(keyType, value)
}

// analyzeKeyRuntimeHistory parses runtime-history key (same format as consensus-state)
func analyzeKeyRuntimeHistory(key []byte) (string, string) {
	return analyzeKeyConsensusState(key)
}

// decodeValueRuntimeHistory decodes runtime history value
func decodeValueRuntimeHistory(keyType string, value []byte) string {
	if len(value) == 0 {
		return "Empty"
	}
	return fmt.Sprintf("Runtime %s data (size: %d bytes)", keyType, len(value))
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
