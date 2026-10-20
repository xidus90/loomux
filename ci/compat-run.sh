#!/bin/sh
# Runs one compatibility entry for real and prints its report:
#   ci/compat-run.sh <loomux binary> <entry key> <root>
# The work copies go to <root>/w, the addons are read from <root>/addons
# (where ci/compat-setup.sh put them), and the report lands in
# <root>/report.md and <root>/report.json. With PROBE set, the entry is the
# one compat-setup.sh wrote to <root>/probe.json instead of the key. The
# report is also appended to GITHUB_STEP_SUMMARY when that is set. Exits with
# the run's status.
set -eu
bin=${1:?usage: ci/compat-run.sh <loomux binary> <entry key> <root>}
key=${2:?entry key}
root=${3:?root}
mkdir -p "$root"

set -- "$key"
[ -z "${PROBE:-}" ] || set -- --entry-json "$root/probe.json"
status=0
"$bin" dev compat run "$@" --work "$root/w" --addons "$root/addons" \
	--json "$root/report.json" >"$root/report.md" || status=$?
cat "$root/report.md"
# Empty when the run itself failed; the log has the error.
if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
	cat "$root/report.md" >>"$GITHUB_STEP_SUMMARY"
fi
exit "$status"
