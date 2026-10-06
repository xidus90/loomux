package apply

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/brain/evidence"
	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// Decisions is `DECISIONS` (apply.py:189): what a human may decide here.
// Putting a case off is no decision; it writes nothing, so the caller
// handles it without calling Approve.
var Decisions = []string{"approve", "reject"}

// humanPrefix is `_HUMAN`: the one trust tier a reviewer may carry into
// `verified`.
const humanPrefix = "human:"

// The file hash and the register advance, as variables so that a test can
// fail a hash of a file that is there, or a register read that already
// succeeded once.
var (
	contentHash     = identity.ContentHash
	advanceRegister = AdvanceRegister
)

// ApplyError is a decision that cannot be carried out and must not proceed
// on a guess. Dirty names what had already been written or deleted when it
// stopped -- vault-relative (a file outside the vault, such as the register
// of a readonly area, is the path itself in slashes), in the order it was
// touched, nil when nothing was. The kind cannot say that: ProposalRefused leaves the evidence check,
// which appended to `audit.md`, and the patch, which wrote nothing.
//
// Nothing on these paths is committed, so whatever Dirty names stands
// unversioned in the working tree.
//
// A failure of the disk or of a reader comes back as an ApplyError too, so
// that it carries Dirty; the failure itself stays reachable through
// errors.Is and errors.As.
type ApplyError struct {
	Msg   string
	Dirty []string
	cause error
}

func (e *ApplyError) Error() string { return e.Msg }

// Unwrap is the failure an ApplyError was made from, nil for a refusal.
func (e *ApplyError) Unwrap() error { return e.cause }

// DirtyFiles is Dirty behind a method. The three kinds below embed
// ApplyError by value, so errors.As on *ApplyError misses them; the method
// is promoted to all of them, and a caller asks one interface whatever kind
// stopped the run.
func (e *ApplyError) DirtyFiles() []string { return e.Dirty }

// mark is promoted to the three kinds below as DirtyFiles is, so one call at
// the boundary fills Dirty whatever kind stopped the run.
func (e *ApplyError) mark(dirty []string) { e.Dirty = dirty }

// TargetMoved is a page that changed under the open case; nothing was
// written to it (arch 10.4).
type TargetMoved struct{ ApplyError }

// SourceMoved is a cited source that changed again after the case was
// formed (arch 10.4).
type SourceMoved struct{ ApplyError }

// ProposalRefused is a proposal of which no claim survived the evidence
// check, or whose surviving claims carry no diff that applies.
type ProposalRefused struct{ ApplyError }

// Options is everything a decision needs besides the case and the
// registry.
type Options struct {
	Amend    string // "" for the proposal in the case
	Decision string
	Reviewer string
	Now      time.Time
	// Scratch is the directory vcs keeps its scratch index in. The CLI hands
	// over `<state>/maintenance`, so the index is the one Python uses.
	Scratch string
	// Lookup is where the registered areas' state lies: Python's required
	// `state_dir`, through which a read-only area's register is found.
	Lookup config.ArtifactLookup
}

// Approve is `approve` (apply.py:307-358): one decision on one case, and one
// commit of exactly what it touched.
//
// The order is fixed: hash guard, source guard, evidence binding, page,
// frontmatter and registers, `log.md` and `audit.md`, the case directory off
// the disk, one commit. A failed commit is reported in Result.Warning, not
// returned as an error: by then everything else is on disk.
func Approve(casePath string, areas []config.Area, o Options) (Result, error) {
	if !slices.Contains(Decisions, o.Decision) {
		return Result{}, &ApplyError{Msg: fmt.Sprintf("decision must be one of %s, found %s",
			strings.Join(Decisions, ", "), pytext.Repr(o.Decision))}
	}
	id, human := strings.CutPrefix(o.Reviewer, humanPrefix)
	if !human || pytext.Strip(id) == "" {
		return Result{}, &ApplyError{Msg: fmt.Sprintf("reviewer must be %s<id> with a non-empty id, found %s",
			humanPrefix, pytext.Repr(o.Reviewer))}
	}
	// Python falls back to a temporary index; vcs has no such fallback, and
	// an empty directory would only surface later as a commit warning.
	if o.Scratch == "" {
		return Result{}, &ApplyError{Msg: "no scratch directory for the commit index"}
	}
	c, err := maintenance.ReadCase(casePath)
	if err != nil {
		return Result{}, failed(err, nil)
	}
	if err := checkTarget(c.Target); err != nil {
		return Result{}, failed(err, nil)
	}
	r, err := resolve(casePath, c, areas)
	if err != nil {
		return Result{}, failed(err, nil)
	}
	p := &place{anchor: r.vault, wiki: r.wiki, registers: registersOf(areas, o.Lookup)}
	if err := p.preflight(r.directory); err != nil {
		return Result{}, failed(err, p.touched)
	}
	a := approval{r: r, p: p, c: c, areas: areas, o: o}
	var result Result
	if o.Decision == "reject" {
		result, err = a.reject()
	} else {
		result, err = a.run()
	}
	if err != nil {
		// Filled here and not at each stop: `place.touched` is the record,
		// and a list kept per stop would go stale.
		return Result{}, failed(err, p.touched)
	}
	return result, nil
}

