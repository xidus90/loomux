package apply

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// The notes a decision leaves in `case.toml` when it stops before the
// commit, verbatim from apply.py:201-206. German, like every line of
// `log.md` and `audit.md`: a person reads them, no code parses them.
const (
	// MovedNote is `_MOVED_NOTE`, apply.py:201-204, the two literals joined.
	MovedNote = "die Zielseite hat sich unter dem Fall bewegt: es wurde nichts geschrieben, " +
		"der Fall steht wieder zur Prüfung an"
	// RefusedNote is `_REFUSED_NOTE`, apply.py:205.
	RefusedNote = "Evidenzbindung: kein Beleg hielt stand, der Vorschlag ist verworfen (§10.6)"
	// AmendNote is `_AMEND_NOTE`, apply.py:206.
	AmendNote = "Evidenzbindung: die eingereichte Nachbesserung hielt nicht stand"
)

const (
	// safeAllowed is `_ALLOWED`, apply.py:216: besides letters and numbers,
	// the only characters a protocol value keeps. The ellipsis and the
	// replacement are in it so that Safe's output survives Safe unchanged.
	safeAllowed = " .,:;/-_()…·"
	// safeReplacement is `_REPLACEMENT`, apply.py:217.
	safeReplacement = '·'
	// safeLineMax is `_LINE_MAX`, apply.py:218, counted in characters.
	safeLineMax = 200
)

// Safe reduces an untrusted protocol value to a name or a path, as `_safe`
// does: a whitelist, not a list of markers to strip, so no markup survives
// and the length cut cannot leave a construction open. Every character that
// is neither a letter nor a number (Unicode categories L and N, all of N)
// nor in safeAllowed becomes one `·`; a result longer than 200 characters
// keeps the first 199 and ends in `…`.
func Safe(value string) string {
	cleaned := make([]rune, 0, len(value))
	for _, char := range value {
		if strings.ContainsRune(safeAllowed, char) || unicode.IsLetter(char) || unicode.IsNumber(char) {
			cleaned = append(cleaned, char)
		} else {
			cleaned = append(cleaned, safeReplacement)
		}
	}
	if len(cleaned) > safeLineMax {
		return string(cleaned[:safeLineMax-1]) + "…"
	}
	return string(cleaned)
}

// LogLine is the block `_append_log` hands to `log.md`: the what-line of a
// decision, dated in UTC as the reference's `datetime.now(UTC)` is, ending
// in a newline.
func LogLine(now time.Time, target string, applied int, caseID string) string {
	return fmt.Sprintf("- %s — `%s`: %d Behauptung(en) eingearbeitet (Fall `%s`)\n",
		now.UTC().Format(time.DateOnly), Safe(target), applied, Safe(caseID))
}

// AuditEntry is what `_append_audit` writes about one outcome. Only the
// number of Claims enters the block.
type AuditEntry struct {
	Now                time.Time
	Target, CaseID     string
	Claims, Complaints []string
	Decided, Changed   string
}

// RenderAudit is the block `_append_audit` hands to `audit.md`: the three
// states of a decision (proposed, decided, actually changed) and one line
// per complaint, every untrusted value through Safe, ending in a newline.
func RenderAudit(e AuditEntry) string {
	lines := []string{
		fmt.Sprintf("## %s — %s (Fall `%s`)", IsoFormat(e.Now), Safe(e.Target), Safe(e.CaseID)),
		"",
		fmt.Sprintf("- vorgeschlagen: %d Behauptung(en), %d ohne Beleg", len(e.Claims), len(e.Complaints)),
		"- entschieden: " + Safe(e.Decided),
		"- tatsächlich geändert: " + Safe(e.Changed),
	}
	for _, complaint := range e.Complaints {
		lines = append(lines, "- verworfen: "+Safe(complaint))
	}
	return strings.Join(append(lines, ""), "\n")
}

// Append is the text `_append` writes: the existing protocol without its
// trailing newlines, one blank line, then block; an empty protocol, or one
// of newlines only, becomes block alone. Only `\n` is trimmed, so a `\r`
// before it stays, as Python's `rstrip("\n")` leaves it.
func Append(existing, block string) string {
	existing = strings.TrimRight(existing, "\n")
	if existing == "" {
		return block
	}
	return existing + "\n\n" + block
}

// Unrecorded reports whether note is a new outcome for a case whose last
// recorded note is last, as `_unrecorded` compares `case.note` with the
// note about to be written. The audit block of a stopped decision is
// appended only then, so one unchanged outcome repeated adds nothing; two
// outcomes alternating still add a block each time.
func Unrecorded(last, note string) bool {
	return last != note
}
