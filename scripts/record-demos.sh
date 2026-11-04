#!/bin/sh
# Record only known mock inputs, never a contributor's dotenv or home config.
set -eu
for tool in vhs jq ffmpeg ttyd; do
  command -v "$tool" >/dev/null 2>&1 || { echo "Demo recording requires $tool on PATH" >&2; exit 1; }
done
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
record_dir=$(mktemp -d "${TMPDIR:-/tmp}/pccli-demos.XXXXXX")
trap 'rm -rf "$record_dir"' EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM
mkdir -p "$record_dir/bin" "$record_dir/home" "$record_dir/scripts" "$record_dir/testdata/mocks" "$record_dir/docs/assets"
case "${1:-}" in
  '') cp "$repo/scripts/banner.ansi" "$repo/internal/cli/banner.ansi"
      (cd "$repo" && go build -o "$record_dir/bin/pccli" ./cmd/pccli) ;;
  --binary) [ "$#" -eq 2 ] || exit 2; cp "$2" "$record_dir/bin/pccli" ;;
  *) echo 'Usage: sh scripts/record-demos.sh [--binary /path/to/pccli]' >&2; exit 2 ;;
esac
cp "$repo"/scripts/*.tape "$record_dir/scripts/"
cp "$repo/testdata/mocks/services-library.json" "$record_dir/testdata/mocks/"
for tape in demo planning-demo attachment-demo json-demo; do
  (
    cd "$record_dir"
    env -i HOME="$record_dir/home" PATH="$record_dir/bin:$PATH" \
      LANG=C.UTF-8 TERM=xterm-256color VHS_NO_SANDBOX="${VHS_NO_SANDBOX:-false}" \
      vhs "scripts/$tape.tape"
  )
done
# Publish only a complete set. Rename also replaces older root-owned artifacts.
mkdir -p "$repo/docs/assets"
for asset in "$record_dir"/docs/assets/*; do
  name=$(basename "$asset")
  cp "$asset" "$repo/docs/assets/$name.next"
  mv -f "$repo/docs/assets/$name.next" "$repo/docs/assets/$name"
done
