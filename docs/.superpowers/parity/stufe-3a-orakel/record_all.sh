#!/usr/bin/env bash
# Records: every case under testdata/cases/3a-source (28).
# Call:    RECORD_DIR=<dir> bash record_all.sh
#          then: loomux dev import-cases --map testdata/cases/3a-map.toml
#                --from testdata/cases/3a-source --to testdata/cases/3a
# Against: ultra-brain tag `loomux-3-source` (3cc72d2), through record.sh.
# Source:  the scratchpad script of the task 16 report (stage 3a), 23 calls.
#          Changed: S, the scratchpad path, now finds record.sh beside this
#          file. Added at the end: the five calls recorded later -- three
#          from the task 16b report, the stale-HEAD case of its fix round,
#          and reindex/globs from the S1-S6 fix round --, with the notes the
#          recordings carry. The S1-S6 report adds PYTHONDONTWRITEBYTECODE=1
#          to the environment of that one recording; record.sh does not set it.
# Every stage-3a recording, in one place.
set -uo pipefail
S="$(cd "$(dirname "$0")" && pwd)"
R() { bash "$S/record.sh" "$@"; }

R reconcile/empty-vault vault-empty "" "a vault whose one area holds no source" "brain-mcp reconcile"
R reconcile/unchanged-source vault-unchanged "" "unchanged sources in a writable and a read-only area raise no case" "brain-mcp reconcile"
R reconcile/changed-source vault-changed "" "a changed source opens a case for the page derived from it" "brain-mcp reconcile"
R reconcile/superseded-case vault-standing "" "a standing case on an older source state is replaced, its proposal carried over" "brain-mcp reconcile"
R reconcile/no-review-centre vault-no-review "" "no registered area declares a review centre" "brain-mcp reconcile"
R reconcile/two-review-centres vault-two-reviews "" "two areas declare two review centres" "brain-mcp reconcile"
R reconcile/unreadable-case vault-unreadable-case "" "a case file in the review centre does not read" "brain-mcp reconcile"
R reconcile/missing-manifest vault-missing-manifest "" "a registered area without a manifest is skipped" "brain-mcp reconcile"

R reindex/nothing-open vault-unchanged message "the catch-up is green and opens nothing, then both areas are indexed" "brain-mcp reindex"
R reindex/cases-opened vault-changed message "the catch-up opens a case, and the index run still happens" "brain-mcp reindex"
R reindex/no-review-centre vault-no-review message "no review centre: a warning, then the index run" "brain-mcp reindex"
R reindex/catch-up-fails vault-two-reviews message "the catch-up fails over two review centres: exit 1, nothing indexed" "brain-mcp reindex"
R reindex/missing-manifest vault-missing-manifest message "an area without a manifest is skipped by the catch-up and the index run" "brain-mcp reindex"

R embed/pending embed-pending "" "vectors are pending: qmd embed runs" "brain-mcp embed"
R embed/nothing-pending embed-nothing "" "nothing is pending: qmd embed runs all the same" "brain-mcp embed"
R embed/no-qmd embed-no-qmd "" "no qmd on PATH" "brain-mcp embed" no-qmd

R area-add/new-area area-new message "a new area with -y" "brain-mcp init -y --path {{WORLD}}/repo-new"
R area-add/known-scope area-known-scope message "the scope is registered already" "brain-mcp init -y --path {{WORLD}}/repo-new --scope project/a"
R area-add/without-yes area-new message "without -y, stdin empty: nothing is asked" "brain-mcp init --path {{WORLD}}/repo-new"
R area-add/no-reindex area-new message "--no-reindex" "brain-mcp init -y --no-reindex --path {{WORLD}}/repo-new"
R area-add/no-registry area-no-registry message "a machine on which no area was ever registered" "brain-mcp init -y --path {{WORLD}}/repo-new"

R reconcile/area-without-include vault-no-include "" "a read-only area whose manifest names no [index] include" "brain-mcp reconcile"
R reindex/area-without-include vault-no-include message "a read-only area whose manifest names no [index] include" "brain-mcp reindex"

R reconcile/merge-event vault-merge "" "a merge of two commits after the base becomes a case for the page that is still planned" "brain-mcp reconcile"
R reconcile/merge-behind-source-case vault-merge-standing "" "a merge steps back behind a source case standing on the same page, and its event stays in the log" "brain-mcp reconcile"
R reconcile/changed-source-baseline vault-changed-baseline "" "a changed source whose last approved state is HEAD: the package diffs against it" "brain-mcp reconcile"
R reconcile/changed-source-stale-head vault-changed-stale-head "" "a changed source whose HEAD is a state the register does not name: no baseline, the package says so" "brain-mcp reconcile"
R reindex/globs vault-globs message "a non-ASCII review centre and a two-star include and exclude that match with zero segments (S1, S2)" "brain-mcp reindex"
