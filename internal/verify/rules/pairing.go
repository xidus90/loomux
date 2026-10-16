package rules

import (
	"cmp"
	_ "embed"
	"fmt"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/verify"
	"github.com/xidus90/loomux/internal/verify/linecov"
)

//go:embed tests.toml
var testsText string

// testProfile describes one test tool; see tests.toml.
type testProfile struct {
	Extension    string   `toml:"extension"`
	Detect       string   `toml:"detect"` // "" always; "plugin.cfg:text" a plugin.cfg mentioning text
	TestSuffixes []string `toml:"test_suffixes"`
	Mapping      string   `toml:"mapping"`
}

// testProfiles holds a function, not parsed data: embedded data is read on
// first use only.
var testProfiles = sync.OnceValues(func() (map[string]testProfile, error) { return parseTests(testsText) })

func parseTests(text string) (map[string]testProfile, error) {
	var doc struct {
		Profile map[string]testProfile `toml:"profile"`
	}
	if _, err := toml.Decode(text, &doc); err != nil {
		return nil, fmt.Errorf("tests.toml: %w", err)
	}
	seen := map[string]string{}
	for _, name := range slices.Sorted(maps.Keys(doc.Profile)) {
		p := doc.Profile[name]
		if p.Mapping != "beside" && p.Mapping != "gdunit4" || p.Extension == "" || len(p.TestSuffixes) == 0 {
			return nil, fmt.Errorf("tests.toml: profile %s: want an extension, test_suffixes and mapping beside or gdunit4", name)
		}
		if p.Mapping == "beside" && p.Detect != "" {
			return nil, fmt.Errorf("tests.toml: profile %s: mapping beside has no detect; only gdunit4 reads one", name)
		}
		if other, dup := seen[p.Extension]; dup {
			return nil, fmt.Errorf("tests.toml: profiles %s and %s both cover %s", other, name, p.Extension)
		}
		seen[p.Extension] = name
	}
	return doc.Profile, nil
}

// godotProject is what Pairing needs of one Godot project.
type godotProject struct {
	installed bool
	folder    string // test_lookup_folder; "" means beside the source
	naming    int    // test_suite_naming_convention
}

// Pairing reports every source of a layer with tests = true that has no test
// module where the test tool of its own layout expects one. The second result
// holds map errors: a layer whose sources no installed test profile covers.
func Pairing(files []File, m *Map) ([]Finding, []string, error) {
	profiles, err := testProfiles()
	if err != nil {
		return nil, nil, err
	}
	content := map[string][]byte{}
	for _, f := range files {
		content[f.Rel] = f.Content
	}
	byExt := map[string]testProfile{}
	for _, p := range profiles {
		byExt[p.Extension] = p
	}
	projects := map[string]godotProject{}
	errSet := map[string]bool{}
	var out []Finding
	for _, f := range files {
		l, ok := m.Of(f.Rel)
		if !ok || !l.Tests || !verify.IsSource(f.Rel) {
			continue
		}
		ext := path.Ext(f.Rel)
		p, ok := byExt[ext]
		if !ok {
			errSet[fmt.Sprintf("layer %s: tests = true, but no test profile knows %s files", l.Name, ext)] = true
			continue
		}
		suffixed := slices.ContainsFunc(p.TestSuffixes, func(s string) bool { return strings.HasSuffix(f.Rel, s) })
		if suffixed && p.Mapping != "gdunit4" {
			continue
		}
		want := strings.TrimSuffix(f.Rel, ext) + p.TestSuffixes[0]
		if p.Mapping == "gdunit4" {
			dir := godotDir(content, f.Rel)
			proj, seen := projects[dir]
			if !seen {
				if proj, err = readProject(content, dir, p.Detect); err != nil {
					return nil, nil, err
				}
				projects[dir] = proj
			}
			if !proj.installed {
				name, text, _ := strings.Cut(p.Detect, ":")
				errSet[fmt.Sprintf("layer %s: tests = true, but %s files need a %s mentioning %s (none found under %s)",
					l.Name, ext, name, text, cmp.Or(dir, "."))] = true
				continue
			}
			rel := strings.TrimPrefix(f.Rel, dir+"/")
			if dir == "" {
				rel = f.Rel
			}
			// With a test folder a file is a test when it lies under it, and the
			// name says nothing. Any path segment equal to the folder counts: the
			// scanner maps a /src/ directory to /<folder>/ and puts the folder
			// under addons/<plugin>/, so it is not always the first segment.
			// Beside the source, only the name tells a test from a source.
			beside := proj.folder == "" || proj.folder == "/"
			if beside && suffixed || !beside && strings.Contains("/"+path.Dir(rel)+"/", "/"+proj.folder+"/") {
				continue
			}
			want = path.Join(dir, gdunit4Path(rel, proj.folder, proj.naming))
		}
		if _, have := content[want]; !have {
			out = append(out, Finding{Path: f.Rel, Text: "has no test module; expected " + want})
		}
	}
	slices.SortFunc(out, func(a, b Finding) int { return cmp.Compare(a.Path, b.Path) })
	errs := make([]string, 0, len(errSet))
	for e := range errSet {
		errs = append(errs, e)
	}
	slices.Sort(errs)
	return out, errs, nil
}

