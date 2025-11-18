#!/bin/bash
# Script to download Oasis node snapshots for Testnet
set -euo pipefail

curl -Lo testnet/20200915-20201104/README.md https://snapshots.oasis.io/node/testnet/20200915-20201104/README.md
curl -Lo testnet/20200915-20201104/consensus_testnet.tar.zst.list https://snapshots.oasis.io/node/testnet/20200915-20201104/consensus_testnet.tar.zst.list
curl -Lo testnet/20200915-20201104/consensus_testnet.tar.zst https://snapshots.oasis.io/node/testnet/20200915-20201104/consensus_testnet.tar.zst

curl -Lo testnet/20201104-20210203/README.md https://snapshots.oasis.io/node/testnet/20201104-20210203/README.md
curl -Lo testnet/20201104-20210203/consensus_testnet.tar.zst.list https://snapshots.oasis.io/node/testnet/20201104-20210203/consensus_testnet.tar.zst.list
curl -Lo testnet/20201104-20210203/consensus_testnet.tar.zst https://snapshots.oasis.io/node/testnet/20201104-20210203/consensus_testnet.tar.zst

curl -Lo testnet/20210203-20210324/README.md https://snapshots.oasis.io/node/testnet/20210203-20210324/README.md
curl -Lo testnet/20210203-20210324/consensus_testnet.tar.zst.list https://snapshots.oasis.io/node/testnet/20210203-20210324/consensus_testnet.tar.zst.list
curl -Lo testnet/20210203-20210324/consensus_testnet.tar.zst https://snapshots.oasis.io/node/testnet/20210203-20210324/consensus_testnet.tar.zst

curl -Lo testnet/20210324-20210413/README.md https://snapshots.oasis.io/node/testnet/20210324-20210413/README.md
curl -Lo testnet/20210324-20210413/consensus_testnet.tar.zst.list https://snapshots.oasis.io/node/testnet/20210324-20210413/consensus_testnet.tar.zst.list
curl -Lo testnet/20210324-20210413/consensus_testnet.tar.zst https://snapshots.oasis.io/node/testnet/20210324-20210413/consensus_testnet.tar.zst

curl -Lo testnet/20210413-20220303/README.md https://snapshots.oasis.io/node/testnet/20210413-20220303/README.md
curl -Lo testnet/20210413-20220303/consensus_testnet.tar.zst.list https://snapshots.oasis.io/node/testnet/20210413-20220303/consensus_testnet.tar.zst.list
curl -Lo testnet/20210413-20220303/consensus_testnet.tar.zst https://snapshots.oasis.io/node/testnet/20210413-20220303/consensus_testnet.tar.zst

curl -Lo testnet/20220303-20231012/README.md https://snapshots.oasis.io/node/testnet/20220303-20231012/README.md
curl -Lo testnet/20220303-20231012/consensus_testnet.tar.zst.list https://snapshots.oasis.io/node/testnet/20220303-20231012/consensus_testnet.tar.zst.list
curl -Lo testnet/20220303-20231012/consensus_testnet.tar.zst https://snapshots.oasis.io/node/testnet/20220303-20231012/consensus_testnet.tar.zst
curl -Lo testnet/20220303-20231012/cipher_testnet.tar.zst.list https://snapshots.oasis.io/node/testnet/20220303-20231012/cipher_testnet.tar.zst.list
curl -Lo testnet/20220303-20231012/cipher_testnet.tar.zst https://snapshots.oasis.io/node/testnet/20220303-20231012/cipher_testnet.tar.zst
curl -Lo testnet/20220303-20231012/emerald_testnet.tar.zst.list https://snapshots.oasis.io/node/testnet/20220303-20231012/emerald_testnet.tar.zst.list
curl -Lo testnet/20220303-20231012/emerald_testnet.tar.zst https://snapshots.oasis.io/node/testnet/20220303-20231012/emerald_testnet.tar.zst
curl -Lo testnet/20220303-20231012/sapphire_testnet.tar.zst.list https://snapshots.oasis.io/node/testnet/20220303-20231012/sapphire_testnet.tar.zst.list
curl -Lo testnet/20220303-20231012/sapphire_testnet.tar.zst https://snapshots.oasis.io/node/testnet/20220303-20231012/sapphire_testnet.tar.zst
