package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/fxamacker/cbor/v2"
)

// AnalysisResult contains all analysis results for a database
type AnalysisResult struct {
	// Database metadata
	DatabasePath    string `json:"database_path"`
	DatabaseType    string `json:"database_type"`
	BadgerDBVersion string `json:"badgerdb_version"`
	DatabaseSize    int64  `json:"database_size"`

	// Statistics
	Keys            KeyStats             `json:"keys"`
	Values          ValueStats           `json:"values"`
	ConsensusBlocks *ConsensusBlockStats `json:"consensus_blocks,omitempty"`
	RuntimeBlocks   *RuntimeBlockStats   `json:"runtime_blocks,omitempty"`
	MKVSStats       *MKVSStats           `json:"mkvs_stats,omitempty"`
	ABCIAnalysis    *ABCIAnalysisStats   `json:"abci_analysis,omitempty"`

	// Quality and checks
	Consistency []CheckResult  `json:"consistency_checks"`
	ErrorCounts map[string]int `json:"error_counts"`
	Anomalies   []string       `json:"anomalies,omitempty"`
}

// ConsensusBlockStats contains statistics for consensus blockstore
type ConsensusBlockStats struct {
	OldestConsensusHeight int64                       `json:"oldest_consensus_height"`
	NewestConsensusHeight int64                       `json:"newest_consensus_height"`
	TotalBlocks           int64                       `json:"total_blocks"`
	IsComplete            bool                        `json:"is_complete"`
	MissingBlocks         []int64                     `json:"missing_blocks,omitempty"`
	OldestTimestamp       string                      `json:"oldest_timestamp,omitempty"` // RFC3339
	NewestTimestamp       string                      `json:"newest_timestamp,omitempty"` // RFC3339
	IntervalStats         *SizeStats                  `json:"interval_stats,omitempty"`
	TransactionStats      *ConsensusTransactionStats  `json:"transaction_stats,omitempty"`
	EventStats            *ConsensusEventStats        `json:"event_stats,omitempty"`
}

// ConsensusTransactionStats contains transaction statistics for consensus blocks
type ConsensusTransactionStats struct {
	Total            int64              `json:"total"`
	PerBlockStats    *SizeStats         `json:"per_block_stats,omitempty"`
	SizeStats        *SizeStats         `json:"size_stats,omitempty"`
	TypeDistribution map[string]int64   `json:"type_distribution"`
}

// ConsensusEventStats contains event statistics for consensus blocks
type ConsensusEventStats struct {
	Total            int64              `json:"total"`
	PerBlockStats    *SizeStats         `json:"per_block_stats,omitempty"`
	TypeDistribution map[string]int64   `json:"type_distribution"`
}

// RuntimeBlockStats contains statistics for runtime history
type RuntimeBlockStats struct {
	OldestConsensusHeight int64                    `json:"oldest_consensus_height"`
	NewestConsensusHeight int64                    `json:"newest_consensus_height"`
	OldestRuntimeHeight   int64                    `json:"oldest_runtime_height"`
	NewestRuntimeHeight   int64                    `json:"newest_runtime_height"`
	OldestTimestamp       string                   `json:"oldest_timestamp,omitempty"`
	NewestTimestamp       string                   `json:"newest_timestamp,omitempty"`
	IsComplete            bool                     `json:"is_complete"`
	TotalBlocks           int64                    `json:"total_blocks"`
	IntervalStats         *SizeStats               `json:"interval_stats,omitempty"`
	TransactionStats      *RuntimeTransactionStats `json:"transaction_stats,omitempty"`
	EventStats            *RuntimeEventStats       `json:"event_stats,omitempty"`
}

// RuntimeTransactionStats contains transaction statistics for runtime blocks
type RuntimeTransactionStats struct {
	Total               int64              `json:"total"`
	EVMTransactions     int64              `json:"evm_transactions"`
	PerBlockStats       *SizeStats         `json:"per_block_stats,omitempty"`
	EVMTypeDistribution map[string]int64   `json:"evm_type_distribution"`
	EVMSizeStats        *SizeStats         `json:"evm_size_stats,omitempty"`
}

// RuntimeEventStats contains event statistics for runtime blocks
type RuntimeEventStats struct {
	Total             int64              `json:"total"`
	EVMEvents         int64              `json:"evm_events"`
	PerBlockStats     *SizeStats         `json:"per_block_stats,omitempty"`
	RuntimeEventTypes map[string]int64   `json:"runtime_event_types"`
	EVMSignatures     map[string]int64   `json:"evm_signatures"`
}

// MKVSStats contains statistics for MKVS databases
type MKVSStats struct {
	NodeTypeDistribution map[string]int64 `json:"node_type_distribution"`
	ModuleDistribution   map[string]int64 `json:"module_distribution,omitempty"`
	TotalModule          int64            `json:"total_module,omitempty"`
	TotalNode            int64            `json:"total_node"`
	WriteLogStats        *WriteLogStats   `json:"write_log_stats,omitempty"`
}

// WriteLogStats contains statistics about available write_log entries
type WriteLogStats struct {
	OldestVersion uint64  `json:"oldest_version"`
	NewestVersion uint64  `json:"newest_version"`
	TotalEntries  int64   `json:"total_entries"`
	VersionRange  uint64  `json:"version_range"`    // NewestVersion - OldestVersion + 1
	Coverage      float64 `json:"coverage_percent"` // (TotalEntries / VersionRange) * 100
}

// ABCIAnalysisStats contains aggregated statistics across all ABCI responses
type ABCIAnalysisStats struct {
	TxMethodCounts  map[string]int64 `json:"tx_method_counts"`
	EventTypeCounts map[string]int64 `json:"event_type_counts"`
	TotalTx         int64            `json:"total_tx"`
	TotalEvent      int64            `json:"total_event"`
}

// SizeStats contains statistical information about sizes
type SizeStats struct {
	Min    int64   `json:"min"`
	Max    int64   `json:"max"`
	Avg    float64 `json:"avg"`
	Total  int64   `json:"total"`
	StdDev float64 `json:"std_dev"`
}

// KeyStats contains statistics about keys
type KeyStats struct {
	TotalCount       int64          `json:"total_count"`
	TypeDistribution map[string]int `json:"type_distribution"`
	SizeStats        SizeStats      `json:"size_stats"`
}

// ValueStats contains statistics about values
type ValueStats struct {
	SizeStats SizeStats `json:"size_stats"`
}

// CheckResult contains the result of a consistency check
type CheckResult struct {
	Name   string   `json:"name"`
	Passed bool     `json:"passed"`
	Issues []string `json:"issues,omitempty"`
}

// SizeStatsAccumulator accumulates size statistics using Welford's online algorithm
type SizeStatsAccumulator struct {
	count int64
	min   int64
	max   int64
	sum   int64
	mean  float64
	m2    float64 // For variance calculation
}

