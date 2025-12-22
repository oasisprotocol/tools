#!/bin/bash
# Script to download Oasis node snapshots for Mainnet
set -euo pipefail

curl -Lo mainnet/20201001-20201118/README.md https://snapshots.oasis.io/node/mainnet/20201001-20201118/README.md
curl -Lo mainnet/20201001-20201118/consensus.tar.zst.list https://snapshots.oasis.io/node/mainnet/20201001-20201118/consensus.tar.zst.list
curl -Lo mainnet/20201001-20201118/consensus.tar.zst https://snapshots.oasis.io/node/mainnet/20201001-20201118/consensus.tar.zst

curl -Lo mainnet/20201118-20210428/README.md https://snapshots.oasis.io/node/mainnet/20201118-20210428/README.md
curl -Lo mainnet/20201118-20210428/consensus.tar.zst.list https://snapshots.oasis.io/node/mainnet/20201118-20210428/consensus.tar.zst.list
curl -Lo mainnet/20201118-20210428/consensus.tar.zst https://snapshots.oasis.io/node/mainnet/20201118-20210428/consensus.tar.zst

curl -Lo mainnet/20210428-20220411/README.md https://snapshots.oasis.io/node/mainnet/20210428-20220411/README.md
curl -Lo mainnet/20210428-20220411/consensus.tar.zst.list https://snapshots.oasis.io/node/mainnet/20210428-20220411/consensus.tar.zst.list
curl -Lo mainnet/20210428-20220411/consensus.tar.zst https://snapshots.oasis.io/node/mainnet/20210428-20220411/consensus.tar.zst
curl -Lo mainnet/20210428-20220411/cipher.tar.zst.list https://snapshots.oasis.io/node/mainnet/20210428-20220411/cipher.tar.zst.list
curl -Lo mainnet/20210428-20220411/cipher.tar.zst https://snapshots.oasis.io/node/mainnet/20210428-20220411/cipher.tar.zst
curl -Lo mainnet/20210428-20220411/emerald.tar.zst.list https://snapshots.oasis.io/node/mainnet/20210428-20220411/emerald.tar.zst.list
curl -Lo mainnet/20210428-20220411/emerald.tar.zst https://snapshots.oasis.io/node/mainnet/20210428-20220411/emerald.tar.zst

curl -Lo mainnet/20220411-20231129/README.md https://snapshots.oasis.io/node/mainnet/20220411-20231129/README.md
curl -Lo mainnet/20220411-20231129/consensus.tar.zst.list https://snapshots.oasis.io/node/mainnet/20220411-20231129/consensus.tar.zst.list
curl -Lo mainnet/20220411-20231129/consensus.tar.zst https://snapshots.oasis.io/node/mainnet/20220411-20231129/consensus.tar.zst
curl -Lo mainnet/20220411-20231129/cipher.tar.zst.list https://snapshots.oasis.io/node/mainnet/20220411-20231129/cipher.tar.zst.list
curl -Lo mainnet/20220411-20231129/cipher.tar.zst https://snapshots.oasis.io/node/mainnet/20220411-20231129/cipher.tar.zst
curl -Lo mainnet/20220411-20231129/emerald.tar.zst.list https://snapshots.oasis.io/node/mainnet/20220411-20231129/emerald.tar.zst.list
curl -Lo mainnet/20220411-20231129/emerald.tar.zst https://snapshots.oasis.io/node/mainnet/20220411-20231129/emerald.tar.zst
curl -Lo mainnet/20220411-20231129/sapphire.tar.zst.list https://snapshots.oasis.io/node/mainnet/20220411-20231129/sapphire.tar.zst.list
curl -Lo mainnet/20220411-20231129/sapphire.tar.zst https://snapshots.oasis.io/node/mainnet/20220411-20231129/sapphire.tar.zst
