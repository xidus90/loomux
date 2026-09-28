// Package notices writes NOTICE.md: the license of every third-party piece
// the loomux binary links -- Go's standard library, each module, each
// tree-sitter grammar whose package is imported -- and the notice of the
// embedded word frequencies. A test holds the committed file to what this
// renders, so a new dependency cannot ship without its notice.
package notices

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/brain/model"
)

// Runner runs the go command in dir.
type Runner func(dir string, args ...string) ([]byte, error)

// GoCommand runs the real go command. It resolves the graph without cgo, as
// the release builds the binary, and keeps what go said in its error: an
// exit status alone does not tell a missing go.mod from a broken one.
func GoCommand(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return out, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exit.Stderr)))
	}
	return out, err
}

const gotreesitter = "github.com/odvcencio/gotreesitter"

// noticeFile is every file a notice quotes, siblings like LICENSE-MIT or
// COPYING.LESSER included; licenseFile is the one a section cannot do without.
var noticeFile = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)^(licen[cs]e|copying|copyright|notice|patents)([._-][a-z0-9._-]+)?$`)
})

var licenseFile = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`(?i)^(licen[cs]e|copying)`) })

// readFile is the seam a test replaces to see a listed license file that
// does not read; neither chmod nor a missing file makes one under Windows.
var readFile = os.ReadFile

type pkg struct {
	ImportPath string
	Standard   bool
	Module     *struct {
		Path, Version, Dir string
		Main               bool
	}
}

type grammar struct {
	Name             string   `json:"name"`
	Repo             string   `json:"repo"`
	Ref              string   `json:"ref"`
	SPDX             string   `json:"spdx"`
	CopyrightHolders []string `json:"copyright_holders"`
	NoticeFile       string   `json:"notice_file"`
	Copyleft         bool     `json:"copyleft"`
}

// Render is NOTICE.md for the binary built from root.
func Render(root string, run Runner) (string, error) {
	out, err := run(root, "list", "-deps", "-json", "./cmd/loomux")
	if err != nil {
		return "", fmt.Errorf("go list: %w", err)
	}
	modules := map[string]pkg{}
	var grammars []string
	standard := false
	for decoder := json.NewDecoder(bytes.NewReader(out)); ; {
		var p pkg
		if err := decoder.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return "", fmt.Errorf("go list: %w", err)
		}
		switch {
		case p.Standard:
			standard = true
		case p.Module == nil || p.Module.Main:
		default:
			// The registry package carries the scanners of every grammar, the
			// copyleft ones among them unless built with
			// gotreesitter_no_copyleft, and the audit has no entry for it.
			if p.ImportPath == gotreesitter+"/grammars" {
				return "", fmt.Errorf("%s links every grammar in gotreesitter's registry, copyleft ones included; import %s/grammars/<name> instead",
					p.ImportPath, gotreesitter)
			}
			modules[p.Module.Path] = p
			// A grammar is its own package directly under grammars/; the
			// runtime, the blobs and the internal scanners are not grammars.
			if name, ok := strings.CutPrefix(p.ImportPath, gotreesitter+"/grammars/"); ok && !strings.Contains(name, "/") &&
				name != "runtime" && name != "grammar_blobs" {
				grammars = append(grammars, name)
			}
		}
	}
	var b strings.Builder
	b.WriteString("# Third-party notices\n\nThe loomux binary links the third-party software and data below. `loomux dev notices` writes this file; do not edit it.\n")
	if standard {
		goroot, err := run(root, "env", "GOROOT")
		if err != nil {
			return "", fmt.Errorf("go env: %w", err)
		}
		if err := section(&b, "Go standard library and runtime", strings.TrimSpace(string(goroot))); err != nil {
			return "", err
		}
	}
	paths := make([]string, 0, len(modules))
	for path := range modules {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	for _, path := range paths {
		m := modules[path].Module
		if err := section(&b, m.Path+" "+m.Version, m.Dir); err != nil {
			return "", err
		}
	}
	if len(grammars) > 0 {
		if err := grammarSections(&b, modules[gotreesitter].Module.Dir, grammars); err != nil {
			return "", err
		}
	}
	b.WriteString("\n" + model.ZipfNotice())
	return b.String(), nil
}

// section is one heading and every license file in dir, verbatim.
func section(b *strings.Builder, heading, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	fmt.Fprintf(b, "\n## %s\n", heading)
	found := false
	for _, e := range entries {
		// A Go file named like a notice is code: go-sdk's root holds
		// copyright_test.go.
		if e.IsDir() || !noticeFile().MatchString(e.Name()) || strings.EqualFold(filepath.Ext(e.Name()), ".go") {
			continue
		}
		text, err := readFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "\n### %s\n\n```\n%s\n```\n", e.Name(), clean(text))
		found = found || licenseFile().MatchString(e.Name())
	}
	if !found {
		return fmt.Errorf("%s: no license file in %s", heading, dir)
	}
	return nil
}

// grammarSections is each linked grammar from gotreesitter's own license
// audit, refusing a copyleft one: its terms would reach the whole binary.
func grammarSections(b *strings.Builder, dir string, names []string) error {
	data, err := os.ReadFile(filepath.Join(dir, "licenses", "grammars.json"))
	if err != nil {
		return err
	}
	var audit struct {
		Entries []grammar `json:"entries"`
	}
	if err := json.Unmarshal(data, &audit); err != nil {
		return fmt.Errorf("grammars.json: %w", err)
	}
	slices.Sort(names)
	for _, name := range names {
		i := slices.IndexFunc(audit.Entries, func(g grammar) bool { return g.Name == name })
		if i < 0 {
			return fmt.Errorf("grammar %s: not in gotreesitter's license audit", name)
		}
		g := audit.Entries[i]
		if g.Copyleft {
			return fmt.Errorf("grammar %s: %s is copyleft and would bind the whole binary", name, g.SPDX)
		}
		text, err := os.ReadFile(filepath.Join(dir, "licenses", "texts", g.SPDX+".txt"))
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "\n## tree-sitter grammar %s\n\n%s@%s, %s\n\n%s\n\n```\n%s\n```\n", name, g.Repo, g.Ref, g.SPDX,
			strings.Join(g.CopyrightHolders, "\n"), clean(text))
		// The audit names a notice file from the module's root
		// ("licenses/notices/elixir-NOTICE.txt"), not from licenses/.
		if g.NoticeFile != "" {
			notice, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(g.NoticeFile)))
			if err != nil {
				return err
			}
			fmt.Fprintf(b, "\n```\n%s\n```\n", clean(notice))
		}
	}
	return nil
}

func clean(text []byte) string {
	return strings.TrimRight(strings.ReplaceAll(string(text), "\r\n", "\n"), "\n")
}
