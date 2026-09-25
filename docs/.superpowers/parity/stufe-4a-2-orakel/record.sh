#!/usr/bin/env bash
# Records: testdata/cases/4a2-source/<verb>/<name>, one case per call.
# Call:    RECORD_DIR=<dir> bash record.sh <verb/name> <world> <compare> <notes> <cmd>
#          record_all.sh makes every call of the stage.
# Against: ultra-brain tag `loomux-3-source` (3cc72d2): brain-mcp.exe from the
#          .venv of the checkout in UB, which has to stand at that tag.
# Source:  stufe-3c-orakel/record.sh with the Python form only (the merge hook
#          has no Go form) and the directories of this stage. The environment
#          is 3a's: BRAIN_STATE_DIR is the staged world (the recorder sets it),
#          everything else the reference could touch points into a sandbox.
set -euo pipefail
# The working directory of a recording: any empty directory holding
# bin/loomux.exe, built from this tree. sandbox/ is created below it.
S="${RECORD_DIR:?set RECORD_DIR to the recording directory}"
WT="$(cd "$(dirname "$0")/../../../.." && pwd)"
UB="C:/Users/micro/Documents/#GIT/ultra-brain"
target="$1"; world="$2"; compare="$3"; notes="$4"; cmd="$5"
sandbox="$S/sandbox"
mkdir -p "$sandbox/localappdata" "$sandbox/loomux-state" "$sandbox/loomux-legacy" "$sandbox/xdg-state" "$sandbox/xdg-cache"
out="$WT/testdata/cases/4a2-source/$target"
rm -rf "$out"
LOOMUX_STATE_DIR="$sandbox/loomux-state" LOOMUX_LEGACY_BRAIN_DIR="$sandbox/loomux-legacy" \
  "$S/bin/loomux.exe" dev record-case --argv "\"$UB/.venv/Scripts/brain-mcp.exe\"" \
  --env "XDG_CONFIG_HOME={{WORLD}}/xdg" \
  --env "LOCALAPPDATA=$sandbox/localappdata" \
  --env "XDG_STATE_HOME=$sandbox/xdg-state" \
  --env "XDG_CACHE_HOME=$sandbox/xdg-cache" \
  --cmd "$cmd" --world "$WT/testdata/cases/4a2-worlds/$world" --out "$out" \
  --notes "loomux-3-source (3cc72d2), brain-mcp: $notes" --compare "$compare"
echo "exit=$(cat "$out/exit") $target"
