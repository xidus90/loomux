#!/usr/bin/env bash
# Records: every case under testdata/cases/3b-source (24).
# Call:    RECORD_DIR=<dir> bash record_all.sh
#          then: loomux dev import-cases --map testdata/cases/3b-map.toml
#                --from testdata/cases/3b-source --to testdata/cases/3b
# Against: ultra-brain tag `loomux-3-source` (3cc72d2), through record.sh.
# Every approve case with a repository pins its commit in git.after, the
# refusals included: it is what proves HEAD did not move.
set -uo pipefail
S="$(cd "$(dirname "$0")" && pwd)"
R() { bash "$S/record.sh" "$@"; }
ID=a-2026-09-21-5bd8

R cases/empty cases-empty "a review centre without a case" "brain-mcp cases"
R cases/three cases-three "three cases in two areas, one of them manual" "brain-mcp cases"
R cases/letter-case cases-letter-case "the queue sorted part by part in lower case: alpha, Beta, gamma, then project-a-later after project-a and before Project-B" "brain-mcp cases"
R cases/renamed-id cases-renamed "the id field and the directory name part: the directory is listed, a warning names both" "brain-mcp cases"
R cases/unreadable cases-unreadable "a case.toml that is not TOML beside a good one: the good one is listed, exit 1" "brain-mcp cases"
R cases/no-review-centre cases-no-review "no registered area declares a review centre" "brain-mcp cases"

R case/open case-open "an open case: sources, package and proposal" "brain-mcp case $ID"
R case/withheld case-withheld "a local_only area: halt line, manual, note, and the files withheld" "brain-mcp case $ID"
R case/package case-withheld "the same case opened deliberately with --package" "brain-mcp case --package $ID"
R case/missing-proposal case-missing-proposal "no proposal.md: the file is named as missing" "brain-mcp case $ID"
R case/unknown case-open "an id no case directory carries" "brain-mcp case nothing-2026-09-21-0000"
R case/ambiguous case-ambiguous "two case directories of one name in two areas" "brain-mcp case $ID"

R approve/success approve-success "a proven proposal is written, committed and followed by the technical update" "brain-mcp approve $ID" git-after
R approve/amend approve-amend "an amendment outside the case replaces the proposal" "brain-mcp approve $ID --amend {{WORLD}}/amend.md" git-after
R approve/reject approve-success "--reject: audit.md, the case removed, one commit, no technical update" "brain-mcp approve $ID --reject" git-after
R approve/defer approve-success "--defer: nothing is written" "brain-mcp approve $ID --defer" git-after
R approve/target-moved approve-target-moved "the page changed after the case was formed: a note and an audit block, nothing committed" "brain-mcp approve $ID" git-after
R approve/source-moved approve-source-moved "the source changed again after the case was formed: a note and an audit block, nothing committed" "brain-mcp approve $ID" git-after
R approve/evidence-fails approve-evidence-fails "a quote the package does not hold: the case turns manual, a note and an audit block" "brain-mcp approve $ID" git-after
R approve/hunk-mismatch approve-hunk-mismatch "a diff whose context the page does not hold: refused, nothing written" "brain-mcp approve $ID" git-after
R approve/rebase approve-rebase "a rebase stands in the repository: written, the commit refused with a warning" "brain-mcp approve $ID" git-after
R approve/no-repo approve-no-repo "a vault without a repository: written, not committed" "brain-mcp approve $ID"
R approve/empty-args approve-success "no case id" "brain-mcp approve" git-after
R approve/unknown approve-success "an id no case directory carries" "brain-mcp approve nothing-2026-09-21-0000" git-after
