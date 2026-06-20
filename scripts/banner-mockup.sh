#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source_file="$script_dir/banner.ansi"

case "${1:-}" in
  --no-color|--mono)
    sed -E $'s/\x1b\\[[0-9;]*m//g' "$source_file"
    ;;
  ""|--color)
    cat "$source_file"
    printf '\033[0m'
    ;;
  *)
    printf 'usage: %s [--color|--no-color|--mono]\n' "${0##*/}" >&2
    exit 2
    ;;
esac