// failed is the `except (ApplyError, OSError)` of `approve`: every error
// leaves as an ApplyError of its kind, carrying what was touched.
func failed(err error, touched []string) error {
	var dirty []string // nil, not empty, when nothing was touched
	dirty = append(dirty, touched...)
	if stopped, ok := err.(interface{ mark([]string) }); ok {
		stopped.mark(dirty)
		return err
	}
	var gate *gateError
	if errors.As(err, &gate) {
		return &ApplyError{Msg: gate.msg, Dirty: dirty}
	}
	return &ApplyError{Msg: err.Error(), Dirty: dirty, cause: err}
}

// approval is `_apply` (apply.py:730-811) with what it reads.
type approval struct {
	r     resolved
	p     *place
	c     maintenance.Case
	areas []config.Area
	o     Options
}

func (a approval) run() (Result, error) {
	page, err := targetPath(a.r, a.c.Target)
	if err != nil {
		return Result{}, err
	}
	if err := a.guard(page); err != nil {
		return Result{}, err
	}
	sources, err := a.guardSources()
	if err != nil {
		return Result{}, err
	}
	claims, err := a.claims()
	if err != nil {
		return Result{}, err
	}
	segments, err := a.segments()
	if err != nil {
		return Result{}, err
	}
	passed, complaints := evidence.CheckEvidence(claims, segments)
	if len(passed) == 0 {
		return Result{}, a.refuse(claims, complaints)
	}
	hunks, err := collect(passed)
	if err != nil {
		return Result{}, err
	}
	current, err := readPage(page)
	if err != nil {
		return Result{}, err
	}
	body, err := ApplyHunks(current, hunks)
	if err != nil {
		return Result{}, refusal(err.Error())
	}
	advanced, err := AdvanceFrontmatter(body, a.updates(), a.o.Reviewer, a.o.Now)
	if err != nil {
		return Result{}, &ApplyError{Msg: a.c.Target + ": " + err.Error()}
	}
	if err := a.p.write(page, advanced); err != nil {
		return Result{}, err
	}
	registers, err := a.advanceRegisters(sources)
	if err != nil {
		return Result{}, err
	}
	log := filepath.Join(a.r.wiki, "log.md")
	line := LogLine(a.o.Now, a.c.Target, len(passed), a.c.ID)
	if err := a.p.rewriteProtocol(log, func(existing string) string { return LogInsert(existing, a.o.Now, line) }); err != nil {
		return Result{}, err
	}
	audit := filepath.Join(a.r.wiki, "audit.md")
	if err := a.p.appendProtocol(audit, RenderAudit(AuditEntry{
		Now: a.o.Now, Target: a.c.Target, CaseID: a.c.ID,
		Claims: headings(claims), Complaints: complaints,
		Decided: "freigegeben durch " + a.o.Reviewer,
		Changed: fmt.Sprintf("%s, %d Behauptung(en) eingearbeitet", a.c.Target, len(passed)),
	})); err != nil {
		return Result{}, err
	}
	// Off the disk before the commit, and here: vcs reads removals from the
	// index and never touches the working tree.
	if err := a.p.remove(a.r.directory); err != nil {
		return Result{}, err
	}
	var add []string
	for _, path := range []string{page, log, audit} {
		staged, err := a.p.staged(path)
		if err != nil {
			return Result{}, err
		}
		add = append(add, staged...)
	}
	sha, warning := commit(a.p.anchor, a.c.ID, "written",
		"Land the reviewed change to "+Safe(a.c.Target),
		append(add, registers...), []string{a.p.relative(a.r.directory)}, a.o.Scratch)
	return Result{Case: a.c, Decision: "approve", Written: true, Commit: sha, Warning: warning, Dropped: complaints}, nil
}

func (a approval) casePath() string { return filepath.Join(a.r.directory, "case.toml") }
func (a approval) proposal() string { return filepath.Join(a.r.directory, "proposal.md") }

// guard is `_guard` (apply.py:913-941): a page that moved since the case was
// formed is not written. `content_hash` folds CRLF, as the one that minted
// `target_hash` did, so a checkout on another platform is no move.
func (a approval) guard(page string) error {
	hash, err := contentHash(page)
	if err != nil {
		return err
	}
	if hash == a.c.TargetHash {
		return nil
	}
	if err := a.halt(MovedNote, "nicht geschrieben (Zielseite bewegt)"); err != nil {
		return err
	}
	return &TargetMoved{ApplyError{Msg: page + ": changed since the case was formed; nothing was written"}}
}

