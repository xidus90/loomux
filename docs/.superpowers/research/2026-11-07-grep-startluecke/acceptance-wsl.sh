#!/usr/bin/env bash
# Acceptance benchmark on Linux (WSL, ext4): loomux graph grep in check mode
# against grep -rnE with the artefact directories excluded. One warm-up run,
# median of 11, interleaved.
# Usage: acceptance-wsl.sh <tree.tar> <loomux linux binary>
set -eu
T=$HOME/bench-grep-tree
rm -rf "$T" && mkdir -p "$T" && tar -xf "$1" -C "$T"
cp "$2" "$HOME/loomux-acceptance" && chmod +x "$HOME/loomux-acceptance"
L=$HOME/loomux-acceptance
cd "$T"
git init -q && git add -A && git -c user.name=bench -c user.email=bench@localhost commit -qm tree
s=$(date +%s%N); "$L" graph build --no-reuse >/dev/null 2>&1; e=$(date +%s%N)
echo "files $(git ls-files | wc -l) build $(( (e-s)/1000000 )) ms"
echo "uname $(uname -r) grep $(grep --version | head -1)"

median() { sort -n | sed -n 6p; }
fmt() { printf '%d.%d' $(($1/10)) $(($1%10)); }
ms() { local s e; s=$(date +%s%N); "$@" >"$OUT" 2>/dev/null || true; e=$(date +%s%N); echo $(( (e-s)/100000 )); }

for spec in "literal|func Search|" "alternation|Search|Grep|" "ignore-case|func search|-i" "no-literal|[A-Z]{3}\\w+|"; do
  P=${spec%%|*}; rest=${spec#*|}; PAT=${rest%|*}
  I=(); [ -z "${rest##*|}" ] || I=(-i)
  G=(grep -rnE --exclude-dir=.git --exclude-dir=.loomux "${I[@]}" "$PAT" .)
  X=("$L" graph grep -- "$PAT" --json --channel cloud "${I[@]}" --max-hits 300)
  OUT=/tmp/acc-g; "${G[@]}" >/dev/null 2>&1 || true
  OUT=/tmp/acc-x; "${X[@]}" >/dev/null 2>&1 || true
  g=(); x=()
  for _ in 1 2 3 4 5 6 7 8 9 10 11; do
    OUT=/tmp/acc-g; g+=("$(ms "${G[@]}")")
    OUT=/tmp/acc-x; x+=("$(ms "${X[@]}")")
  done
  gm=$(printf '%s\n' "${g[@]}" | median); xm=$(printf '%s\n' "${x[@]}" | median)
  hits=$(grep -oE '"(total_hits|truncated_hits|index)": *("[a-z]+"|[0-9]+)' /tmp/acc-x | tr '\n' ' ')
  verdict=PASS; [ $((xm * 10)) -le $((gm * 12)) ] || verdict=FAIL
  printf '%-12s grep-excl %s ms (%s lines)  loomux %s ms (%s)  limit 1.2x  %s\n' \
    "$P" "$(fmt "$gm")" "$(wc -l </tmp/acc-g)" "$(fmt "$xm")" "$hits" "$verdict"
done