// Add adds a new value to the accumulator
func (s *SizeStatsAccumulator) Add(value int64) {
	s.count++
	s.sum += value

	if s.count == 1 {
		s.min = value
		s.max = value
		s.mean = float64(value)
		s.m2 = 0
	} else {
		if value < s.min {
			s.min = value
		}
		if value > s.max {
			s.max = value
		}

		// Welford's online algorithm for variance
		delta := float64(value) - s.mean
		s.mean += delta / float64(s.count)
		delta2 := float64(value) - s.mean
		s.m2 += delta * delta2
	}
}

// Finalize returns the final SizeStats
func (s *SizeStatsAccumulator) Finalize() SizeStats {
	if s.count == 0 {
		return SizeStats{}
	}

	stdDev := 0.0
	if s.count > 1 {
		variance := s.m2 / float64(s.count)
		stdDev = math.Sqrt(variance)
	}

	return SizeStats{
		Min:    s.min,
		Max:    s.max,
		Avg:    s.mean,
		Total:  s.sum,
		StdDev: stdDev,
	}
}

// consensusBlockstoreCollector collects data for consensus-blockstore analysis
type consensusBlockstoreCollector struct {
	keyTypes   map[string]int
	heights    []int64
	timestamps []time.Time
	blocks     map[int64]bool
	commits    map[int64]bool
	seenCommits map[int64]bool
}

// runtimeHistoryCollector collects data for runtime-history analysis
type runtimeHistoryCollector struct {
	keyTypes          map[string]int
	heights           []int64
	timestamps        []time.Time
	consensusHeights  []int64
}

// consensusMkvsCollector collects data for consensus-mkvs analysis
type consensusMkvsCollector struct {
	keyTypes         map[string]int
	nodeTypes        map[string]int64
	modules          map[string]int64
	writeLogVersions []uint64 // For version completeness checks
	totalLeaves      int64
	totalInternal    int64

	// Consensus-specific height-based aggregation
	leafCountsByHeight  map[int64]int // height -> leaf count
	txCountsByHeight    map[int64]int // height -> tx count
	eventCountsByHeight map[int64]int // height -> event count
}

// runtimeMkvsCollector collects data for runtime-mkvs analysis
type runtimeMkvsCollector struct {
	keyTypes         map[string]int
	nodeTypes        map[string]int64
	modules          map[string]int64
	writeLogVersions []uint64 // For version completeness checks
	totalLeaves      int64
	totalInternal    int64

	// Per-height transaction/event aggregation
	txCountsByHeight       map[uint64]int            // height -> total tx count
	evmTxCountsByHeight    map[uint64]int            // height -> EVM tx count
	evmTxTypesByHeight     map[uint64]map[string]int // height -> type -> count
	eventCountsByHeight    map[uint64]int            // height -> total event count
	evmEventCountsByHeight map[uint64]int            // height -> EVM event count
	evmEventSigsByHeight   map[uint64]map[string]int // height -> signature -> count
	runtimeEventsByHeight  map[uint64]map[string]int // height -> event type -> count
}

// consensusEvidenceCollector collects data for consensus-evidence analysis
type consensusEvidenceCollector struct {
	keyTypes         map[string]int
	consensusHeights []int64
}

// consensusStateCollector collects data for consensus-state analysis
type consensusStateCollector struct {
	keyTypes            map[string]int
	abciResponseHeights []int64

	// Per-height aggregated data (NOT full objects)
	txCountsByHeight     map[int64]int            // height -> tx count
	txMethodsByHeight    map[int64]map[string]int // height -> method -> count
	eventTypesByHeight   map[int64]map[string]int // height -> event type -> count
}

