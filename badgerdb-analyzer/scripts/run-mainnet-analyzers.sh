#!/bin/bash
# Script to run all badgerdb-analyzers on Mainnet snapshots
set -euo pipefail

ANALYZER_V2="./bin/badgerdb-analyzer-v2"
ANALYZER_V3="./bin/badgerdb-analyzer-v3"
pre_run() {
  mkdir -p "${OUTPUT_DIR}"
  "$(dirname "${0}")"/snapshots-extract.sh --extract "${DATA_DIR%.dir}.tar.zst"
  echo "Processing: ${DATA_DIR}..."
}
post_run() {
  "$(dirname "${0}")"/snapshots-extract.sh --clean "${DATA_DIR%.dir}.tar.zst"
  echo "Processing done."
  echo
}


# # mainnet-20201001-20201118 (BadgerDB v2)
# (
# DATA_DIR="/snapshots/mainnet/20201001-20201118/consensus.dir"
# OUTPUT_DIR="./outputs/mainnet-$(basename $(dirname "${DATA_DIR}"))"
# pre_run
# $ANALYZER_V2 consensus-blockstore-v2 $DATA_DIR/tendermint/data/blockstore.badger.db $OUTPUT_DIR/consensus-blockstore-v2.json > $OUTPUT_DIR/consensus-blockstore-v2.log 2>&1 || true
# $ANALYZER_V2 consensus-evidence-v2 $DATA_DIR/tendermint/data/evidence.badger.db $OUTPUT_DIR/consensus-evidence-v2.json > $OUTPUT_DIR/consensus-evidence-v2.log 2>&1 || true
# $ANALYZER_V2 consensus-state-v2 $DATA_DIR/tendermint/data/state.badger.db $OUTPUT_DIR/consensus-state-v2.json > $OUTPUT_DIR/consensus-state-v2.log 2>&1 || true
# $ANALYZER_V2 consensus-mkvs-v2 $DATA_DIR/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT_DIR/consensus-mkvs-v2.json > $OUTPUT_DIR/consensus-mkvs-v2.log 2>&1 || true
# post_run
# )

# # mainnet-20201118-20210428 (BadgerDB v2)
# (
# DATA_DIR="/snapshots/mainnet/20201118-20210428/consensus.dir"
# OUTPUT_DIR="./outputs/mainnet-$(basename $(dirname "${DATA_DIR}"))"
# pre_run
# $ANALYZER_V2 consensus-blockstore-v2 $DATA_DIR/tendermint/data/blockstore.badger.db $OUTPUT_DIR/consensus-blockstore-v2.json > $OUTPUT_DIR/consensus-blockstore-v2.log 2>&1 || true
# $ANALYZER_V2 consensus-evidence-v2 $DATA_DIR/tendermint/data/evidence.badger.db $OUTPUT_DIR/consensus-evidence-v2.json > $OUTPUT_DIR/consensus-evidence-v2.log 2>&1 || true
# $ANALYZER_V2 consensus-state-v2 $DATA_DIR/tendermint/data/state.badger.db $OUTPUT_DIR/consensus-state-v2.json > $OUTPUT_DIR/consensus-state-v2.log 2>&1 || true
# $ANALYZER_V2 consensus-mkvs-v2 $DATA_DIR/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT_DIR/consensus-mkvs-v2.json > $OUTPUT_DIR/consensus-mkvs-v2.log 2>&1 || true
# post_run
# )

# # mainnet-20210428-20220411 (BadgerDB v3)
# (
# DATA_DIR="/snapshots/mainnet/20210428-20220411/consensus.dir"
# OUTPUT_DIR="./outputs/mainnet-$(basename $(dirname "${DATA_DIR}"))"
# pre_run
# $ANALYZER_V3 consensus-blockstore-v3 $DATA_DIR/tendermint/data/blockstore.badger.db $OUTPUT_DIR/consensus-blockstore-v3.json > $OUTPUT_DIR/consensus-blockstore-v3.log 2>&1 || true
# $ANALYZER_V3 consensus-evidence-v3 $DATA_DIR/tendermint/data/evidence.badger.db $OUTPUT_DIR/consensus-evidence-v3.json > $OUTPUT_DIR/consensus-evidence-v3.log 2>&1 || true
# $ANALYZER_V3 consensus-state-v3 $DATA_DIR/tendermint/data/state.badger.db $OUTPUT_DIR/consensus-state-v3.json > $OUTPUT_DIR/consensus-state-v3.log 2>&1 || true
# $ANALYZER_V3 consensus-mkvs-v3 $DATA_DIR/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT_DIR/consensus-mkvs-v3.json > $OUTPUT_DIR/consensus-mkvs-v3.log 2>&1 || true
# post_run
# )
# (
# DATA_DIR="/snapshots/mainnet/20210428-20220411/cipher.dir"
# OUTPUT_DIR="./outputs/mainnet-$(basename $(dirname "${DATA_DIR}"))"
# pre_run
# $ANALYZER_V3 runtime-mkvs-v3 $DATA_DIR/runtimes/000000000000000000000000000000000000000000000000e199119c992377cb/mkvs_storage.badger.db $OUTPUT_DIR/cipher-runtime-mkvs-v3.json > $OUTPUT_DIR/cipher-runtime-mkvs-v3.log 2>&1 || true
# $ANALYZER_V3 runtime-history-v3 $DATA_DIR/runtimes/000000000000000000000000000000000000000000000000e199119c992377cb/history.db $OUTPUT_DIR/cipher-runtime-history-v3.json > $OUTPUT_DIR/cipher-runtime-history-v3.log 2>&1 || true
# post_run
# )
# (
# DATA_DIR="/snapshots/mainnet/20210428-20220411/emerald.dir"
# OUTPUT_DIR="./outputs/mainnet-$(basename $(dirname "${DATA_DIR}"))"
# pre_run
# $ANALYZER_V3 runtime-history-v3 $DATA_DIR/runtimes/000000000000000000000000000000000000000000000000e2eaa99fc008f87f/history.db $OUTPUT_DIR/emerald-runtime-history-v3.json > $OUTPUT_DIR/emerald-runtime-history-v3.log 2>&1 || true
# $ANALYZER_V3 runtime-mkvs-v3 $DATA_DIR/runtimes/000000000000000000000000000000000000000000000000e2eaa99fc008f87f/mkvs_storage.badger.db $OUTPUT_DIR/emerald-runtime-mkvs-v3.json > $OUTPUT_DIR/emerald-runtime-mkvs-v3.log 2>&1 || true
# post_run
# )

