//go:build badgerv2

package main

import (
	"fmt"
	"os"
	"strings"

	badger "github.com/dgraph-io/badger/v2"
)

type (
	DB              = badger.DB
	Txn             = badger.Txn
	Item            = badger.Item
	Iterator        = badger.Iterator
	IteratorOptions = badger.IteratorOptions
)

var DefaultIteratorOptions = badger.DefaultIteratorOptions

func init() {
	BadgerVersion = "v2.2007.2"
}

// openDatabase tries to open the database in ReadOnly mode first,
// then falls back to local mirror mode for corrupted/improperly-closed databases on FUSE
func openDatabase(path string) (*DB, error) {
	// Try ReadOnly mode first (clean databases)
	opts := badger.DefaultOptions(path)
	opts.ReadOnly = true
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err == nil {
		// Successfully opened
		fmt.Fprintf(os.Stderr, "Opened database in ReadOnly mode\n")
		return db, nil
	}

	// Check for version incompatibility
	if strings.Contains(err.Error(), "unsupported version") {
		return nil, fmt.Errorf("database version incompatible with this tool. Try a different badgerdb-sampler version.\nOriginal error: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Failed, trying local mirror workaround for FUSE...\n")

	// Create local mirror with symlinks to avoid mmap SIGBUS errors on FUSE
	tmpDir, cleanup, err := createLocalMirror(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create local mirror: %w", err)
	}
	// Note: cleanup is not called here - tmp files remain in /tmp and will be cleaned by OS
	// This is acceptable for read-only analysis tools. For long-running services, use defer cleanup()
	_ = cleanup

	// Open from local mirror in read-write mode (needed for corrupted/improperly closed DBs)
	// Unified configuration across all BadgerDB versions
	opts = badger.DefaultOptions(tmpDir)
	opts.ReadOnly = false
	opts.SyncWrites = false
	opts.NumMemtables = 1          // Minimize memtable usage
	opts.MemTableSize = 1 << 20    // 1MB memtable (minimal)
	opts.ValueThreshold = 1 << 17  // 128KB - must be less than max batch size (~157KB with 1MB memtable)
	opts.NumLevelZeroTables = 100
	opts.NumLevelZeroTablesStall = 200
	opts.NumCompactors = 0
	opts.CompactL0OnClose = false
	opts.BypassLockGuard = true    // Allow concurrent access
	opts.DetectConflicts = false   // Reduce overhead
	opts.Logger = nil

	db, err = badger.Open(opts)
	if err == nil {
		// Successfully opened
		fmt.Fprintf(os.Stderr, "Opened database from local mirror (tmp dir: %s)\n", tmpDir)
		return db, nil
	}

	// Check again for version incompatibility
	if strings.Contains(err.Error(), "unsupported version") {
		return nil, fmt.Errorf("database version incompatible with this tool. Try a different badgerdb-sampler version.\nOriginal error: %v", err)
	}
	return nil, fmt.Errorf("failed to open database from local mirror: %v", err)
}
