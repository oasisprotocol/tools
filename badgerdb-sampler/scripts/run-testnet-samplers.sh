#!/bin/bash
# Script to run all badgerdb-samplers on Testnet
set -euo pipefail

SAMPLER_V2="./bin/badgerdb-sampler-v2"
SAMPLER_V3="./bin/badgerdb-sampler-v3"
OUTPUT="./outputs"

mkdir -p "$OUTPUT"

# $OUTPUT/testnet/20200915-20201104/ (BadgerDB v2)
(
mkdir -p "$OUTPUT/testnet-20200915-20201104"
$SAMPLER_V2 consensus-blockstore-v2 /snapshots/testnet/20200915-20201104/consensus_testnet.mount/tendermint/data/blockstore.badger.db $OUTPUT/testnet-20200915-20201104/ > $OUTPUT/testnet-20200915-20201104/consensus-blockstore-v2.log 2>&1 || true
$SAMPLER_V2 consensus-evidence-v2 /snapshots/testnet/20200915-20201104/consensus_testnet.mount/tendermint/data/evidence.badger.db $OUTPUT/testnet-20200915-20201104/ > $OUTPUT/testnet-20200915-20201104/consensus-evidence-v2.log 2>&1 || true
$SAMPLER_V2 consensus-state-v2 /snapshots/testnet/20200915-20201104/consensus_testnet.mount/tendermint/data/state.badger.db $OUTPUT/testnet-20200915-20201104/ > $OUTPUT/testnet-20200915-20201104/consensus-state-v2.log 2>&1 || true
$SAMPLER_V2 consensus-mkvs-v2 /snapshots/testnet/20200915-20201104/consensus_testnet.mount/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT/testnet-20200915-20201104/ > $OUTPUT/testnet-20200915-20201104/consensus-mkvs-v2.log 2>&1 || true
) &

# $OUTPUT/testnet/20201104-20210203/ (BadgerDB v2)
(
mkdir -p "$OUTPUT/testnet-20201104-20210203"
$SAMPLER_V2 consensus-blockstore-v2 /snapshots/testnet/20201104-20210203/consensus_testnet.mount/tendermint/data/blockstore.badger.db $OUTPUT/testnet-20201104-20210203/ > $OUTPUT/testnet-20201104-20210203/consensus-blockstore-v2.log 2>&1 || true
$SAMPLER_V2 consensus-evidence-v2 /snapshots/testnet/20201104-20210203/consensus_testnet.mount/tendermint/data/evidence.badger.db $OUTPUT/testnet-20201104-20210203/ > $OUTPUT/testnet-20201104-20210203/consensus-evidence-v2.log 2>&1 || true
$SAMPLER_V2 consensus-state-v2 /snapshots/testnet/20201104-20210203/consensus_testnet.mount/tendermint/data/state.badger.db $OUTPUT/testnet-20201104-20210203/ > $OUTPUT/testnet-20201104-20210203/consensus-state-v2.log 2>&1 || true
$SAMPLER_V2 consensus-mkvs-v2 /snapshots/testnet/20201104-20210203/consensus_testnet.mount/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT/testnet-20201104-20210203/ > $OUTPUT/testnet-20201104-20210203/consensus-mkvs-v2.log 2>&1 || true
) &

# $OUTPUT/testnet/20210203-20210324/ (BadgerDB v2)
(
mkdir -p "$OUTPUT/testnet-20210203-20210324"
$SAMPLER_V2 consensus-blockstore-v2 /snapshots/testnet/20210203-20210324/consensus_testnet.mount/tendermint/data/blockstore.badger.db $OUTPUT/testnet-20210203-20210324/ > $OUTPUT/testnet-20210203-20210324/consensus-blockstore-v2.log 2>&1 || true
$SAMPLER_V2 consensus-evidence-v2 /snapshots/testnet/20210203-20210324/consensus_testnet.mount/tendermint/data/evidence.badger.db $OUTPUT/testnet-20210203-20210324/ > $OUTPUT/testnet-20210203-20210324/consensus-evidence-v2.log 2>&1 || true
$SAMPLER_V2 consensus-state-v2 /snapshots/testnet/20210203-20210324/consensus_testnet.mount/tendermint/data/state.badger.db $OUTPUT/testnet-20210203-20210324/ > $OUTPUT/testnet-20210203-20210324/consensus-state-v2.log 2>&1 || true
$SAMPLER_V2 consensus-mkvs-v2 /snapshots/testnet/20210203-20210324/consensus_testnet.mount/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT/testnet-20210203-20210324/ > $OUTPUT/testnet-20210203-20210324/consensus-mkvs-v2.log 2>&1 || true
) &

# $OUTPUT/testnet/20210324-20210413/ (BadgerDB v2)
(
mkdir -p "$OUTPUT/testnet-20210324-20210413"
$SAMPLER_V2 consensus-blockstore-v2 /snapshots/testnet/20210324-20210413/consensus_testnet.mount/tendermint/data/blockstore.badger.db $OUTPUT/testnet-20210324-20210413/ > $OUTPUT/testnet-20210324-20210413/consensus-blockstore-v2.log 2>&1 || true
$SAMPLER_V2 consensus-evidence-v2 /snapshots/testnet/20210324-20210413/consensus_testnet.mount/tendermint/data/evidence.badger.db $OUTPUT/testnet-20210324-20210413/ > $OUTPUT/testnet-20210324-20210413/consensus-evidence-v2.log 2>&1 || true
$SAMPLER_V2 consensus-state-v2 /snapshots/testnet/20210324-20210413/consensus_testnet.mount/tendermint/data/state.badger.db $OUTPUT/testnet-20210324-20210413/ > $OUTPUT/testnet-20210324-20210413/consensus-state-v2.log 2>&1 || true
$SAMPLER_V2 consensus-mkvs-v2 /snapshots/testnet/20210324-20210413/consensus_testnet.mount/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT/testnet-20210324-20210413/ > $OUTPUT/testnet-20210324-20210413/consensus-mkvs-v2.log 2>&1 || true
) &

