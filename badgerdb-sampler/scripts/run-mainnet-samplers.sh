#!/bin/bash
# Script to run all badgerdb-samplers on Mainnet
set -euo pipefail

SAMPLER_V2="./bin/badgerdb-sampler-v2"
SAMPLER_V3="./bin/badgerdb-sampler-v3"
OUTPUT="./outputs"

mkdir -p "$OUTPUT"

# $OUTPUT/mainnet/20201001-20201118/ (BadgerDB v2)
mkdir -p "$OUTPUT/mainnet-20201001-20201118"
(
$SAMPLER_V2 consensus-blockstore-v2 /snapshots/mainnet/20201001-20201118/consensus.mount/tendermint/data/blockstore.badger.db $OUTPUT/mainnet-20201001-20201118/consensus-blockstore-v2.json > $OUTPUT/mainnet-20201001-20201118/consensus-blockstore-v2.log 2>&1 || true
$SAMPLER_V2 consensus-evidence-v2 /snapshots/mainnet/20201001-20201118/consensus.mount/tendermint/data/evidence.badger.db $OUTPUT/mainnet-20201001-20201118/consensus-evidence-v2.json > $OUTPUT/mainnet-20201001-20201118/consensus-evidence-v2.log 2>&1 || true
$SAMPLER_V2 consensus-state-v2 /snapshots/mainnet/20201001-20201118/consensus.mount/tendermint/data/state.badger.db $OUTPUT/mainnet-20201001-20201118/consensus-state-v2.json > $OUTPUT/mainnet-20201001-20201118/consensus-state-v2.log 2>&1 || true
$SAMPLER_V2 consensus-mkvs-v2 /snapshots/mainnet/20201001-20201118/consensus.mount/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT/mainnet-20201001-20201118/consensus-mkvs-v2.json > $OUTPUT/mainnet-20201001-20201118/consensus-mkvs-v2.log 2>&1 || true
) &

# $OUTPUT/mainnet/20201118-20210428/ (BadgerDB v2)
mkdir -p "$OUTPUT/mainnet-20201118-20210428"
(
$SAMPLER_V2 consensus-blockstore-v2 /snapshots/mainnet/20201118-20210428/consensus.mount/tendermint/data/blockstore.badger.db $OUTPUT/mainnet-20201118-20210428/consensus-blockstore-v2.json > $OUTPUT/mainnet-20201118-20210428/consensus-blockstore-v2.log 2>&1 || true
$SAMPLER_V2 consensus-evidence-v2 /snapshots/mainnet/20201118-20210428/consensus.mount/tendermint/data/evidence.badger.db $OUTPUT/mainnet-20201118-20210428/consensus-evidence-v2.json > $OUTPUT/mainnet-20201118-20210428/consensus-evidence-v2.log 2>&1 || true
$SAMPLER_V2 consensus-state-v2 /snapshots/mainnet/20201118-20210428/consensus.mount/tendermint/data/state.badger.db $OUTPUT/mainnet-20201118-20210428/consensus-state-v2.json > $OUTPUT/mainnet-20201118-20210428/consensus-state-v2.log 2>&1 || true
$SAMPLER_V2 consensus-mkvs-v2 /snapshots/mainnet/20201118-20210428/consensus.mount/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT/mainnet-20201118-20210428/consensus-mkvs-v2.json > $OUTPUT/mainnet-20201118-20210428/consensus-mkvs-v2.log 2>&1 || true
) &

# $OUTPUT/mainnet/20210428-20220411/ (BadgerDB v3)
mkdir -p "$OUTPUT/mainnet-20210428-20220411"
(
$SAMPLER_V3 consensus-blockstore-v3 /snapshots/mainnet/20210428-20220411/consensus.mount/tendermint/data/blockstore.badger.db $OUTPUT/mainnet-20210428-20220411/consensus-blockstore-v3.json > $OUTPUT/mainnet-20210428-20220411/consensus-blockstore-v3.log 2>&1 || true
$SAMPLER_V3 consensus-evidence-v3 /snapshots/mainnet/20210428-20220411/consensus.mount/tendermint/data/evidence.badger.db $OUTPUT/mainnet-20210428-20220411/consensus-evidence-v3.json > $OUTPUT/mainnet-20210428-20220411/consensus-evidence-v3.log 2>&1 || true
$SAMPLER_V3 consensus-state-v3 /snapshots/mainnet/20210428-20220411/consensus.mount/tendermint/data/state.badger.db $OUTPUT/mainnet-20210428-20220411/consensus-state-v3.json > $OUTPUT/mainnet-20210428-20220411/consensus-state-v3.log 2>&1 || true
$SAMPLER_V3 consensus-mkvs-v3 /snapshots/mainnet/20210428-20220411/consensus.mount/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT/mainnet-20210428-20220411/consensus-mkvs-v3.json > $OUTPUT/mainnet-20210428-20220411/consensus-mkvs-v3.log 2>&1 || true
) &
(
$SAMPLER_V3 runtime-mkvs-v3 /snapshots/mainnet/20210428-20220411/cipher.mount/runtimes/000000000000000000000000000000000000000000000000e199119c992377cb/mkvs_storage.badger.db $OUTPUT/mainnet-20210428-20220411/cipher-runtime-mkvs-v3.json > $OUTPUT/mainnet-20210428-20220411/cipher-runtime-mkvs-v3.log 2>&1 || true
$SAMPLER_V3 runtime-history-v3 /snapshots/mainnet/20210428-20220411/cipher.mount/runtimes/000000000000000000000000000000000000000000000000e199119c992377cb/history.db $OUTPUT/mainnet-20210428-20220411/cipher-runtime-history-v3.json > $OUTPUT/mainnet-20210428-20220411/cipher-runtime-history-v3.log 2>&1 || true
) &
(
$SAMPLER_V3 runtime-history-v3 /snapshots/mainnet/20210428-20220411/emerald.mount/runtimes/000000000000000000000000000000000000000000000000e2eaa99fc008f87f/history.db $OUTPUT/mainnet-20210428-20220411/emerald-runtime-history-v3.json > $OUTPUT/mainnet-20210428-20220411/emerald-runtime-history-v3.log 2>&1 || true
$SAMPLER_V3 runtime-mkvs-v3 /snapshots/mainnet/20210428-20220411/emerald.mount/runtimes/000000000000000000000000000000000000000000000000e2eaa99fc008f87f/mkvs_storage.badger.db $OUTPUT/mainnet-20210428-20220411/emerald-runtime-mkvs-v3.json > $OUTPUT/mainnet-20210428-20220411/emerald-runtime-mkvs-v3.log 2>&1 || true
) &

