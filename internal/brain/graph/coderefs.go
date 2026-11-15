package graph

import (
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// CodeRef is a place in the repository that a document names.
type CodeRef struct {
	From string `json:"from"`
	Path string `json:"path"`
	Line int    `json:"line"`
	End  int    `json:"end"`
}

var (
	// One pass over spans and links keeps the candidates in text order.
	refToken  = regexp.MustCompile("`([^`\\n]+)`|\\[[^\\]]*\\]\\(([^)\\s]+)\\)")
	spanPath  = regexp.MustCompile(`^([A-Za-z0-9_.\-/]+)(?::(\d{1,9})(?:-(\d{1,9}))?)?$`)
	hasScheme = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.\-]*:`)
)

// CandidateRefs lists the repository paths a document names, in text order and
// without checking that they exist. Spans resolve from the repository root,
// Markdown links from the document's folder.
func CandidateRefs(docRel, text string) []CodeRef {
	var refs []CodeRef
	for _, m := range refToken.FindAllStringSubmatch(stripFences(text), -1) {
		var ref CodeRef
		var ok bool
		if m[2] != "" {
			ref, ok = linkRef(docRel, m[2])
		} else {
			ref, ok = spanRef(m[1])
		}
		if ok {
			ref.From = docRel
			refs = append(refs, ref)
		}
	}
	return refs
}

// stripFences drops fenced code blocks, which show code instead of naming it.
func stripFences(text string) string {
	var out []string
	var fence string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case fence != "":
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
		case strings.HasPrefix(trimmed, "```"):
			fence = "```"
		case strings.HasPrefix(trimmed, "~~~"):
			fence = "~~~"
		default:
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func spanRef(span string) (CodeRef, bool) {
	m := spanPath.FindStringSubmatch(span)
	if m == nil || !strings.Contains(m[1], "/") {
		return CodeRef{}, false
	}
	p, ok := rootRelative(path.Clean(strings.TrimSuffix(m[1], "/")))
	if !ok {
		return CodeRef{}, false
	}
	// The pattern admits digits only, so the conversions cannot fail.
	line, _ := strconv.Atoi(m[2])
	end, _ := strconv.Atoi(m[3])
	return CodeRef{Path: p, Line: line, End: end}, true
}

func linkRef(docRel, target string) (CodeRef, bool) {
	target, _, _ = strings.Cut(target, "#")
	if target == "" || hasScheme.MatchString(target) || strings.HasPrefix(target, "/") {
		return CodeRef{}, false
	}
	p, ok := rootRelative(path.Join(path.Dir(docRel), target))
	return CodeRef{Path: p}, ok
}

// rootRelative accepts only a cleaned path that stays inside the repository
// and outside the directories that hold no project content.
func rootRelative(p string) (string, bool) {
	first, _, _ := strings.Cut(p, "/")
	if p == "." || path.IsAbs(p) || first == ".." || first == ".git" || first == ".loomux" {
		return "", false
	}
	return p, true
}

// ExistsExact reports whether rel names an existing file or folder under root
// with every segment spelled exactly, so a case-insensitive file system
// cannot make "Internal/web" match "internal/web".
func ExistsExact(root, rel string) bool {
	dir := root
	for _, seg := range strings.Split(rel, "/") {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return false
		}
		found := false
		for _, e := range entries {
			if e.Name() == seg {
				found = true
				break
			}
		}
		if !found {
			return false
		}
		dir = dir + "/" + seg
	}
	return true
}
