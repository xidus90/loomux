package query

import (
	"fmt"
	"strings"

	"github.com/xidus90/loomux/internal/code/skeleton"
)

// SkeletonAnswer encapsulates the extracted entries for a file.
type SkeletonAnswer struct {
	Path    string           `json:"path"`
	Entries []skeleton.Entry `json:"entries"`
}

// SkeletonOptions configures the skeleton extraction.
type SkeletonOptions struct {
	NoRefresh bool
	Keep      func(path string) bool
}

// Skeleton resolves file and extracts its symbol signatures, types, and line spans.
func Skeleton(root, file string, opts SkeletonOptions) (SkeletonAnswer, []string, error) {
	g, notes, err := loadGraph(root, opts.NoRefresh)
	if err != nil {
		return SkeletonAnswer{}, notes, err
	}

	resolvedPath, entries, err := skeleton.Extract(g, file, opts.Keep)
	if err != nil {
		return SkeletonAnswer{}, notes, err
	}
	return SkeletonAnswer{Path: resolvedPath, Entries: entries}, notes, nil
}

// SkeletonReport formats SkeletonAnswer for human-readable CLI display.
func SkeletonReport(a SkeletonAnswer) string {
	if a.Path == "" {
		return "no file\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s:\n", a.Path)
	if len(a.Entries) == 0 {
		b.WriteString("  (no symbols defined)\n")
		return b.String()
	}
	for _, e := range a.Entries {
		sig := e.Signature
		if sig == "" {
			sig = e.Name
		}
		fmt.Fprintf(&b, "  %-10s %s (%s)\n", e.Span, sig, e.Kind)
	}
	return b.String()
}
