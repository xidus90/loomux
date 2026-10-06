package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"
)

// readBytes is os.ReadFile, as a variable so that a test can fail a read of
// a file that is there -- the one failure `is_file` followed by
// `read_text` leaves open.
var readBytes = os.ReadFile

// reject is `_reject` (apply.py:698-727), healed: a human said no to the
// proposal, and the sources the case was formed over are acknowledged all
// the same -- the page's `sources[]` and the register move on, so the next
// reconcile does not open the same case again. A source that moved once more
// since the case was formed halts it as it halts an approval: that newer
// state was never under review. The page's text, `generated` and `verified`
// stay: it was neither regenerated nor confirmed.
//
// The audit block is written even so -- a rejected proposal stays
// traceable -- and the case directory is removed in the same commit: leaving
// the case standing would put a decided case back in the queue on every
// later pass. No proposal is needed and the page's hash is not checked; a
// missing proposal is counted as no claims.
func (a approval) reject() (Result, error) {
	// targetPlace and not targetPath: a page deleted or renamed since the
	// case was formed must not keep the rejection from closing it, as the
	// unhealed path, which never read the page, did not either.
	page, err := targetPlace(a.r, a.c.Target)
	if err != nil {
		return Result{}, err
	}
	sources, err := a.guardSources()
	if err != nil {
		return Result{}, err
	}
	claims, err := claimHeadings(a.proposal())
	if err != nil {
		return Result{}, err
	}
	advanced, changed := "", false
	if isFile(page) {
		current, err := readPage(page)
		if err != nil {
			return Result{}, err
		}
		advanced, changed, err = AdvanceSources(current, a.updates())
		if err != nil {
			return Result{}, &ApplyError{Msg: a.c.Target + ": " + err.Error(), cause: err}
		}
	}
	var add []string
	if changed {
		if err := a.p.write(page, advanced); err != nil {
			return Result{}, err
		}
		staged, err := a.p.staged(page)
		if err != nil {
			return Result{}, err
		}
		add = append(add, staged...)
	}
	registers, err := a.advanceRegisters(sources)
	if err != nil {
		return Result{}, err
	}
	audit := filepath.Join(a.r.wiki, "audit.md")
	block := RenderAudit(AuditEntry{
		Now: a.o.Now, Target: a.c.Target, CaseID: a.c.ID, Claims: claims,
		Decided: "abgelehnt durch " + a.o.Reviewer, Changed: "nichts",
	})
	if err := a.p.appendProtocol(audit, block); err != nil {
		return Result{}, err
	}
	if err := a.p.remove(a.r.directory); err != nil {
		return Result{}, err
	}
	staged, err := a.p.staged(audit)
	if err != nil {
		return Result{}, err
	}
	add = append(add, staged...)
	sha, warning := commit(a.p.anchor, a.c.ID, "decision recorded",
		"Reject the proposed change to "+Safe(a.c.Target),
		append(add, registers...), []string{a.p.relative(a.r.directory)}, a.o.Scratch)
	return Result{Case: a.c, Decision: "reject", Written: false, Commit: sha, Warning: warning}, nil
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
	return p.rewriteProtocol(path, func(existing string) string { return Append(existing, block) })
}

// rewriteProtocol reads a protocol the way appendProtocol describes and
// writes back what edit makes of it; `log.md` takes LogInsert here, where
// the reference appends it like the audit.
func (p *place) rewriteProtocol(path string, edit func(existing string) string) error {
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
	return p.writeScaffold(path, edit(existing))
}
