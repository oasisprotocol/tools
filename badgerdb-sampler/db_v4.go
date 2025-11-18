//go:build badgerv4

package main

import (
	"fmt"
	"os"
	"strings"

	badger "github.com/dgraph-io/badger/v4"
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
	BadgerVersion = "v4.x.x"
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

	// Check for BadgerDB v2/v3 incompatibility
	if strings.Contains(err.Error(), "manifest has unsupported version") ||
		strings.Contains(err.Error(), "unsupported version") {
		return nil, fmt.Errorf("database may use BadgerDB v2 or v3 (incompatible). Please use badgerdb-sampler-v2 or badgerdb-sampler-v3 instead.\nOriginal error: %v", err)
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
	opts.Logger = nil

	db, err = badger.Open(opts)
	if err != nil {
		// Check again for v2/v3 incompatibility in FileIO mode
		if strings.Contains(err.Error(), "manifest has unsupported version") ||
			strings.Contains(err.Error(), "unsupported version") {
			return nil, fmt.Errorf("database may use BadgerDB v2 or v3 (incompatible). Please use badgerdb-sampler-v2 or badgerdb-sampler-v3 instead.\nOriginal error: %v", err)
		}
		return nil, fmt.Errorf("failed to open database with both ReadOnly and FileIO modes: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Opened database in FileIO mode\n")
	return db, nil
}
