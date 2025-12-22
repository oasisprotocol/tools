#!/bin/bash
# Script to run all badgerdb-analyzers on Testnet snapshots
set -euo pipefail

ANALYZER_V2="./bin/badgerdb-analyzer-v2"
ANALYZER_V3="./bin/badgerdb-analyzer-v3"
ANALYZER_V4="./bin/badgerdb-analyzer-v4"
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


# testnet-20200915-20201104 (BadgerDB v2)
(
DATA_DIR="/snapshots/testnet/20200915-20201104/consensus_testnet.dir"
OUTPUT_DIR="./outputs/testnet-$(basename $(dirname "${DATA_DIR}"))"
pre_run
$ANALYZER_V2 consensus-blockstore-v2 $DATA_DIR/tendermint/data/blockstore.badger.db $OUTPUT_DIR/consensus-blockstore-v2.json > $OUTPUT_DIR/consensus-blockstore-v2.log 2>&1 || true
$ANALYZER_V2 consensus-evidence-v2 $DATA_DIR/tendermint/data/evidence.badger.db $OUTPUT_DIR/consensus-evidence-v2.json > $OUTPUT_DIR/consensus-evidence-v2.log 2>&1 || true
$ANALYZER_V2 consensus-state-v2 $DATA_DIR/tendermint/data/state.badger.db $OUTPUT_DIR/consensus-state-v2.json > $OUTPUT_DIR/consensus-state-v2.log 2>&1 || true
$ANALYZER_V2 consensus-mkvs-v2 $DATA_DIR/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT_DIR/consensus-mkvs-v2.json > $OUTPUT_DIR/consensus-mkvs-v2.log 2>&1 || true
post_run
)

# testnet-20201104-20210203 (BadgerDB v2)
(
DATA_DIR="/snapshots/testnet/20201104-20210203/consensus_testnet.dir"
OUTPUT_DIR="./outputs/testnet-$(basename $(dirname "${DATA_DIR}"))"
pre_run
$ANALYZER_V2 consensus-blockstore-v2 $DATA_DIR/tendermint/data/blockstore.badger.db $OUTPUT_DIR/consensus-blockstore-v2.json > $OUTPUT_DIR/consensus-blockstore-v2.log 2>&1 || true
$ANALYZER_V2 consensus-evidence-v2 $DATA_DIR/tendermint/data/evidence.badger.db $OUTPUT_DIR/consensus-evidence-v2.json > $OUTPUT_DIR/consensus-evidence-v2.log 2>&1 || true
$ANALYZER_V2 consensus-state-v2 $DATA_DIR/tendermint/data/state.badger.db $OUTPUT_DIR/consensus-state-v2.json > $OUTPUT_DIR/consensus-state-v2.log 2>&1 || true
$ANALYZER_V2 consensus-mkvs-v2 $DATA_DIR/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT_DIR/consensus-mkvs-v2.json > $OUTPUT_DIR/consensus-mkvs-v2.log 2>&1 || true
post_run
)

# testnet-20210203-20210324 (BadgerDB v2)
(
DATA_DIR="/snapshots/testnet/20210203-20210324/consensus_testnet.dir"
OUTPUT_DIR="./outputs/testnet-$(basename $(dirname "${DATA_DIR}"))"
pre_run
$ANALYZER_V2 consensus-blockstore-v2 $DATA_DIR/tendermint/data/blockstore.badger.db $OUTPUT_DIR/consensus-blockstore-v2.json > $OUTPUT_DIR/consensus-blockstore-v2.log 2>&1 || true
$ANALYZER_V2 consensus-evidence-v2 $DATA_DIR/tendermint/data/evidence.badger.db $OUTPUT_DIR/consensus-evidence-v2.json > $OUTPUT_DIR/consensus-evidence-v2.log 2>&1 || true
$ANALYZER_V2 consensus-state-v2 $DATA_DIR/tendermint/data/state.badger.db $OUTPUT_DIR/consensus-state-v2.json > $OUTPUT_DIR/consensus-state-v2.log 2>&1 || true
$ANALYZER_V2 consensus-mkvs-v2 $DATA_DIR/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT_DIR/consensus-mkvs-v2.json > $OUTPUT_DIR/consensus-mkvs-v2.log 2>&1 || true
post_run
)

# testnet-20210324-20210413 (BadgerDB v2)
(
DATA_DIR="/snapshots/testnet/20210324-20210413/consensus_testnet.dir"
OUTPUT_DIR="./outputs/testnet-$(basename $(dirname "${DATA_DIR}"))"
pre_run
$ANALYZER_V2 consensus-blockstore-v2 $DATA_DIR/tendermint/data/blockstore.badger.db $OUTPUT_DIR/consensus-blockstore-v2.json > $OUTPUT_DIR/consensus-blockstore-v2.log 2>&1 || true
$ANALYZER_V2 consensus-evidence-v2 $DATA_DIR/tendermint/data/evidence.badger.db $OUTPUT_DIR/consensus-evidence-v2.json > $OUTPUT_DIR/consensus-evidence-v2.log 2>&1 || true
$ANALYZER_V2 consensus-state-v2 $DATA_DIR/tendermint/data/state.badger.db $OUTPUT_DIR/consensus-state-v2.json > $OUTPUT_DIR/consensus-state-v2.log 2>&1 || true
$ANALYZER_V2 consensus-mkvs-v2 $DATA_DIR/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT_DIR/consensus-mkvs-v2.json > $OUTPUT_DIR/consensus-mkvs-v2.log 2>&1 || true
post_run
)

