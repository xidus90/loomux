package load

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/worktree/junction"
)

// ownFlow is a project flow of its own that loads against the real blocks.
const ownFlow = "schema_version = 1\n[flow]\nstart = \"stop\"\n" +
	"[[node]]\nname = \"stop\"\nkind = \"exit\"\ncode = 4\nmessage = \"done\"\n" +
	"[[edge]]\nfrom = \"stop\"\nto = \"END\"\n"

// bundledFlows is a catalog holding the example flow, laid out the way the
// binary's catalog is, README included.
func bundledFlows(t *testing.T) fstest.MapFS {
	t.Helper()
	flows := fstest.MapFS{"example/README.md": {Data: []byte("What the example does.\n")}}
	for _, file := range []string{"flow.toml", "instructions/draft.md", "questions/approve.md"} {
		flows["example/"+file] = &fstest.MapFile{Data: []byte(mustRead(t, filepath.Join("testdata", "example", filepath.FromSlash(file))))}
	}
	return flows
}

// project is a project root holding files, named by their slash path below
// the root. A name ending in "/" is an empty folder.
func project(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if strings.HasSuffix(name, "/") {
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func readFound(t *testing.T, found Found, name string) string {
	t.Helper()
	raw, err := fs.ReadFile(found.Files, name)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

var mayOverride = config.FlowSettings{Overrides: []string{"example"}}

func TestFindABundledFlow(t *testing.T) {
	found, err := Find(project(t, nil), bundledFlows(t), config.FlowSettings{}, "example")
	if err != nil {
		t.Fatal(err)
	}
	if found.Name != "example" || found.Origin != "bundled" || found.File != "bundled:example/flow.toml" {
		t.Fatalf("found = %+v", found)
	}
	if got := readFound(t, found, "questions/approve.md"); got != mustRead(t, filepath.Join("testdata", "example", "questions", "approve.md")) {
		t.Fatalf("approve.md = %q", got)
	}
	graph, err := Load(found, realCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	if graph.Origin != OriginBundled || graph.File != "bundled:example/flow.toml" {
		t.Fatalf("origin = %q, file = %q", graph.Origin, graph.File)
	}
}

func TestFindAProjectFlow(t *testing.T) {
	root := project(t, map[string]string{".loomux/flows/mine/flow.toml": ownFlow})
	found, err := Find(root, bundledFlows(t), config.FlowSettings{}, "mine")
	if err != nil {
		t.Fatal(err)
	}
	if found.Origin != "project" || found.File != ".loomux/flows/mine/flow.toml" || found.Warnings != nil {
		t.Fatalf("found = %+v", found)
	}
	if _, err := Load(found, realCatalog(t)); err != nil {
		t.Fatal(err)
	}
}

// A project flow may keep texts of its own beside its flow.toml.
func TestFindAProjectFlowWithItsTexts(t *testing.T) {
	root := project(t, map[string]string{
		".loomux/flows/mine/flow.toml":        ownFlow,
		".loomux/flows/mine/questions/ask.md": "Ship?",
	})
	found, err := Find(root, bundledFlows(t), config.FlowSettings{}, "mine")
	if err != nil {
		t.Fatal(err)
	}
	if found.Origin != OriginProject || found.Overlays != nil || readFound(t, found, "questions/ask.md") != "Ship?" {
		t.Fatalf("found = %+v", found)
	}
}

func TestFindHidesABundledFlowOnlyWithAnOverride(t *testing.T) {
	root := project(t, map[string]string{".loomux/flows/example/flow.toml": ownFlow})
	found, err := Find(root, bundledFlows(t), config.FlowSettings{}, "example")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".loomux/flows/example is ignored: [flow] overrides does not name it"}
	if found.Origin != "bundled" || !reflect.DeepEqual(found.Warnings, want) {
		t.Fatalf("without an override: %+v", found)
	}
	found, err = Find(root, bundledFlows(t), mayOverride, "example")
	if err != nil {
		t.Fatal(err)
	}
	if found.Origin != "project (hides bundled)" || found.File != ".loomux/flows/example/flow.toml" || found.Warnings != nil {
		t.Fatalf("with an override: %+v", found)
	}
	if readFound(t, found, "flow.toml") != ownFlow {
		t.Fatal("the hiding flow reads the bundled flow.toml")
	}
}

func TestFindOverlaysSingleFiles(t *testing.T) {
	root := project(t, map[string]string{".loomux/flows/example/questions/approve.md": "Really?"})
	found, err := Find(root, bundledFlows(t), mayOverride, "example")
	if err != nil {
		t.Fatal(err)
	}
	if found.Origin != "bundled+overlay" || found.File != "bundled:example/flow.toml" ||
		!reflect.DeepEqual(found.Overlays, []string{"questions/approve.md"}) {
		t.Fatalf("found = %+v", found)
	}
	if got := readFound(t, found, "questions/approve.md"); got != "Really?" {
		t.Fatalf("approve.md = %q", got)
	}
	if got := readFound(t, found, "instructions/draft.md"); got != mustRead(t, filepath.Join("testdata", "example", "instructions", "draft.md")) {
		t.Fatalf("draft.md = %q", got)
	}
	graph, err := Load(found, realCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	if graph.Origin != OriginOverlay || !reflect.DeepEqual(graph.Overlays, found.Overlays) {
		t.Fatalf("origin = %q, overlays = %v", graph.Origin, graph.Overlays)
	}
}

func TestFindIgnoresAnOverlayWithoutAnOverride(t *testing.T) {
	root := project(t, map[string]string{".loomux/flows/example/questions/approve.md": "Really?"})
	found, err := Find(root, bundledFlows(t), config.FlowSettings{}, "example")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".loomux/flows/example is ignored: [flow] overrides does not name it"}
	if found.Origin != "bundled" || found.Overlays != nil || !reflect.DeepEqual(found.Warnings, want) {
		t.Fatalf("found = %+v", found)
	}
	if got := readFound(t, found, "questions/approve.md"); got == "Really?" {
		t.Fatal("the overlay was laid over without an override")
	}
}

func TestFindRefusesAnOverlayOfNothing(t *testing.T) {
	root := project(t, map[string]string{".loomux/flows/example/questions/nope.md": "?"})
	_, err := Find(root, bundledFlows(t), mayOverride, "example")
	if err == nil || err.Error() != `.loomux/flows/example overlays nothing: bundled flow "example" has no questions/nope.md` {
		t.Fatalf("err = %v", err)
	}
}

func TestFindRefusesAnOverlayWithoutABundledFlow(t *testing.T) {
	root := project(t, map[string]string{".loomux/flows/ghost/questions/q.md": "?"})
	_, err := Find(root, bundledFlows(t), config.FlowSettings{}, "ghost")
	if err == nil || err.Error() != `.loomux/flows/ghost overlays nothing: there is no bundled flow "ghost"` {
		t.Fatalf("err = %v", err)
	}
}

// A stray file is named, not laid over and not ignored: a README beside an
// overlay would otherwise look like one file the project meant to replace.
func TestFindRefusesAStrayFileInAProjectFolder(t *testing.T) {
	for name, c := range map[string]struct {
		files map[string]string
		flow  string
		want  string
	}{
		"a README alone": {map[string]string{".loomux/flows/example/README.md": "notes"}, "example",
			".loomux/flows/example/README.md is neither flow.toml nor a file under instructions/ or questions/"},
		// An overlay replaces texts and nothing else: its README would describe
		// a flow it does not define.
		"a README beside an overlay": {map[string]string{
			".loomux/flows/example/questions/approve.md": "Really?",
			".loomux/flows/example/README.md":            "notes",
		}, "example", ".loomux/flows/example/README.md is neither flow.toml nor a file under instructions/ or questions/"},
		"tests beside an overlay": {map[string]string{
			".loomux/flows/example/questions/approve.md": "Really?",
			".loomux/flows/example/_test/journal.jsonl":  "{}",
		}, "example", ".loomux/flows/example/_test/journal.jsonl is neither flow.toml nor a file under instructions/ or questions/"},
		"a folder beside an overlay": {map[string]string{
			".loomux/flows/example/questions/approve.md": "Really?",
			".loomux/flows/example/notes/todo.md":        "later",
		}, "example", ".loomux/flows/example/notes/todo.md is neither flow.toml nor a file under instructions/ or questions/"},
		"a file beside a flow.toml": {map[string]string{
			".loomux/flows/mine/flow.toml": ownFlow,
			".loomux/flows/mine/README.md": "What mine does.",
			".loomux/flows/mine/draft.md":  "Draft.",
		}, "mine", ".loomux/flows/mine/draft.md is neither flow.toml, README.md nor a file under instructions/, questions/ or _test/"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Find(project(t, c.files), bundledFlows(t), mayOverride, c.flow)
			if err == nil || err.Error() != c.want {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
		})
	}
}

// A catalog flow copied as a template carries its README and its tests; the
// copy loads unchanged, and the loader reads neither.
func TestFindTakesACopiedCatalogFlowAsItIs(t *testing.T) {
	files := map[string]string{".loomux/flows/example/_test/cases/one/journal.jsonl": "{}"}
	for name, file := range bundledFlows(t) {
		files[".loomux/flows/"+name] = string(file.Data)
	}
	found, err := Find(project(t, files), bundledFlows(t), mayOverride, "example")
	if err != nil {
		t.Fatal(err)
	}
	if found.Origin != OriginHides {
		t.Fatalf("found = %+v", found)
	}
	if _, err := Load(found, realCatalog(t)); err != nil {
		t.Fatal(err)
	}
}

func TestFindRefusesAFileWhereAFolderBelongs(t *testing.T) {
	root := project(t, map[string]string{".loomux/flows/mine": ownFlow})
	_, err := Find(root, bundledFlows(t), config.FlowSettings{}, "mine")
	if err == nil || err.Error() != ".loomux/flows/mine is a file; a flow is a folder with flow.toml" {
		t.Fatalf("err = %v", err)
	}
}

// unreadable is a folder whose instructions/ cannot be listed.
type unreadable struct{ fstest.MapFS }

func (u unreadable) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == "instructions" {
		return nil, fs.ErrPermission
	}
	return u.MapFS.ReadDir(name)
}

// A folder that cannot be read through is not taken for what the readable
// part of it holds, and the refusal names the folder: the file system's own
// words name only a path inside it.
func TestAFolderThatCannotBeReadIsAnError(t *testing.T) {
	folder := unreadable{fstest.MapFS{"flow.toml": {Data: []byte(ownFlow)}, "instructions/a.md": {Data: []byte("A.")}}}
	_, _, err := contents(folder, ".loomux/flows/mine")
	if !errors.Is(err, fs.ErrPermission) || !strings.HasPrefix(err.Error(), ".loomux/flows/mine cannot be read: ") {
		t.Fatalf("err = %v", err)
	}
}

// A link where a flow's folder belongs -- a symbolic link, or a junction on
// Windows -- is refused as a link, not taken for a file.
func TestFindRefusesALinkWhereAFolderBelongs(t *testing.T) {
	root := project(t, map[string]string{"elsewhere/flow.toml": ownFlow, ".loomux/flows/": ""})
	link(t, filepath.Join(root, "elsewhere"), filepath.Join(root, ".loomux", "flows", "mine"))
	const want = ".loomux/flows/mine is a link; a flow is a folder with flow.toml"
	if _, err := Find(root, bundledFlows(t), config.FlowSettings{}, "mine"); err == nil || err.Error() != want {
		t.Fatalf("err = %v", err)
	}
	got := List(root, bundledFlows(t), config.FlowSettings{}, realCatalog(t))
	if len(got) != 2 || got[1].Name != "mine" || got[1].Problem != want {
		t.Fatalf("list = %+v", got)
	}
}

// link makes link point at the folder target: a junction on Windows, where a
// symbolic link needs a privilege, and a symbolic link elsewhere.
func link(t *testing.T, target, link string) {
	t.Helper()
	err := junction.Create(link, target)
	if errors.Is(err, junction.ErrUnsupported) {
		err = os.Symlink(target, link)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestFindRefusesAnEmptyProjectFolder(t *testing.T) {
	root := project(t, map[string]string{".loomux/flows/mine/instructions/": ""})
	_, err := Find(root, bundledFlows(t), config.FlowSettings{}, "mine")
	if err == nil || err.Error() != ".loomux/flows/mine holds no flow.toml and nothing to overlay" {
		t.Fatalf("err = %v", err)
	}
}

// The name reaches Find from the command line and from a run's marker, and it
// is joined into a path. "../../evil" is exactly what a name is not.
func TestFindRefusesABadName(t *testing.T) {
	for _, name := range []string{"Dev", "../../evil", "with space", "1st", ""} {
		_, err := Find(project(t, nil), bundledFlows(t), config.FlowSettings{}, name)
		want := `"` + name + `" is not a flow name; a flow name is [a-z][a-z0-9-]*`
		if err == nil || err.Error() != want {
			t.Fatalf("%q: err = %v", name, err)
		}
	}
}

func TestFindNamesTheKnownFlows(t *testing.T) {
	root := project(t, map[string]string{".loomux/flows/mine/flow.toml": ownFlow})
	_, err := Find(root, bundledFlows(t), config.FlowSettings{}, "nope")
	if err == nil || err.Error() != `no flow named "nope"; known flows: example, mine` {
		t.Fatalf("err = %v", err)
	}
}

// A flow that will not load is listed with its reason rather than left out:
// a flow that disappears from the list reads as a flow nobody wrote.
func TestListNamesEveryFlowOnceWithItsOrigin(t *testing.T) {
	root := project(t, map[string]string{
		".loomux/flows/mine/flow.toml":                ownFlow,
		".loomux/flows/bad/flow.toml":                 "schema_version = 2\n",
		".loomux/flows/1st/flow.toml":                 ownFlow,
		".loomux/flows/notes.md":                      "not a flow",
		".loomux/flows/.gitkeep":                      "",
		".loomux/flows/.draft/flow.toml":              ownFlow,
		".loomux/flows/example/questions/approve.md":  "Really?",
		".loomux/flows/review/instructions/review.md": "Review.",
	})
	flows := bundledFlows(t)
	flows["review/flow.toml"] = &fstest.MapFile{Data: []byte("schema_version = 1\n")}
	flows["review/instructions/review.md"] = &fstest.MapFile{Data: []byte("Review it.")}
	flows["drafts/notes.md"] = &fstest.MapFile{Data: []byte("no flow.toml, no flow")}
	settings := config.FlowSettings{Default: "mine", Overrides: []string{"review"}}
	got := List(root, flows, settings, realCatalog(t))
	var names []string
	for _, entry := range got {
		names = append(names, entry.Name)
	}
	// Sorted, so a list is the same list twice running. notes.md is listed as
	// what it is not; drafts/ ships no flow, and .gitkeep and .draft/ are
	// passed over like every name starting with ".".
	if want := []string{"1st", "bad", "example", "mine", "notes.md", "review"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	if got[0].Origin != "" || !strings.Contains(got[0].Problem, "is not a flow name") {
		t.Fatalf("1st: %+v", got[0])
	}
	if got[1].Origin != OriginProject || !strings.Contains(got[1].Problem, "schema_version 2 is unknown") {
		t.Fatalf("bad: %+v", got[1])
	}
	warning := []string{".loomux/flows/example is ignored: [flow] overrides does not name it"}
	if got[2].Origin != OriginBundled || got[2].Problem != "" || !reflect.DeepEqual(got[2].Warnings, warning) || got[2].Default {
		t.Fatalf("example: %+v", got[2])
	}
	if got[3].Origin != OriginProject || got[3].Problem != "" || !got[3].Default {
		t.Fatalf("mine: %+v", got[3])
	}
	if got[4].Origin != "" || got[4].Problem != ".loomux/flows/notes.md is a file; a flow is a folder with flow.toml" {
		t.Fatalf("notes.md: %+v", got[4])
	}
	if got[5].Origin != OriginOverlay || !strings.Contains(got[5].Problem, "[flow] is missing") {
		t.Fatalf("review: %+v", got[5])
	}
}

// A file where a bundled flow's folder would be is listed once, under the
// flow, with what Find says about it: refused where [flow] overrides names
// the flow, ignored with a warning where it does not.
func TestListNamesAFileInAFlowsPlaceOnce(t *testing.T) {
	root := project(t, map[string]string{".loomux/flows/example": ownFlow})
	got := List(root, bundledFlows(t), mayOverride, realCatalog(t))
	if len(got) != 1 || got[0].Name != "example" || got[0].Problem != ".loomux/flows/example is a file; a flow is a folder with flow.toml" {
		t.Fatalf("%+v", got)
	}
	got = List(root, bundledFlows(t), config.FlowSettings{}, realCatalog(t))
	if len(got) != 1 || got[0].Problem != "" || got[0].Origin != OriginBundled || len(got[0].Warnings) != 1 {
		t.Fatalf("%+v", got)
	}
}

// Without [flow] overrides the bundled flow runs, and nothing the project
// folder holds can stop it: not a stray file, not an empty folder, not a file
// in the folder's place. The folder is named in a warning and read no further.
func TestFindRunsTheBundledFlowWhateverAnIgnoredFolderHolds(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"a README alone":               {".loomux/flows/example/README.md": "notes"},
		"a stray file by an overlay":   {".loomux/flows/example/questions/approve.md": "Really?", ".loomux/flows/example/notes.md": "later"},
		"an overlay of nothing":        {".loomux/flows/example/questions/nope.md": "?"},
		"an empty folder":              {".loomux/flows/example/instructions/": ""},
		"a file in the folder's place": {".loomux/flows/example": ownFlow},
		"a stray file by a flow.toml":  {".loomux/flows/example/flow.toml": ownFlow, ".loomux/flows/example/draft.md": "Draft."},
	} {
		t.Run(name, func(t *testing.T) {
			found, err := Find(project(t, files), bundledFlows(t), config.FlowSettings{}, "example")
			if err != nil {
				t.Fatal(err)
			}
			want := []string{".loomux/flows/example is ignored: [flow] overrides does not name it"}
			if found.Origin != OriginBundled || found.Overlays != nil || !reflect.DeepEqual(found.Warnings, want) {
				t.Fatalf("found = %+v", found)
			}
			if got := readFound(t, found, "questions/approve.md"); got != mustRead(t, filepath.Join("testdata", "example", "questions", "approve.md")) {
				t.Fatalf("approve.md = %q", got)
			}
		})
	}
}

// A folder counts only when its name is spelled exactly like the flow's: on
// Windows and macOS the file system would find Example for example, on Linux
// it would not, and a flow must be found the same way everywhere.
func TestFindTrustsOnlyAFolderSpelledLikeTheFlow(t *testing.T) {
	root := project(t, map[string]string{".loomux/flows/Example/flow.toml": ownFlow})
	found, err := Find(root, bundledFlows(t), mayOverride, "example")
	if err != nil {
		t.Fatal(err)
	}
	if found.Origin != OriginBundled || found.Warnings != nil {
		t.Fatalf("found = %+v", found)
	}
	_, err = Find(root, fstest.MapFS{}, config.FlowSettings{}, "example")
	if err == nil || err.Error() != `no flow named "example"; known flows: Example` {
		t.Fatalf("err = %v", err)
	}
	var names []string
	for _, entry := range List(root, bundledFlows(t), mayOverride, realCatalog(t)) {
		names = append(names, entry.Name)
	}
	if want := []string{"Example", "example"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v", names)
	}
}

// A flows folder that cannot be read as one says so. It stops a project flow
// and a bundled flow the project may override; a bundled flow it may not
// override runs, with the reason in a warning.
func TestFindSaysSoWhenTheFlowsFolderCannotBeRead(t *testing.T) {
	root := project(t, map[string]string{".loomux/flows": "not a folder"})
	const prefix = ".loomux/flows cannot be read as a folder of flows: "
	if _, err := Find(root, bundledFlows(t), config.FlowSettings{}, "mine"); err == nil || !strings.HasPrefix(err.Error(), prefix) {
		t.Fatalf("mine: err = %v", err)
	}
	if _, err := Find(root, bundledFlows(t), mayOverride, "example"); err == nil || !strings.HasPrefix(err.Error(), prefix) {
		t.Fatalf("example with an override: err = %v", err)
	}
	found, err := Find(root, bundledFlows(t), config.FlowSettings{}, "example")
	if err != nil {
		t.Fatal(err)
	}
	if found.Origin != OriginBundled || len(found.Warnings) != 1 || !strings.HasPrefix(found.Warnings[0], prefix) {
		t.Fatalf("found = %+v", found)
	}
}

// A name starting with "." is nobody's flow text: a folder passes over
// .DS_Store, .gitkeep and an editor's .draft/ wherever they sit.
func TestFindPassesOverDotNames(t *testing.T) {
	root := project(t, map[string]string{
		".loomux/flows/example/questions/approve.md":  "Really?",
		".loomux/flows/example/.DS_Store":             "",
		".loomux/flows/example/.draft/questions/x.md": "?",
		".loomux/flows/example/instructions/.gitkeep": "",
		".loomux/flows/mine/flow.toml":                ownFlow,
		".loomux/flows/mine/.DS_Store":                "",
		".loomux/flows/empty/.gitkeep":                "",
	})
	found, err := Find(root, bundledFlows(t), mayOverride, "example")
	if err != nil {
		t.Fatal(err)
	}
	if found.Origin != OriginOverlay || !reflect.DeepEqual(found.Overlays, []string{"questions/approve.md"}) {
		t.Fatalf("example: %+v", found)
	}
	if found, err := Find(root, bundledFlows(t), config.FlowSettings{}, "mine"); err != nil || found.Origin != OriginProject {
		t.Fatalf("mine: %+v, %v", found, err)
	}
	_, err = Find(root, bundledFlows(t), config.FlowSettings{}, "empty")
	if err == nil || err.Error() != ".loomux/flows/empty holds no flow.toml and nothing to overlay" {
		t.Fatalf("empty: err = %v", err)
	}
}

func TestListOfAProjectWithoutFlowsIsTheCatalog(t *testing.T) {
	got := List(project(t, nil), bundledFlows(t), config.FlowSettings{}, realCatalog(t))
	if len(got) != 1 || got[0].Name != "example" || got[0].Origin != OriginBundled || got[0].Problem != "" {
		t.Fatalf("%+v", got)
	}
}

// An override of a flow the catalog does not ship is a warning; a default
// that names no flow is not one of them, it is MissingDefault's refusal.
func TestSettingsWarningsNameWhatIsNotThere(t *testing.T) {
	got := SettingsWarnings(bundledFlows(t), config.FlowSettings{Default: "ghost", Overrides: []string{"example", "ghost"}})
	if want := []string{`[flow] overrides names "ghost", which no bundled flow has`}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	// An override of a project flow is not there either, for a project flow
	// hides nothing.
	got = SettingsWarnings(bundledFlows(t), config.FlowSettings{Overrides: []string{"mine"}})
	if want := []string{`[flow] overrides names "mine", which no bundled flow has`}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
	if got := SettingsWarnings(bundledFlows(t), config.FlowSettings{}); got != nil {
		t.Fatalf("got %q", got)
	}
}

// A default is a flow of the project or of the catalog; one that is neither
// is refused, and an unset default is no mistake.
func TestMissingDefaultRefusesADefaultThatNamesNoFlow(t *testing.T) {
	root := project(t, map[string]string{".loomux/flows/mine/flow.toml": ownFlow})
	err := MissingDefault(root, bundledFlows(t), config.FlowSettings{Default: "ghost"})
	if err == nil || err.Error() != `[flow] default names "ghost", which is no flow here` {
		t.Fatalf("err = %v", err)
	}
	for _, name := range []string{"", "mine", "example"} {
		if err := MissingDefault(root, bundledFlows(t), config.FlowSettings{Default: name}); err != nil {
			t.Fatalf("default %q: %v", name, err)
		}
	}
}

func TestFindSaysSoWhenThereAreNoFlowsAtAll(t *testing.T) {
	_, err := Find(project(t, nil), fstest.MapFS{}, config.FlowSettings{}, "other")
	if err == nil || err.Error() != `no flow named "other"; known flows: none` {
		t.Fatalf("err = %v", err)
	}
}
