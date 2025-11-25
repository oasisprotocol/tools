package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// BadgerVersion is set by build tags
var BadgerVersion string

// Sample contains both raw and decoded key/value information
type Sample struct {
	KeyRaw    string      `json:"key_raw"`
	KeySize   int         `json:"key_size"`
	KeyType   string      `json:"key_type"`
	Key       interface{} `json:"key"`                   // Decoded key (renamed from Key)
	KeyError  string      `json:"key_error,omitempty"`   // Error decoding key
	ValueRaw  string      `json:"value_raw"`
	ValueSize int         `json:"value_size"`
	Value     interface{} `json:"value"`                 // Decoded value (renamed from Value)
	ValueError string     `json:"value_error,omitempty"` // Error decoding value
	Timestamp int64       `json:"timestamp,omitempty"`
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
		opts.PrefetchSize = 100  // Optimal prefetch size for BadgerDB
		opts.AllVersions = false // Only latest version of each key
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
				KeyRaw:     truncateHex(key, TruncateLongSize),
				KeySize:    len(key),
				KeyType:    "unknown",
				Key: nil,
			}

			// Decode key based on database type
			switch stats.DatabaseType {
			case "consensus-blockstore":
				keyInfo := decodeConsensusBlockstoreKey(key)
				sample.KeyType = keyInfo.KeyType
				sample.Key = keyInfo
			case "consensus-evidence":
				keyInfo := decodeConsensusEvidenceKey(key)
				sample.KeyType = keyInfo.KeyType
				sample.Key = keyInfo
			case "consensus-mkvs":
				keyInfo := decodeConsensusMkvsKey(key)
				sample.KeyType = keyInfo.KeyType
				sample.Key = keyInfo
			case "consensus-state":
				keyInfo := decodeConsensusStateKey(key)
				sample.KeyType = keyInfo.KeyType
				sample.Key = keyInfo
			case "runtime-mkvs":
				keyInfo := decodeRuntimeMkvsKey(key)
				sample.KeyType = keyInfo.KeyType
				sample.Key = keyInfo
			case "runtime-history":
				keyInfo := decodeRuntimeHistoryKey(key)
				sample.KeyType = keyInfo.KeyType
				sample.Key = keyInfo
			}

			// Fetch and decode value
			err := item.Value(func(val []byte) error {
				sample.ValueSize = len(val)
				sample.ValueRaw = truncateHex(val, TruncateLongSize)
				sample.Value = nil

				// Decode value based on database type
				switch stats.DatabaseType {
				case "consensus-blockstore":
					valueInfo := decodeConsensusBlockstoreValue(sample.KeyType, val)
					sample.Value = valueInfo
					sample.Timestamp = valueInfo.Timestamp
				case "consensus-evidence":
					value, err := decodeConsensusEvidenceValue(sample.KeyType, val)
					sample.Value = value
					sample.ValueError = err
				case "consensus-mkvs":
					sample.Value = decodeConsensusMkvsValue(sample.KeyType, val)
				case "consensus-state":
					sample.Value = decodeConsensusStateValue(sample.KeyType, val)
				case "runtime-mkvs":
					sample.Value = decodeRuntimeMkvsValue(sample.KeyType, val)
				case "runtime-history":
					sample.Value = decodeRuntimeHistoryValue(sample.KeyType, val)
				}
				return nil
			})
			if err != nil {
				log.Printf("Error reading value for key %s: %v", sample.KeyRaw, err)
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
