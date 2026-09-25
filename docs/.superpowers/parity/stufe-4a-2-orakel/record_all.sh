#!/usr/bin/env bash
# Records: every case under testdata/cases/4a2-source (14).
# Call:    RECORD_DIR=<dir> bash record_all.sh
#          then: loomux dev import-cases --map testdata/cases/4a2-map.toml
#                --from testdata/cases/4a2-source --to testdata/cases/4a2
# Against: ultra-brain tag `loomux-3-source` (3cc72d2), through record.sh.
# Compare: message where the reference answers with its German sentence
# (no area, no hook), data everywhere else.
set -uo pipefail
S="$(cd "$(dirname "$0")" && pwd)"
R() { bash "$S/record.sh" "$@"; }

R hook/install-consenting hook-consenting "" "one consenting area in a fresh repository" "brain-mcp hook install"
R hook/install-no-consent hook-no-consent message "the only area says on_merge = false" "brain-mcp hook install"
R hook/install-foreign hook-foreign "" "a post-merge of the user's own stands where the hook goes" "brain-mcp hook install"
R hook/install-own-earlier hook-own-earlier "" "the reference's own hook stands there, unrecorded" "brain-mcp hook install"
R hook/install-no-repository hook-no-repository "" "the consenting area is in no repository" "brain-mcp hook install"
R hook/install-orphaned hook-orphaned "" "hooks.tsv names an area the registry no longer has" "brain-mcp hook install"
R hook/install-empty hook-empty message "a registry without areas" "brain-mcp hook install"

R hook/status-installed hook-installed "" "the hook and its record, as install left them" "brain-mcp hook status"
R hook/status-own-earlier hook-own-earlier "" "the reference's own hook without a record" "brain-mcp hook status"
R hook/status-orphaned hook-orphaned "" "a record for an area the registry no longer has" "brain-mcp hook status"
R hook/status-empty hook-empty message "a registry without areas" "brain-mcp hook status"

R hook/remove-installed hook-installed "" "the hook and its record, as install left them" "brain-mcp hook remove"
R hook/remove-foreign hook-foreign message "a post-merge of the user's own, no record" "brain-mcp hook remove"
R hook/remove-empty hook-empty message "a registry without areas" "brain-mcp hook remove"
