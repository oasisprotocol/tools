package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BadgerVersion is set by build tags
var BadgerVersion string

func main() {
	// Parse arguments
	dbType, dbPath, outputPath, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		printUsage()
		os.Exit(3) // Invalid args
	}

	// Calculate database directory size
	dbSize, err := getDatabaseSize(dbPath)
	if err != nil {
		logProgress("Warning: Could not calculate database size: %v", err)
		dbSize = 0
	}

	// Log initial info
	logProgress("Database: %s (type: %s, size: %s)", dbPath, dbType, formatBytes(dbSize))
	logProgress("Opening database...")

	// Open database with multi-stage strategy (ReadOnly → Minimal RW → Local Mirror)
	db, err := openDatabase(dbPath)
	if err != nil {
		logProgress("Failed to open database: %v", err)
		os.Exit(2) // Open failed
	}
	defer db.Close()

	// Analyze database
	logProgress("Starting analysis...")
	result, err := analyzeDatabase(db, dbType, dbPath, BadgerVersion, dbSize)
	if err != nil {
		logProgress("Analysis failed: %v", err)
		os.Exit(4) // Error
	}

	// Output JSON
	if err := outputJSON(result, outputPath); err != nil {
		logProgress("Failed to output JSON: %v", err)
		os.Exit(4) // Error
	}

	// Determine exit code based on consistency checks
	exitCode := 0
	if result.Consistency != nil {
		for _, check := range result.Consistency {
			if !check.Passed {
				exitCode = 1 // Issues found
				break
			}
		}
	}

	logProgress("Analysis complete")
	os.Exit(exitCode)
}

// parseArgs parses command-line arguments
func parseArgs() (dbType, dbPath, outputPath string, err error) {
	if len(os.Args) < 3 {
		return "", "", "", fmt.Errorf("insufficient arguments")
	}

	dbType = normalizeDBType(os.Args[1])
	dbPath = os.Args[2]

	// Validate database type
	validTypes := []string{
		"consensus-blockstore",
		"consensus-evidence",
		"consensus-mkvs",
		"consensus-state",
		"runtime-mkvs",
		"runtime-history",
	}
	valid := false
	for _, validType := range validTypes {
		if dbType == validType {
			valid = true
			break
		}
	}
	if !valid {
		return "", "", "", fmt.Errorf("invalid database type: %s (valid types: %s)",
			dbType, strings.Join(validTypes, ", "))
	}

	// Optional output path
	if len(os.Args) > 3 {
		outputPath = os.Args[3]
	}

	return dbType, dbPath, outputPath, nil
}

// normalizeDBType removes version suffix from database type
func normalizeDBType(dbType string) string {
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

// outputJSON outputs the analysis result as JSON to stdout or file
func outputJSON(result *AnalysisResult, outputPath string) error {
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if outputPath == "" {
		// Output to stdout
		fmt.Println(string(output))
	} else {
		// Create directory if needed
		dir := filepath.Dir(outputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Write to file
		if err := os.WriteFile(outputPath, output, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", outputPath, err)
		}

		logProgress("Results saved to: %s", outputPath)
	}

	return nil
}

// logProgress logs a progress message to stderr
func logProgress(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// printUsage prints usage information
func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <database-type> <database-path> [output-json]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nDatabase types:\n")
	fmt.Fprintf(os.Stderr, "  consensus-blockstore - Block metadata and commit info\n")
	fmt.Fprintf(os.Stderr, "  consensus-evidence   - Byzantine validator evidence\n")
	fmt.Fprintf(os.Stderr, "  consensus-mkvs       - Consensus state Merkle tree\n")
	fmt.Fprintf(os.Stderr, "  consensus-state      - Tendermint consensus state\n")
	fmt.Fprintf(os.Stderr, "  runtime-mkvs         - Runtime state Merkle tree\n")
	fmt.Fprintf(os.Stderr, "  runtime-history      - Runtime block history\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  %s consensus-blockstore /path/to/blockstore.badger.db\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s runtime-history /path/to/history.badger.db analysis.json\n", os.Args[0])
}

// formatBytes formats a byte count as a human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
