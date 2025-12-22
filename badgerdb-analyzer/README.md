# BadgerDB Analyzer

This **badgerdb-analyzer** tool is for analyzing and checking the consistency of BadgerDB databases from Oasis node snapshots. It performs a full database scan, analysis with statistics, consistency validation, and integrity checks. It handles multiple BadgerDB versions (v2-v4) and data representations use by Oasis nodes (v20.x-v25.x). The decoding logic is from the **badgerdb-sampler** tool.

**Warning:** For experimental purposes only. Decoded data might be incorrect.

## Features

- Full database analysis in single pass
- Comprehensive statistics (key/value sizes, block ranges, transaction counts)
- Consistency checks (block sequences, timestamp validation, referential integrity)
- Multi-version support (BadgerDB v2, v3, v4)
- Six database types (consensus-blockstore, consensus-evidence, consensus-mkvs, consensus-state, runtime-history, runtime-mkvs)
- JSON output for programmatic analysis

## Supported Database Types

| Database Type          | Description                         | Key Statistics |
|------------------------|-------------------------------------|----------------|
| `consensus-blockstore` | Block metadata and commit info      | Consensus block sequences, timestamps, transactions, events |
| `consensus-evidence`   | Byzantine validator evidence        | Evidence heights, gap detection |
| `consensus-mkvs`       | Consensus state Merkle tree         | Node types, write log versions |
| `consensus-state`      | Tendermint/CometBFT consensus state | ABCI responses completeness |
| `runtime-history`      | Runtime block history               | Runtime/consensus heights, timestamps, intervals |
| `runtime-mkvs`         | Runtime state Merkle tree           | Node types, module distribution, write log versions |

## Installation

```bash
make build-all
```

This builds three binaries:
- `./bin/badgerdb-analyzer-v2` - For BadgerDB v2 databases
- `./bin/badgerdb-analyzer-v3` - For BadgerDB v3 databases
- `./bin/badgerdb-analyzer-v4` - For BadgerDB v4 databases

## Usage

```bash
badgerdb-analyzer-vX <database-type> <database-path> [output-json]
```

**Arguments:**
- `database-type`: One of: `consensus-blockstore`, `consensus-evidence`, `consensus-mkvs`, `consensus-state`, `runtime-mkvs`, `runtime-history`
- `database-path`: Path to the BadgerDB directory
- `output-json`: (Optional) Path to JSON output file. If omitted, outputs to stdout

### Examples

**Analyze consensus blockstore:**
```bash
./bin/badgerdb-analyzer-v3 consensus-blockstore \
  /snapshots/mainnet/consensus/tendermint/data/blockstore.badger.db \
  blockstore-analysis.json
```

**Analyze runtime history:**
```bash
./bin/badgerdb-analyzer-v3 runtime-history \
  /snapshots/mainnet/runtimes/<runtime-id>/history.db \
  runtime-history-analysis.json
```

## Exit Codes

- `0` - Success, all checks passed
- `1` - Issues found (checks failed)
- `2` - Database open failed
- `3` - Invalid arguments
- `4` - Analysis error

## Related Tools

- **badgerdb-sampler**: Sample-based database exploration
