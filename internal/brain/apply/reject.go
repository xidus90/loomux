package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/brain/maintenance"
)

// readBytes is os.ReadFile, as a variable so that a test can fail a read of
// a file that is there -- the one failure `is_file` followed by
// `read_text` leaves open.
var readBytes = os.ReadFile

// reject is `_reject` (apply.py:698-727): a human said no. The wiki stays as
// it is, but the case is closed and gone.
//
// The audit block is written even so -- a rejected proposal stays
// traceable -- and the case directory is removed in the same commit, which
// carries audit.md alone: leaving the case standing would put a decided case
// back in the queue on every later pass. No proposal is needed and no hash
// is checked; a missing proposal is counted as no claims.
//
// Inherited from the reference, not healed: neither the page's `sources[]`
// nor the register is advanced, so the next reconcile opens the same case
// again (parity record, "Geerbt").
func reject(r resolved, p *place, c maintenance.Case, reviewer string, now time.Time, scratch string) (Result, error) {
	claims, err := claimHeadings(filepath.Join(r.directory, "proposal.md"))
	if err != nil {
		return Result{}, err
	}
	audit := filepath.Join(r.wiki, "audit.md")
	block := RenderAudit(AuditEntry{
		Now: now, Target: c.Target, CaseID: c.ID, Claims: claims,
		Decided: "abgelehnt durch " + reviewer, Changed: "nichts",
	})
	if err := p.appendProtocol(audit, block); err != nil {
		return Result{}, err
	}
	if err := p.remove(r.directory); err != nil {
		return Result{}, err
	}
	add, err := p.staged(audit)
	if err != nil {
		return Result{}, err
	}
	sha, warning := commit(p.anchor, c.ID, "decision recorded",
		"Reject the proposed change to "+Safe(c.Target),
		add, []string{p.relative(r.directory)}, scratch)
	return Result{Case: c, Decision: "reject", Written: false, Commit: sha, Warning: warning}, nil
}

// claimHeadings is `_claims` (apply.py:1217-1221): the claims of a proposal
// that may legitimately not be there at all. Only their number enters the
// audit block. Bytes that are not UTF-8 are replaced, as `read_text(errors=
// "replace")` does, rather than failing a decision that needs no proposal.
func claimHeadings(proposal string) ([]string, error) {
	if !isFile(proposal) {
		return nil, nil
	}
	claims, err := readClaims(proposal)
	if err != nil {
		return nil, err
	}
	return headings(claims), nil
}

// appendProtocol is `_append` (apply.py:1301-1307): block below what the
// protocol holds, the file created if the bundle never scaffolded one. The
// protocol is read as it stands, line endings and all, and strictly as
// UTF-8, as `read_text(encoding="utf-8", newline="")` reads it.
func (p *place) appendProtocol(path, block string) error {
	existing := ""
	if isFile(path) {
		data, err := readBytes(path)
		if err != nil {
			return err
		}
		if !utf8.Valid(data) {
			return fmt.Errorf("%s: not valid UTF-8", path)
		}
		existing = string(data)
	}
	return p.writeScaffold(path, Append(existing, block))
}
