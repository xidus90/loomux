#!/usr/bin/env bash
# Records: testdata/cases/3b-source/<verb>/<name>, one case per call.
# Call:    RECORD_DIR=<dir> bash record.sh <verb/name> <world> <notes> <cmd> [git-after]
#          record_all.sh makes every call of the stage.
# Against: ultra-brain tag `loomux-3-source` (3cc72d2): brain-mcp.exe from the
#          .venv of the checkout named in UB, which has to stand at that tag.
# Source:  stufe-3a-orakel/record.sh, with the worktree of this stage, the
#          3b directories, no --compare (every 3b case compares data) and a
#          fifth argument that pins the commit in git.after.
set -euo pipefail
# The working directory of a recording: any empty directory that holds
# bin/loomux.exe (built from this tree) and fakeqmd/qmd.exe (the fake qmd,
# built from internal/dev/fakeqmd/_qmd). sandbox/ is created below it.
S="${RECORD_DIR:?set RECORD_DIR to the recording directory}"
WT="$(cd "$(dirname "$0")/../../../.." && pwd)"
UB="C:/Users/micro/Documents/#GIT/ultra-brain"
target="$1"; world="$2"; notes="$3"; cmd="$4"; mode="${5:-}"
sandbox="$S/sandbox"
mkdir -p "$sandbox/localappdata" "$sandbox/loomux-state" "$sandbox/loomux-legacy" "$sandbox/xdg-state" "$sandbox/xdg-cache"
out="$WT/testdata/cases/3b-source/$target"
rm -rf "$out"
args=(dev record-case
  --argv "\"$UB/.venv/Scripts/brain-mcp.exe\""
  --env "XDG_CONFIG_HOME={{WORLD}}/xdg"
  --env "LOCALAPPDATA=$sandbox/localappdata"
  --env "XDG_STATE_HOME=$sandbox/xdg-state"
  --env "XDG_CACHE_HOME=$sandbox/xdg-cache"
  --path-prepend "$S/fakeqmd" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json"
  --cmd "$cmd" --world "$WT/testdata/cases/3b-worlds/$world" --out "$out"
  --notes "loomux-3-source (3cc72d2), brain-mcp over fakeqmd: $notes")
if [ "$mode" = "git-after" ]; then
  args+=(--git-after)
fi
LOOMUX_STATE_DIR="$sandbox/loomux-state" LOOMUX_LEGACY_BRAIN_DIR="$sandbox/loomux-legacy" \
  "$S/bin/loomux.exe" "${args[@]}"
echo "exit=$(cat "$out/exit") $target"
