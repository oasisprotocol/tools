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

	badger "github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
	"github.com/gogo/protobuf/proto"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmstore "github.com/tendermint/tendermint/proto/tendermint/store"
)

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
		fmt.Fprintf(os.Stderr, "Usage: %s <database-type> <path-to-db> [output-subdir] [output-prefix] [max-samples]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s consensus-blockstore-v2 /path/to/blockstore.badger.db\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s runtime-mkvs-v2 /path/to/mkvs_storage.badger.db testnet-20220303 emerald 100\n", os.Args[0])
		os.Exit(1)
	}

	dbType := os.Args[1]
	dbPath := os.Args[2]
	outputSubdir := ""
	outputPrefix := ""
	maxSamples := 200 // default
	if len(os.Args) >= 4 {
		outputSubdir = os.Args[3]
	}
	if len(os.Args) >= 5 {
		outputPrefix = os.Args[4]
	}
	if len(os.Args) >= 6 {
		var err error
		maxSamples, err = strconv.Atoi(os.Args[5])
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
	db, err := openDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	stats := DBStats{
		DatabasePath:    dbPath,
		DatabaseType:    dbType,
		BadgerDBVersion: "v2.2007.2",
		DatabaseSize:    dbSize,
		KeyTypeCounts:   make(map[string]int),
		Samples:         []Sample{},
	}

	// Collect samples
	err = collectSamples(db, &stats, maxSamples)
	if err != nil {
		log.Fatalf("Error collecting samples: %v", err)
	}

	// Output JSON
	output, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
	}

	fmt.Println(string(output))

	// Save results in JSON files in per-snapshot dir and per-database type with optional prefix
	if outputSubdir != "" {
		outputDir := filepath.Join("/workspace/sampler-outputs", outputSubdir)
		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			log.Printf("Warning: Could not create directory %s: %v", outputDir, err)
		} else {
			filename := outputPrefix + dbType + ".json"
			outputFile := filepath.Join(outputDir, filename)
			err = os.WriteFile(outputFile, output, 0644)
			if err != nil {
				log.Printf("Warning: Could not write to %s: %v", outputFile, err)
			} else {
				fmt.Fprintf(os.Stderr, "\nResults saved to: %s\n", outputFile)
			}
		}
	}
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