# mainnet-20220411-20231129 (BadgerDB v3)
(
DATA_DIR="/snapshots/mainnet/20220411-20231129/consensus.dir"
OUTPUT_DIR="./outputs/mainnet-$(basename $(dirname "${DATA_DIR}"))"
pre_run
$ANALYZER_V3 consensus-blockstore-v3 $DATA_DIR/tendermint/data/blockstore.badger.db $OUTPUT_DIR/consensus-blockstore-v3.json > $OUTPUT_DIR/consensus-blockstore-v3.log 2>&1 || true
$ANALYZER_V3 consensus-evidence-v3 $DATA_DIR/tendermint/data/evidence.badger.db $OUTPUT_DIR/consensus-evidence-v3.json > $OUTPUT_DIR/consensus-evidence-v3.log 2>&1 || true
$ANALYZER_V3 consensus-state-v3 $DATA_DIR/tendermint/data/state.badger.db $OUTPUT_DIR/consensus-state-v3.json > $OUTPUT_DIR/consensus-state-v3.log 2>&1 || true
$ANALYZER_V3 consensus-mkvs-v3 $DATA_DIR/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT_DIR/consensus-mkvs-v3.json > $OUTPUT_DIR/consensus-mkvs-v3.log 2>&1 || true
post_run
)
(
DATA_DIR="/snapshots/mainnet/20220411-20231129/cipher.dir"
OUTPUT_DIR="./outputs/mainnet-$(basename $(dirname "${DATA_DIR}"))"
pre_run
$ANALYZER_V3 runtime-mkvs-v3 $DATA_DIR/runtimes/000000000000000000000000000000000000000000000000e199119c992377cb/mkvs_storage.badger.db $OUTPUT_DIR/cipher-runtime-mkvs-v3.json > $OUTPUT_DIR/cipher-runtime-mkvs-v3.log 2>&1 || true
$ANALYZER_V3 runtime-history-v3 $DATA_DIR/runtimes/000000000000000000000000000000000000000000000000e199119c992377cb/history.db $OUTPUT_DIR/cipher-runtime-history-v3.json > $OUTPUT_DIR/cipher-runtime-history-v3.log 2>&1 || true
post_run
)
(
DATA_DIR="/snapshots/mainnet/20220411-20231129/emerald.dir"
OUTPUT_DIR="./outputs/mainnet-$(basename $(dirname "${DATA_DIR}"))"
pre_run
$ANALYZER_V3 runtime-mkvs-v3 $DATA_DIR/runtimes/000000000000000000000000000000000000000000000000e2eaa99fc008f87f/mkvs_storage.badger.db $OUTPUT_DIR/emerald-runtime-mkvs-v3.json > $OUTPUT_DIR/emerald-runtime-mkvs-v3.log 2>&1 || true
$ANALYZER_V3 runtime-history-v3 $DATA_DIR/runtimes/000000000000000000000000000000000000000000000000e2eaa99fc008f87f/history.db $OUTPUT_DIR/emerald-runtime-history-v3.json > $OUTPUT_DIR/emerald-runtime-history-v3.log 2>&1 || true
post_run
)
(
DATA_DIR="/snapshots/mainnet/20220411-20231129/sapphire.dir"
OUTPUT_DIR="./outputs/mainnet-$(basename $(dirname "${DATA_DIR}"))"
pre_run
$ANALYZER_V3 runtime-mkvs-v3 $DATA_DIR/runtimes/000000000000000000000000000000000000000000000000f80306c9858e7279/mkvs_storage.badger.db $OUTPUT_DIR/sapphire-runtime-mkvs-v3.json > $OUTPUT_DIR/sapphire-runtime-mkvs-v3.log 2>&1 || true
$ANALYZER_V3 runtime-history-v3 $DATA_DIR/runtimes/000000000000000000000000000000000000000000000000f80306c9858e7279/history.db $OUTPUT_DIR/sapphire-runtime-history-v3.json > $OUTPUT_DIR/sapphire-runtime-history-v3.log 2>&1 || true
post_run
)

echo "All samplers finished!"

ls -ald outputs/*/* || true