// guardSources is `_guard_sources` (apply.py:944-981): a source that changed
// again after the case was formed is not certified. Sources are looked up
// in every registered area; one no register knows is skipped, and so is
// one whose file is gone. The lookup is handed on to the register advance,
// which Python makes a second time.
func (a approval) guardSources() (map[string]sourceFile, error) {
	docIDs := make([]string, 0, len(a.c.Sources))
	for _, state := range a.c.Sources {
		docIDs = append(docIDs, state.DocID)
	}
	found, err := resolveSources(a.areas, a.o.Lookup, docIDs)
	if err != nil {
		return nil, err
	}
	for _, state := range a.c.Sources {
		source, ok := found[state.DocID]
		if !ok || !isFile(source.path) {
			continue
		}
		hash, err := contentHash(source.path)
		if err != nil {
			return nil, err
		}
		if hash == state.ContentHash {
			continue
		}
		note := fmt.Sprintf("Quelle %s hat sich seit der Fallbildung erneut geändert", source.relative)
		if err := a.halt(note, "nicht geschrieben (Quelle bewegt)"); err != nil {
			return nil, err
		}
		return nil, &SourceMoved{ApplyError{Msg: source.path + ": " + note}}
	}
	return found, nil
}

// halt records a guard's note in `case.toml` and, for an outcome not
// recorded last time, an audit block counting the claims of the case's own
// proposal. The case file is written every time; write-if-changed makes a
// repeat free.
func (a approval) halt(note, decided string) error {
	next := a.c
	next.Note = note
	if err := a.p.recordCase(a.casePath(), next); err != nil {
		return err
	}
	if !Unrecorded(a.c.Note, note) {
		return nil
	}
	claims, err := claimHeadings(a.proposal())
	if err != nil {
		return err
	}
	return a.audit(AuditEntry{Claims: claims, Decided: decided, Changed: "nichts"})
}

func (a approval) audit(entry AuditEntry) error {
	entry.Now, entry.Target, entry.CaseID = a.o.Now, a.c.Target, a.c.ID
	return a.p.appendProtocol(filepath.Join(a.r.wiki, "audit.md"), RenderAudit(entry))
}

// claims reads the proposal to approve: the amendment if there is one, the
// case's own otherwise. An amendment that is the proposal, by content, is
// refused, or the model's text could be resubmitted as the reviewer's any
// number of times, each exempt from `manual`.
func (a approval) claims() ([]evidence.Claim, error) {
	source := a.proposal()
	if a.o.Amend != "" {
		same, err := sameFile(a.o.Amend, source)
		if err != nil {
			return nil, err
		}
		if same {
			return nil, &ApplyError{Msg: a.o.Amend + ": an amendment may not be the case's own proposal"}
		}
		source = a.o.Amend
	}
	if !isFile(source) {
		return nil, &ApplyError{Msg: source + ": no proposal to approve"}
	}
	return readClaims(source)
}

// segments reads `package.md`. A package that does not read is no fault of
// the proposal, so it is a plain ApplyError and marks nothing `manual`.
func (a approval) segments() ([]evidence.Segment, error) {
	path := filepath.Join(a.r.directory, "package.md")
	text, err := pytext.ReadText(path)
	if err != nil {
		return nil, err
	}
	segments, err := evidence.ReadPackage(text)
	if err != nil {
		return nil, &ApplyError{Msg: path + ": " + err.Error()}
	}
	return segments, nil
}

// refuse is `_refuse` (apply.py:1032-1062): nothing held, so the case stays,
// with its note, and there is no second attempt (arch 10.6). `manual` is set
// only for the model's own proposal: an amendment is the reviewer's text,
// and a typo of theirs must not close the skill path.
func (a approval) refuse(claims []evidence.Claim, complaints []string) error {
	amended := a.o.Amend != ""
	note := RefusedNote
	if amended {
		note = AmendNote
	}
	next := a.c
	next.Note = note
	next.Manual = a.c.Manual || !amended
	if err := a.p.recordCase(a.casePath(), next); err != nil {
		return err
	}
	if Unrecorded(a.c.Note, note) {
		if err := a.audit(AuditEntry{
			Claims: headings(claims), Complaints: complaints,
			Decided: "verworfen (Evidenzbindung)", Changed: "nichts",
		}); err != nil {
			return err
		}
	}
	return refusal(a.c.ID + ": " + note)
}

