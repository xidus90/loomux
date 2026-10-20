package compat

import (
	"io/fs"
	"strings"
	"testing"
)

func TestTheListLoads(t *testing.T) {
	list, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("the list is empty")
	}
	for _, e := range list {
		if e.Stack == "" || e.Runtime == "" || e.Version == "" || e.Fixture == "" || len(e.Kinds) == 0 || e.Line == 0 {
			t.Errorf("incomplete entry %+v", e)
		}
	}
	if got, want := list[0].Key(), "gdscript/godot-3.6/godot3-gut"; got != want {
		t.Errorf("Key() = %q, want %q", got, want)
	}
	if list[0].Addons[0] != (Addon{Name: "GUT", Version: "7.4.3"}) || list[1].Tools["gdtoolkit"] != "3.*" {
		t.Errorf("addons or tools lost: %+v %+v", list[0], list[1])
	}
	if got, want := list[1].Key(), "gdscript/godot-3.6/godot3-plain"; got != want {
		t.Errorf("Key() = %q, want %q", got, want)
	}
}

const good = `[[entry]]
stack = "python"
runtime = "python"
version = "3.12"
fixture = "python-mypy"
kinds = ["lint", "types"]
tools = { ruff = "0.17.*" }
`

func TestTheListRefuses(t *testing.T) {
	cases := []struct{ name, doc, want string }{
		{"syntax", "[[entry", "compat.toml"},
		{"unknown key", good + "colour = 1\n", `compat.toml:8: unknown key "colour"`},
		{"unknown stack", strings.Replace(good, `stack = "python"`, `stack = "cobol"`, 1), `compat.toml:1: stack "cobol"`},
		{"wrong runtime", strings.Replace(good, `runtime = "python"`, `runtime = "godot"`, 1), `compat.toml:1: runtime "godot"`},
		{"bad version", strings.Replace(good, `"3.12"`, `"3.12.1"`, 1), `compat.toml:1: version "3.12.1"`},
		{"unknown kind", strings.Replace(good, `"types"`, `"magic"`, 1), `compat.toml:1: kind "magic"`},
		{"duplicate", good + "\n" + good, `compat.toml:9: duplicate entry python/python-3.12/python-mypy`},
		{"unknown addon", good + `addons = [{ name = "Nope", version = "1" }]` + "\n", `compat.toml:1: addon "Nope"`},
		{"unknown tool", strings.Replace(good, "ruff", "hammer", 1), `compat.toml:1: tool "hammer"`},
		{"no fixture folder", strings.Replace(good, `"python-mypy"`, `"nowhere"`, 1), `compat.toml:1: fixture "nowhere"`},
		{"no kinds", strings.Replace(good, "kinds = [\"lint\", \"types\"]", "kinds = []", 1), `compat.toml:1: entry has no kinds`},
		{"no fixture", strings.Replace(good, "fixture = \"python-mypy\"\n", "", 1), `compat.toml:1: entry has no fixture`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parse([]byte(c.doc), "compat.toml")
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want it to contain %q", err, c.want)
			}
		})
	}
}

func TestAnInlineKeyHasNoLine(t *testing.T) {
	doc := good + `addons = [{ name = "GUT", version = "1", colour = "x" }]` + "\n"
	_, err := parse([]byte(doc), "compat.toml")
	if err == nil || !strings.HasPrefix(err.Error(), `compat.toml: unknown key "colour"`) {
		t.Errorf("err = %v", err)
	}
}

func TestTheListLoadsFromAnyDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(Fixtures(), "go-module/go.mod.txt"); err != nil {
		t.Error(err)
	}
	if _, err := fs.Stat(Fixtures(), "godot3-gut/project.godot"); err != nil {
		t.Error(err)
	}
}
