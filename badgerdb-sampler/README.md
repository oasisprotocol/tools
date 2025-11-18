# BadgerDB Sampler

A tool for sampling BadgerDB v2, v3, and v4 databases used by Oasis nodes.

## Building

```bash
# Build all versions
make build-all
```

## Usage

### Prerequisites

- Go 1.21 or higher
- BadgerDB databases from Oasis nodes (see https://snapshots.oasis.io/)

### Run

```bash
# Same interface for all versions
./badgerdb-sampler-v2 <database-type> <path-to-db> [output-subdir] [output-prefix] [max-samples]
./badgerdb-sampler-v3 <database-type> <path-to-db> [output-subdir] [output-prefix] [max-samples]
```

Database types (version suffixes are optional):

- `consensus-blockstore` (or `consensus-blockstore-v2`, `consensus-blockstore-v3`)
- `consensus-evidence`
- `consensus-mkvs`
- `consensus-state`
- `runtime-mkvs`
- `runtime-history`

### Examples

```bash
# Analyze v2 consensus blockstore
./badgerdb-sampler-v2 consensus-blockstore /path/to/blockstore.badger.db ./outputs/testnet-20220303/

# Analyze v3 runtime MKVS
./badgerdb-sampler-v3 runtime-mkvs /path/to/mkvs_storage.badger.db ./outputs/testnet-20220303/emerald- 500
```
