package query

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
)

// AuditOptions picks the change and when an untested area is a finding.
type AuditOptions struct {
	Base            string
	Cached          bool
	Threshold       int
	SkipTestCallers bool
}

// AuditFinding is an area no changed test reaches, with the seeds that have
// at least the threshold's callers.
type AuditFinding struct {
	Path   string       `json:"path"`
	Signal blast.Signal `json:"signal"`
	Seeds  []blast.Seed `json:"seeds"` // only the seeds at or above the threshold
}

// AuditAnswer is the audit of the change git names in Range.
type AuditAnswer struct {
	Range    string         `json:"range"`
	Areas    int            `json:"areas"`
	Findings []AuditFinding `json:"findings"`
}

// Audit is blast-audit: an area is a finding when no changed test reaches it
// (none or stale) and one of its seeds has at least Threshold callers. It never
// rebuilds the graph; that is graph-fresh's job, and two rebuilds would meet
// at the same lock.
func Audit(root string, opts AuditOptions) (AuditAnswer, []string, error) {
	if opts.Threshold < 1 {
		return AuditAnswer{}, nil, errors.New("threshold must be at least 1")
	}
	ans, g, notes, err := blastWith(root, BlastOptions{Base: opts.Base, Cached: opts.Cached, NoRefresh: true})
	if err != nil {
		return AuditAnswer{}, notes, err
	}
	out := AuditAnswer{Range: ans.Range, Areas: len(ans.Areas)}
	var x *blast.Index
	if opts.SkipTestCallers {
		x = blast.New(g)
	}
	for _, a := range ans.Areas {
		if a.Signal != blast.SignalNone && a.Signal != blast.SignalStale {
			continue
		}
		f := AuditFinding{Path: a.Path, Signal: a.Signal}
		for _, s := range a.Seeds {
			n := s.InDegree
			if x != nil {
				n = x.InDegreeWhere(s.Node.ID, func(m *model.Node) bool { return m != nil && !blast.IsTestPath(m.Path) })
			}
			if n >= opts.Threshold {
				f.Seeds = append(f.Seeds, blast.Seed{Node: s.Node, InDegree: n})
			}
		}
		if len(f.Seeds) > 0 {
			out.Findings = append(out.Findings, f)
		}
	}
	return out, notes, nil
}

// AuditReport is one line per finding; without one, a line that says so.
func AuditReport(a AuditAnswer, threshold int) string {
	if len(a.Findings) == 0 {
		return fmt.Sprintf("no area at or above %d callers lacks a changed test (%d areas)\n", threshold, a.Areas)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "blast audit: %s, threshold %d\n", a.Range, threshold)
	for _, f := range a.Findings {
		seeds := make([]string, len(f.Seeds))
		for i, s := range f.Seeds {
			seeds[i] = fmt.Sprintf("%s in-degree %d", s.Node.Name, s.InDegree)
		}
		fmt.Fprintf(&b, "%s [%s]: %s\n", f.Path, f.Signal, strings.Join(seeds, ", "))
	}
	return b.String()
}