// updates is the case's state of every source it names, for the page.
// Whether a register still knows a source does not matter here.
func (a approval) updates() []SourceUpdate {
	updates := make([]SourceUpdate, 0, len(a.c.Sources))
	for _, state := range a.c.Sources {
		updates = append(updates, SourceUpdate{DocID: state.DocID, ContentHash: state.ContentHash, Revision: state.Revision})
	}
	return updates
}

// advanceRegisters is `_advance_register` (apply.py:1181-1214): every
// register that knows one of the case's sources moves them on, in the order
// the case names them, and those inside the vault are staged. A source no
// register knows is left to the next reconciliation; a row invented here
// would mint an identity nobody can check.
//
// Each register is advanced under the lock of its area, which `reindex`
// holds from reading that register to writing it back; Python takes none,
// and a reindex in between lost the advanced row. A read-only area's stock
// is moved to loomux's state directory under the same lock first, since
// its register is written only there.
func (a approval) advanceRegisters(found map[string]sourceFile) ([]string, error) {
	var order []string
	rows := map[string]map[string]identity.Identity{}
	for _, state := range a.c.Sources {
		source, ok := found[state.DocID]
		if !ok {
			continue
		}
		if rows[source.register] == nil {
			order = append(order, source.register)
			rows[source.register] = map[string]identity.Identity{}
		}
		rows[source.register][source.relative] = identity.Identity{
			DocID: state.DocID, Relative: source.relative,
			ContentHash: state.ContentHash, Revision: state.Revision,
		}
	}
	var staged []string
	for _, register := range order {
		add, err := a.advanceRegisterLocked(register, rows[register])
		if err != nil {
			return nil, err
		}
		staged = append(staged, add...)
	}
	return staged, nil
}

// advanceRegisterLocked advances one register while the lock of every
// area writing it is held, and releases them once it is written and
// staged. The registry refuses two scopes that share a state directory,
// so no lock is taken twice here.
func (a approval) advanceRegisterLocked(register string, rows map[string]identity.Identity) ([]string, error) {
	for _, area := range a.areas {
		if registerOf(area, a.o.Lookup) != register {
			continue
		}
		release, err := config.LockArea(area, a.o.Lookup.Primary)
		if err != nil {
			return nil, err
		}
		defer release()
		if err := recoverStock(area, a.o.Lookup); err != nil {
			return nil, err
		}
	}
	text, err := advanceRegister(register, rows)
	if err != nil {
		return nil, err
	}
	if err := a.p.writeScaffold(register, text); err != nil {
		return nil, err
	}
	return a.p.staged(register)
}

// collect is `_collect` (apply.py:1065-1074): every diff fence of every
// surviving claim, as hunks. A refusal here names the claim; one from
// ApplyHunks cannot, since it concerns the hunks of all claims together.
func collect(passed []evidence.Claim) ([]Hunk, error) {
	var hunks []Hunk
	for _, claim := range passed {
		diffs, err := CollectDiff(claim.Body)
		if err != nil {
			return nil, refusal(claim.Heading + ": " + err.Error())
		}
		for _, diff := range diffs {
			parsed, err := ParseUnifiedDiff(diff)
			if err != nil {
				return nil, refusal(claim.Heading + ": " + err.Error())
			}
			hunks = append(hunks, parsed...)
		}
	}
	return hunks, nil
}

func refusal(msg string) *ProposalRefused {
	return &ProposalRefused{ApplyError{Msg: msg}}
}

// readPage is `read_text(encoding="utf-8", newline="")` folded to LF, the
// way `content_hash` folds before hashing: a page saved from a Windows
// editor passes the guard, and a diff written against its LF form must
// still apply. Only CRLF is folded; a lone CR stays, as Python leaves it.
func readPage(page string) (string, error) {
	data, err := readBytes(page)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("%s: not valid UTF-8", page)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n"), nil
}

// sameFile is `_same_file` (apply.py:1013-1029): two files with the same
// folded content. A stumble guard against passing the proposal, or a copy
// of it, as an amendment -- not a lock: one changed byte gets past it.
func sameFile(one, other string) (bool, error) {
	if !isFile(one) || !isFile(other) {
		return false, nil
	}
	first, err := contentHash(one)
	if err != nil {
		return false, err
	}
	second, err := contentHash(other)
	if err != nil {
		return false, err
	}
	return first == second, nil
}

// readClaims is a proposal's claims, read as `read_text(errors="replace")`
// reads it: bytes that are not UTF-8 are replaced rather than refused.
func readClaims(proposal string) ([]evidence.Claim, error) {
	data, err := readBytes(proposal)
	if err != nil {
		return nil, err
	}
	return evidence.ReadProposal(strings.ToValidUTF8(string(data), "�")), nil
}

func headings(claims []evidence.Claim) []string {
	var out []string
	for _, claim := range claims {
		out = append(out, claim.Heading)
	}
	return out
}
