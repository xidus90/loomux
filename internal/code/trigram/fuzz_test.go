package trigram

import (
	"regexp"
	"testing"
)

// FuzzCandidatesSuperset is the correctness net of the whole index: for every
// pattern whose literal conditions the index understands, every file the real
// regexp matches has to be among the candidates. A candidate set that misses a
// matching file would silently drop a hit; a candidate set that is too large
// only costs time.
func FuzzCandidatesSuperset(f *testing.F) {
	files := []File{
		{Rel: "a.txt", Data: []byte("abcde")},
		{Rel: "b.txt", Data: []byte("abcxx")},
		{Rel: "c.txt", Data: []byte("zzzzz")},
		{Rel: "d.txt", Data: []byte("bcd")},
		{Rel: "e.txt", Data: []byte("Search|Grep")},
		{Rel: "f.txt", Data: []byte("func Search() {}")},
	}
	ix, err := decode(buildTest(f, files))
	if err != nil {
		f.Fatalf("decode: %v", err)
	}
	f.Add("abc")
	f.Add("abcd")
	f.Add("abc|bcd")
	f.Add("(?i)ABC")
	f.Add(`bcd[\s\S]x`)
	f.Add("func (Search|Grep)")
	f.Fuzz(func(t *testing.T, pattern string) {
		sets, ok, err := Literals(pattern, false, false)
		if err != nil || !ok {
			return
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return
		}
		if _, used := ix.Candidates(sets); !used {
			return
		}
		candidates, _ := ix.Candidates(sets)
		has := make(map[string]bool, len(candidates))
		for _, rel := range candidates {
			has[rel] = true
		}
		for _, file := range files {
			if re.Match(file.Data) && !has[file.Rel] {
				t.Fatalf("pattern %q matches %s, but the candidates are %v", pattern, file.Rel, candidates)
			}
		}
	})
}
