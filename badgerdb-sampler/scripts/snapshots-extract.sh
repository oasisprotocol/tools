#!/bin/bash
# Script to extract or mount .tar.zst snapshot archives
#
# Usage:
#   ./snapshots-extract.sh [--extract|--mount|--clean] <snapshot1> [<snapshot2>...]
#
# Examples:
#   ./snapshots-extract.sh --extract testnet/*/*.tar.zst
#   ./snapshots-extract.sh --mount testnet/*/*.tar.zst
#   ./snapshots-extract.sh --clean testnet/*/*.tar.zst
set -euo pipefail

usage() {
  echo "Usage: $0 [--extract|--mount|--clean] <snapshot1> [<snapshot2>...]"
  exit 1
}

mode="${1:-}"
if [[ "$mode" != "--extract" && "$mode" != "--mount" && "$mode" != "--clean" ]]; then
  usage
fi
shift

if [[ $# -eq 0 ]]; then
  echo "Error: No snapshot files specified"
  usage
fi

for archive in "$@"; do
  basename="${archive%.tar.zst}"
  rootdir="${basename}.dir"
  overlaydir="${basename}.overlay"

  if [[ "$mode" == "--extract" ]]; then
    # Extract mode
    if [[ -d "$rootdir" ]]; then
      echo "Already extracted: $rootdir"
      continue
    fi
    echo "Extracting: $archive -> $rootdir"
    mkdir -p "$rootdir"
    tar --zstd -C "$rootdir" -xf "$archive"

  elif [[ "$mode" == "--mount" ]]; then
    # Mount mode
    if mountpoint -q "$rootdir" 2>/dev/null; then
      echo "Already mounted: $rootdir"
      continue
    fi
    echo "Mounting: $archive -> $rootdir"
    mkdir -p "$rootdir"
    ./ratarmount -o allow_other,uid=1000,gid=1000 -w "$overlaydir" "$archive" "$rootdir"

  elif [[ "$mode" == "--clean" ]]; then
    # Cleanup mode
    echo "Cleaning: $basename"
    if mountpoint -q "$rootdir" 2>/dev/null; then
      timeout 120 fusermount -u "$rootdir" || sudo umount -f "$rootdir"
    fi
    rm -rf "$rootdir" "$overlaydir"
  fi
done

echo "Extracting done."
