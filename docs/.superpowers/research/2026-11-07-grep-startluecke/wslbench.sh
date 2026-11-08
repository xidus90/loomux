#!/usr/bin/env bash
# SPIKE benchmark on Linux (WSL, ext4): grep -rn, git grep and loomux with all
# spike switches, on a copy of the spike worktree.
set -u
SRC=/mnt/c/Users/micro/AppData/Local/Temp/claude/5f6807b516805710/fcefffaa-a51a-403b-8401-af2bb7918cb7/scratchpad
WT=/mnt/c/Users/micro/Documents/#GIT/loomux/.claude/worktrees/spike-grep-startup
T=$HOME/bench-grep-tree
rm -rf "$T" && mkdir -p "$T" && tar -xf "$SRC/tree.tar" -C "$T"
cp "$WT/bin/spike-linux" "$HOME/spike-linux" && chmod +x "$HOME/spike-linux"
cd "$T"
git init -q && git add -A && git -c user.name=bench -c user.email=bench@localhost commit -qm tree
L="$HOME/spike-linux"
export LMX_SPIKE_PARWALK=1 LMX_SPIKE_ONEWALK=1 LMX_SPIKE_SPANS=1 LMX_SPIKE_OVERLAY=1 LMX_SPIKE_SCANPAR=1
if [ "${S10:-0}" = 1 ]; then export LMX_SPIKE_LAZYIDX=1 LMX_SPIKE_RECGOB=1 LMX_SPIKE_LAZYSPANS=1 LMX_SPIKE_GCOFF=1; fi
if [ "${S12:-0}" = 1 ]; then export LMX_SPIKE_LITPRE=1 LMX_SPIKE_WHOLE=1; fi
LABEL="loomux ${SPIKES:-S1-S5}"
s=$(date +%s%N); "$L" graph build --no-reuse >/dev/null 2>&1; e=$(date +%s%N)
echo "build loomux $(( (e-s)/1000000 )) ms"
echo "uname $(uname -r) grep $(grep --version | head -1)"

measure() { # name, then the command
  local name=$1; shift
  "$@" >/dev/null 2>&1
  local ts=()
  for _ in 1 2 3 4 5 6 7; do
    local s e; s=$(date +%s%N); "$@" >/tmp/bench-out 2>/dev/null; e=$(date +%s%N)
    ts+=($(( (e-s)/100000 )))
  done
  local med; med=$(printf '%s\n' "${ts[@]}" | sort -n | sed -n 4p)
  printf '%-14s %-24s median %5d.%d ms  lines %s\n' "$P" "$name" $((med/10)) $((med%10)) "$(wc -l </tmp/bench-out)"
}

for spec in "literal|func Search|" "alternation|Search|Grep|" "ignore-case|func search|-i" "no-literal|[A-Z]{3}\\w+|"; do
  P=${spec%%|*}; rest=${spec#*|}; I=${rest##*|}; PAT=${rest%|*}
  measure "grep -rnE ." grep -rnE $I "$PAT" .
  measure "grep -rnE excl" grep -rnE --exclude-dir=.git --exclude-dir=.loomux $I "$PAT" .
  measure "git grep -nE" git grep -nE $I "$PAT"
  measure "$LABEL" "$L" graph grep -- "$PAT" --json --channel cloud $I --max-hits 300
  echo "               loomux hits: $(grep -oE '"(total_hits|truncated_hits|index)": *("[a-z]+"|[0-9]+)' /tmp/bench-out | tr '\n' ' ')"
done
