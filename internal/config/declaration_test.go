package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// declarationFile writes body to a config.toml in a fresh directory and
// answers its path.
func declarationFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const areaX = "[area]\nscope = \"x\"\n"

func TestADeclarationRefusesWhatItCannotUse(t *testing.T) {
	for name, row := range map[string]struct{ body, want string }{
		"area a string":      {"area = \"x\"\n", "[area] must be a table, found string"},
		"privacy an integer": {"privacy = 5\n" + areaX, "[privacy] must be a table, found integer"},
		"wiki a datetime":    {"wiki = 1979-05-27\n" + areaX, "[wiki] must be a table, found datetime"},
		"maintenance a string": {"maintenance = \"x\"\n" + areaX,
			"[maintenance] must be a table, found string"},
		"model an array":   {"model = [1]\n" + areaX, "[model] must be a table, found array"},
		"layout a boolean": {"layout = true\n" + areaX, "[layout] must be a table, found boolean"},
		"index a float":    {"index = 1.5\n" + areaX, "[index] must be a table, found float"},
		"missing scope":    {"[area]\nname = \"x\"\n", `[area] is missing "scope"`},
		"scope an integer": {"[area]\nscope = 3\n",
			"[area] scope must be a non-empty string, found integer"},
		"empty scope": {"[area]\nscope = \"\"\n", `[area] scope must be a non-empty string, found ""`},
		"scope before mode": {"[area]\nscope = \"\"\n\n[privacy]\nmode = \"cloud\"\n",
			`[area] scope must be a non-empty string, found ""`},
		"unknown mode": {areaX + "\n[privacy]\nmode = \"cloud\"\n",
			`[privacy] mode must be one of automatic_cloud, local_only, manual_cloud, found "cloud"`},
		"mode an integer": {areaX + "\n[privacy]\nmode = 1\n",
			"[privacy] mode must be one of automatic_cloud, local_only, manual_cloud, found integer"},
		"types a string": {areaX + "\n[wiki]\ntypes = \"Topic\"\n",
			"[wiki] types must be an array of strings, found string"},
		"types with an integer": {areaX + "\n[wiki]\ntypes = [\"Topic\", 1]\n",
			"[wiki] types #2 must be a string, found integer"},
		"on_merge a string": {areaX + "\n[maintenance]\non_merge = \"yes\"\n",
			"[maintenance] on_merge must be a boolean, found string"},
		"empty branch": {areaX + "\n[maintenance]\nbranch = \"\"\n",
			`[maintenance] branch must be a non-empty string, found ""`},
		"enabled a string": {areaX + "\n[model]\nenabled = \"yes\"\n",
			"[model] enabled must be a boolean, found string"},
		"roles an integer": {areaX + "\n[model]\nroles = 5\n",
			"[model] roles must be a table, found integer"},
		"unknown roles": {areaX + "\n[model.roles]\nzap = false\nplace = true\nguess = true\n",
			`[model] roles has unknown "guess", "zap"; known are describe, place, propose`},
		"role a string": {areaX + "\n[model.roles]\nplace = \"yes\"\n",
			"[model] roles.place must be a boolean, found string"},
		"hub an integer": {areaX + "\n[layout]\nhub = 1\n",
			"[layout] hub must be a string, found integer"},
		"include a string": {areaX + "\n[index]\ninclude = \"*.md\"\n",
			"[index] include must be an array of strings, found string"},
		"exclude with an integer": {areaX + "\n[index]\nexclude = [\"a\", 3]\n",
			"[index] exclude #2 must be a string, found integer"},
		"never a string": {areaX + "\n[privacy]\nnever = \"x\"\n",
			"[privacy] never must be an array of strings, found string"},
		"untouched_days zero": {areaX + "\n[wiki]\nuntouched_days = 0\n",
			"[wiki] untouched_days must be an integer >= 1, found 0"},
		"untouched_days true": {areaX + "\n[wiki]\nuntouched_days = true\n",
			"[wiki] untouched_days must be an integer >= 1, found boolean"},
		"untouched_days a string": {areaX + "\n[wiki]\nuntouched_days = \"5\"\n",
			"[wiki] untouched_days must be an integer >= 1, found string"},
	} {
		t.Run(name, func(t *testing.T) {
			path := declarationFile(t, row.body)
			manifest, err := ReadDeclaration(path)
			want := path + ": " + row.want
			if err == nil || err.Error() != want {
				t.Fatalf("ReadDeclaration = %+v, %v; want %q", manifest, err, want)
			}
		})
	}
}

func TestADeclarationCarriesWhatItDeclares(t *testing.T) {
	path := declarationFile(t, "[area]\nscope = \"project/demo\"\n\n"+
		"[privacy]\nmode = \"local_only\"\nnever = [\"secret/**\"]\n\n"+
		"[wiki]\ntypes = [\"Runbook\"]\nuntouched_days = 30\n\n"+
		"[maintenance]\non_merge = true\nbranch = \"main\"\n\n"+
		"[model]\nenabled = false\n\n[model.roles]\nplace = true\n\n"+
		"[layout]\nwiki = \"docs/wiki\"\nhub = \"hub\"\nreview = \"95\"\ninbox = \"00 In\"\n\n"+
		"[index]\ninclude = [\"**/*.md\"]\nexclude = [\"drafts/**\"]\nunsearched = [\"archive/**\"]\n")
	m, err := ReadDeclaration(path)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join([]string{m.Path, m.Scope, m.PrivacyMode, strings.Join(m.NeverGlobs, ","),
		strings.Join(m.DeclaredTypes, ","), m.LayoutWiki, m.LayoutHub, m.LayoutReview, m.LayoutInbox,
		strings.Join(m.IndexInclude, ","), strings.Join(m.IndexExclude, ","), strings.Join(m.IndexUnsearched, ",")}, "|")
	want := path + "|project/demo|local_only|secret/**|Runbook|docs/wiki|hub|95|00 In|**/*.md|drafts/**|archive/**"
	if got != want || m.UntouchedDays != 30 {
		t.Errorf("got %q and %d days, want %q and 30", got, m.UntouchedDays, want)
	}
}