# $OUTPUT/testnet/20210413-20220303/ (BadgerDB v3)
(
mkdir -p "$OUTPUT/testnet-20210413-20220303"
$SAMPLER_V3 consensus-blockstore-v3 /snapshots/testnet/20210413-20220303/consensus_testnet.mount/tendermint/data/blockstore.badger.db $OUTPUT/testnet-20210413-20220303/ > $OUTPUT/testnet-20210413-20220303/consensus-blockstore-v3.log 2>&1 || true
$SAMPLER_V3 consensus-evidence-v3 /snapshots/testnet/20210413-20220303/consensus_testnet.mount/tendermint/data/evidence.badger.db $OUTPUT/testnet-20210413-20220303/ > $OUTPUT/testnet-20210413-20220303/consensus-evidence-v3.log 2>&1 || true
$SAMPLER_V3 consensus-state-v3 /snapshots/testnet/20210413-20220303/consensus_testnet.mount/tendermint/data/state.badger.db $OUTPUT/testnet-20210413-20220303/ > $OUTPUT/testnet-20210413-20220303/consensus-state-v3.log 2>&1 || true
$SAMPLER_V3 consensus-mkvs-v3 /snapshots/testnet/20210413-20220303/consensus_testnet.mount/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT/testnet-20210413-20220303/ > $OUTPUT/testnet-20210413-20220303/consensus-mkvs-v3.log 2>&1 || true
) &

# $OUTPUT/testnet/20220303-20231012/ (BadgerDB v3)
(
mkdir -p "$OUTPUT/testnet-20220303-20231012"
$SAMPLER_V3 consensus-blockstore-v3 /snapshots/testnet/20220303-20231012/consensus_testnet.mount/tendermint/data/blockstore.badger.db $OUTPUT/testnet-20220303-20231012/ > $OUTPUT/testnet-20220303-20231012/consensus-blockstore-v3.log 2>&1 || true
$SAMPLER_V3 consensus-evidence-v3 /snapshots/testnet/20220303-20231012/consensus_testnet.mount/tendermint/data/evidence.badger.db $OUTPUT/testnet-20220303-20231012/ > $OUTPUT/testnet-20220303-20231012/consensus-evidence-v3.log 2>&1 || true
$SAMPLER_V3 consensus-state-v3 /snapshots/testnet/20220303-20231012/consensus_testnet.mount/tendermint/data/state.badger.db $OUTPUT/testnet-20220303-20231012/ > $OUTPUT/testnet-20220303-20231012/consensus-state-v3.log 2>&1 || true
$SAMPLER_V3 consensus-mkvs-v3 /snapshots/testnet/20220303-20231012/consensus_testnet.mount/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT/testnet-20220303-20231012/ > $OUTPUT/testnet-20220303-20231012/consensus-mkvs-v3.log 2>&1 || true

$SAMPLER_V3 runtime-mkvs-v3 /snapshots/testnet/20220303-20231012/cipher_testnet.mount/runtimes/0000000000000000000000000000000000000000000000000000000000000000/mkvs_storage.badger.db $OUTPUT/testnet-20220303-20231012/cipher- > $OUTPUT/testnet-20220303-20231012/cipher-runtime-mkvs-v3.log 2>&1 || true
$SAMPLER_V3 runtime-mkvs-v3 /snapshots/testnet/20220303-20231012/emerald_testnet.mount/runtimes/00000000000000000000000000000000000000000000000072c8215e60d5bca7/mkvs_storage.badger.db $OUTPUT/testnet-20220303-20231012/emerald- > $OUTPUT/testnet-20220303-20231012/emerald-runtime-mkvs-v3.log 2>&1 || true
$SAMPLER_V3 runtime-mkvs-v3 /snapshots/testnet/20220303-20231012/sapphire_testnet.mount/runtimes/000000000000000000000000000000000000000000000000a6d1e3ebf60dff6c/mkvs_storage.badger.db $OUTPUT/testnet-20220303-20231012/sapphire- > $OUTPUT/testnet-20220303-20231012/sapphire-runtime-mkvs-v3.log 2>&1 || true

$SAMPLER_V3 runtime-history-v3 /snapshots/testnet/20220303-20231012/cipher_testnet.mount/runtimes/0000000000000000000000000000000000000000000000000000000000000000/history.db $OUTPUT/testnet-20220303-20231012/cipher- > $OUTPUT/testnet-20220303-20231012/cipher-runtime-history-v3.log 2>&1 || true
$SAMPLER_V3 runtime-history-v3 /snapshots/testnet/20220303-20231012/emerald_testnet.mount/runtimes/00000000000000000000000000000000000000000000000072c8215e60d5bca7/history.db $OUTPUT/testnet-20220303-20231012/emerald- > $OUTPUT/testnet-20220303-20231012/emerald-runtime-history-v3.log 2>&1 || true
$SAMPLER_V3 runtime-history-v3 /snapshots/testnet/20220303-20231012/sapphire_testnet.mount/runtimes/000000000000000000000000000000000000000000000000a6d1e3ebf60dff6c/history.db $OUTPUT/testnet-20220303-20231012/sapphire- > $OUTPUT/testnet-20220303-20231012/sapphire-runtime-history-v3.log 2>&1 || true
) &

echo "All samplers launched in background!"
ps aux | grep badgerdb-sampler
