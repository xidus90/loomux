package hooks

import (
	"slices"
	"strings"
)

// maxLexical is how many spellings lexicalSpellings makes of one path at
// most; every element with an alias or a trailing dot doubles them.
const maxLexical = 16

// lexicalSpellings are rel as Windows may read it, without asking the file
// system: each element without its trailing dots and blanks, which Windows
// drops, and each element that is an 8.3 alias of one of elements (the
// literal names of the protected globs) spelled as that name. rel itself is
// not among them. The default mode has no resolution, so a short name or a
// trailing dot must not hide a protected path there; strict mode resolves
// what exists, and this adds what does not exist yet.
func lexicalSpellings(rel string, elements []string) []string {
	spellings := []string{""}
	for i, part := range strings.Split(rel, "/") {
		forms := []string{part}
		// . and .. trim to nothing and stay as they are.
		if trimmed := strings.TrimRight(part, ". "); trimmed != "" {
			forms = append(forms, trimmed)
			for _, e := range elements {
				if aliasMatches(trimmed, e) {
					forms = append(forms, e)
				}
			}
		}
		var next []string
		for _, s := range spellings {
			for _, f := range forms {
				if i > 0 {
					f = s + "/" + f
				}
				if len(next) < maxLexical && !slices.Contains(next, f) {
					next = append(next, f)
				}
			}
		}
		spellings = next
	}
	return slices.DeleteFunc(spellings, func(s string) bool { return s == rel })
}

// aliasMatches says whether alias is an 8.3 short name Windows may have
// made of long: a stem of at least two of long's first six letters, or two
// of them and four hex digits, then ~ and digits, then the first three
// letters of long's extension when it has one. Leading dots and blanks of
// long are dropped, its other dots and blanks too, and + , ; = [ ] become _.
func aliasMatches(alias, long string) bool {
	alias = strings.ToUpper(alias)
	// Without a ~ the number is empty, and the name no alias.
	stem, number, _ := strings.Cut(alias, "~")
	ext := ""
	if dot := strings.LastIndexByte(number, '.'); dot >= 0 {
		number, ext = number[:dot], number[dot+1:]
	}
	if number == "" || strings.Trim(number, "0123456789") != "" {
		return false
	}
	name := strings.ToUpper(strings.TrimLeft(long, ". "))
	longExt := ""
	if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
		name, longExt = name[:dot], name[dot+1:]
	}
	name = strings.NewReplacer(" ", "", ".", "", "+", "_", ",", "_", ";", "_", "=", "_", "[", "_", "]", "_").Replace(name)
	full := name[:min(6, len(name))]
	if ext != longExt[:min(3, len(longExt))] {
		return false
	}
	hashed := len(stem) == 6 && len(full) >= 2 && stem[:2] == full[:2] && strings.Trim(stem[2:], "0123456789ABCDEF") == ""
	return len(stem) >= 2 && strings.HasPrefix(full, stem) || hashed
}

// protectedElements are the literal elements of every protected glob, in
// lower case, each once: what an 8.3 alias in a path may stand for.
func (j judge) protectedElements() []string {
	var out []string
	for _, g := range j.protectedGlobs() {
		for _, e := range strings.Split(strings.ToLower(g.glob), "/") {
			if !strings.ContainsAny(e, "*?[") && e != "" && !slices.Contains(out, e) {
				out = append(out, e)
			}
		}
	}
	return out
}
