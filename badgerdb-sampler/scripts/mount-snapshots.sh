#!/bin/bash
# Script to mount .tar.zst snapshot archives without extracting
#
#   ./mount-snapshots.sh testnet
#
# Manually:
#   ./ratarmount -o allow_other,uid=1000,gid=1000 -w "testnet/20220303-20231012/cipher_testnet.overlay" "testnet/20220303-20231012/cipher_testnet.tar.zst" "testnet/20220303-20231012/cipher_testnet.mount"
#   fusermount -u "testnet/20220303-20231012/cipher_testnet.mount" && rm -rf "testnet/20220303-20231012/cipher_testnet.mount" "testnet/20220303-20231012/cipher_testnet.overlay"
#
set -euo pipefail

cd "$(dirname "$0")"
search_dir="${1:-.}"

find "$search_dir" -name "*.tar.zst" -type f | while read -r archive; do
  basename="${archive%.tar.zst}"
  mountpoint="${basename}.mount"
  overlay="${basename}.overlay"

  if mountpoint -q "$mountpoint" 2>/dev/null; then
    echo "Already mounted: $mountpoint"
    continue
  fi

  echo "Mounting: $archive -> $mountpoint"
  mkdir -p "$mountpoint"
  ./ratarmount -o allow_other,uid=1000,gid=1000 -w "$overlay" "$archive" "$mountpoint"
done

echo "Done!"
