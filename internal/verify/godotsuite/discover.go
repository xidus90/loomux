// Package godotsuite finds what check godot-suite has to run and guard in a
// Godot project.
package godotsuite

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/verify/linecov"
)

var (
	suiteLine = regexp.MustCompile(`(?m)^\s*extends\s+GdUnitTestSuite\b`)
	classLine = regexp.MustCompile(`(?m)^\s*class_name\s+([A-Za-z_][A-Za-z0-9_]*)`)
	cacheName = regexp.MustCompile(`"class"\s*:\s*&?"([^"]+)"`)
	resPath   = regexp.MustCompile(`"(res://[^"]+\.gd)"`)
)

const bom = "\xef\xbb\xbf"

// readFile is a variable so a test can fail one read; no portable file
// refuses reads for its owner.
var readFile = os.ReadFile

// walkGD visits the project's own GDScript: addons, the import cache and
// nested checkouts belong to someone else.
func walkGD(root string, fn func(rel string, data []byte) error) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if p != root && (rel == "addons" || d.Name() == ".godot" || isCheckout(p)) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(p) != ".gd" {
			return nil
		}
		data, err := readFile(p)
		if err != nil {
			return err
		}
		return fn(rel, bytes.TrimPrefix(data, []byte(bom)))
	})
}

func isCheckout(dir string) bool {
	_, err := os.Lstat(filepath.Join(dir, ".git"))
	return err == nil
}

// Suites lists the res:// paths of the gdUnit4 suites, sorted.
func Suites(root string) ([]string, error) {
	var out []string
	err := walkGD(root, func(rel string, data []byte) error {
		if suiteLine.Match(data) {
			out = append(out, "res://"+rel)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

// ClassCacheFindings compares the declared class names with the editor's
// class cache; rootArg is the --root the rescan hint repeats.
func ClassCacheFindings(root, rootArg string) []string {
	rescan := "run loomux godot rescan"
	if rootArg != "." {
		rescan += " --root " + rootArg
	}
	const cache = ".godot/global_script_class_cache.cfg"
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(cache)))
	if os.IsNotExist(err) {
		return []string{cache + " does not exist; " + rescan}
	}
	if err != nil {
		return []string{"unreadable class cache: " + err.Error()}
	}
	have := map[string]bool{}
	for _, m := range cacheName.FindAllSubmatch(data, -1) {
		have[string(m[1])] = true
	}
	var missing []string
	err = walkGD(root, func(rel string, src []byte) error {
		if m := classLine.FindSubmatch(src); m != nil {
			if name := string(m[1]); !have[name] {
				missing = append(missing, fmt.Sprintf("%s (%s)", name, rel))
			}
		}
		return nil
	})
	if err != nil {
		return []string{"unreadable scripts: " + err.Error()}
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{"class_name not in the class cache: " + strings.Join(missing, ", ") + "; " + rescan}
}

// NeedsDisk says whether an autoload script or scene is one Nano Coverage would
// instrument; those load before the in-memory hook runs, so only the files
// on disk can carry the counters.
func NeedsDisk(root string) (bool, error) {
	cfg, err := os.ReadFile(filepath.Join(root, "project.godot"))
	if err != nil {
		return false, err
	}
	autoload, found, err := linecov.GodotCfg(cfg, "autoload")
	if err != nil || !found {
		return false, err
	}
	// The first read parsed the whole file, so this one cannot fail.
	nano, _, _ := linecov.GodotCfg(cfg, "nano_coverage")
	patterns, ok := nano["general/ignore_paths"]
	if !ok {
		patterns = []string{"**/*_test.gd"}
	}
	var ignored []linecov.Matcher
	for _, p := range patterns {
		m, err := linecov.NanoMatcher(p)
		if err != nil {
			return false, err
		}
		ignored = append(ignored, m)
	}
	addons := len(nano["general/ignore_addons"]) != 1 || nano["general/ignore_addons"][0] != "false"
	for _, vals := range autoload {
		for _, v := range vals {
			rel, ok := strings.CutPrefix(strings.TrimPrefix(v, "*"), "res://")
			if !ok || (addons && strings.HasPrefix(rel, "addons/")) {
				continue
			}
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
				continue
			}
			if !anyMatch(ignored, rel) {
				return true, nil
			}
		}
	}
	return false, nil
}

func anyMatch(ms []linecov.Matcher, rel string) bool {
	for _, m := range ms {
		if m(rel) {
			return true
		}
	}
	return false
}

// HookConflict names a coverage session hook registered in project.godot, or
// returns "". The value is a Dictionary literal over several lines, so the
// paths come from the raw text, not from the section reader.
func HookConflict(root string) (string, error) {
	cfg, err := os.ReadFile(filepath.Join(root, "project.godot"))
	if err != nil {
		return "", err
	}
	_, rest, ok := strings.Cut(string(cfg), "[gdunit4]")
	if !ok {
		return "", nil
	}
	section, _, _ := strings.Cut(rest, "\n[")
	_, value, ok := strings.Cut(section, "hooks/session_hooks=")
	if !ok {
		return "", nil
	}
	value, _, _ = strings.Cut(value, ")")
	for _, m := range resPath.FindAllStringSubmatch(value, -1) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(m[1], "res://"))))
		if err != nil {
			continue
		}
		if bytes.Contains(data, []byte("CoverageApi")) || bytes.Contains(data, []byte("NanoCoverage")) {
			return "project.godot registers a coverage session hook, " + m[1] +
				"; remove it from [gdunit4] hooks/session_hooks and set [nano_coverage] integrations/gdunit4=false, or the Godot editor registers it again", nil
		}
	}
	return "", nil
}

// loadProfiles is a variable so a test can fail the profile load.
var loadProfiles = linecov.LoadProfiles

// instrumentedMarker is the first line nanocoverage leaves in a script it
// instrumented; the profile in linecov owns it.
func instrumentedMarker() (string, error) {
	ps, err := loadProfiles()
	if err != nil {
		return "", err
	}
	for _, p := range ps {
		if p.Name == "nanocoverage" && p.Instrumented != "" {
			return p.Instrumented, nil
		}
	}
	return "", errors.New("no nanocoverage profile names its instrumentation marker")
}

// Instrumented lists the scripts a killed run left instrumented.
func Instrumented(root string) ([]string, error) {
	marker, err := instrumentedMarker()
	if err != nil {
		return nil, err
	}
	var out []string
	err = walkGD(root, func(rel string, data []byte) error {
		first, _, _ := bytes.Cut(data, []byte("\n"))
		if string(bytes.TrimSuffix(first, []byte("\r"))) == marker {
			out = append(out, rel)
		}
		return nil
	})
	return out, err
}
