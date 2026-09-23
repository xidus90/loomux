#!/usr/bin/env bash
# Records: testdata/cases/3c-source/<verb>/<name>, one case per call.
# Call:    RECORD_DIR=<dir> bash record.sh <go|py> <verb/name> <world> <compare> <notes> <cmd>
#          record_all.sh makes every call of the stage.
# Against: ultra-brain tag `loomux-3-source` (3cc72d2). `go` runs the Go
#          binary built from the tag (brain check has no Python form, E4),
#          `py` runs brain-mcp.exe from the .venv of the checkout in UB, which
#          has to stand at that tag.
# Source:  stufe-3b-orakel/record.sh, without fakeqmd (no command of this
#          stage searches) and with the choice of reference.
set -euo pipefail
# The working directory of a recording: any empty directory holding
# bin/loomux.exe, built from this tree. sandbox/ is created below it.
S="${RECORD_DIR:?set RECORD_DIR to the recording directory}"
WT="$(cd "$(dirname "$0")/../../../.." && pwd)"
UB="C:/Users/micro/Documents/#GIT/ultra-brain"
BRAIN="${BRAIN_EXE:-C:/Users/micro/Documents/#GIT/brain-3c.exe}"
form="$1"; target="$2"; world="$3"; compare="$4"; notes="$5"; cmd="$6"
sandbox="$S/sandbox"
mkdir -p "$sandbox/localappdata" "$sandbox/loomux-state" "$sandbox/loomux-legacy" "$sandbox/xdg-state" "$sandbox/xdg-cache"
out="$WT/testdata/cases/3c-source/$target"
rm -rf "$out"
if [ "$form" = "go" ]; then
  reference=(--exe "$BRAIN")
  said="brain.exe"
else
  reference=(--argv "\"$UB/.venv/Scripts/brain-mcp.exe\"")
  said="brain-mcp"
fi
LOOMUX_STATE_DIR="$sandbox/loomux-state" LOOMUX_LEGACY_BRAIN_DIR="$sandbox/loomux-legacy" \
  "$S/bin/loomux.exe" dev record-case "${reference[@]}" \
  --env "XDG_CONFIG_HOME={{WORLD}}/xdg" \
  --env "LOCALAPPDATA=$sandbox/localappdata" \
  --env "XDG_STATE_HOME=$sandbox/xdg-state" \
  --env "XDG_CACHE_HOME=$sandbox/xdg-cache" \
  --cmd "$cmd" --world "$WT/testdata/cases/3c-worlds/$world" --out "$out" \
  --notes "loomux-3-source (3cc72d2), $said: $notes" --compare "$compare"
echo "exit=$(cat "$out/exit") $target"