# testnet-20210413-20220303 (BadgerDB v3)
(
DATA_DIR="/snapshots/testnet/20210413-20220303/consensus_testnet.dir"
OUTPUT_DIR="./outputs/testnet-$(basename $(dirname "${DATA_DIR}"))"
pre_run
$ANALYZER_V3 consensus-blockstore-v3 $DATA_DIR/tendermint/data/blockstore.badger.db _DIR-20220303/consensus-blockstore-v3.json > $OUTPUT_DIR/consensus-blockstore-v3.log 2>&1 || true
$ANALYZER_V3 consensus-evidence-v3 $DATA_DIR/tendermint/data/evidence.badger.db $OUTPUT_DIR/consensus-evidence-v3.json > $OUTPUT_DIR/consensus-evidence-v3.log 2>&1 || true
$ANALYZER_V3 consensus-state-v3 $DATA_DIR/tendermint/data/state.badger.db $OUTPUT_DIR/consensus-state-v3.json > $OUTPUT_DIR/consensus-state-v3.log 2>&1 || true
$ANALYZER_V3 consensus-mkvs-v3 $DATA_DIR/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT_DIR/consensus-mkvs-v3.json > $OUTPUT_DIR/consensus-mkvs-v3.log 2>&1 || true
post_run
)

# testnet-20220303-20231012 (BadgerDB v3)
(
DATA_DIR="/snapshots/testnet/20220303-20231012/consensus_testnet.dir"
OUTPUT_DIR="./outputs/testnet-$(basename $(dirname "${DATA_DIR}"))"
pre_run
$ANALYZER_V3 consensus-blockstore-v3 $DATA_DIR/tendermint/data/blockstore.badger.db $OUTPUT_DIR/consensus-blockstore-v3.json > $OUTPUT_DIR/consensus-blockstore-v3.log 2>&1 || true
$ANALYZER_V3 consensus-evidence-v3 $DATA_DIR/tendermint/data/evidence.badger.db $OUTPUT_DIR/consensus-evidence-v3.json > $OUTPUT_DIR/consensus-evidence-v3.log 2>&1 || true
$ANALYZER_V3 consensus-state-v3 $DATA_DIR/tendermint/data/state.badger.db $OUTPUT_DIR/consensus-state-v3.json > $OUTPUT_DIR/consensus-state-v3.log 2>&1 || true
$ANALYZER_V3 consensus-mkvs-v3 $DATA_DIR/tendermint/abci-state/mkvs_storage.badger.db $OUTPUT_DIR/consensus-mkvs-v3.json > $OUTPUT_DIR/consensus-mkvs-v3.log 2>&1 || true
post_run
)
(
DATA_DIR="/snapshots/testnet/20220303-20231012/cipher_testnet.dir"
OUTPUT_DIR="./outputs/testnet-$(basename $(dirname "${DATA_DIR}"))"
pre_run
$ANALYZER_V3 runtime-mkvs-v3 $DATA_DIR/runtimes/0000000000000000000000000000000000000000000000000000000000000000/mkvs_storage.badger.db $OUTPUT_DIR/cipher-runtime-mkvs-v3.json > $OUTPUT_DIR/cipher-runtime-mkvs-v3.log 2>&1 || true
$ANALYZER_V3 runtime-history-v3 $DATA_DIR/runtimes/0000000000000000000000000000000000000000000000000000000000000000/history.db $OUTPUT_DIR/cipher-runtime-history-v3.json > $OUTPUT_DIR/cipher-runtime-history-v3.log 2>&1 || true
post_run
)
(
DATA_DIR="/snapshots/testnet/20220303-20231012/emerald_testnet.dir"
OUTPUT_DIR="./outputs/testnet-$(basename $(dirname "${DATA_DIR}"))"
pre_run
$ANALYZER_V3 runtime-mkvs-v3 $DATA_DIR/runtimes/00000000000000000000000000000000000000000000000072c8215e60d5bca7/mkvs_storage.badger.db $OUTPUT_DIR/emerald-runtime-mkvs-v3.json > $OUTPUT_DIR/emerald-runtime-mkvs-v3.log 2>&1 || true
$ANALYZER_V3 runtime-history-v3 $DATA_DIR/runtimes/00000000000000000000000000000000000000000000000072c8215e60d5bca7/history.db $OUTPUT_DIR/emerald-runtime-history-v3.json > $OUTPUT_DIR/emerald-runtime-history-v3.log 2>&1 || true
post_run
)
(
DATA_DIR="/snapshots/testnet/20220303-20231012/sapphire_testnet.dir"
OUTPUT_DIR="./outputs/testnet-$(basename $(dirname "${DATA_DIR}"))"
pre_run
$ANALYZER_V3 runtime-mkvs-v3 $DATA_DIR/runtimes/000000000000000000000000000000000000000000000000a6d1e3ebf60dff6c/mkvs_storage.badger.db $OUTPUT_DIR/sapphire-runtime-mkvs-v3.json > $OUTPUT_DIR/sapphire-runtime-mkvs-v3.log 2>&1 || true
$ANALYZER_V3 runtime-history-v3 $DATA_DIR/runtimes/000000000000000000000000000000000000000000000000a6d1e3ebf60dff6c/history.db $OUTPUT_DIR/sapphire-runtime-history-v3.json > $OUTPUT_DIR/sapphire-runtime-history-v3.log 2>&1 || true
post_run
)

echo "All samplers finished!"

ls -ald outputs/*/* || true
