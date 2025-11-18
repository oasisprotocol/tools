#!/bin/bash
# Script to unmount .tar.zst snapshot archives and remove overlays
set -euo pipefail

cd "$(dirname "$0")"
search_dir="${1:-.}"

find "$search_dir" -name "*.tar.zst" -type f | while read -r archive; do
  basename="${archive%.tar.zst}"
  mountpoint="${basename}.mount"
  overlay="${basename}.overlay"

  if mountpoint -q "$mountpoint" 2>/dev/null; then
    echo "Unmounting: $mountpoint"
    fusermount -u "$mountpoint"
    rm -rf "$mountpoint" "$overlay"
  fi
done

echo "Done!"
