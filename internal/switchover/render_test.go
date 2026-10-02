package switchover

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const testSum = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// fullParams sets every field, each to a value that passes.
func fullParams() Params {
	return Params{
		Name:        "x",
		Project:     "/p/x",
		Loomux:      "/bin/loomux",
		ConfigNew:   "/prep/config.toml",
		Registry:    "/state/registry.toml",
		RegistryNew: "/prep/registry.toml",
		RegistrySum: testSum,
		WikiSrcs:    []string{"/v/91 Projekte/x", "/state/areas/project-x"},
		WikiDst:     "/p/x/docs/wiki",
		Vault:       "/v",
		VaultOld:    "91 Projekte/x",
		StateArea:   "/state/areas/project-x",
		OldFiles:    []string{".ultraloom", ".brain.toml"},
		OldHooks:    []string{"ulguard", "brain wiki-gate"},
	}
}

// minimalParams sets only what is always required.
func minimalParams() Params {
	return Params{Name: "x", Project: "/p/x", Loomux: "loomux", ConfigNew: "/prep/config.toml"}
}

func TestRenderNamesTheRequiredFieldThatIsMissing(t *testing.T) {
	cases := []struct {
		want   string
		change func(*Params)
	}{
		{"name is required", func(p *Params) { p.Name = "" }},
		{"project is required", func(p *Params) { p.Project = "" }},
		{"loomux is required", func(p *Params) { p.Loomux = "" }},
		{"config_new is required", func(p *Params) { p.ConfigNew = "" }},
		{"registry is required with registry_new", func(p *Params) { p.Registry = "" }},
		{"registry_sum is required with registry_new", func(p *Params) { p.RegistrySum = "" }},
		{"wiki_dst is required with wiki_srcs", func(p *Params) { p.WikiDst = "" }},
		{"vault is required with vault_old", func(p *Params) { p.Vault = "" }},
	}
	for _, c := range cases {
		p := fullParams()
		c.change(&p)
		_, err := render("#!/bin/sh\n", p)
		if err == nil || err.Error() != c.want {
			t.Errorf("%s: err %v", c.want, err)
		}
	}
}

func TestRenderTakesOnlyAModuleChoiceForInit(t *testing.T) {
	for _, arg := range []string{"--brain=none", "--hooks=all", "--graph=each"} {
		p := minimalParams()
		p.InitArgs = []string{arg}
		if _, err := render("#!/bin/sh\n", p); err != nil {
			t.Errorf("%s: %v", arg, err)
		}
	}
	for _, arg := range []string{"--brain", "--brain none", "--base=none", "--brain=off", "--dry-run", "--root=/x", "--brain=none;x", "",
		// A module choice at the end or at the start only is none.
		"--dry-run --brain=none", "--brain=none --dry-run", "x--brain=none", "--brain=nonex"} {
		p := minimalParams()
		p.InitArgs = []string{arg}
		_, err := render("#!/bin/sh\n", p)
		if err == nil || !strings.HasPrefix(err.Error(), "init_args: ") {
			t.Errorf("%q: err %v", arg, err)
		}
	}
}

