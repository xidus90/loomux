// Package resolve turns the raw edges of every extracted file into edges with
// node ids and a confidence, and drops what stays ambiguous.
//
// The two-tier provenance is the original's (src/graph/resolve.ts):
//
//   - extracted -- the target is certain: a hit in the same file, an import
//     specifier, structural containment, or a package selector resolved
//     against the one package it can mean
//   - inferred  -- a bare name resolved through exactly one match across files
//
// Ambiguous means dropped, never guessed. The original's own header says why:
// name guessing "halved precision", and one same-named symbol once collected
// 1040 in-edges across 476 files, so every pull request touching it dragged a
// whole backend into its blast radius.
//
// Ported from src/graph/resolve.ts (MIT; origin under
// "Ported sources" in NOTICE.md).
package resolve

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Module is one go.mod, or one tsconfig.json or jsconfig.json with path
// aliases, of the repository.
type Module struct {
	Dir  string
	Path string    // go.mod: the module path; "" for a tsconfig
	TS   *TSConfig // nil for a go.mod
}

// TSConfig is the alias part of a tsconfig.json or jsconfig.json.
type TSConfig struct {
	BaseURL string              // relative to Dir, as written; "" without one
	Paths   map[string][]string // compilerOptions.paths as written
}

// ModuleFile says whether rel is a file Modules reads.
func ModuleFile(rel string) bool {
	switch path.Base(rel) {
	case "go.mod", "tsconfig.json", "jsconfig.json":
		return true
	}
	return false
}

// Modules reads the module directive of every go.mod and the path aliases of
// every tsconfig.json and jsconfig.json among files.
//
// Only that directive, by hand: golang.org/x/mod would parse the whole file and
// is a dependency this stage does not take. A go.mod without a module
// directive declares nothing and is skipped.
//
// Only an unreadable go.mod fails: a tsconfig.json or jsconfig.json that
// cannot be read -- a dangling symlink in an example, a locked file -- reads
// as empty, which ModulesFrom skips like any config that does not parse, and
// an unreadable tsconfig.json still hides the jsconfig.json beside it.
func Modules(root string, files []string) ([]Module, error) {
	bodies := map[string]string{}
	for _, rel := range files {
		if !ModuleFile(rel) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil && path.Base(rel) == "go.mod" {
			return nil, err
		}
		bodies[rel] = string(b)
	}
	return ModulesFrom(bodies)
}

// ModulesFrom is Modules over contents already read: the repo-relative path
// of each module file to its text, for a caller that reads the index and not
// the disk. A path ModuleFile rejects is skipped.
//
// A jsconfig.json beside a tsconfig.json is skipped, as tsserver prefers the
// tsconfig.json beside it. A config without paths and baseUrl aliases
// nothing, and one that does not parse is skipped rather than failing the
// build: unlike a go.mod, it is hand-written JSONC that tsc alone validates.
func ModulesFrom(bodies map[string]string) ([]Module, error) {
	rels := make([]string, 0, len(bodies))
	for rel := range bodies {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	var out []Module
	for _, rel := range rels {
		dir := dirOf(rel)
		switch path.Base(rel) {
		case "go.mod":
			if mod, ok := moduleDirective(bodies[rel]); ok {
				out = append(out, Module{Dir: dir, Path: mod})
			}
		case "jsconfig.json":
			if _, ok := bodies[path.Join(path.Dir(rel), "tsconfig.json")]; ok {
				continue
			}
			fallthrough
		case "tsconfig.json":
			if ts, ok := tsConfig(bodies[rel]); ok {
				out = append(out, Module{Dir: dir, TS: ts})
			}
		}
	}
	// Longest path first, so resolveImport can take the first match and have
	// the most specific module. A tsconfig has no path and keeps its place
	// behind every go.mod.
	sort.SliceStable(out, func(i, j int) bool { return len(out[i].Path) > len(out[j].Path) })
	return out, nil
}

// tsConfig reads the aliases of one tsconfig.json or jsconfig.json; false
// when it does not parse or aliases nothing.
func tsConfig(body string) (*TSConfig, bool) {
	var cfg struct {
		CompilerOptions struct {
			BaseURL string              `json:"baseUrl"`
			Paths   map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}
	// tsc strips the byte order mark Windows editors write; encoding/json
	// rejects it.
	body = strings.TrimPrefix(body, string(rune(0xfeff)))
	if json.Unmarshal([]byte(stripJSONC(body)), &cfg) != nil {
		return nil, false
	}
	o := cfg.CompilerOptions
	if o.BaseURL == "" && len(o.Paths) == 0 {
		return nil, false
	}
	return &TSConfig{BaseURL: o.BaseURL, Paths: o.Paths}, true
}

// stripJSONC turns the JSON with comments and trailing commas that tsc reads
// into JSON: comments go first, so a comma before a comment and a closing
// bracket is seen as trailing.
func stripJSONC(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '"':
			end := stringEnd(s, i)
			b.WriteString(s[i:end])
			i = end - 1
		case strings.HasPrefix(s[i:], "//"):
			// Up to the newline, which stays.
			n := strings.IndexByte(s[i:], '\n')
			if n < 0 {
				n = len(s) - i
			}
			i += n - 1
		case strings.HasPrefix(s[i:], "/*"):
			n := strings.Index(s[i+2:], "*/")
			if n < 0 {
				n = len(s)
			}
			i += n + 3
		default:
			b.WriteByte(s[i])
		}
	}
	return dropTrailingCommas(b.String())
}

