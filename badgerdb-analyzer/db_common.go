package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// createLocalMirror creates a temporary local directory with database files copied/symlinked
// to avoid mmap issues on FUSE filesystems. Returns temp path and cleanup function.
func createLocalMirror(fusePath string) (string, func(), error) {
	// Create temp directory on local filesystem (not FUSE)
	tmpDir, err := os.MkdirTemp("", "badgerdb-temp-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	cleanup := func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to clean up temp directory %s: %v\n", tmpDir, err)
		}
	}

	fmt.Fprintf(os.Stderr, "  Creating local mirror in: %s\n", tmpDir)

	entries, err := os.ReadDir(fusePath)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to read database directory: %w", err)
	}

	copiedCount := 0
	linkedCount := 0
	skippedCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		srcPath := filepath.Join(fusePath, name)
		dstPath := filepath.Join(tmpDir, name)

		// Skip memtable files - these will be created fresh
		if strings.HasSuffix(name, ".mem") {
			fmt.Fprintf(os.Stderr, "  Skipping memtable file: %s\n", name)
			skippedCount++
			continue
		}

		// Copy small metadata files (MANIFEST, DISCARD, etc.)
		// Symlink large data files (*.sst, *.vlog)
		if strings.HasSuffix(name, ".sst") || strings.HasSuffix(name, ".vlog") {
			// Symlink large data files
			if err := os.Symlink(srcPath, dstPath); err != nil {
				cleanup()
				return "", nil, fmt.Errorf("failed to symlink %s: %w", name, err)
			}
			linkedCount++
		} else {
			// Copy small metadata files
			if err := copyFile(srcPath, dstPath); err != nil {
				cleanup()
				return "", nil, fmt.Errorf("failed to copy %s: %w", name, err)
			}
			copiedCount++
		}
	}

	fmt.Fprintf(os.Stderr, "  Mirror created: %d files copied, %d files symlinked, %d files skipped\n",
		copiedCount, linkedCount, skippedCount)

	return tmpDir, cleanup, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}

// removeMemtableFiles renames .mem files by appending .bak suffix and returns a restore function.
// This allows opening databases with corrupted memtables by letting BadgerDB create fresh memtable files.
func removeMemtableFiles(dbPath string) (backupDir string, restore func() error, err error) {
	entries, err := os.ReadDir(dbPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read database directory: %w", err)
	}

	// Rename all .mem files to .mem.bak
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".mem") {
			oldPath := filepath.Join(dbPath, entry.Name())
			newPath := oldPath + ".bak"
			if err := os.Rename(oldPath, newPath); err != nil {
				return "", nil, fmt.Errorf("failed to rename %s: %w", entry.Name(), err)
			}
			count++
		}
	}

	fmt.Fprintf(os.Stderr, "  Renamed %d memtable file(s) to *.mem.bak\n", count)

	// Restore function renames .mem.bak files back to .mem
	restore = func() error {
		entries, err := os.ReadDir(dbPath)
		if err != nil {
			return fmt.Errorf("failed to read database directory: %w", err)
		}
		count := 0
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".mem.bak") {
				oldPath := filepath.Join(dbPath, entry.Name())
				newPath := strings.TrimSuffix(oldPath, ".bak")
				if err := os.Rename(oldPath, newPath); err != nil {
					return fmt.Errorf("failed to restore %s: %w", entry.Name(), err)
				}
				count++
			}
		}
		fmt.Fprintf(os.Stderr, "  Restored %d memtable file(s) from *.mem.bak\n", count)
		return nil
	}

	return dbPath, restore, nil
}