// One day is the smallest value the rule allows, and it is allowed.
func TestADeclarationAcceptsOneUntouchedDay(t *testing.T) {
	m, err := ReadDeclaration(declarationFile(t, areaX+"\n[wiki]\nuntouched_days = 1\n"))
	if err != nil || m.UntouchedDays != 1 {
		t.Fatalf("ReadDeclaration = %+v, %v; want one day", m, err)
	}
}

func TestADeclarationThatSaysNothingElseGetsTheDefaults(t *testing.T) {
	m, err := ReadDeclaration(declarationFile(t, areaX))
	if err != nil {
		t.Fatal(err)
	}
	if m.PrivacyMode != "manual_cloud" || m.UntouchedDays != DefaultUntouchedDays ||
		m.NeverGlobs != nil || m.DeclaredTypes != nil || m.LayoutInbox != "" {
		t.Errorf("ReadDeclaration = %+v", m)
	}
}

func TestAConfigurationWithoutAnAreaIsErrNoArea(t *testing.T) {
	path := declarationFile(t, "[layout]\nwiki = 1\n")
	_, err := ReadDeclaration(path)
	if !errors.Is(err, ErrNoArea) || err.Error() != path+": the configuration declares no [area]" {
		t.Fatalf("err = %v, want ErrNoArea naming the file", err)
	}
}

func TestADeclarationThatCannotBeReadIsNoAbsence(t *testing.T) {
	// The file that vanishes between a stat and the read: an error, and
	// neither ErrNoArea nor anything a caller could take for "no manifest".
	path := filepath.Join(t.TempDir(), "gone.toml")
	_, err := ReadDeclaration(path)
	if errors.Is(err, ErrNoArea) || !errors.Is(err, fs.ErrNotExist) ||
		!strings.HasPrefix(err.Error(), path+": cannot be read: ") {
		t.Fatalf("err = %v", err)
	}
}

func TestADeclarationThatIsNoTomlNamesTheFile(t *testing.T) {
	path := declarationFile(t, "[area\n")
	_, err := ReadDeclaration(path)
	if err == nil || !strings.HasPrefix(err.Error(), path+": not valid TOML: ") {
		t.Fatalf("err = %v", err)
	}
}

func TestAnInboxMustStayInsideTheArea(t *testing.T) {
	outside := t.TempDir()
	m := &Manifest{Path: "p/config.toml", LayoutInbox: outside}
	if _, err := m.InboxLayout(); err == nil ||
		err.Error() != "p/config.toml: [layout] inbox must be relative to the area, found "+strconv.Quote(outside) {
		t.Errorf("InboxLayout = %v", err)
	}
	for _, value := range []string{"", "00 In"} {
		m := &Manifest{LayoutInbox: value}
		if got, err := m.InboxLayout(); err != nil || got != value {
			t.Errorf("InboxLayout(%q) = %q, %v", value, got, err)
		}
	}
}

func TestDeclarationKeysNameEverySectionTheReaderChecks(t *testing.T) {
	keys := DeclarationKeys()
	for _, section := range declarationSections() {
		if _, ok := keys[section]; !ok {
			t.Errorf("DeclarationKeys lacks [%s]", section)
		}
	}
	if !slices.Equal(keys["layout"], []string{"wiki", "hub", "review", "inbox"}) {
		t.Errorf("layout keys %v", keys["layout"])
	}
}

// declared reads body as a declaration and fails the test on any refusal.
func declared(t *testing.T, body string) *Manifest {
	t.Helper()
	m, err := ReadDeclaration(declarationFile(t, body))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestTheDeclarationCarriesTheMergeConsent(t *testing.T) {
	m := declared(t, "[area]\nscope = \"project/x\"\n[maintenance]\non_merge = true\nbranch = \"master\"\n")
	if !m.OnMerge || m.MergeBranch != "master" {
		t.Fatalf("OnMerge=%v MergeBranch=%q", m.OnMerge, m.MergeBranch)
	}
}

func TestTheMergeBranchDefaultsToMain(t *testing.T) {
	m := declared(t, "[area]\nscope = \"project/x\"\n")
	if m.OnMerge || m.MergeBranch != "main" {
		t.Fatalf("OnMerge=%v MergeBranch=%q", m.OnMerge, m.MergeBranch)
	}
}

// Unknown keys are not judged, so the old name passes without a word and
// without effect.
func TestTheOldMergeBranchNameIsNotRead(t *testing.T) {
	m := declared(t, "[area]\nscope = \"project/x\"\n[maintenance]\non_merge = true\nmerge_branch = \"dev\"\n")
	if m.MergeBranch != "main" {
		t.Fatalf("merge_branch must stay unread, got %q", m.MergeBranch)
	}
}