// godotDir is the directory of the nearest project.godot above rel; "" is
// the root of the file set, also when there is none.
func godotDir(content map[string][]byte, rel string) string {
	for dir := path.Dir(rel); dir != "."; dir = path.Dir(dir) {
		if _, ok := content[dir+"/project.godot"]; ok {
			return dir
		}
	}
	return ""
}

// readProject reads the test settings of the Godot project at dir and
// whether a plugin.cfg mentioning the profile's tool lies below it.
func readProject(content map[string][]byte, dir, detect string) (godotProject, error) {
	proj := godotProject{folder: "test"}
	cfg, ok := content[path.Join(dir, "project.godot")]
	if !ok {
		return proj, nil
	}
	name, text, _ := strings.Cut(detect, ":")
	for rel, c := range content {
		if path.Base(rel) == name && (dir == "" || strings.HasPrefix(rel, dir+"/")) && strings.Contains(string(c), text) {
			proj.installed = true
		}
	}
	if !proj.installed {
		return proj, nil
	}
	vals, _, err := linecov.GodotCfgKeepEmpty(cfg, "gdunit4")
	if err != nil {
		return proj, fmt.Errorf("%s: %w", path.Join(dir, "project.godot"), err)
	}
	if v := vals["settings/test/test_lookup_folder"]; len(v) > 0 {
		proj.folder = v[0]
	}
	if v := vals["settings/test/test_suite_naming_convention"]; len(v) > 0 {
		proj.naming, _ = strconv.Atoi(v[0])
	}
	return proj, nil
}

// gdunit4Path is GdUnitTestSuiteScanner.resolve_test_suite_path for a res://
// source, quirks included: the final replace runs over the whole path.
func gdunit4Path(src, folder string, naming int) string {
	p := "res://" + src
	name := strings.TrimSuffix(path.Base(src), path.Ext(src))
	suite := suiteName(name, naming)
	if folder == "" || folder == "/" {
		return strings.TrimPrefix(strings.ReplaceAll(p, name, suite), "res://")
	}
	var t string
	if strings.Contains(p, "/src/") {
		t = strings.ReplaceAll(p, "/src/", "/"+folder+"/")
	} else {
		parts := slices.DeleteFunc(strings.Split(p, "/"), func(s string) bool { return s == "" })
		from := 1
		t = parts[0] + "//"
		if parts[1] == "addons" {
			t += "addons/" + parts[2] + "/"
			from = 3
		}
		t += folder
		for _, s := range parts[from:] {
			t += "/" + s
		}
	}
	t = strings.ReplaceAll(t, "///", "/")
	return strings.TrimPrefix(strings.ReplaceAll(t, name, suite), "res://")
}

// suiteName follows _to_naming_convention: 0 auto-detect, 1 snake, 2 pascal.
func suiteName(name string, naming int) string {
	switch naming {
	case 0:
		if isSnakeCase(name) {
			return toSnakeCase(name + "Test")
		}
		return toPascalCase(name + "Test")
	case 1:
		return toSnakeCase(name + "Test")
	case 2:
		return toPascalCase(name + "Test")
	}
	return "-<Unexpected>-"
}

func isSnakeCase(s string) bool {
	for _, r := range s {
		if r != '_' && string(r) == strings.ToUpper(string(r)) {
			return false
		}
	}
	return true
}

// toSnakeCase is GdObjects.to_snake_case: an underscore before every capital
// once more than one element has been written.
func toSnakeCase(s string) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		lower := strings.ToLower(string(r))
		if string(r) != lower && n > 1 {
			b.WriteByte('_')
			n++
		}
		b.WriteString(lower)
		n++
	}
	return b.String()
}

// toPascalCase is GdObjects.to_pascal_case: String.capitalize() without spaces.
func toPascalCase(s string) string {
	var b strings.Builder
	for _, word := range strings.Fields(strings.ReplaceAll(camelToUnderscore(s), "_", " ")) {
		r := []rune(word)
		b.WriteString(string(unicode.ToUpper(r[0])) + string(r[1:]))
	}
	return b.String()
}

// camelToUnderscore is Godot's String.camelcase_to_underscore.
func camelToUnderscore(s string) string {
	r := []rune(s)
	digit := func(c rune) bool { return c >= '0' && c <= '9' }
	var b strings.Builder
	start := 0
	for i := 1; i < len(r); i++ {
		upper, number := unicode.IsUpper(r[i]), digit(r[i])
		prevUpper, prevNumber := unicode.IsUpper(r[i-1]), digit(r[i-1])
		next2Lower := i+2 < len(r) && unicode.IsLower(r[i+1]) && unicode.IsLower(r[i+2])
		nextLower := i+1 < len(r) && unicode.IsLower(r[i+1])
		nextNumber := i+1 < len(r) && digit(r[i+1])
		condA := upper && !prevUpper && !prevNumber
		condB := prevUpper && upper && next2Lower
		numberLetter := number && !prevNumber && nextLower
		condC := number && !prevNumber && !numberLetter
		letterNumber := !number && prevNumber && (nextLower || nextNumber)
		if condA || condB || condC || letterNumber {
			b.WriteString(string(r[start:i]) + "_")
			start = i
		}
	}
	b.WriteString(string(r[start:]))
	return strings.ToLower(b.String())
}