func TestRenderTakesTheFieldsOnlyTheirPartiesNeed(t *testing.T) {
	for name, p := range map[string]Params{"minimal": minimalParams(), "full": fullParams()} {
		if _, err := render("#!/bin/sh\n", p); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestRenderQuotesEveryValueForTheShell(t *testing.T) {
	p := fullParams()
	p.Name = "it's"
	p.Project = "/p/it's"
	p.Registry, p.RegistryNew, p.RegistrySum = "", "", ""
	out, err := render("N=@NAME@\nP=@PROJECT@\nR=@REGISTRY@\n", p)
	if err != nil {
		t.Fatal(err)
	}
	want := "N='it'\\''s'\nP='/p/it'\\''s'\nR=''\n"
	if out != want {
		t.Fatalf("got\n%s\nwant\n%s", out, want)
	}
}

func TestRenderJoinsTheListsWithTheSeparatorTheScriptSplitsOn(t *testing.T) {
	out, err := render("F=@OLD_FILES@\nH=@OLD_HOOKS@\nW=@WIKI_SRCS@\n", fullParams())
	if err != nil {
		t.Fatal(err)
	}
	want := "F='.ultraloom .brain.toml'\nH='ulguard|brain wiki-gate'\nW='/v/91 Projekte/x|/state/areas/project-x'\n"
	if out != want {
		t.Fatalf("got\n%s\nwant\n%s", out, want)
	}
}

func TestRenderSetsEveryToken(t *testing.T) {
	names := map[string]string{
		"@NAME@": "'x'", "@PROJECT@": "'/p/x'", "@LOOMUX@": "'/bin/loomux'",
		"@CONFIG_NEW@": "'/prep/config.toml'", "@REGISTRY@": "'/state/registry.toml'",
		"@REGISTRY_NEW@": "'/prep/registry.toml'", "@REGISTRY_SUM@": "'" + testSum + "'",
		"@WIKI_DST@": "'/p/x/docs/wiki'", "@VAULT@": "'/v'", "@VAULT_OLD@": "'91 Projekte/x'",
		"@STATE_AREA@": "'/state/areas/project-x'", "@VAULT_SRC@": "'/v/91 Projekte/x'",
	}
	for token, want := range names {
		out, err := render(token, fullParams())
		if err != nil || out != want {
			t.Errorf("%s: got %q, %v; want %q", token, out, err, want)
		}
	}
}

func TestRenderRefusesATokenItHasNoValueFor(t *testing.T) {
	_, err := render("A=@PROJECT@\nB=@BOGUS@\n", fullParams())
	if err == nil || !strings.Contains(err.Error(), "@BOGUS@") {
		t.Fatalf("err %v", err)
	}
}

func TestRenderLeavesATokenInsideAValueAlone(t *testing.T) {
	p := fullParams()
	p.Project = "/p/@VAULT@"
	out, err := render("P=@PROJECT@\n", p)
	if err != nil || out != "P='/p/@VAULT@'\n" {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestRenderLeavesNoTokenInTheScript(t *testing.T) {
	out, err := Render(fullParams())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "#!/bin/sh\n") {
		t.Fatalf("the script does not start with its shebang:\n%.200s", out)
	}
	if left := regexp.MustCompile(`@[A-Z_]+@`).FindString(out); left != "" {
		t.Fatalf("token %s left in the script", left)
	}
	for _, line := range []string{"PROJECT='/p/x'", "OLD_HOOKS='ulguard|brain wiki-gate'", "VAULT_OLD='91 Projekte/x'"} {
		if !strings.Contains(out, "\n"+line+"\n") {
			t.Errorf("the script lacks the line %s", line)
		}
	}
}

func TestRenderDropsEveryCarriageReturn(t *testing.T) {
	p := fullParams()
	p.Name = "x\r"
	p.OldHooks = []string{"ulguard\r"}
	out, err := render("A=@NAME@\r\nB=@OLD_HOOKS@\r\n", p)
	if err != nil || out != "A='x'\nB='ulguard'\n" {
		t.Fatalf("got %q, %v", out, err)
	}
	if script, _ := Render(fullParams()); strings.Contains(script, "\r") {
		t.Fatal("the rendered script holds a carriage return")
	}
}

const everyToken = "@NAME@ @PROJECT@ @LOOMUX@ @CONFIG_NEW@ @REGISTRY@ @REGISTRY_NEW@ @REGISTRY_SUM@ " +
	"@WIKI_SRCS@ @WIKI_DST@ @VAULT@ @VAULT_OLD@ @VAULT_SRC@ @STATE_AREA@ @OLD_FILES@ @OLD_HOOKS@\n"

func TestRenderDropsTheCarriageReturnOfEveryField(t *testing.T) {
	want, err := render(everyToken, fullParams())
	if err != nil {
		t.Fatal(err)
	}
	cr := func(s string) string { return s + "\r" }
	each := func(list []string) []string {
		var out []string
		for _, s := range list {
			out = append(out, cr(s))
		}
		return out
	}
	p := fullParams()
	p.Name, p.Project, p.Loomux, p.ConfigNew = cr(p.Name), cr(p.Project), cr(p.Loomux), cr(p.ConfigNew)
	p.Registry, p.RegistryNew, p.RegistrySum = cr(p.Registry), cr(p.RegistryNew), cr(p.RegistrySum)
	p.WikiDst, p.Vault, p.VaultOld, p.StateArea = cr(p.WikiDst), cr(p.Vault), cr(p.VaultOld), cr(p.StateArea)
	p.WikiSrcs, p.OldFiles, p.OldHooks = each(p.WikiSrcs), each(p.OldFiles), each(p.OldHooks)
	got, err := render(everyToken, p)
	if err != nil || got != want {
		t.Fatalf("got %q, %v\nwant %q", got, err, want)
	}
}

func TestRenderNamesTheSourceThatIsTheVaultFolder(t *testing.T) {
	p := fullParams()
	p.VaultOld = "91 Projekte/x/"
	p.WikiSrcs = []string{"/state/areas/project-x", "/v/91 Projekte/x/"}
	out, err := render("S=@VAULT_SRC@ O=@VAULT_OLD@\n", p)
	// The script compares the source as it is spelt in WIKI_SRCS, and
	// strips the folder's name off what git lists.
	if err != nil || out != "S='/v/91 Projekte/x/' O='91 Projekte/x'\n" {
		t.Fatalf("got %q, %v", out, err)
	}
	out, err = render("S=@VAULT_SRC@\n", minimalParams())
	if err != nil || out != "S=''\n" {
		t.Fatalf("without a vault: %q, %v", out, err)
	}
	// A vault without a folder of it names no source, not even one that is
	// the vault itself.
	p = fullParams()
	p.VaultOld = ""
	p.WikiSrcs = []string{"/v"}
	out, err = render("S=@VAULT_SRC@\n", p)
	if err != nil || out != "S=''\n" {
		t.Fatalf("without vault_old: %q, %v", out, err)
	}
	// The source has to be spelt as the folder is: the script compares the
	// two as text.
	p = fullParams()
	p.WikiSrcs = []string{"/V/91 projekte/X"}
	if _, err := render("S=@VAULT_SRC@\n", p); err == nil || !strings.HasPrefix(err.Error(), "vault_old: ") {
		t.Fatalf("a source in another case: err %v", err)
	}
}

func TestRenderKeepsWhatInitWritesOutOfTheOldFiles(t *testing.T) {
	for _, f := range []string{".claude", ".claude/settings.json", ".loomux", ".loomux/config.toml", "./.loomux",
		"docs", "docs/wiki", "docs/wiki/index.md"} {
		p := fullParams()
		p.OldFiles = []string{".ultraloom", f}
		_, err := render("#!/bin/sh\n", p)
		if err == nil || !strings.HasPrefix(err.Error(), "old_files: "+f+" would remove what init writes or the wiki") {
			t.Errorf("%q: err %v", f, err)
		}
	}
	p := fullParams()
	p.OldFiles = []string{".claude/ultraloom.json", "doc", "docs/wikis", ".loomuxx", ".brain.toml"}
	if _, err := render("#!/bin/sh\n", p); err != nil {
		t.Errorf("neighbours: %v", err)
	}
	// A wiki outside the project protects nothing inside it.
	p.WikiDst = "/elsewhere/docs/wiki"
	p.OldFiles = []string{"docs"}
	if _, err := render("#!/bin/sh\n", p); err != nil {
		t.Errorf("wiki elsewhere: %v", err)
	}
}

// The script runs on a disk that ignores case: `rm -rf "$PROJECT/.GIT"` takes
// git's directory, and `.Loomux` is the configuration init has just written.
func TestRenderJudgesAnOldFileWithoutRegardToCase(t *testing.T) {
	for f, want := range map[string]string{
		".GIT":                  "old_files: .GIT is git's",
		".Git/config":           "old_files: .Git/config is git's",
		"./.gIt":                "old_files: ./.gIt is git's",
		".LOOMUX":               "old_files: .LOOMUX would remove what init writes or the wiki",
		".Loomux/config.toml":   "old_files: .Loomux/config.toml would remove what init writes or the wiki",
		".Claude":               "old_files: .Claude would remove what init writes or the wiki",
		".CLAUDE/settings.json": "old_files: .CLAUDE/settings.json would remove what init writes or the wiki",
		".claude/Settings.JSON": "old_files: .claude/Settings.JSON would remove what init writes or the wiki",
		"DOCS":                  "old_files: DOCS would remove what init writes or the wiki",
		"Docs/Wiki":             "old_files: Docs/Wiki would remove what init writes or the wiki",
		"docs/WIKI/index.md":    "old_files: docs/WIKI/index.md would remove what init writes or the wiki",
	} {
		p := fullParams()
		p.OldFiles = []string{".ultraloom", f}
		if _, err := render("#!/bin/sh\n", p); err == nil || err.Error() != want {
			t.Errorf("%q: err %v, want %s", f, err, want)
		}
	}
	// The wiki's place is found under a project spelt in another case too,
	// whichever of the two carries the capitals.
	for _, c := range [][2]string{{"/p/x", "/P/X/Docs/Wiki"}, {"/P/X", "/p/x/docs/wiki"}} {
		p := fullParams()
		p.Project, p.WikiDst = c[0], c[1]
		p.OldFiles = []string{"docs/wiki"}
		if _, err := render("#!/bin/sh\n", p); err == nil || err.Error() != "old_files: docs/wiki would remove what init writes or the wiki" {
			t.Errorf("project %s, wiki %s: err %v", c[0], c[1], err)
		}
	}
	p := fullParams()
	p.VaultOld = ".GIT"
	p.WikiSrcs = []string{"/v/.GIT"}
	if _, err := render("#!/bin/sh\n", p); err == nil || err.Error() != "vault_old: .GIT is git's" {
		t.Errorf("vault_old: err %v", err)
	}
	// What only begins like one of them is another file in any case.
	p = fullParams()
	p.OldFiles = []string{".GITHUB", ".Gitignore", ".LoomuxOld", ".Claude-old", "Docs/Wikis", ".Ultraloom"}
	if _, err := render("#!/bin/sh\n", p); err != nil {
		t.Errorf("neighbours: %v", err)
	}
}

// NTFS keeps a second, short name for a file: GIT~1 is .git and LOOMUX~1 is
// .loomux there, and `rm -rf "$PROJECT/GIT~1"` takes the repository.
func TestRenderRefusesAnOldFileSpeltLikeAShortName(t *testing.T) {
	for _, f := range []string{"GIT~1", "git~1/config", "sub/LOOMUX~2", "a~1", "CLAUDE~1", "DOCS~1/wiki", "x~0", "x/~9", "a~12b"} {
		p := fullParams()
		p.OldFiles = []string{".ultraloom", f}
		want := "old_files: " + f + " holds ~ before a digit, the short name a disk of Windows keeps for another file, " +
			"which may be .git or what init writes"
		if _, err := render("#!/bin/sh\n", p); err == nil || err.Error() != want {
			t.Errorf("%q: err %v", f, err)
		}
	}
	// A ~ alone is no short name, and neither is one a digit does not follow
	// in the same element.
	p := fullParams()
	p.OldFiles = []string{"notes~", "~backup", "file~name.txt", "v1~beta", "1~", "a~/1", ".brain.toml", ".ultraloom"}
	if _, err := render("#!/bin/sh\n", p); err != nil {
		t.Errorf("neighbours: %v", err)
	}
	// The vault's folder goes through git, which knows a file by the name it
	// tracks only: a folder named like this is its own.
	p = fullParams()
	p.VaultOld = "91 Projekte/x~1"
	p.WikiSrcs = []string{"/v/91 Projekte/x~1"}
	if _, err := render("#!/bin/sh\n", p); err != nil {
		t.Errorf("vault_old: %v", err)
	}
}

// A repository below the project has a .git of its own, and the script would
// remove that one as it would the project's.
func TestRenderRefusesGitsDirectoryAtAnyDepth(t *testing.T) {
	for _, f := range []string{"sub/.git", "sub/.GIT/hooks", "a/b/.Git", "./sub/x/../.git"} {
		p := fullParams()
		p.OldFiles = []string{".ultraloom", f}
		if _, err := render("#!/bin/sh\n", p); err == nil || err.Error() != "old_files: "+f+" is git's" {
			t.Errorf("%q: err %v", f, err)
		}
	}
	p := fullParams()
	p.VaultOld = "91 Projekte/.git"
	p.WikiSrcs = []string{"/v/91 Projekte/.git"}
	if _, err := render("#!/bin/sh\n", p); err == nil || err.Error() != "vault_old: 91 Projekte/.git is git's" {
		t.Errorf("vault_old: err %v", err)
	}
	p = fullParams()
	// The path counts as the script's rm resolves it: .git/../x is x.
	p.OldFiles = []string{"sub/.gitignore", "sub/.github", "sub/git", "sub/x.git", "sub/.gitx/.gi", ".gitmodules", ".git/../x"}
	if _, err := render("#!/bin/sh\n", p); err != nil {
		t.Errorf("neighbours: %v", err)
	}
}

func TestRenderRefusesALineBreakInAValue(t *testing.T) {
	cases := map[string]func(*Params){
		"name":         func(p *Params) { p.Name = "x\ny" },
		"project":      func(p *Params) { p.Project = "/p\n/x" },
		"loomux":       func(p *Params) { p.Loomux = "loomux\n" },
		"config_new":   func(p *Params) { p.ConfigNew = "/c\r\n" },
		"registry":     func(p *Params) { p.Registry = "/r\n" },
		"registry_new": func(p *Params) { p.RegistryNew = "/r\n" },
		"registry_sum": func(p *Params) { p.RegistrySum = testSum + "\n" },
		"wiki_srcs":    func(p *Params) { p.WikiSrcs = []string{"/a", "/b\n"} },
		"wiki_dst":     func(p *Params) { p.WikiDst = "/d\n" },
		"vault":        func(p *Params) { p.Vault = "/v\n" },
		"vault_old":    func(p *Params) { p.VaultOld = "91 Projekte/x\n" },
		"state_area":   func(p *Params) { p.StateArea = "/s\n" },
		"old_files":    func(p *Params) { p.OldFiles = []string{".a", ".b\n"} },
		"old_hooks":    func(p *Params) { p.OldHooks = []string{"a", "b\nc"} },
	}
	for field, change := range cases {
		p := fullParams()
		change(&p)
		_, err := render("#!/bin/sh\n", p)
		if err == nil || err.Error() != field+": a value holds a line break" {
			t.Errorf("%s: err %v", field, err)
		}
	}
	p := fullParams()
	p.Name = "x\x00"
	if _, err := render("#!/bin/sh\n", p); err == nil || err.Error() != "name: a value holds a NUL" {
		t.Errorf("NUL: err %v", err)
	}
}

func TestRenderWantsAbsolutePaths(t *testing.T) {
	cases := map[string]func(*Params){
		"project":      func(p *Params) { p.Project = "p/x" },
		"config_new":   func(p *Params) { p.ConfigNew = "config.toml" },
		"registry":     func(p *Params) { p.Registry = "registry.toml" },
		"registry_new": func(p *Params) { p.RegistryNew = "./registry.toml" },
		"wiki_srcs":    func(p *Params) { p.WikiSrcs = []string{"/a", "b"} },
		"wiki_dst":     func(p *Params) { p.WikiDst = "docs/wiki" },
		"vault":        func(p *Params) { p.Vault = "v" },
		"state_area":   func(p *Params) { p.StateArea = "areas/x" },
	}
	for field, change := range cases {
		p := fullParams()
		change(&p)
		_, err := render("#!/bin/sh\n", p)
		if err == nil || err.Error() != field+": not an absolute path" {
			t.Errorf("%s: err %v", field, err)
		}
	}
	// The loomux of the script may be a name found on PATH.
	p := fullParams()
	p.Loomux = "loomux"
	if _, err := render("#!/bin/sh\n", p); err != nil {
		t.Errorf("loomux by name: %v", err)
	}
}

func TestRenderTakesAPathOfThisSystem(t *testing.T) {
	p := fullParams()
	abs, err := filepath.Abs("x")
	if err != nil {
		t.Fatal(err)
	}
	p.Project = abs
	if _, err := render("#!/bin/sh\n", p); err != nil {
		t.Fatalf("%s: %v", abs, err)
	}
}

func TestRenderWantsAPlainSHA256(t *testing.T) {
	for _, sum := range []string{"abc", strings.ToUpper(testSum), testSum + "0", testSum[1:], "g" + testSum[1:]} {
		p := fullParams()
		p.RegistrySum = sum
		if _, err := render("#!/bin/sh\n", p); err == nil || err.Error() != "registry_sum: not a SHA-256 in lower-case hex" {
			t.Errorf("%q: err %v", sum, err)
		}
	}
}

func TestRenderRefusesAnOldFileOutsideTheProject(t *testing.T) {
	for _, f := range []string{"", ".", "..", "./", "a/../..", "a/..", "../x", "/abs", "C:/x", "c:x",
		"a b", "a\tb", " a", "a\U000000a0b", `a\b`, ".git", ".git/hooks", "./.git"} {
		p := fullParams()
		p.OldFiles = []string{".ultraloom", f}
		_, err := render("#!/bin/sh\n", p)
		if err == nil || !strings.HasPrefix(err.Error(), "old_files: ") {
			t.Errorf("%q: err %v", f, err)
		}
	}
	p := fullParams()
	p.OldFiles = []string{".ultraloom", ".claude/old.json", ".brain.toml", ".ultraloom/vendor"}
	if _, err := render("#!/bin/sh\n", p); err != nil {
		t.Errorf("valid old files: %v", err)
	}
}

func TestRenderRefusesAVaultFolderOutsideTheVault(t *testing.T) {
	for _, f := range []string{".", "..", "../x", "/abs", "C:/x", `a\b`, ".git", "x/../..", "91 Projekte/.."} {
		p := fullParams()
		p.VaultOld = f
		p.WikiSrcs = []string{"/v/" + f}
		_, err := render("#!/bin/sh\n", p)
		if err == nil || !strings.HasPrefix(err.Error(), "vault_old: ") {
			t.Errorf("%q: err %v", f, err)
		}
	}
}

func TestRenderRemovesOnlyAVaultFolderItMerges(t *testing.T) {
	p := fullParams()
	p.WikiSrcs = []string{"/v/91 Projekte/y"}
	_, err := render("#!/bin/sh\n", p)
	if err == nil || err.Error() != "vault_old: /v/91 Projekte/x is not one of wiki_srcs; the vault would lose what was never merged" {
		t.Fatalf("err %v", err)
	}
	p.WikiSrcs = nil
	p.WikiDst = ""
	if _, err := render("#!/bin/sh\n", p); err == nil || !strings.HasPrefix(err.Error(), "vault_old: ") {
		t.Fatalf("without wiki_srcs: err %v", err)
	}
	for _, alike := range []Params{
		{Vault: "/v/", VaultOld: "91 Projekte/x", WikiSrcs: []string{"/v/91 Projekte/x"}},
		{Vault: "/v", VaultOld: "91 Projekte/x/", WikiSrcs: []string{"/v/91 Projekte/x"}},
		{Vault: "/v", VaultOld: "91 Projekte/x", WikiSrcs: []string{"/state", "/v/91 Projekte/x/"}},
		{Vault: `C:\v`, VaultOld: "91 Projekte/x", WikiSrcs: []string{"C:/v/91 Projekte/x"}},
	} {
		q := fullParams()
		q.Vault, q.VaultOld, q.WikiSrcs = alike.Vault, alike.VaultOld, alike.WikiSrcs
		if strings.HasPrefix(q.Vault, "C:") && !filepath.IsAbs(q.Vault) {
			continue // a drive letter is an absolute path on Windows only
		}
		if _, err := render("#!/bin/sh\n", q); err != nil {
			t.Errorf("%+v: %v", alike, err)
		}
	}
}

func TestRenderRefusesTheSeparatorInsideAListItem(t *testing.T) {
	cases := map[string]func(*Params){
		"wiki_srcs: an entry is empty":  func(p *Params) { p.WikiSrcs = []string{"/a", ""} },
		"wiki_srcs: an entry holds |":   func(p *Params) { p.WikiSrcs = []string{"/a|/b"} },
		"old_hooks: an entry is empty":  func(p *Params) { p.OldHooks = []string{"ulguard", ""} },
		"old_hooks: an entry holds |":   func(p *Params) { p.OldHooks = []string{"a|b"} },
		"old_files: an entry is empty":  func(p *Params) { p.OldFiles = []string{""} },
		"old_files: .. leaves the root": func(p *Params) { p.OldFiles = []string{".."} },
	}
	for want, change := range cases {
		p := fullParams()
		change(&p)
		_, err := render("#!/bin/sh\n", p)
		if err == nil || err.Error() != want {
			t.Errorf("%s: err %v", want, err)
		}
	}
}

func TestRenderRefusesASourceNamedTwice(t *testing.T) {
	// Named twice, the vault's folder would be staged once and counted as
	// a second source that is never removed.
	for _, second := range []string{"/v/91 Projekte/x/", "/v/91 projekte/X", "/v/./91 Projekte/x"} {
		p := fullParams()
		p.WikiSrcs = []string{"/v/91 Projekte/x", second}
		_, err := render("#!/bin/sh\n", p)
		if err == nil || err.Error() != "wiki_srcs: "+second+" names /v/91 Projekte/x again" {
			t.Errorf("%q: err %v", second, err)
		}
	}
}

func TestTheScriptParses(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh on PATH")
	}
	for name, p := range map[string]Params{"minimal": minimalParams(), "full": fullParams()} {
		out, err := Render(p)
		if err != nil {
			t.Fatal(err)
		}
		script := filepath.Join(t.TempDir(), "apply.sh")
		if err := os.WriteFile(script, []byte(out), 0o644); err != nil {
			t.Fatal(err)
		}
		if msg, err := exec.Command(sh, "-n", script).CombinedOutput(); err != nil || len(out) < 100 {
			t.Errorf("%s: sh -n: %v %s (script of %d bytes)", name, err, msg, len(out))
		}
	}
}
