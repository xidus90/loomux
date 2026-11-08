#!/usr/bin/env bash
# SPIKE: phase trace of loomux graph grep on Linux.
cd ~/bench-grep-tree || exit 1
export LMX_SPIKE_PARWALK=1 LMX_SPIKE_ONEWALK=1 LMX_SPIKE_SPANS=1 LMX_SPIKE_OVERLAY=1 LMX_SPIKE_SCANPAR=1
export LMX_SPIKE_LAZYIDX=1 LMX_SPIKE_RECGOB=1 LMX_SPIKE_LAZYSPANS=1 LMX_SPIKE_GCOFF=1 LMX_SPIKE_TRACE=1
for _ in 1 2 3 4; do
  s=$(date +%s%N)
  ~/spike-linux graph grep -- "func Search" --json --channel cloud 2>&1 >/dev/null | awk '{printf "%s=%s ", $2, $3}'
  e=$(date +%s%N)
  echo " total=$(( (e-s)/1000000 ))"
done
for _ in 1 2 3 4; do
  s=$(date +%s%N); ~/spike-linux version >/dev/null; e=$(date +%s%N)
  echo -n "$(( (e-s)/1000000 )) "
done
echo "ms version"
for _ in 1 2 3; do
  s=$(date +%s%N); /bin/true; e=$(date +%s%N)
  echo -n "$(( (e-s)/1000000 )) "
done
echo "ms /bin/true"
