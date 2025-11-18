//go:build badgerv3

package main

import (
	"fmt"
	"os"
	"strings"

	badger "github.com/dgraph-io/badger/v3"
)

type (
	DB               = badger.DB
	Txn              = badger.Txn
	Item             = badger.Item
	Iterator         = badger.Iterator
	IteratorOptions  = badger.IteratorOptions
)

var DefaultIteratorOptions = badger.DefaultIteratorOptions

func init() {
	BadgerVersion = "v3.2103.5"
}

// openDatabase tries to open the database in ReadOnly mode first,
// then falls back to FileIO mode for corrupted/improperly-closed databases
func openDatabase(path string) (*DB, error) {
	// Try ReadOnly mode first (clean databases)
	opts := badger.DefaultOptions(path)
	opts.ReadOnly = true
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err == nil {
		fmt.Fprintf(os.Stderr, "Opened database in ReadOnly mode\n")
		return db, nil
	}

	// Check for BadgerDB v2/v4 incompatibility
	if strings.Contains(err.Error(), "manifest has unsupported version") ||
		strings.Contains(err.Error(), "unsupported version") {
		return nil, fmt.Errorf("database may use BadgerDB v2 or v4 (incompatible). Please use badgerdb-sampler-v2 or badgerdb-sampler-v4 instead.\nOriginal error: %v", err)
	}

	fmt.Fprintf(os.Stderr, "ReadOnly mode failed, trying FileIO workaround...\n")

	// FileIO workaround mode - for improperly closed databases
	opts = badger.DefaultOptions(path)
	opts.ReadOnly = false
	opts.SyncWrites = false
	opts.NumMemtables = 5
	opts.NumLevelZeroTables = 100
	opts.NumLevelZeroTablesStall = 200
	opts.NumCompactors = 0
	opts.CompactL0OnClose = false
	// Note: ValueLogLoadingMode and TableLoadingMode were removed in BadgerDB v3
	opts.Logger = nil

	db, err = badger.Open(opts)
	if err != nil {
		// Check again for v2/v4 incompatibility in FileIO mode
		if strings.Contains(err.Error(), "manifest has unsupported version") ||
			strings.Contains(err.Error(), "unsupported version") {
			return nil, fmt.Errorf("database may use BadgerDB v2 or v4 (incompatible). Please use badgerdb-sampler-v2 or badgerdb-sampler-v4 instead.\nOriginal error: %v", err)
		}
		return nil, fmt.Errorf("failed to open database with both ReadOnly and FileIO modes: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Opened database in FileIO mode\n")
	return db, nil
}
