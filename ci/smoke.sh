#!/bin/sh
# Smoke test of a built loomux binary: the three answers every build must
# give, whatever platform or forge runs it.
set -u
bin=${1:?usage: ci/smoke.sh <binary>}
fail=0
out=$("$bin" version) || { echo "smoke: version exited $?" >&2; fail=1; }
case $out in
"loomux "*) ;;
*) echo "smoke: version answered '$out'" >&2; fail=1 ;;
esac
"$bin" help >/dev/null || { echo "smoke: help exited $?" >&2; fail=1; }
"$bin" gibtesnicht >/dev/null 2>&1
rc=$?
[ "$rc" -eq 2 ] || { echo "smoke: unknown command exited $rc, want 2" >&2; fail=1; }
exit $fail
