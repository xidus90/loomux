package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/compat"
)

type stubUpstream struct {
	rels map[string][]string
	err  error
}

func (s stubUpstream) Releases(src string) ([]string, error) { return s.rels[src], s.err }

// stubScout swaps the upstream sources; every source answers the same list.
func stubScout(t *testing.T, up compat.Upstream) {
	t.Helper()
	old := compatUpstream
	t.Cleanup(func() { compatUpstream = old })
	compatUpstream = func() compat.Upstream { return up }
}

type allSame []string

func (a allSame) Releases(string) ([]string, error) { return a, nil }

func TestDevCompatScoutReportsFindings(t *testing.T) {
	stubScout(t, allSame{"99.0", "0.1"})
	jsonPath := filepath.Join(t.TempDir(), "f.json")
	code, out, errOut := run("dev", "compat", "scout", "--json", jsonPath)
	if code != 0 || !strings.Contains(out, "compat: godot4-gdunit4/godot 99.0 is out\n") || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	var fs []compat.Finding
	if err := json.Unmarshal(data, &fs); err != nil || len(fs) == 0 || fs[0].Probe.Fixture == "" || !strings.HasPrefix(fs[0].TOML, "[[entry]]") {
		t.Fatalf("json %v: %s", err, data)
	}
}

func TestCompatUpstreamIsHTTP(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "tok")
	if h, ok := compatUpstream().(compat.HTTPUpstream); !ok || h.Token != "tok" {
		t.Fatalf("%#v", compatUpstream())
	}
}

func TestDevCompatScoutNothingNew(t *testing.T) {
	stubScout(t, allSame{"0.0.1"})
	jsonPath := filepath.Join(t.TempDir(), "f.json")
	code, out, _ := run("dev", "compat", "scout", "--json", jsonPath)
	data, _ := os.ReadFile(jsonPath)
	if code != 0 || out != "nothing new\n" || string(data) != "[]\n" {
		t.Fatalf("code %d, out %q, json %q", code, out, data)
	}
}

func TestDevCompatScoutFailures(t *testing.T) {
	stubScout(t, stubUpstream{err: errors.New("down")})
	if code, _, errOut := run("dev", "compat", "scout"); code != 1 || !strings.Contains(errOut, "down") {
		t.Errorf("source error: code %d, err %q", code, errOut)
	}
	stubScout(t, allSame{"0.0.1"})
	if code, _, errOut := run("dev", "compat", "scout", "--json", filepath.Join(t.TempDir(), "no", "f.json")); code != 1 || errOut == "" {
		t.Errorf("unwritable: code %d, err %q", code, errOut)
	}
	for _, args := range [][]string{{"dev", "compat", "scout", "extra"}, {"dev", "compat", "scout", "--bogus"}} {
		if code, _, _ := run(args...); code != 2 {
			t.Errorf("%v: code %d", args, code)
		}
	}
}

func TestDevCompatScoutResolved(t *testing.T) {
	list, _ := compat.Load()
	e := list[0]
	for title, want := range map[string]string{
		"compat: " + e.Fixture + "/" + e.Runtime + " " + e.Version + " is out": "resolved\n",
		"compat: " + e.Fixture + "/" + e.Runtime + " 99.0 is out":              "open\n",
	} {
		if code, out, _ := run("dev", "compat", "scout", "--resolved", title); code != 0 || out != want {
			t.Errorf("%q: code %d, out %q", title, code, out)
		}
	}
}

func writeProbe(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "probe.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const probeJSON = `{"stack":"go","runtime":"go","version":"1.28","fixture":"go-module","kinds":["lint"]}`

func TestDevCompatRunEntryJSON(t *testing.T) {
	got := stubCompatRun(t, compat.Report{Entry: "go/go-1.28/go-module", Green: true, Lines: []string{}}, nil)
	p := writeProbe(t, probeJSON)
	if code, out, errOut := run("dev", "compat", "run", "--entry-json", p, "--work", "w"); code != 0 || !strings.Contains(out, "go/go-1.28/go-module") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if got.Entry == nil || got.Entry.Version != "1.28" {
		t.Fatalf("options %+v", got)
	}
	code, out, _ := run("dev", "compat", "run", "--entry-json", p, "--describe")
	if code != 0 || !strings.Contains(out, `"version":"1.28"`) {
		t.Errorf("describe: code %d, out %q", code, out)
	}
	list, _ := compat.Load()
	var gd compat.Entry
	for _, e := range list {
		if len(e.Addons) > 0 {
			gd = e
		}
	}
	data, _ := json.Marshal(gd)
	if code, out, _ := run("dev", "compat", "run", "--entry-json", writeProbe(t, string(data)), "--describe"); code != 0 || !strings.Contains(out, `"addons":[{"name"`) {
		t.Errorf("describe addons: code %d, out %q", code, out)
	}
	// A framework of the stack, but no addon whose source CI knows.
	sourceless := writeProbe(t, `{"stack":"gdscript","runtime":"godot","version":"3.6","fixture":"godot3-gut","kinds":["lint"],"addons":[{"name":"godot-code-coverage","version":"1.0.0"}]}`)
	if code, out, errOut := run("dev", "compat", "run", "--entry-json", sourceless, "--describe"); code != 2 || out != "" || !strings.Contains(errOut, "no source known") {
		t.Errorf("describe without a source: code %d, out %q, err %q", code, out, errOut)
	}
	for _, args := range [][]string{
		{"dev", "compat", "run", "--entry-json", writeProbe(t, `{"stack":"go"}`), "--work", "w"},
		{"dev", "compat", "run", "--entry-json", p + ".missing", "--work", "w"},
		{"dev", "compat", "run", list[0].Key(), "--entry-json", p, "--work", "w"},
	} {
		if code, _, _ := run(args...); code != 2 {
			t.Errorf("%v: code %d", args, code)
		}
	}
}

func TestDevCompatMatrixProbes(t *testing.T) {
	p := writeProbe(t, "["+probeJSON+"]")
	code, out, _ := run("dev", "compat", "matrix", "--probes", p)
	jobs := matrixJobs(t, out)
	if code != 0 || len(jobs) != 2 || jobs[0].Probe == "" || jobs[0].Entry != "go/go-1.28/go-module" {
		t.Fatalf("code %d, out %q", code, out)
	}
	if code, _, errOut := run("dev", "compat", "matrix", "--probes", p+".missing"); code != 1 || errOut == "" {
		t.Errorf("missing: code %d, err %q", code, errOut)
	}
}
