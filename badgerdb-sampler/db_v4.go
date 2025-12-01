//go:build badgerv4

package main

import (
	"fmt"
	"os"
	"strings"

	badger "github.com/dgraph-io/badger/v4"
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
	BadgerVersion = "v4.x.x"
}

// openDatabase tries three strategies in order:
// 1. ReadOnly mode (cleanly closed databases)
// 2. Minimal read-write mode (allows log replay on original path)
// 3. Local mirror mode (FUSE workaround with mmap-safe copy)
func openDatabase(path string) (*DB, error) {
	// Stage 1: Try read-only mode first (clean databases)
	fmt.Fprintf(os.Stderr, "Opening in read-only mode...\n")
	opts := badger.DefaultOptions(path)
	opts.ReadOnly = true
	// Keep default logger enabled for diagnostics

	db, err := badger.Open(opts)
	if err == nil {
		fmt.Fprintf(os.Stderr, "Successfully opened in read-only mode\n")
		return db, nil
	}

	// Check for version incompatibility
	if strings.Contains(err.Error(), "unsupported version") {
		return nil, fmt.Errorf("database version incompatible with this tool. Try a different badgerdb-sampler version.\nOriginal error: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Failed to open in read-only mode: %v\n", err)

	// Stage 2: Try minimal read-write mode (allows log replay)
	fmt.Fprintf(os.Stderr, "Opening in minimal read-write mode (allows log replay)...\n")
	opts = badger.DefaultOptions(path)
	opts.ReadOnly = false
	opts.SyncWrites = false
	opts.NumMemtables = 1          // Minimize memtable usage
	opts.NumLevelZeroTables = 100
	opts.NumLevelZeroTablesStall = 200
	opts.NumCompactors = 0
	opts.CompactL0OnClose = false
	opts.BypassLockGuard = true    // Allow concurrent access
	opts.DetectConflicts = false   // Reduce overhead
	// Keep default logger enabled for diagnostics

	db, err = badger.Open(opts)
	if err == nil {
		fmt.Fprintf(os.Stderr, "Successfully opened in minimal read-write mode\n")
		return db, nil
	}

	// Check again for version incompatibility
	if strings.Contains(err.Error(), "unsupported version") {
		return nil, fmt.Errorf("database version incompatible with this tool. Try a different badgerdb-sampler version.\nOriginal error: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Failed to open in minimal read-write mode: %v\n", err)

	// Stage 3: Try removing corrupted memtable files and retry
	fmt.Fprintf(os.Stderr, "Opening without memtable files...\n")
	if backupDir, restoreMemFiles, err := removeMemtableFiles(path); backupDir != "" {
		// Memtable files were removed, retry opening
		opts = badger.DefaultOptions(path)
		opts.ReadOnly = false
		opts.SyncWrites = false
		opts.NumMemtables = 1          // Minimize memtable usage
		opts.NumLevelZeroTables = 100
		opts.NumLevelZeroTablesStall = 200
		opts.NumCompactors = 0
		opts.CompactL0OnClose = false
		opts.BypassLockGuard = true    // Allow concurrent access
		opts.DetectConflicts = false   // Reduce overhead
		// Keep default logger enabled for diagnostics

		db, err = badger.Open(opts)
		if err == nil {
			fmt.Fprintf(os.Stderr, "Successfully opened without memtable files\n")
			return db, nil
		}

		// Check for version incompatibility
		if strings.Contains(err.Error(), "unsupported version") {
			return nil, fmt.Errorf("database version incompatible with this tool. Try a different badgerdb-sampler version.\nOriginal error: %v", err)
		}

		// Restore memtable files before continuing
		fmt.Fprintf(os.Stderr, "Restoring memtable files...\n")
		if restoreErr := restoreMemFiles(); restoreErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to restore memtable files: %v\n", restoreErr)
		} else {
			fmt.Fprintf(os.Stderr, "Memtable files restored\n")
		}
	}

	fmt.Fprintf(os.Stderr, "Failed to open without memtable files: %v\n", err)

	// Stage 4: Try local mirror workaround (FUSE compatibility)
	// BadgerDB v4 uses mmap for memtables similar to v3, which doesn't work on FUSE filesystems
	fmt.Fprintf(os.Stderr, "Opening with local mirror (workaround for SIGBUS error on FUSE)...\n")

	// Create local mirror with symlinks to avoid mmap SIGBUS errors on FUSE
	tmpDir, cleanup, err := createLocalMirror(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create local mirror: %w", err)
	}
	// Note: cleanup is not called here - tmp files remain in /tmp and will be cleaned by OS
	// This is acceptable for read-only analysis tools. For long-running services, use defer cleanup()
	_ = cleanup

	// Open from local mirror with same minimal read-write settings
	opts = badger.DefaultOptions(tmpDir)
	opts.ReadOnly = false
	opts.SyncWrites = false
	opts.NumMemtables = 1          // Minimize memtable usage
	opts.NumLevelZeroTables = 100
	opts.NumLevelZeroTablesStall = 200
	opts.NumCompactors = 0
	opts.CompactL0OnClose = false
	opts.BypassLockGuard = true    // Allow concurrent access
	opts.DetectConflicts = false   // Reduce overhead
	// Keep default logger enabled for diagnostics

	db, err = badger.Open(opts)
	if err == nil {
		fmt.Fprintf(os.Stderr, "Successfully opened with local mirror (tmp dir: %s)\n", tmpDir)
		return db, nil
	}

	// Check again for version incompatibility
	if strings.Contains(err.Error(), "unsupported version") {
		return nil, fmt.Errorf("database version incompatible with this tool. Try a different badgerdb-sampler version.\nOriginal error: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Failed to open with local mirror (tmp dir: %s): %v\n", tmpDir, err)
	return nil, fmt.Errorf("failure: all opening strategies failed")
}