// dropTrailingCommas drops each comma outside a string that only whitespace
// separates from a closing bracket.
func dropTrailingCommas(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			end := stringEnd(s, i)
			b.WriteString(s[i:end])
			i = end - 1
		case ',':
			rest := strings.TrimLeft(s[i+1:], " \t\r\n")
			if rest != "" && (rest[0] == '}' || rest[0] == ']') {
				continue
			}
			b.WriteByte(',')
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// stringEnd is the index after the closing quote of the JSON string opening
// at start, or len(s) when it never closes.
func stringEnd(s string, start int) int {
	for i := start + 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '"':
			return i + 1
		}
	}
	return len(s)
}

// nearestTSConfig is the tsconfig whose directory holds file and lies
// deepest, as tsc picks the config nearest a file; nil when none holds it.
func nearestTSConfig(mods []Module, file string) *Module {
	var best *Module
	for i := range mods {
		m := &mods[i]
		if m.TS == nil || (m.Dir != "" && !strings.HasPrefix(file, m.Dir+"/")) {
			continue
		}
		if best == nil || len(m.Dir) > len(best.Dir) {
			best = m
		}
	}
	return best
}

// moduleDirective is the argument of the first `module` line.
func moduleDirective(body string) (string, bool) {
	for _, line := range strings.Split(body, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1], true
		}
	}
	return "", false
}

// importDir is the repo-relative directory an import path names.
//
// The bool is not decoration: the root package of a root module legitimately
// lives at "", and "" is also what a caller would use for "not found". Returning
// both apart is what keeps `import "example.com/repo"` from reading as the
// standard library. The original has the same distinction and spells it with a
// sentinel; in Go the pair is the honest form.
//
// The longest module path wins, so a nested module in a monorepo beats its
// parent.
func importDir(spec string, mods []Module) (string, bool) {
	for _, m := range mods {
		if m.TS != nil {
			continue
		}
		var sub string
		switch {
		case spec == m.Path:
			sub = ""
		case strings.HasPrefix(spec, m.Path+"/"):
			sub = spec[len(m.Path)+1:]
		default:
			continue
		}
		switch {
		case m.Dir == "":
			return sub, true
		case sub == "":
			return m.Dir, true
		default:
			return m.Dir + "/" + sub, true
		}
	}
	return "", false
}

// dirOf is the directory a node's path lies in, with path.Dir's "." for a file
// at the repository root spelled as "" -- the same form importDir returns, so
// the two can be compared at all.
func dirOf(p string) string {
	dir := path.Dir(p)
	if dir == "." {
		return ""
	}
	return dir
}