// analyzeDatabase performs full analysis of a database
func analyzeDatabase(db *DB, dbType string, dbPath string, dbVersion string, dbSize int64) (*AnalysisResult, error) {
	// Initialize result
	result := &AnalysisResult{
		DatabasePath:    dbPath,
		DatabaseType:    dbType,
		BadgerDBVersion: dbVersion,
		DatabaseSize:    dbSize,
		Keys: KeyStats{
			TotalCount:       0,
			TypeDistribution: make(map[string]int),
			SizeStats:        SizeStats{},
		},
		Values: ValueStats{
			SizeStats: SizeStats{},
		},
		ErrorCounts: make(map[string]int),
		Consistency: []CheckResult{},
		Anomalies:   []string{},
	}

	// Initialize statistics accumulators
	keySizeStats := &SizeStatsAccumulator{}
	valueSizeStats := &SizeStatsAccumulator{}

	// Database-specific collectors
	var consensusBlockstoreData *consensusBlockstoreCollector
	var runtimeHistoryData *runtimeHistoryCollector
	var consensusMkvsData *consensusMkvsCollector
	var runtimeMkvsData *runtimeMkvsCollector
	var consensusEvidenceData *consensusEvidenceCollector
	var consensusStateData *consensusStateCollector

	switch dbType {
	case "consensus-blockstore":
		consensusBlockstoreData = &consensusBlockstoreCollector{
			keyTypes:    make(map[string]int),
			heights:     []int64{},
			timestamps:  []time.Time{},
			blocks:      make(map[int64]bool),
			commits:     make(map[int64]bool),
			seenCommits: make(map[int64]bool),
		}
	case "runtime-history":
		runtimeHistoryData = &runtimeHistoryCollector{
			keyTypes:         make(map[string]int),
			heights:          []int64{},
			timestamps:       []time.Time{},
			consensusHeights: []int64{},
		}
	case "consensus-mkvs":
		consensusMkvsData = &consensusMkvsCollector{
			keyTypes:            make(map[string]int),
			nodeTypes:           make(map[string]int64),
			modules:             make(map[string]int64),
			writeLogVersions:    []uint64{},
			totalLeaves:         0,
			totalInternal:       0,
			leafCountsByHeight:  make(map[int64]int),
			txCountsByHeight:    make(map[int64]int),
			eventCountsByHeight: make(map[int64]int),
		}
	case "runtime-mkvs":
		runtimeMkvsData = &runtimeMkvsCollector{
			keyTypes:               make(map[string]int),
			nodeTypes:              make(map[string]int64),
			modules:                make(map[string]int64),
			writeLogVersions:       []uint64{},
			totalLeaves:            0,
			totalInternal:          0,
			txCountsByHeight:       make(map[uint64]int),
			evmTxCountsByHeight:    make(map[uint64]int),
			evmTxTypesByHeight:     make(map[uint64]map[string]int),
			eventCountsByHeight:    make(map[uint64]int),
			evmEventCountsByHeight: make(map[uint64]int),
			evmEventSigsByHeight:   make(map[uint64]map[string]int),
			runtimeEventsByHeight:  make(map[uint64]map[string]int),
		}
	case "consensus-evidence":
		consensusEvidenceData = &consensusEvidenceCollector{
			keyTypes:         make(map[string]int),
			consensusHeights: []int64{},
		}
	case "consensus-state":
		consensusStateData = &consensusStateCollector{
			keyTypes:            make(map[string]int),
			abciResponseHeights: []int64{},
			txCountsByHeight:    make(map[int64]int),
			txMethodsByHeight:   make(map[int64]map[string]int),
			eventTypesByHeight:  make(map[int64]map[string]int),
		}
	}

	// Single-pass iteration
	logProgress("Phase 1: Collecting statistics...")
	keyCount := int64(0)

	err := db.View(func(txn *Txn) error {
		opts := DefaultIteratorOptions
		opts.PrefetchValues = true
		opts.PrefetchSize = 100
		opts.AllVersions = false
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()
			keySizeStats.Add(int64(len(key)))

			value, _ := item.ValueCopy(nil)
			if value != nil {
				valueSizeStats.Add(int64(len(value)))
			}

			// Process based on database type
			switch dbType {
			case "consensus-blockstore":
				processConsensusBlockstoreEntry(key, value, consensusBlockstoreData, result.ErrorCounts)
			case "runtime-history":
				processRuntimeHistoryEntry(key, value, runtimeHistoryData, result.ErrorCounts)
			case "consensus-mkvs":
				processConsensusMkvsEntry(key, value, consensusMkvsData, result.ErrorCounts)
			case "runtime-mkvs":
				processRuntimeMkvsEntry(key, value, runtimeMkvsData, result.ErrorCounts)
			case "consensus-evidence":
				processConsensusEvidenceEntry(key, value, consensusEvidenceData, result.ErrorCounts)
			case "consensus-state":
				processConsensusStateEntry(key, value, consensusStateData, result.ErrorCounts)
			}

			keyCount++
			if keyCount%100000 == 0 {
				logProgress("  Processed %d keys...", keyCount)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Finalize statistics
	result.Keys.TotalCount = keyCount
	result.Keys.SizeStats = keySizeStats.Finalize()
	result.Values.SizeStats = valueSizeStats.Finalize()

	logProgress("Phase 2: Building statistics and running checks...")

	// Build database-specific stats and checks
	switch dbType {
	case "consensus-blockstore":
		result.ConsensusBlocks = buildConsensusBlockStats(consensusBlockstoreData)
		result.Consistency = runConsensusBlockstoreChecks(consensusBlockstoreData)
		result.Keys.TypeDistribution = consensusBlockstoreData.keyTypes
	case "runtime-history":
		result.RuntimeBlocks = buildRuntimeBlockStats(runtimeHistoryData)
		result.Consistency = runRuntimeHistoryChecks(runtimeHistoryData)
		result.Keys.TypeDistribution = runtimeHistoryData.keyTypes
	case "consensus-mkvs":
		result.MKVSStats = buildConsensusMkvsStats(consensusMkvsData)
		result.Consistency = runConsensusMkvsChecks(consensusMkvsData)
		result.Keys.TypeDistribution = consensusMkvsData.keyTypes
	case "runtime-mkvs":
		result.MKVSStats = buildRuntimeMkvsStats(runtimeMkvsData)
		// Populate transaction/event stats if RuntimeBlockStats exists
		if result.RuntimeBlocks != nil {
			populateRuntimeTransactionEventStats(result.RuntimeBlocks, runtimeMkvsData)
		}
		result.Consistency = runRuntimeMkvsChecks(runtimeMkvsData)
		result.Keys.TypeDistribution = runtimeMkvsData.keyTypes
	case "consensus-evidence":
		result.Consistency = runConsensusEvidenceChecks(consensusEvidenceData)
		result.Keys.TypeDistribution = consensusEvidenceData.keyTypes
	case "consensus-state":
		result.ABCIAnalysis = buildABCIAnalysisStats(consensusStateData)
		result.Consistency = runConsensusStateChecks(consensusStateData)
		result.Keys.TypeDistribution = consensusStateData.keyTypes
	}

	return result, nil
}

// processConsensusBlockstoreEntry processes a single consensus-blockstore entry
func processConsensusBlockstoreEntry(key, value []byte, collector *consensusBlockstoreCollector, errorCounts map[string]int) {
	keyInfo := decodeConsensusBlockstoreKey(key)
	collector.keyTypes[keyInfo.KeyType]++

	if value == nil {
		return
	}

	valueInfo := decodeConsensusBlockstoreValue(keyInfo.KeyType, value)

	// Extract heights and timestamps from block metadata
	if keyInfo.KeyType == "block_meta" {
		if keyInfo.ConsensusHeight > 0 {
			collector.heights = append(collector.heights, keyInfo.ConsensusHeight)
			collector.blocks[keyInfo.ConsensusHeight] = true
		}
		if valueInfo.Timestamp > 0 {
			collector.timestamps = append(collector.timestamps, time.Unix(valueInfo.Timestamp, 0))
		}
	}

	// Track commits
	if keyInfo.KeyType == "block_commit" {
		if keyInfo.ConsensusHeight > 0 {
			collector.commits[keyInfo.ConsensusHeight] = true
		}
	}

	// Track seen commits
	if keyInfo.KeyType == "seen_commit" {
		if keyInfo.ConsensusHeight > 0 {
			collector.seenCommits[keyInfo.ConsensusHeight] = true
		}
	}

	// Count decode errors
	if valueInfo.RawError != "" {
		errorCounts[valueInfo.RawError]++
	}
}

// buildConsensusBlockStats builds consensus block statistics
func buildConsensusBlockStats(collector *consensusBlockstoreCollector) *ConsensusBlockStats {
	if len(collector.heights) == 0 {
		return &ConsensusBlockStats{}
	}

	// Sort heights
	sort.Slice(collector.heights, func(i, j int) bool {
		return collector.heights[i] < collector.heights[j]
	})

	// Sort timestamps
	sort.Slice(collector.timestamps, func(i, j int) bool {
		return collector.timestamps[i].Before(collector.timestamps[j])
	})

	stats := &ConsensusBlockStats{
		OldestConsensusHeight: collector.heights[0],
		NewestConsensusHeight: collector.heights[len(collector.heights)-1],
		TotalBlocks:           int64(len(collector.heights)),
	}

	// Check if complete
	expectedBlocks := stats.NewestConsensusHeight - stats.OldestConsensusHeight + 1
	stats.IsComplete = (stats.TotalBlocks == expectedBlocks)

	// Find missing blocks
	if !stats.IsComplete {
		heightSet := make(map[int64]bool)
		for _, h := range collector.heights {
			heightSet[h] = true
		}
		for h := stats.OldestConsensusHeight; h <= stats.NewestConsensusHeight; h++ {
			if !heightSet[h] {
				stats.MissingBlocks = append(stats.MissingBlocks, h)
			}
		}
	}

	// Timestamps
	if len(collector.timestamps) > 0 {
		stats.OldestTimestamp = collector.timestamps[0].Format(time.RFC3339)
		stats.NewestTimestamp = collector.timestamps[len(collector.timestamps)-1].Format(time.RFC3339)

		// Calculate interval statistics
		if len(collector.timestamps) > 1 {
			intervalStats := &SizeStatsAccumulator{}
			for i := 1; i < len(collector.timestamps); i++ {
				interval := collector.timestamps[i].Sub(collector.timestamps[i-1]).Milliseconds()
				if interval >= 0 {
					intervalStats.Add(interval)
				}
			}
			finalStats := intervalStats.Finalize()
			stats.IntervalStats = &finalStats
		}
	}

	return stats
}

// runConsensusBlockstoreChecks runs all consistency checks for consensus-blockstore
func runConsensusBlockstoreChecks(collector *consensusBlockstoreCollector) []CheckResult {
	checks := []CheckResult{}

	// block_sequence_complete
	checks = append(checks, checkBlockSequence(collector.heights))

	// timestamp_monotonicity
	checks = append(checks, checkTimestampMonotonicity(collector.timestamps))

	// block_interval_anomalies
	checks = append(checks, checkBlockIntervalAnomalies(collector.heights, collector.timestamps))

	// referential_integrity
	var oldestHeight, newestHeight int64
	if len(collector.heights) > 0 {
		oldestHeight = collector.heights[0]
		newestHeight = collector.heights[len(collector.heights)-1]
	}
	checks = append(checks, checkReferentialIntegrity(collector.blocks, collector.commits, oldestHeight, newestHeight))

	// block_size_anomalies (placeholder for now)
	checks = append(checks, CheckResult{Name: "block_size_anomalies", Passed: true})

	return checks
}

// checkBlockSequence checks if blocks are sequential with no gaps
func checkBlockSequence(heights []int64) CheckResult {
	result := CheckResult{
		Name:   "block_sequence_complete",
		Passed: true,
	}

	if len(heights) == 0 {
		return result
	}

	// Heights should already be sorted
	minHeight := heights[0]
	maxHeight := heights[len(heights)-1]
	expected := maxHeight - minHeight + 1

	if int64(len(heights)) != expected {
		result.Passed = false

		// Find missing blocks
		heightSet := make(map[int64]bool)
		for _, h := range heights {
			heightSet[h] = true
		}

		missingCount := 0
		for h := minHeight; h <= maxHeight && missingCount < 100; h++ {
			if !heightSet[h] {
				result.Issues = append(result.Issues, fmt.Sprintf("Missing block at consensus height: %d", h))
				missingCount++
			}
		}
		if expected-int64(len(heights)) > 100 {
			result.Issues = append(result.Issues, fmt.Sprintf("... and %d more missing blocks", expected-int64(len(heights))-100))
		}
	}

	return result
}

// checkTimestampMonotonicity checks if timestamps increase monotonically
func checkTimestampMonotonicity(timestamps []time.Time) CheckResult {
	result := CheckResult{
		Name:   "timestamp_monotonicity",
		Passed: true,
	}

	if len(timestamps) < 2 {
		return result
	}

	issueCount := 0
	for i := 1; i < len(timestamps) && issueCount < 10; i++ {
		if timestamps[i].Before(timestamps[i-1]) {
			result.Passed = false
			result.Issues = append(result.Issues,
				fmt.Sprintf("Timestamp at index %d (%s) is before index %d (%s)",
					i, timestamps[i].Format(time.RFC3339),
					i-1, timestamps[i-1].Format(time.RFC3339)))
			issueCount++
		}
	}

	return result
}

// checkBlockIntervalAnomalies checks if block intervals are within 3σ of mean
func checkBlockIntervalAnomalies(heights []int64, timestamps []time.Time) CheckResult {
	result := CheckResult{
		Name:   "block_interval_anomalies",
		Passed: true,
	}

	if len(timestamps) < 3 || len(heights) != len(timestamps) {
		return result
	}

	// Calculate intervals
	intervals := make([]float64, 0, len(timestamps)-1)
	for i := 1; i < len(timestamps); i++ {
		interval := timestamps[i].Sub(timestamps[i-1]).Seconds()
		if interval >= 0 {
			intervals = append(intervals, interval)
		}
	}

	if len(intervals) < 3 {
		return result
	}

	// Calculate mean and stddev
	sum := 0.0
	for _, v := range intervals {
		sum += v
	}
	mean := sum / float64(len(intervals))

	variance := 0.0
	for _, v := range intervals {
		variance += (v - mean) * (v - mean)
	}
	stddev := math.Sqrt(variance / float64(len(intervals)))

	// Find anomalies (more than 3σ from mean)
	threshold := 3.0 * stddev
	issueCount := 0
	for i, interval := range intervals {
		if issueCount >= 10 {
			break
		}
		if math.Abs(interval-mean) > threshold {
			result.Passed = false
			// Report consensus heights where the interval spans (intervals[i] is between heights[i] and heights[i+1])
			result.Issues = append(result.Issues,
				fmt.Sprintf("Anomalous interval at consensus heights %d-%d: %.1fs",
					heights[i], heights[i+1], interval))
			issueCount++
		}
	}

	return result
}

// checkReferentialIntegrity checks if every block has a commit and vice versa
func checkReferentialIntegrity(blocks, commits map[int64]bool, oldestHeight, newestHeight int64) CheckResult {
	result := CheckResult{
		Name:   "referential_integrity",
		Passed: true,
	}

	// Check if every block has a commit
	for height := range blocks {
		if !commits[height] {
			// Skip expected edge case: newest block may not have commit yet
			if height == newestHeight {
				result.Issues = append(result.Issues,
					fmt.Sprintf("Block at consensus height %d has no commit (expected: last height)", height))
				continue
			}
			result.Passed = false
			result.Issues = append(result.Issues,
				fmt.Sprintf("Block at consensus height %d has no commit", height))
			if len(result.Issues) >= 10 {
				break
			}
		}
	}

	// Check if every commit has a block
	for height := range commits {
		if !blocks[height] {
			// Skip expected edge case: oldest commit may reference previous block
			if height == oldestHeight-1 {
				result.Issues = append(result.Issues,
					fmt.Sprintf("Commit at consensus height %d has no block (expected: first height-1)", height))
				continue
			}
			result.Passed = false
			result.Issues = append(result.Issues,
				fmt.Sprintf("Commit at consensus height %d has no block", height))
			if len(result.Issues) >= 10 {
				break
			}
		}
	}

	return result
}

// processRuntimeHistoryEntry processes a single runtime-history entry
func processRuntimeHistoryEntry(key, value []byte, collector *runtimeHistoryCollector, errorCounts map[string]int) {
	keyInfo := decodeRuntimeHistoryKey(key)
	collector.keyTypes[keyInfo.KeyType]++

	if value == nil {
		return
	}

	// Only process block entries for height/timestamp/consensus height extraction
	if keyInfo.KeyType != "block" {
		return
	}

	valueInfo := decodeRuntimeHistoryValue(keyInfo.KeyType, value)

	// Count decode errors
	if valueInfo.RawError != "" {
		errorCounts[valueInfo.RawError]++
		return
	}

	// Extract data from block info
	if valueInfo.Block != nil && !valueInfo.Block.BlockNil {
		// Add runtime height
		if valueInfo.Block.RuntimeHeight > 0 {
			collector.heights = append(collector.heights, int64(valueInfo.Block.RuntimeHeight))
		}

		// Add consensus height
		if valueInfo.Block.ConsensusHeight > 0 {
			collector.consensusHeights = append(collector.consensusHeights, valueInfo.Block.ConsensusHeight)
		}

		// Add timestamp
		if valueInfo.Block.Timestamp != "" {
			ts, err := time.Parse(time.RFC3339, valueInfo.Block.Timestamp)
			if err == nil {
				collector.timestamps = append(collector.timestamps, ts)
			} else {
				errorCounts["timestamp_parse_error"]++
			}
		}
	}
}

func buildRuntimeBlockStats(collector *runtimeHistoryCollector) *RuntimeBlockStats {
	if len(collector.heights) == 0 {
		return &RuntimeBlockStats{}
	}

	// Sort runtime heights
	sort.Slice(collector.heights, func(i, j int) bool {
		return collector.heights[i] < collector.heights[j]
	})

	// Sort consensus heights
	sort.Slice(collector.consensusHeights, func(i, j int) bool {
		return collector.consensusHeights[i] < collector.consensusHeights[j]
	})

	// Sort timestamps
	sort.Slice(collector.timestamps, func(i, j int) bool {
		return collector.timestamps[i].Before(collector.timestamps[j])
	})

	stats := &RuntimeBlockStats{
		OldestRuntimeHeight: collector.heights[0],
		NewestRuntimeHeight: collector.heights[len(collector.heights)-1],
		TotalBlocks:         int64(len(collector.heights)),
	}

	// Set consensus heights
	if len(collector.consensusHeights) > 0 {
		stats.OldestConsensusHeight = collector.consensusHeights[0]
		stats.NewestConsensusHeight = collector.consensusHeights[len(collector.consensusHeights)-1]
	}

	// Check if runtime block sequence is complete
	expectedBlocks := stats.NewestRuntimeHeight - stats.OldestRuntimeHeight + 1
	stats.IsComplete = (stats.TotalBlocks == expectedBlocks)

	// Timestamps
	if len(collector.timestamps) > 0 {
		stats.OldestTimestamp = collector.timestamps[0].Format(time.RFC3339)
		stats.NewestTimestamp = collector.timestamps[len(collector.timestamps)-1].Format(time.RFC3339)

		// Calculate interval statistics
		if len(collector.timestamps) > 1 {
			intervalStats := &SizeStatsAccumulator{}
			for i := 1; i < len(collector.timestamps); i++ {
				interval := collector.timestamps[i].Sub(collector.timestamps[i-1]).Milliseconds()
				if interval >= 0 {
					intervalStats.Add(interval)
				}
			}
			finalStats := intervalStats.Finalize()
			stats.IntervalStats = &finalStats
		}
	}

	// Note: Transaction and event stats would be populated from runtime-mkvs analysis
	// For now, these remain nil as runtime-history only contains block metadata

	return stats
}

// runRuntimeHistoryChecks runs all consistency checks for runtime-history
func runRuntimeHistoryChecks(collector *runtimeHistoryCollector) []CheckResult {
	checks := []CheckResult{}

	// runtime_block_sequence_complete
	checks = append(checks, checkRuntimeBlockSequence(collector.heights))

	// consensus_height_mapping
	checks = append(checks, checkConsensusHeightMapping(collector.consensusHeights, collector.heights))

	// timestamp_monotonicity
	checks = append(checks, checkTimestampMonotonicity(collector.timestamps))

	return checks
}

// checkRuntimeBlockSequence checks if runtime blocks are sequential with no gaps
func checkRuntimeBlockSequence(heights []int64) CheckResult {
	result := CheckResult{
		Name:   "runtime_block_sequence_complete",
		Passed: true,
	}

	if len(heights) == 0 {
		return result
	}

	// Heights should already be sorted
	minHeight := heights[0]
	maxHeight := heights[len(heights)-1]
	expected := maxHeight - minHeight + 1

	if int64(len(heights)) != expected {
		result.Passed = false

		// Find missing blocks
		heightSet := make(map[int64]bool)
		for _, h := range heights {
			heightSet[h] = true
		}

		missingCount := 0
		for h := minHeight; h <= maxHeight && missingCount < 100; h++ {
			if !heightSet[h] {
				result.Issues = append(result.Issues, fmt.Sprintf("Missing runtime block at height: %d", h))
				missingCount++
			}
		}
		if expected-int64(len(heights)) > 100 {
			result.Issues = append(result.Issues, fmt.Sprintf("... and %d more missing blocks", expected-int64(len(heights))-100))
		}
	}

	return result
}

// checkConsensusHeightMapping checks if consensus height mapping is valid
func checkConsensusHeightMapping(consensusHeights []int64, runtimeHeights []int64) CheckResult {
	result := CheckResult{
		Name:   "consensus_height_mapping",
		Passed: true,
	}

	if len(consensusHeights) == 0 {
		return result
	}

	// Check for monotonicity in consensus heights
	issueCount := 0
	for i := 1; i < len(consensusHeights) && issueCount < 10; i++ {
		if consensusHeights[i] < consensusHeights[i-1] {
			result.Passed = false
			runtimeRange := ""
			if i < len(runtimeHeights) {
				runtimeRange = fmt.Sprintf(", runtime: %d-%d", runtimeHeights[i-1], runtimeHeights[i])
			}
			result.Issues = append(result.Issues,
				fmt.Sprintf("Consensus height decreased: %d to %d (consensus: %d-%d%s)",
					consensusHeights[i-1], consensusHeights[i],
					consensusHeights[i-1], consensusHeights[i], runtimeRange))
			issueCount++
		}
	}

	// Check for large gaps in consensus heights (>10000 blocks)
	const maxGap = 10000
	for i := 1; i < len(consensusHeights) && issueCount < 10; i++ {
		gap := consensusHeights[i] - consensusHeights[i-1]
		if gap > maxGap {
			result.Passed = false
			runtimeRange := ""
			if i < len(runtimeHeights) {
				runtimeRange = fmt.Sprintf(", runtime: %d-%d", runtimeHeights[i-1], runtimeHeights[i])
			}
			result.Issues = append(result.Issues,
				fmt.Sprintf("Large consensus height gap: %d blocks (consensus: %d-%d%s)",
					gap, consensusHeights[i-1], consensusHeights[i], runtimeRange))
			issueCount++
		}
	}

	return result
}

// processConsensusMkvsEntry processes a single consensus-mkvs entry
func processConsensusMkvsEntry(key, value []byte, collector *consensusMkvsCollector, errorCounts map[string]int) {
	keyInfo := decodeConsensusMkvsKey(key)
	collector.keyTypes[keyInfo.KeyType]++

	// Track write_log versions for consistency checks
	if keyInfo.KeyType == "write_log" && keyInfo.ConsensusHeight > 0 {
		collector.writeLogVersions = append(collector.writeLogVersions, uint64(keyInfo.ConsensusHeight))
	}

	// Process node values to extract node types
	if keyInfo.KeyType == "node" && value != nil {
		valueInfo := decodeConsensusMkvsValue(keyInfo.KeyType, value)

		if valueInfo.RawError != "" {
			errorCounts[valueInfo.RawError]++
		}

		// Count node types
		if valueInfo.NodeType != "" {
			collector.nodeTypes[valueInfo.NodeType]++

			if valueInfo.NodeType == "leaf" {
				collector.totalLeaves++

				// Inline height extraction (module-specific)
				var height int64
				if valueInfo.Leaf != nil && valueInfo.Leaf.Module == "roothash" && len(key) >= 41 {
					height = int64(binary.BigEndian.Uint64(key[len(key)-8:]))
					if height <= 0 || height >= 100000000 {
						height = 0 // sanity check
					}
				}

				if height > 0 {
					collector.leafCountsByHeight[height]++

					// Try to count transactions/events from CBOR value
					if valueInfo.Leaf != nil && valueInfo.Leaf.ValueType == "cbor" && len(value) > 0 {
						// Try to decode CBOR and look for transaction/event arrays
						var decoded map[string]interface{}
						if err := cbor.Unmarshal(value, &decoded); err == nil {
							if txArray, ok := decoded["transactions"].([]interface{}); ok {
								collector.txCountsByHeight[height] += len(txArray)
							}
							if evArray, ok := decoded["events"].([]interface{}); ok {
								collector.eventCountsByHeight[height] += len(evArray)
							}
						}
					}
				}

				// Track module distribution
				if valueInfo.Leaf != nil && valueInfo.Leaf.Module != "" {
					collector.modules[valueInfo.Leaf.Module]++
				}
			} else if valueInfo.NodeType == "internal" {
				collector.totalInternal++
			}
		}
	}
}

// processRuntimeMkvsEntry processes a single runtime-mkvs entry
func processRuntimeMkvsEntry(key, value []byte, collector *runtimeMkvsCollector, errorCounts map[string]int) {
	keyInfo := decodeRuntimeMkvsKey(key)
	collector.keyTypes[keyInfo.KeyType]++

	// Track write_log versions and count transactions/events
	if keyInfo.KeyType == "write_log" && keyInfo.RuntimeHeight > 0 {
		collector.writeLogVersions = append(collector.writeLogVersions, keyInfo.RuntimeHeight)

		// Decode write_log and count transactions/events
		if value != nil {
			valueInfo := decodeRuntimeMkvsValue(keyInfo.KeyType, value)

			if valueInfo.WriteLog != nil {
				height := keyInfo.RuntimeHeight

				for _, entry := range valueInfo.WriteLog {
					if len(entry.Key) < 1 || entry.Value == nil {
						continue
					}

					module := decodeRuntimeModuleKey(entry.Key)
					leafValue := decodeRuntimeLeafValue(module, entry.Key, entry.Value)
					if leafValue == nil {
						continue
					}

					// Count transactions
					if leafValue.EVMTxInput != nil {
						collector.txCountsByHeight[height]++
						collector.evmTxCountsByHeight[height]++
						if leafValue.EVMTxInput.EVMTx != nil && leafValue.EVMTxInput.EVMTx.Type != "" {
							if collector.evmTxTypesByHeight[height] == nil {
								collector.evmTxTypesByHeight[height] = make(map[string]int)
							}
							collector.evmTxTypesByHeight[height][leafValue.EVMTxInput.EVMTx.Type]++
						}
					}

					// Count events
					if leafValue.EVMEvent != nil {
						collector.eventCountsByHeight[height]++
						collector.evmEventCountsByHeight[height]++
						if leafValue.EVMEvent.EventSignature != "" {
							if collector.evmEventSigsByHeight[height] == nil {
								collector.evmEventSigsByHeight[height] = make(map[string]int)
							}
							collector.evmEventSigsByHeight[height][leafValue.EVMEvent.EventSignature]++
						}
					} else if leafValue.RuntimeConsensusEvent != nil || leafValue.ValueType == "io_event" {
						collector.eventCountsByHeight[height]++
						if collector.runtimeEventsByHeight[height] == nil {
							collector.runtimeEventsByHeight[height] = make(map[string]int)
						}
						collector.runtimeEventsByHeight[height][module]++
					}
				}
			}
		}
	}

	// Process node values to extract node types and modules
	if keyInfo.KeyType == "node" && value != nil {
		valueInfo := decodeRuntimeMkvsValue(keyInfo.KeyType, value)

		if valueInfo.RawError != "" {
			errorCounts[valueInfo.RawError]++
		}

		// Count node types
		if valueInfo.NodeType != "" {
			collector.nodeTypes[valueInfo.NodeType]++

			if valueInfo.NodeType == "leaf" {
				collector.totalLeaves++
				// Extract module from leaf
				if valueInfo.Leaf != nil && valueInfo.Leaf.Module != "" {
					collector.modules[valueInfo.Leaf.Module]++
				}
			} else if valueInfo.NodeType == "internal" {
				collector.totalInternal++
			}
		}
	}
}

func buildConsensusMkvsStats(collector *consensusMkvsCollector) *MKVSStats {
	stats := &MKVSStats{
		NodeTypeDistribution: collector.nodeTypes,
		TotalNode:            collector.totalLeaves + collector.totalInternal,
	}

	// Include module distribution for consensus MKVS
	if len(collector.modules) > 0 {
		stats.ModuleDistribution = collector.modules
		// Calculate total across all modules
		for _, count := range collector.modules {
			stats.TotalModule += count
		}
	}

	// Add write_log statistics
	if len(collector.writeLogVersions) > 0 {
		stats.WriteLogStats = calculateWriteLogStats(collector.writeLogVersions)
	}

	return stats
}

func buildRuntimeMkvsStats(collector *runtimeMkvsCollector) *MKVSStats {
	stats := &MKVSStats{
		NodeTypeDistribution: collector.nodeTypes,
		TotalNode:            collector.totalLeaves + collector.totalInternal,
	}

	// Include module distribution for runtime MKVS
	if len(collector.modules) > 0 {
		stats.ModuleDistribution = collector.modules
		// Calculate total across all modules
		for _, count := range collector.modules {
			stats.TotalModule += count
		}
	}

	// Add write_log statistics
	if len(collector.writeLogVersions) > 0 {
		stats.WriteLogStats = calculateWriteLogStats(collector.writeLogVersions)
	}

	return stats
}

// calculateWriteLogStats computes statistics about available write_log entries
func calculateWriteLogStats(versions []uint64) *WriteLogStats {
	if len(versions) == 0 {
		return nil
	}

	// Sort to find range
	sortedVersions := make([]uint64, len(versions))
	copy(sortedVersions, versions)
	sort.Slice(sortedVersions, func(i, j int) bool {
		return sortedVersions[i] < sortedVersions[j]
	})

	oldest := sortedVersions[0]
	newest := sortedVersions[len(sortedVersions)-1]
	versionRange := newest - oldest + 1

	// Calculate coverage (what percentage of the range is present)
	coverage := 0.0
	if versionRange > 0 {
		coverage = (float64(len(versions)) / float64(versionRange)) * 100.0
	}

	return &WriteLogStats{
		OldestVersion: oldest,
		NewestVersion: newest,
		TotalEntries:  int64(len(versions)),
		VersionRange:  versionRange,
		Coverage:      coverage,
	}
}

// populateRuntimeTransactionEventStats builds transaction and event stats from write_log data
func populateRuntimeTransactionEventStats(stats *RuntimeBlockStats, collector *runtimeMkvsCollector) {
	// Build transaction stats
	if len(collector.txCountsByHeight) > 0 {
		txStats := &RuntimeTransactionStats{
			EVMTypeDistribution: make(map[string]int64),
		}

		// Calculate per-block stats
		perBlockStats := &SizeStatsAccumulator{}
		evmPerBlockStats := &SizeStatsAccumulator{}

		for height, count := range collector.txCountsByHeight {
			txStats.Total += int64(count)
			perBlockStats.Add(int64(count))

			// EVM transaction counts
			if evmCount, exists := collector.evmTxCountsByHeight[height]; exists {
				txStats.EVMTransactions += int64(evmCount)
				evmPerBlockStats.Add(int64(evmCount))
			}

			// Aggregate EVM type distribution
			if types, exists := collector.evmTxTypesByHeight[height]; exists {
				for txType, typeCount := range types {
					txStats.EVMTypeDistribution[txType] += int64(typeCount)
				}
			}
		}

		finalPerBlock := perBlockStats.Finalize()
		txStats.PerBlockStats = &finalPerBlock

		if evmPerBlockStats.count > 0 {
			finalEVM := evmPerBlockStats.Finalize()
			txStats.EVMSizeStats = &finalEVM
		}

		stats.TransactionStats = txStats
	}

	// Build event stats
	if len(collector.eventCountsByHeight) > 0 {
		eventStats := &RuntimeEventStats{
			RuntimeEventTypes: make(map[string]int64),
			EVMSignatures:     make(map[string]int64),
		}

		perBlockStats := &SizeStatsAccumulator{}

		for height, count := range collector.eventCountsByHeight {
			eventStats.Total += int64(count)
			perBlockStats.Add(int64(count))

			// EVM event counts
			if evmCount, exists := collector.evmEventCountsByHeight[height]; exists {
				eventStats.EVMEvents += int64(evmCount)
			}

			// Aggregate EVM signatures
			if sigs, exists := collector.evmEventSigsByHeight[height]; exists {
				for sig, sigCount := range sigs {
					eventStats.EVMSignatures[sig] += int64(sigCount)
				}
			}

			// Aggregate runtime event types
			if types, exists := collector.runtimeEventsByHeight[height]; exists {
				for eventType, typeCount := range types {
					eventStats.RuntimeEventTypes[eventType] += int64(typeCount)
				}
			}
		}

		finalPerBlock := perBlockStats.Finalize()
		eventStats.PerBlockStats = &finalPerBlock

		stats.EventStats = eventStats
	}
}

// checkConsensusMKVSHeightAnomalies detects unusual patterns in leaf counts per height
func checkConsensusMKVSHeightAnomalies(leafCountsByHeight map[int64]int) CheckResult {
	result := CheckResult{Name: "consensus_mkvs_height_anomalies", Passed: true}
	if len(leafCountsByHeight) < 3 {
		return result
	}

	// Calculate mean and stddev of leaf counts
	sum := 0.0
	for _, count := range leafCountsByHeight {
		sum += float64(count)
	}
	mean := sum / float64(len(leafCountsByHeight))

	variance := 0.0
	for _, count := range leafCountsByHeight {
		variance += (float64(count) - mean) * (float64(count) - mean)
	}
	stddev := math.Sqrt(variance / float64(len(leafCountsByHeight)))

	// Find anomalies (more than 3σ from mean)
	threshold := 3.0 * stddev
	if stddev == 0 {
		return result
	}

	// Sort heights for reporting
	heights := make([]int64, 0, len(leafCountsByHeight))
	for height := range leafCountsByHeight {
		heights = append(heights, height)
	}
	sort.Slice(heights, func(i, j int) bool {
		return heights[i] < heights[j]
	})

	issueCount := 0
	for _, height := range heights {
		count := float64(leafCountsByHeight[height])
		if math.Abs(count-mean) > threshold {
			result.Passed = false
			result.Issues = append(result.Issues,
				fmt.Sprintf("Anomalous leaf count at height %d: %d nodes", height, int(count)))
			issueCount++
			if issueCount >= 10 {
				break
			}
		}
	}

	return result
}

// runConsensusMkvsChecks runs all consistency checks for consensus-mkvs databases
func runConsensusMkvsChecks(collector *consensusMkvsCollector) []CheckResult {
	checks := []CheckResult{}

	// version_monotonicity
	checks = append(checks, checkVersionMonotonicity(collector.writeLogVersions))

	// consensus_mkvs_height_anomalies - run if height data available
	if len(collector.leafCountsByHeight) > 0 {
		checks = append(checks, checkConsensusMKVSHeightAnomalies(collector.leafCountsByHeight))
	}

	return checks
}

// runRuntimeMkvsChecks runs all consistency checks for runtime-mkvs databases
func runRuntimeMkvsChecks(collector *runtimeMkvsCollector) []CheckResult {
	checks := []CheckResult{}

	// version_monotonicity
	checks = append(checks, checkVersionMonotonicity(collector.writeLogVersions))

	// No height anomaly check for runtime MKVS (no height-based aggregation)

	return checks
}

// checkVersionMonotonicity checks if versions increase monotonically (allowing duplicates)
func checkVersionMonotonicity(versions []uint64) CheckResult {
	result := CheckResult{
		Name:   "version_monotonicity",
		Passed: true,
	}

	if len(versions) < 2 {
		return result
	}

	// Check if versions are non-decreasing
	issueCount := 0
	for i := 1; i < len(versions) && issueCount < 10; i++ {
		if versions[i] < versions[i-1] {
			result.Passed = false
			result.Issues = append(result.Issues,
				fmt.Sprintf("Version decreased: %d to %d at index %d",
					versions[i-1], versions[i], i))
			issueCount++
		}
	}

	return result
}

func processConsensusEvidenceEntry(key, value []byte, collector *consensusEvidenceCollector, errorCounts map[string]int) {
	keyInfo := decodeConsensusEvidenceKey(key)
	collector.keyTypes[keyInfo.KeyType]++

	// Collect consensus heights for gap checking
	if keyInfo.ConsensusHeight > 0 {
		collector.consensusHeights = append(collector.consensusHeights, keyInfo.ConsensusHeight)
	}
}

func processConsensusStateEntry(key, value []byte, collector *consensusStateCollector, errorCounts map[string]int) {
	keyInfo := decodeConsensusStateKey(key)
	collector.keyTypes[keyInfo.KeyType]++

	// Collect ABCI response heights and data for analysis
	if keyInfo.KeyType == "abci_responses" && keyInfo.ConsensusHeight > 0 {
		height := keyInfo.ConsensusHeight
		collector.abciResponseHeights = append(collector.abciResponseHeights, height)

		// Decode ABCI response (reuses existing logic from decode_consensus.go)
		valueInfo := decodeConsensusStateValue(keyInfo.KeyType, value)
		if valueInfo.ABCIResponse != nil {
			resp := valueInfo.ABCIResponse

			// Extract tx count
			collector.txCountsByHeight[height] = resp.TxResultCount

			// Extract transaction method counts
			if resp.TransactionSummary != nil {
				collector.txMethodsByHeight[height] = resp.TransactionSummary.MethodCounts
			}

			// Extract event type counts
			if resp.EventSummary != nil {
				collector.eventTypesByHeight[height] = resp.EventSummary.EventTypeCounts
			}
		}
	}
}

// runConsensusEvidenceChecks runs consistency checks for consensus-evidence
func runConsensusEvidenceChecks(collector *consensusEvidenceCollector) []CheckResult {
	checks := []CheckResult{}

	// consensus_height_sequence - check for large gaps
	checks = append(checks, checkConsensusHeightSequence(collector.consensusHeights))

	return checks
}

// buildABCIAnalysisStats builds aggregated stats from collected data
func buildABCIAnalysisStats(collector *consensusStateCollector) *ABCIAnalysisStats {
	if len(collector.txCountsByHeight) == 0 {
		return nil
	}

	stats := &ABCIAnalysisStats{
		TxMethodCounts:  make(map[string]int64),
		EventTypeCounts: make(map[string]int64),
	}

	// Aggregate across all heights
	for _, txCounts := range collector.txMethodsByHeight {
		for method, count := range txCounts {
			stats.TxMethodCounts[method] += int64(count)
		}
	}

	for _, eventCounts := range collector.eventTypesByHeight {
		for eventType, count := range eventCounts {
			stats.EventTypeCounts[eventType] += int64(count)
		}
	}

	// Calculate totals
	for _, count := range stats.TxMethodCounts {
		stats.TotalTx += count
	}

	for _, count := range stats.EventTypeCounts {
		stats.TotalEvent += count
	}

	return stats
}

// checkABCIResponseAnomalies detects unusual patterns using pre-computed tx counts
func checkABCIResponseAnomalies(txCountsByHeight map[int64]int) CheckResult {
	result := CheckResult{Name: "abci_response_anomalies", Passed: true}

	if len(txCountsByHeight) < 3 {
		return result
	}

	// Calculate mean and stddev of tx counts
	sum := 0.0
	for _, count := range txCountsByHeight {
		sum += float64(count)
	}
	mean := sum / float64(len(txCountsByHeight))

	variance := 0.0
	for _, count := range txCountsByHeight {
		variance += (float64(count) - mean) * (float64(count) - mean)
	}
	stddev := math.Sqrt(variance / float64(len(txCountsByHeight)))

	// Find anomalies (more than 3σ from mean)
	threshold := 3.0 * stddev
	if stddev == 0 {
		return result
	}

	// Sort heights for reporting
	heights := make([]int64, 0, len(txCountsByHeight))
	for height := range txCountsByHeight {
		heights = append(heights, height)
	}
	sort.Slice(heights, func(i, j int) bool {
		return heights[i] < heights[j]
	})

	issueCount := 0
	for _, height := range heights {
		count := float64(txCountsByHeight[height])
		if math.Abs(count-mean) > threshold {
			result.Passed = false
			result.Issues = append(result.Issues,
				fmt.Sprintf("Anomalous tx count at height %d: %d transactions", height, int(count)))
			issueCount++
			if issueCount >= 10 {
				break
			}
		}
	}

	return result
}

// runConsensusStateChecks runs consistency checks for consensus-state
func runConsensusStateChecks(collector *consensusStateCollector) []CheckResult {
	checks := []CheckResult{}

	// abci_response_complete - check that all heights have ABCI responses
	checks = append(checks, checkABCIResponseComplete(collector.abciResponseHeights))

	// abci_response_anomalies - check for unusual tx count patterns
	checks = append(checks, checkABCIResponseAnomalies(collector.txCountsByHeight))

	return checks
}

// checkConsensusHeightSequence checks for large gaps in consensus heights (>1000 blocks)
func checkConsensusHeightSequence(consensusHeights []int64) CheckResult {
	result := CheckResult{
		Name:   "consensus_height_sequence",
		Passed: true,
	}

	if len(consensusHeights) < 2 {
		return result
	}

	// Sort heights
	sorted := make([]int64, len(consensusHeights))
	copy(sorted, consensusHeights)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Check for large gaps (>1000 blocks)
	const maxGap = 1000
	issueCount := 0
	for i := 1; i < len(sorted) && issueCount < 10; i++ {
		gap := sorted[i] - sorted[i-1]
		if gap > maxGap {
			result.Passed = false
			result.Issues = append(result.Issues,
				fmt.Sprintf("Large consensus height gap: %d blocks (consensus: %d-%d)",
					gap, sorted[i-1], sorted[i]))
			issueCount++
		}
	}

	return result
}

// checkABCIResponseComplete checks if ABCI responses form a complete sequence
func checkABCIResponseComplete(abciHeights []int64) CheckResult {
	result := CheckResult{
		Name:   "abci_response_complete",
		Passed: true,
	}

	if len(abciHeights) == 0 {
		return result
	}

	// Sort heights
	sorted := make([]int64, len(abciHeights))
	copy(sorted, abciHeights)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	minHeight := sorted[0]
	maxHeight := sorted[len(sorted)-1]
	expected := maxHeight - minHeight + 1

	if int64(len(sorted)) != expected {
		result.Passed = false

		// Find missing heights
		heightSet := make(map[int64]bool)
		for _, h := range sorted {
			heightSet[h] = true
		}

		missingCount := 0
		for h := minHeight; h <= maxHeight && missingCount < 100; h++ {
			if !heightSet[h] {
				result.Issues = append(result.Issues, fmt.Sprintf("Missing ABCI response at consensus height: %d", h))
				missingCount++
			}
		}
		if expected-int64(len(sorted)) > 100 {
			result.Issues = append(result.Issues, fmt.Sprintf("... and %d more missing ABCI responses", expected-int64(len(sorted))-100))
		}
	}

	return result
}