# $OUTPUT/mainnet/20220411-20231129/ (BadgerDB v3)
mkdir -p "$OUTPUT/mainnet-20220411-20231129"
(
$SAMPLER_V3 consensus-blockstore-v3 /snapshots/mainnet/20220411-20231129/consensus.mount/tendermint/data/blockstore.badger.db $OUTPUT/mainnet-20220411-20231129/consensus-blockstore-v3.json > $OUTPUT/mainnet-20220411-20231129/consensus-blockstore-v3.log 2>&1 || true
$SAMPLER_V3 consensus-evidence-v3 /snapshots/mainnet/20220411-20231129/consensus.mount/tendermint/data/evidence.badger.db $OUTPUT/mainnet-20220411-20231129/consensus-evidence-v3.json > $OUTPUT/mainnet-20220411-20231129/consensus-evidence-v3.log 2>&1 || true
$SAMPLER_V3 consensus-state-v3 /snapshots/mainnet/20220411-20231129/consensus.mount/tendermint/data/state.badger.db $OUTPUT/mainnet-20220411-20231129/consensus-state-v3.json > $OUTPUT/mainnet-20220411-20231129/consensus-state-v3.log 2>&1 || true
$SAMPLER_V3 consensus-mkvs-v3 /snapshots/mainnet/20220411-20231129/consensus.mount/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT/mainnet-20220411-20231129/consensus-mkvs-v3.json > $OUTPUT/mainnet-20220411-20231129/consensus-mkvs-v3.log 2>&1 || true
) &
(
$SAMPLER_V3 runtime-mkvs-v3 /snapshots/mainnet/20220411-20231129/cipher.mount/runtimes/000000000000000000000000000000000000000000000000e199119c992377cb/mkvs_storage.badger.db $OUTPUT/mainnet-20220411-20231129/cipher-runtime-mkvs-v3.json > $OUTPUT/mainnet-20220411-20231129/cipher-runtime-mkvs-v3.log 2>&1 || true
$SAMPLER_V3 runtime-history-v3 /snapshots/mainnet/20220411-20231129/cipher.mount/runtimes/000000000000000000000000000000000000000000000000e199119c992377cb/history.db $OUTPUT/mainnet-20220411-20231129/cipher-runtime-history-v3.json > $OUTPUT/mainnet-20220411-20231129/cipher-runtime-history-v3.log 2>&1 || true
) &
(
$SAMPLER_V3 runtime-mkvs-v3 /snapshots/mainnet/20220411-20231129/emerald.mount/runtimes/000000000000000000000000000000000000000000000000e2eaa99fc008f87f/mkvs_storage.badger.db $OUTPUT/mainnet-20220411-20231129/emerald-runtime-mkvs-v3.json > $OUTPUT/mainnet-20220411-20231129/emerald-runtime-mkvs-v3.log 2>&1 || true
$SAMPLER_V3 runtime-history-v3 /snapshots/mainnet/20220411-20231129/emerald.mount/runtimes/000000000000000000000000000000000000000000000000e2eaa99fc008f87f/history.db $OUTPUT/mainnet-20220411-20231129/emerald-runtime-history-v3.json > $OUTPUT/mainnet-20220411-20231129/emerald-runtime-history-v3.log 2>&1 || true
) &
(
$SAMPLER_V3 runtime-mkvs-v3 /snapshots/mainnet/20220411-20231129/sapphire.mount/runtimes/000000000000000000000000000000000000000000000000f80306c9858e7279/mkvs_storage.badger.db $OUTPUT/mainnet-20220411-20231129/sapphire-runtime-mkvs-v3.json > $OUTPUT/mainnet-20220411-20231129/sapphire-runtime-mkvs-v3.log 2>&1 || true
$SAMPLER_V3 runtime-history-v3 /snapshots/mainnet/20220411-20231129/sapphire.mount/runtimes/000000000000000000000000000000000000000000000000f80306c9858e7279/history.db $OUTPUT/mainnet-20220411-20231129/sapphire-runtime-history-v3.json > $OUTPUT/mainnet-20220411-20231129/sapphire-runtime-history-v3.log 2>&1 || true
) &

echo "All samplers launched in background!"
ps aux | grep badgerdb-sampler
