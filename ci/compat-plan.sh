#!/bin/sh
# Prints the job matrix of a compatibility run:
#   ci/compat-plan.sh <loomux binary>
# From the environment: EVENT (the triggering event), ENTRIES (entry keys,
# a JSON list or comma separated; empty = none named) and LABELS (the pull
# request's labels). Named entries run as named; a pull request without
# names runs what its changes touch, or everything under release:major; any
# other run, a dispatch with no names included, runs everything.
# PROBES, a JSON list of whole entries that are not in the list (what
# `dev compat scout` finds), is run instead of all of that.
set -eu
bin=${1:?usage: ci/compat-plan.sh <loomux binary>}

if [ -n "${PROBES:-}" ]; then
	file=$(mktemp)
	printf '%s' "$PROBES" >"$file"
	status=0
	"$bin" dev compat matrix --probes "$file" || status=$?
	rm -f "$file"
	exit "$status"
fi

set --
entries=$(printf '%s' "${ENTRIES:-}" | tr -d '[]" \n' | tr ',' ' ')
set -f
for key in $entries; do
	set -- "$@" --entry "$key"
done

if [ $# -gt 0 ]; then
	:
elif [ "${EVENT:-}" != pull_request ] || printf '%s' "${LABELS:-}" | grep -q 'release:major'; then
	set -- --full
else
	# ChangedSince diffs the two refs directly, so the base is the point the
	# branch left master, not master's tip. An assignment, so that set -e
	# stops at a failing merge-base instead of planning against "".
	base=$(git merge-base origin/master HEAD)
	set -- --changed-since "$base"
fi
"$bin" dev compat matrix "$@"
