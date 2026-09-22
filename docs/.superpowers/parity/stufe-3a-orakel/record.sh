#!/usr/bin/env bash
# Records: testdata/cases/3a-source/<verb>/<name>, one case per call.
# Call:    RECORD_DIR=<dir> bash record.sh <verb/name> <world> <compare> <notes> <cmd> [no-qmd]
#          record_all.sh makes every call of the stage.
# Against: ultra-brain tag `loomux-3-source` (3cc72d2): brain-mcp.exe from the
#          .venv of the checkout named in UB, which has to stand at that tag.
# Source:  the scratchpad script of the task 16 report (stage 3a); the report
#          itself shows only its shape. Changed: the scratchpad path in S,
#          which is now RECORD_DIR. WT and UB are this machine's checkouts and
#          stay as they were.
# record.sh <verb/name> <world> <compare> <notes> <cmd> [no-qmd]
# Records one stage-3a case from the Python reference, isolated from every
# productive state directory of this machine.
set -euo pipefail
# The working directory of a recording. It was a session scratchpad; set it
# to any empty directory that holds bin/loomux.exe (built from this tree) and
# fakeqmd/ (the fake qmd, built from internal/dev/fakeqmd/_qmd). sandbox/ is
# created below it.
S="${RECORD_DIR:?set RECORD_DIR to the recording directory}"
WT="C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/planung-von-3-c56c81"
UB="C:/Users/micro/Documents/#GIT/ultra-brain"
target="$1"; world="$2"; compare="$3"; notes="$4"; cmd="$5"; mode="${6:-}"
sandbox="$S/sandbox"
mkdir -p "$sandbox/localappdata" "$sandbox/loomux-state" "$sandbox/loomux-legacy" "$sandbox/xdg-state" "$sandbox/xdg-cache"
out="$WT/testdata/cases/3a-source/$target"
rm -rf "$out"
args=(dev record-case
  --argv "\"$UB/.venv/Scripts/brain-mcp.exe\""
  --env "XDG_CONFIG_HOME={{WORLD}}/xdg"
  --env "LOCALAPPDATA=$sandbox/localappdata"
  --env "XDG_STATE_HOME=$sandbox/xdg-state"
  --env "XDG_CACHE_HOME=$sandbox/xdg-cache"
  --cmd "$cmd" --world "$WT/testdata/cases/3a-worlds/$world" --out "$out"
  --notes "loomux-3-source (3cc72d2), brain-mcp over fakeqmd: $notes" --compare "$compare")
if [ "$mode" = "no-qmd" ]; then
  # No qmd anywhere on PATH: the reference finds its python by the launcher's
  # own absolute path, so System32 is all the process needs.
  args+=(--env "PATH=C:/Windows/System32")
else
  args+=(--path-prepend "$S/fakeqmd" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json")
fi
LOOMUX_STATE_DIR="$sandbox/loomux-state" LOOMUX_LEGACY_BRAIN_DIR="$sandbox/loomux-legacy" \
  "$S/bin/loomux.exe" "${args[@]}"
echo "exit=$(cat "$out/exit") $target"
