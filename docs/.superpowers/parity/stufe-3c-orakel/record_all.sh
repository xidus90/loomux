#!/usr/bin/env bash
# Records: every case under testdata/cases/3c-source.
# Call:    RECORD_DIR=<dir> bash record_all.sh
#          then: loomux dev import-cases --map testdata/cases/3c-map.toml
#                --from testdata/cases/3c-source --to testdata/cases/3c
# Against: ultra-brain tag `loomux-3-source` (3cc72d2), through record.sh.
# Compare: data (stdout exact) for check, lint and types; message (exit and
# world exact) for wiki init and retype, which print the paths they wrote.
set -uo pipefail
S="$(cd "$(dirname "$0")" && pwd)"
R() { bash "$S/record.sh" "$@"; }

R go check/file-clean one-area "" "one clean page of a registered bundle" "brain check file {{WORLD}}/repo-a/wiki/a.md"
R go check/file-broken one-area "" "a page whose frontmatter does not parse, outside every bundle" "brain check file {{WORLD}}/loose/broken.md"
R go check/file-missing one-area "" "a path that does not exist" "brain check file {{WORLD}}/loose/gone.md"
R go check/file-directory one-area "" "a directory named like a page" "brain check file {{WORLD}}/loose/folder.md"
R go check/file-notes one-area "" "--notes after the path" "brain check file {{WORLD}}/repo-a/wiki/a.md --notes"
R go check/bundle-clean one-area "" "a clean bundle" "brain check bundle --scope project/a"
R go check/bundle-findings findings "" "no sources, a dead link, a link out of the area and a passed stale_after" "brain check bundle --scope project/a"
R go check/bundle-unknown one-area "" "a scope nobody registered" "brain check bundle --scope project/x"
R go check/bundle-no-scope one-area "" "no --scope" "brain check bundle"
R go check/all-federation federation "" "a signpost naming nobody and a shared area citing a project" "brain check all"
R go check/all-no-registry no-registry "" "no registry" "brain check all"
R go check/all-notes federation "" "--notes" "brain check all --notes"
R go check/code one-area "" "the fourth width of the reference, which loomux drops" "brain check code"

R py lint/sweep-clean one-area "" "one clean bundle" "brain-mcp lint"
R py lint/sweep-findings findings "" "errors and a warning" "brain-mcp lint"
R py lint/sweep-federation federation "" "the signpost's duty and the shared area's direction" "brain-mcp lint --scope all"
R py lint/scope federation "" "one area of three" "brain-mcp lint --scope project/a"
R py lint/unknown-scope refusals "" "a scope nobody registered" "brain-mcp lint --scope project/x"
R py lint/no-wiki-path refusals "" "an area without a wiki path" "brain-mcp lint --scope project/nowiki"
R py lint/wiki-missing refusals "" "a wiki path that is not a directory, named" "brain-mcp lint --scope project/gone"
R py lint/sweep-wiki-missing refusals "" "the same, found by the sweep" "brain-mcp lint"
R py lint/single-file one-area "" "one page by path: the 1a path" "brain-mcp lint {{WORLD}}/repo-a/wiki/a.md"

R py wiki-init/new refusals message "a registered wiki that does not exist yet" "brain-mcp wiki init --scope project/gone"
R py wiki-init/partial partial message "an index of its own stays, the rest is written" "brain-mcp wiki init --scope project/a"
R py wiki-init/complete one-area message "nothing missing, nothing written" "brain-mcp wiki init --scope project/a"
R py wiki-init/unknown-scope refusals message "a scope nobody registered" "brain-mcp wiki init --scope project/x"
R py wiki-init/no-wiki-path refusals message "an area without a wiki path" "brain-mcp wiki init --scope project/nowiki"

R py types/empty types-empty "" "no area with a wiki" "brain-mcp types"
R py types/three types-three "" "three areas: a declared type, an alias, an untyped page, a lower-case type" "brain-mcp types"
R py types/no-registry no-registry "" "no registry" "brain-mcp types"

R py retype/rename retype message "two pages renamed; a longer type and a quoted value stay" "brain-mcp retype --scope project/a --from \"Design Decision\" --to Decision"
R py retype/second-run retype-done message "a bundle already renamed: nothing written" "brain-mcp retype --scope project/a --from \"Design Decision\" --to Decision"
R py retype/unknown-target retype message "a target no rank knows: a warning, the run goes on" "brain-mcp retype --scope project/a --from \"Design Decision\" --to Topik"
R py retype/read-only retype message "a read-only area" "brain-mcp retype --scope project/ro --from Topic --to Decision"
R py retype/unknown-scope retype message "a scope nobody registered" "brain-mcp retype --scope project/x --from Topic --to Decision"