// openDatabase tries to open the database in ReadOnly mode first,
// then falls back to FileIO mode for corrupted/improperly-closed databases
func openDatabase(path string) (*badger.DB, error) {
	// Try ReadOnly mode first (clean databases)
	opts := badger.DefaultOptions(path)
	opts.ReadOnly = true
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err == nil {
		fmt.Fprintf(os.Stderr, "Opened database in ReadOnly mode\n")
		return db, nil
	}

	// Check for BadgerDB v3 incompatibility
	if strings.Contains(err.Error(), "manifest has unsupported version") {
		return nil, fmt.Errorf("database uses BadgerDB v3 (incompatible). Please use sampler-v3 instead.\nOriginal error: %v", err)
	}

	fmt.Fprintf(os.Stderr, "ReadOnly mode failed, trying FileIO workaround...\n")

	// FileIO workaround mode - for improperly closed databases
	opts = badger.DefaultOptions(path)
	opts.ReadOnly = false                      // Must use read-write for improperly closed DB
	opts.SyncWrites = false                    // Don't sync writes
	opts.NumMemtables = 5                      // More memtables to avoid flushing
	opts.NumLevelZeroTables = 100              // High threshold to avoid compaction
	opts.NumLevelZeroTablesStall = 200         // Very high stall threshold
	opts.NumCompactors = 0                     // Disable background compaction
	opts.CompactL0OnClose = false              // Don't compact on close
	opts.ValueLogLoadingMode = options.FileIO  // Use FileIO instead of mmap
	opts.TableLoadingMode = options.FileIO     // Use FileIO instead of mmap
	opts.Logger = nil

	db, err = badger.Open(opts)
	if err != nil {
		// Check again for v3 incompatibility in FileIO mode
		if strings.Contains(err.Error(), "manifest has unsupported version") {
			return nil, fmt.Errorf("database uses BadgerDB v3 (incompatible). Please use sampler-v3 instead.\nOriginal error: %v", err)
		}
		return nil, fmt.Errorf("failed to open database with both ReadOnly and FileIO modes: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Opened database in FileIO mode\n")
	return db, nil
}

// collectSamples iterates through keys and collects samples with full decoding
func collectSamples(db *badger.DB, stats *DBStats, maxSamples int) error {
	return db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true // Fetch values for sampling
		opts.PrefetchSize = maxSamples // Only prefetch what we need
		it := txn.NewIterator(opts)
		defer it.Close()

		sampleCount := 0

		for it.Rewind(); it.Valid(); it.Next() {
			// Only collect samples up to maxSamples
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
			case "consensus-blockstore-v2":
				keyType, decodedKey := analyzeKeyConsensusBlockstoreV2(key)
				sample.KeyType = keyType
				sample.DecodedKey = decodedKey
			case "consensus-evidence-v2":
				keyType, decodedKey := analyzeKeyConsensusEvidenceV2(key)
				sample.KeyType = keyType
				sample.DecodedKey = decodedKey
			case "consensus-mkvs-v2":
				keyType, decodedKey := analyzeKeyConsensusMkvsV2(key)
				sample.KeyType = keyType
				sample.DecodedKey = decodedKey
			case "consensus-state-v2":
				keyType, decodedKey := analyzeKeyConsensusStateV2(key)
				sample.KeyType = keyType
				sample.DecodedKey = decodedKey
			case "runtime-mkvs-v2":
				keyType, decodedKey := analyzeKeyRuntimeMkvsV2(key)
				sample.KeyType = keyType
				sample.DecodedKey = decodedKey
			case "runtime-history-v2":
				keyType, decodedKey := analyzeKeyRuntimeHistoryV2(key)
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
				case "consensus-blockstore-v2":
					decoded, ts := decodeValueConsensusBlockstoreV2(sample.KeyType, val)
					sample.DecodedValue = decoded
					sample.Timestamp = ts
				case "consensus-evidence-v2":
					sample.DecodedValue = decodeValueConsensusEvidenceV2(sample.KeyType, val)
				case "consensus-mkvs-v2":
					sample.DecodedValue = decodeValueConsensusMkvsV2(sample.KeyType, val)
				case "consensus-state-v2":
					sample.DecodedValue = decodeValueConsensusStateV2(sample.KeyType, val)
				case "runtime-mkvs-v2":
					sample.DecodedValue = decodeValueRuntimeMkvsV2(sample.KeyType, val)
				case "runtime-history-v2":
					sample.DecodedValue = decodeValueRuntimeHistoryV2(sample.KeyType, val)
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

// analyzeKeyConsensusBlockstoreV2 parses consensus-blockstore-v2 key and returns key type and decoded representation
func analyzeKeyConsensusBlockstoreV2(key []byte) (string, string) {
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

// decodeValueConsensusBlockstoreV2 attempts to decode protobuf value based on key type
// Returns: (decoded_string, timestamp_unix)
func decodeValueConsensusBlockstoreV2(keyType string, value []byte) (string, int64) {
	switch keyType {
	case "blockstore_state":
		var state tmstore.BlockStoreState
		if err := proto.Unmarshal(value, &state); err == nil {
			return fmt.Sprintf("BlockStoreState{base: %d, height: %d}", state.Base, state.Height), 0
		}
		return fmt.Sprintf("Failed to decode BlockStoreState (size: %d bytes)", len(value)), 0

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
		return fmt.Sprintf("Failed to decode BlockMeta (size: %d bytes)", len(value)), 0

	case "block_part":
		var part tmproto.Part
		if err := proto.Unmarshal(value, &part); err == nil {
			return fmt.Sprintf("Part{index: %d, bytes_size: %d, proof_total: %d}",
				part.Index, len(part.Bytes), part.Proof.Total), 0
		}
		return fmt.Sprintf("Failed to decode Part (size: %d bytes)", len(value)), 0

	case "block_commit":
		var commit tmproto.Commit
		if err := proto.Unmarshal(value, &commit); err == nil {
			// Commits don't have a direct timestamp field; timestamp comes from block header
			return fmt.Sprintf("Commit{height: %d, round: %d, signatures: %d}",
				commit.Height, commit.Round, len(commit.Signatures)), 0
		}
		return fmt.Sprintf("Failed to decode Commit (size: %d bytes)", len(value)), 0

	case "seen_commit":
		var commit tmproto.Commit
		if err := proto.Unmarshal(value, &commit); err == nil {
			// Commits don't have a direct timestamp field; timestamp comes from block header
			return fmt.Sprintf("SeenCommit{height: %d, round: %d, signatures: %d}",
				commit.Height, commit.Round, len(commit.Signatures)), 0
		}
		return fmt.Sprintf("Failed to decode SeenCommit (size: %d bytes)", len(value)), 0

	case "block_hash":
		// Block hash values are plain strings containing height numbers
		height, err := strconv.ParseInt(string(value), 10, 64)
		if err == nil {
			return fmt.Sprintf("Height: %d", height), 0
		}
		return fmt.Sprintf("String: %s", string(value)), 0

	default:
		return fmt.Sprintf("Unknown type (size: %d bytes)", len(value)), 0
	}
}

// analyzeKeyConsensusEvidenceV2 parses consensus-evidence-v2 key and returns key type and decoded representation
func analyzeKeyConsensusEvidenceV2(key []byte) (string, string) {
	if len(key) < 2 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// All keys start with 0x01 (dbVersion from Oasis BadgerDB wrapper)
	if key[0] != 0x01 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// Evidence DB is typically empty or uses simple key patterns
	// Return the hex representation for now
	return fmt.Sprintf("type_%02x", key[1]), fmt.Sprintf("%x", key[1:])
}

// decodeValueConsensusEvidenceV2 attempts to decode evidence value
func decodeValueConsensusEvidenceV2(keyType string, value []byte) string {
	// Evidence values are protobuf-encoded DuplicateVoteEvidence or LightClientAttackEvidence
	// For now, just show size as evidence DB is typically empty
	if len(value) == 0 {
		return "Empty"
	}

	// Try to decode as DuplicateVoteEvidence
	var evidence tmproto.DuplicateVoteEvidence
	if err := proto.Unmarshal(value, &evidence); err == nil {
		return fmt.Sprintf("DuplicateVoteEvidence{vote_a_height: %d, vote_b_height: %d}",
			evidence.VoteA.Height, evidence.VoteB.Height)
	}

	return fmt.Sprintf("Evidence (size: %d bytes, type: %s)", len(value), keyType)
}

// analyzeKeyConsensusMkvsV2 parses consensus-mkvs-v2 key and returns key type and decoded representation
// MKVS keys: 0xNN (type prefix) + type-specific data (NO dbVersion prefix)
// Key formats from oasis-core/go/storage/mkvs/db/badger/badger.go:
//   0x00: node (hash)
//   0x01: writeLog (version uint64, new root, old root)
//   0x02: rootsMetadata (version uint64)
//   0x03: rootUpdatedNodes (version uint64, root)
//   0x04: metadata
//   0x05: multipartRestoreNodeLog (hash)
//   0x06: rootNode (typed hash)
func analyzeKeyConsensusMkvsV2(key []byte) (string, string) {
	if len(key) < 1 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// MKVS keys start directly with the type prefix byte (0x00-0x06)
	// NO dbVersion prefix for MKVS storage
	prefixByte := key[0]
	data := key[1:]

	switch prefixByte {
	case 0x00:
		// nodeKeyFmt: hash.Hash (32 bytes)
		if len(data) >= 32 {
			return "node", fmt.Sprintf("node:hash=%x", data[:32])
		}
		return "node", fmt.Sprintf("node:hash=%x", data)
	case 0x01:
		// writeLogKeyFmt: uint64(version) + TypedHash(new root) + TypedHash(old root)
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			if len(data) >= 41 {
				return "write_log", fmt.Sprintf("write_log:v=%d,new_root=%x,old_root=%x",
					version, truncateBytes(data[8:41], 8), truncateBytes(data[41:], 8))
			}
			return "write_log", fmt.Sprintf("write_log:v=%d", version)
		}
		return "write_log", "write_log:<malformed>"
	case 0x02:
		// rootsMetadataKeyFmt: uint64(version)
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			return "roots_metadata", fmt.Sprintf("roots_metadata:v=%d", version)
		}
		return "roots_metadata", "roots_metadata:<malformed>"
	case 0x03:
		// rootUpdatedNodesKeyFmt: uint64(version) + TypedHash(root)
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			if len(data) >= 41 {
				return "root_updated_nodes", fmt.Sprintf("root_updated_nodes:v=%d,root=%x",
					version, truncateBytes(data[8:41], 8))
			}
			return "root_updated_nodes", fmt.Sprintf("root_updated_nodes:v=%d", version)
		}
		return "root_updated_nodes", "root_updated_nodes:<malformed>"
	case 0x04:
		// metadataKeyFmt: no additional data
		return "metadata", "metadata"
	case 0x05:
		// multipartRestoreNodeLogKeyFmt: TypedHash (33 bytes: type byte + 32-byte hash)
		if len(data) >= 33 {
			return "multipart_restore_log", fmt.Sprintf("multipart_restore_log:hash=%x", data[:33])
		}
		return "multipart_restore_log", fmt.Sprintf("multipart_restore_log:hash=%x", data)
	case 0x06:
		// rootNodeKeyFmt: TypedHash (33 bytes: type byte + 32-byte hash)
		if len(data) >= 33 {
			return "root_node", fmt.Sprintf("root_node:hash=%x", data[:33])
		}
		return "root_node", fmt.Sprintf("root_node:hash=%x", data)
	default:
		return fmt.Sprintf("unknown_%02x", prefixByte), fmt.Sprintf("unknown_%02x:%x", prefixByte, truncateBytes(data, 8))
	}
}

// decodeValueConsensusMkvsV2 decodes MKVS value
// Value types:
//   - node: Serialized node data (binary)
//   - write_log: CBOR-serialized write log
//   - roots_metadata: CBOR-serialized rootsMetadata
//   - root_updated_nodes: CBOR-serialized []updatedNode
//   - metadata: CBOR-serialized metadata
func decodeValueConsensusMkvsV2(keyType string, value []byte) string {
	if len(value) == 0 {
		return "Empty"
	}

	// For CBOR-encoded values, we could decode but it's complex
	// Just report size and encoding type
	switch keyType {
	case "node":
		return fmt.Sprintf("Serialized node (size: %d bytes)", len(value))
	case "write_log", "roots_metadata", "root_updated_nodes", "metadata":
		return fmt.Sprintf("CBOR-encoded %s (size: %d bytes)", keyType, len(value))
	default:
		return fmt.Sprintf("MKVS %s data (size: %d bytes)", keyType, len(value))
	}
}

// analyzeKeyConsensusStateV2 parses consensus-state-v2 key and returns key type and decoded representation
// State keys are ASCII strings: 0x01 + "<key_name>:<height>"
func analyzeKeyConsensusStateV2(key []byte) (string, string) {
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

// decodeValueConsensusStateV2 decodes state value (protobuf-encoded Tendermint state data)
func decodeValueConsensusStateV2(keyType string, value []byte) string {
	// State values are protobuf-encoded, but the exact message type depends on the key
	// For simplicity, just show size and type info
	if len(value) == 0 {
		return "Empty"
	}

	return fmt.Sprintf("Tendermint %s data (size: %d bytes)", keyType, len(value))
}

// analyzeKeyRuntimeMkvsV2 parses runtime-mkvs-v2 key and returns key type and decoded representation
// MKVS keys: 0xNN (type prefix) + type-specific data (NO dbVersion prefix)
// Same format as consensus-mkvs-v2 from oasis-core/go/storage/mkvs/db/badger/badger.go
func analyzeKeyRuntimeMkvsV2(key []byte) (string, string) {
	if len(key) < 1 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// MKVS keys start directly with the type prefix byte (0x00-0x06)
	// NO dbVersion prefix for MKVS storage
	prefixByte := key[0]
	data := key[1:]

	switch prefixByte {
	case 0x00:
		// nodeKeyFmt: hash.Hash (32 bytes)
		if len(data) >= 32 {
			return "node", fmt.Sprintf("node:hash=%x", data[:32])
		}
		return "node", fmt.Sprintf("node:hash=%x", data)
	case 0x01:
		// writeLogKeyFmt: uint64(version) + TypedHash(new root) + TypedHash(old root)
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			if len(data) >= 41 {
				return "write_log", fmt.Sprintf("write_log:v=%d,new_root=%x,old_root=%x",
					version, truncateBytes(data[8:41], 8), truncateBytes(data[41:], 8))
			}
			return "write_log", fmt.Sprintf("write_log:v=%d", version)
		}
		return "write_log", "write_log:<malformed>"
	case 0x02:
		// rootsMetadataKeyFmt: uint64(version)
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			return "roots_metadata", fmt.Sprintf("roots_metadata:v=%d", version)
		}
		return "roots_metadata", "roots_metadata:<malformed>"
	case 0x03:
		// rootUpdatedNodesKeyFmt: uint64(version) + TypedHash(root)
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			if len(data) >= 41 {
				return "root_updated_nodes", fmt.Sprintf("root_updated_nodes:v=%d,root=%x",
					version, truncateBytes(data[8:41], 8))
			}
			return "root_updated_nodes", fmt.Sprintf("root_updated_nodes:v=%d", version)
		}
		return "root_updated_nodes", "root_updated_nodes:<malformed>"
	case 0x04:
		// metadataKeyFmt: no additional data
		return "metadata", "metadata"
	case 0x05:
		// multipartRestoreNodeLogKeyFmt: TypedHash (33 bytes: type byte + 32-byte hash)
		if len(data) >= 33 {
			return "multipart_restore_log", fmt.Sprintf("multipart_restore_log:hash=%x", data[:33])
		}
		return "multipart_restore_log", fmt.Sprintf("multipart_restore_log:hash=%x", data)
	case 0x06:
		// rootNodeKeyFmt: TypedHash (33 bytes: type byte + 32-byte hash)
		if len(data) >= 33 {
			return "root_node", fmt.Sprintf("root_node:hash=%x", data[:33])
		}
		return "root_node", fmt.Sprintf("root_node:hash=%x", data)
	default:
		return fmt.Sprintf("unknown_%02x", prefixByte), fmt.Sprintf("unknown_%02x:%x", prefixByte, truncateBytes(data, 8))
	}
}

// decodeValueRuntimeMkvsV2 decodes runtime MKVS value
// Same format as consensus MKVS (CBOR-encoded for most types)
func decodeValueRuntimeMkvsV2(keyType string, value []byte) string {
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

// analyzeKeyRuntimeHistoryV2 parses runtime-history-v2 key and returns key type and decoded representation
// History keys are ASCII strings: 0x01 + "<key_name>:<height>"
func analyzeKeyRuntimeHistoryV2(key []byte) (string, string) {
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

// decodeValueRuntimeHistoryV2 decodes runtime history value
func decodeValueRuntimeHistoryV2(keyType string, value []byte) string {
	// Runtime history values are similar to consensus state
	// For simplicity, just show size and type info
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
