# BadgerDB Sampler

This **badgerdb-sampler** tool implements a comprehensive data extraction and decoding logic for BadgerDB databases used in Oasis node snapshots. This tool can extract and decode most data structures from consensus and runtime databases, even EVM data. It handles multiple BadgerDB versions (v2-v4) and data representations use by Oasis nodes (v20.x-v25.x). The decoding logic handles various data formats and reports decoding issues.

## Features

- **Multi-version BadgerDB Support**: Compatible with BadgerDB v2, v3, and v4
- **Comprehensive Decoding**: Handles consensus state, runtime state, and EVM-specific data
- **Error Tracking**: Collects and reports decoding errors with detailed error counts
- **Statistics Generation**: Provides key type distributions, database size, and sample counts
- **Module-aware Parsing**: Recognizes and routes data to appropriate decoders (evm, accounts, contracts, core)
- **EVM Event Decoding**: Includes event signature database for human-readable EVM event names
- **FUSE Filesystem Support**: Works with databases on FUSE mounts using intelligent fallback strategies
- **Read-only Access**: Can analyze databases currently in use by nodes via `BypassLockGuard`

## Supported Database Types

- `consensus-blockstore` - Block metadata and commit info
- `consensus-evidence` - Byzantine validator evidence
- `consensus-mkvs` - Consensus state Merkle tree
- `consensus-state` - Tendermint/CometBFT consensus state
- `runtime-mkvs` - Runtime state Merkle tree (includes EVM storage)
- `runtime-history` - Runtime block history with CBOR-encoded data (includes EVM events/transactions)

## Building

```bash
# Build all versions (recommended)
make build-all

# Clean build artifacts
make clean
```

## Usage

### Prerequisites

- Go 1.21 or higher
- BadgerDB databases from Oasis nodes (see https://snapshots.oasis.io/)

### Command Syntax

```bash
./bin/badgerdb-sampler-v{2,3,4} <database-type> <path-to-db> [output-json] [max-samples]
```

**Parameters:**
- `database-type`: One of the supported database types (see above)
- `path-to-db`: Path to the BadgerDB database directory
- `output-json`: Optional path to save JSON results (default: stdout only)
- `max-samples`: Optional maximum number of samples to collect (default: 1000)

### Examples

```bash
# Analyze v2 consensus blockstore (default 1000 samples, stdout only)
./bin/badgerdb-sampler-v2 consensus-blockstore /path/to/blockstore.badger.db

# Analyze v3 runtime MKVS with JSON output
./bin/badgerdb-sampler-v3 runtime-mkvs /path/to/mkvs_storage.badger.db ./outputs/testnet-20220303/emerald-runtime-mkvs.json

# Analyze runtime history with custom sample limit
./bin/badgerdb-sampler-v3 runtime-history /path/to/history.badger.db ./outputs/runtime-history.json 500

# Analyze currently running node's database (read-only)
./bin/badgerdb-sampler-v3 consensus-state /var/lib/oasis/node/consensus/state.badger.db ./outputs/consensus-state.json
```

## Output Format

The tool outputs JSON with comprehensive statistics and samples:

```json
{
  "database_path": "/path/to/mkvs_storage.badger.db",
  "database_type": "runtime-mkvs",
  "badgerdb_version": "v3",
  "database_size_bytes": 1234567890,
  "sample_count": 1000,
  "key_type_counts": {
    "mkvs:node": 850,
    "mkvs:root": 150
  },
  "error_counts": {
    "failed to decode CBOR value: unexpected EOF": 5,
    "unknown module prefix": 2
  },
  "samples": [
    {
      "key_type": "mkvs:node",
      "key": { /* decoded key structure */ },
      "value": { /* decoded value structure */ },
    }
  ]
}
```

**Key Fields:**
- `key_type_counts`: Distribution of different key types in the database
- `error_counts`: Aggregated decoding errors across all samples
- `samples`: Array of individual key-value pairs with full decoding details

## Architecture

The tool uses a three-layer design:

1. **Database Access** (`db_v*.go`, `db_common.go`)
   - Version-specific BadgerDB initialization via build tags
   - FUSE workaround with temp directory and symlinks
   - Read-only mode with lock bypass for in-use databases

2. **Decoding Logic** (`decode_*.go`)
   - `decode_consensus.go`: Tendermint protobuf parsing
   - `decode_runtime.go`: CBOR/MKVS parsing with module routing
   - `decode_evm.go`: EVM-specific parsing (contracts, events, transactions)

3. **Type System** (`types_*.go`)
   - Separated deserialization and output types
   - EVM-specific output structures
   - Event signature database for human-readable names
