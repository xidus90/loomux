package hooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/verify"
)

func editEnv(t *testing.T, answer func(child.Spec) child.Result, seen *[]string) EditEnv {
	t.Helper()
	var mu sync.Mutex
	return EditEnv{
		Start: func(s child.Spec) child.Result {
			mu.Lock()
			*seen = append(*seen, s.Dir+"|"+strings.Join(s.Argv, " "))
			mu.Unlock()
			return answer(s)
		},
		Look:   func(s string) (string, error) { return s, nil },
		Loomux: "loomux",
		Budget: DefaultBudget,
		Now:    time.Now,
	}
}

func passing(child.Spec) child.Result { return child.Result{} }

// goProject is a root detection calls a Go module.
func goProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module m\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// postEdit runs one payload against root and answers every tool with answer.
func postEdit(t *testing.T, root, payload string, answer func(child.Spec) child.Result) (int, string, string, []string) {
	t.Helper()
	seen := []string{}
	var so, se bytes.Buffer
	code := RunPostEdit(strings.NewReader(payload), &so, &se, root, editEnv(t, answer, &seen))
	return code, so.String(), se.String(), seen
}

func filePayload(t *testing.T, path string) string {
	t.Helper()
	return `{"tool_name":"Edit","tool_input":{"file_path":` + asJSON(t, path) + `}}`
}

func TestPostEditRunsTheEditProfileOfTheFilesStack(t *testing.T) {
	root := goProject(t)
	code, _, se, seen := postEdit(t, root, filePayload(t, filepath.Join(root, "a.go")), passing)
	if code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	got := strings.Join(seen, "\n")
	if !strings.Contains(got, root+"|go vet ./...") || !strings.Contains(got, "loomux check gofmt a.go") || strings.Contains(got, "go test") {
		t.Fatalf("edit runs lint and types only, at the root: %v", seen)
	}
}

func TestPostEditBlocksOnARedLane(t *testing.T) {
	root := goProject(t)
	code, _, se, _ := postEdit(t, root, `{"tool_input":{"file_path":"a.go"}}`, func(child.Spec) child.Result {
		return child.Result{Code: 1, Stdout: "vet: bad\n"}
	})
	if code != ExitDenied || !strings.Contains(se, "vet: bad") || !strings.Contains(se, "lint/go: failed") {
		t.Fatalf("%d %q", code, se)
	}
}

// A lane whose tool is missing blocks no edit, but it is named where the
// harness reads a hook that exited 0: hookSpecificOutput.additionalContext.
func TestPostEditSkipsAMissingToolOutLoud(t *testing.T) {
	root := goProject(t)
	seen := []string{}
	env := editEnv(t, passing, &seen)
	env.Look = func(string) (string, error) { return "", errors.New("not found") }
	var so, se bytes.Buffer
	code := RunPostEdit(strings.NewReader(`{"tool_input":{"file_path":"a.go"}}`), &so, &se, root, env)
	if code != ExitOK || !strings.Contains(so.String(), "lane skipped") {
		t.Fatalf("%d %q", code, so.String())
	}
	// Walked by key, not decoded into the struct the code wrote, which would
	// pass whatever field the code chose.
	var said map[string]any
	if err := json.Unmarshal(so.Bytes(), &said); err != nil {
		t.Fatalf("stdout has to be one JSON document, got %q: %v", so.String(), err)
	}
	specific, _ := said["hookSpecificOutput"].(map[string]any)
	context, _ := specific["additionalContext"].(string)
	if specific["hookEventName"] != "PostToolUse" || !strings.Contains(context, `"go" is not on PATH`) {
		t.Fatalf("the notice is not where the harness reads it: %q", so.String())
	}
	if len(seen) != 0 {
		t.Fatalf("no tool may start when it is missing: %v", seen)
	}
}

// A payload that names no file, or a file nothing checks, starts nothing and
// blocks nothing.
func TestPostEditLeavesAlonePayloadsAndFilesItHasNoLaneFor(t *testing.T) {
	root := goProject(t)
	outside := filepath.Join(t.TempDir(), "b.go")
	for name, payload := range map[string]string{
		"not json":          `not json`,
		"no tool input":     `{"tool_name":"Edit"}`,
		"empty file_path":   `{"tool_input":{"file_path":""}}`,
		"empty both":        `{"tool_input":{"file_path":"","notebook_path":""}}`,
		"ignored extension": `{"tool_input":{"file_path":"data.json"}}`,
		"ignored upper":     `{"tool_input":{"file_path":"DATA.JSON"}}`,
		"unknown extension": `{"tool_input":{"file_path":"a.xyz"}}`,
		"no extension":      `{"tool_input":{"file_path":"Makefile"}}`,
		"relative outside":  `{"tool_input":{"file_path":"../elsewhere/a.go"}}`,
		"absolute outside":  filePayload(t, outside),
		"inactive stack":    `{"tool_input":{"file_path":"a.py"}}`,
	} {
		code, so, se, seen := postEdit(t, root, payload, passing)
		if code != ExitOK || so != "" || se != "" || len(seen) != 0 {
			t.Errorf("%s: %d %q %q %v", name, code, so, se, seen)
		}
	}
}

// A notebook edit names its file under notebook_path; it is checked like any
// other file of its stack.
func TestPostEditReadsTheNotebookPath(t *testing.T) {
	root := goProject(t)
	code, _, se, seen := postEdit(t, root, `{"tool_input":{"notebook_path":"a.go"}}`, passing)
	if code != ExitOK || len(seen) == 0 {
		t.Fatalf("%d %q %v", code, se, seen)
	}
}

// A broken config blocks no edit, but it says why: exit 1 shows the message
// without refusing the change.
func TestPostEditReportsAConfigItCannotRead(t *testing.T) {
	for name, body := range map[string]string{
		"toml":     "[verify\n",
		"schema":   "[verify]\nlint = \"x\"\n",
		"resolved": "[verify.go.test]\nafter = \"coverage\"\n",
	} {
		root := goProject(t)
		writeManifest(t, root, body)
		code, _, se, seen := postEdit(t, root, `{"tool_input":{"file_path":"data.json"}}`, passing)
		if code != ExitInternal || !strings.HasPrefix(se, "loomux hook post-tool-use: ") || len(seen) != 0 {
			t.Errorf("%s: %d %q", name, code, se)
		}
	}
}

// Presets that fail to load and a plan that fails cannot be provoked from a
// project; they stand in through their seams and end like a broken config.
func TestPostEditReportsPresetsAndPlansThatFail(t *testing.T) {
	boom := errors.New("boom")
	loadPresets, plan := editPresets, editPlan
	t.Cleanup(func() { editPresets, editPlan = loadPresets, plan })

	editPresets = func() (*verify.Presets, error) { return nil, boom }
	if code, _, se, _ := postEdit(t, goProject(t), `{"tool_input":{"file_path":"a.go"}}`, passing); code != ExitInternal || se != "loomux hook post-tool-use: boom\n" {
		t.Fatalf("presets: %d %q", code, se)
	}
	editPresets = loadPresets

	editPlan = func(verify.Effective, verify.Request, verify.PlanEnv) ([]verify.Job, error) { return nil, boom }
	if code, _, se, _ := postEdit(t, goProject(t), `{"tool_input":{"file_path":"a.go"}}`, passing); code != ExitInternal || se != "loomux hook post-tool-use: boom\n" {
		t.Fatalf("plan: %d %q", code, se)
	}
}

// measuringEdit stands in for go test: it writes the profile it is told to,
// and like go test it fails when the directory for it is not there.
func measuringEdit(s child.Spec) child.Result {
	for _, a := range s.Argv {
		if p, ok := strings.CutPrefix(a, "-coverprofile="); ok {
			if err := os.WriteFile(p, []byte("mode: set\n"), 0o644); err != nil {
				return child.Result{Code: 1, Stderr: err.Error()}
			}
		}
	}
	return child.Result{}
}

// An edit profile that measures coverage needs the cover directory, which
// the agent cannot make: the policy refuses writes under .loomux/state. The
// hook makes it, and a green run leaves nothing of its own behind.
func TestPostEditMakesTheCoverDirectoryForAMeasuringProfile(t *testing.T) {
	root := goProject(t)
	os.WriteFile(filepath.Join(root, "a_test.go"), []byte("package m\n"), 0o644)
	writeManifest(t, root, "[verify.profiles]\nedit = [\"lint\", \"test\", \"coverage\"]\n")
	code, so, se, seen := postEdit(t, root, `{"tool_input":{"file_path":"a.go"}}`, measuringEdit)
	if code != ExitOK {
		t.Fatalf("%d %q %q %v", code, so, se, seen)
	}
	if !strings.Contains(strings.Join(seen, "\n"), "-coverprofile=") {
		t.Fatalf("test must measure: %v", seen)
	}
	entries, err := os.ReadDir(filepath.Join(root, ".loomux", "state", "cover"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("a green edit cleans its own profile: %v %v", entries, err)
	}
}

// A cover directory that cannot be made ends the hook like a broken config,
// before any tool starts.
func TestPostEditReportsACoverDirectoryItCannotMake(t *testing.T) {
	root := goProject(t)
	os.MkdirAll(filepath.Join(root, ".loomux"), 0o755)
	os.WriteFile(filepath.Join(root, ".loomux", "state"), nil, 0o644)
	code, _, se, seen := postEdit(t, root, `{"tool_input":{"file_path":"a.go"}}`, passing)
	if code != ExitInternal || !strings.HasPrefix(se, "loomux hook post-tool-use: ") || len(seen) != 0 {
		t.Fatalf("%d %q %v", code, se, seen)
	}
}

// Files left behind cost disk, not correctness: the edit passes and the
// hook says what it could not remove.
func TestPostEditWarnsWhenItCannotCleanTheCoverDirectory(t *testing.T) {
	root := goProject(t)
	stale := filepath.Join(root, ".loomux", "state", "cover", "old")
	os.MkdirAll(stale, 0o755)
	os.WriteFile(filepath.Join(stale, "x"), nil, 0o644)
	past := time.Now().Add(-48 * time.Hour)
	os.Chtimes(stale, past, past)
	code, _, se, _ := postEdit(t, root, `{"tool_input":{"file_path":"a.go"}}`, passing)
	if code != ExitOK || !strings.HasPrefix(se, "loomux hook post-tool-use: cleaning coverage files: ") {
		t.Fatalf("%d %q", code, se)
	}
}

// A lane runs in the area holding the file, and {file} names it from there.
func TestPostEditRunsATypeScriptLaneInItsArea(t *testing.T) {
	root := t.TempDir()
	web := filepath.Join(root, "web")
	if err := os.MkdirAll(filepath.Join(web, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"package.json", "tsconfig.json"} {
		if err := os.WriteFile(filepath.Join(web, name), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	code, _, se, seen := postEdit(t, root, filePayload(t, filepath.Join(web, "src", "a.ts")), passing)
	if code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	got := strings.Join(seen, "\n")
	if !strings.Contains(got, web+"|npx eslint --cache src/a.ts") || !strings.Contains(got, web+"|npx tsc --noEmit") {
		t.Fatalf("the lanes run in web/ on src/a.ts: %v", seen)
	}
}

// A spent budget drops the lanes it did not reach and says so; an edit is
// never refused for want of time.
func TestPostEditPassesALaneTheBudgetDidNotReach(t *testing.T) {
	root := goProject(t)
	seen := []string{}
	env := editEnv(t, passing, &seen)
	var calls atomic.Int64
	start := time.Now()
	// Every reading of the clock lies 30s after the one before, so whatever
	// reads it after the deadline was taken finds the 10s budget spent.
	env.Now = func() time.Time { return start.Add(time.Duration(calls.Add(1)) * 30 * time.Second) }
	env.Budget = 10 * time.Second
	var so, se bytes.Buffer
	code := RunPostEdit(strings.NewReader(`{"tool_input":{"file_path":"a.go"}}`), &so, &se, root, env)
	if code != ExitOK || !strings.Contains(so.String(), "the edit budget ran out: lint/go") || len(seen) != 0 {
		t.Fatalf("%d %q %q %v", code, so.String(), se.String(), seen)
	}
}

// wikiProject declares docs/wiki and holds one page in it.
func wikiProject(t *testing.T, manifest, body string) (string, string) {
	t.Helper()
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"docs/wiki\"\n"+manifest)
	return root, writeWikiPage(t, root, "page.md", body)
}

const cleanPage = "---\ntitle: Sample Concept\ntype: concept\ndescription: A valid OKF test document\n---\n\n# Sample Concept\nThis is a test concept.\n"

// A wiki page is linted in this process: no tool starts for it, and a page
// the lint refuses blocks the edit, whether the path comes relative or
// absolute.
func TestPostEditLintsAWikiPageInProcess(t *testing.T) {
	root, page := wikiProject(t, "", "no frontmatter at all\n")
	for _, payload := range []string{`{"tool_input":{"file_path":"docs/wiki/page.md"}}`, filePayload(t, page)} {
		code, _, se, seen := postEdit(t, root, payload, passing)
		if code != ExitDenied || !strings.Contains(se, "lint/wiki: failed") || !strings.Contains(se, "page.md") || len(seen) != 0 {
			t.Fatalf("%d %q %v", code, se, seen)
		}
	}
}

func TestPostEditLetsACleanWikiPageThrough(t *testing.T) {
	root, _ := wikiProject(t, "", cleanPage)
	code, so, se, _ := postEdit(t, root, `{"tool_input":{"file_path":"docs/wiki/page.md"}}`, passing)
	if code != ExitOK || so != "" || se != "" {
		t.Fatalf("%d %q %q", code, so, se)
	}
}

// Markdown outside the wiki, a wiki nobody declared, and a wiki lane switched
// off check nothing.
func TestPostEditLeavesMarkdownAloneWhereNoWikiLaneRuns(t *testing.T) {
	root, _ := wikiProject(t, "", "no frontmatter at all\n")
	if code, _, se, _ := postEdit(t, root, `{"tool_input":{"file_path":"README.md"}}`, passing); code != ExitOK || se != "" {
		t.Errorf("outside the wiki: %d %q", code, se)
	}
	off, _ := wikiProject(t, "[verify.wiki]\nlint = false\n", "no frontmatter at all\n")
	if code, _, se, _ := postEdit(t, off, `{"tool_input":{"file_path":"docs/wiki/page.md"}}`, passing); code != ExitOK || se != "" {
		t.Errorf("lint = false: %d %q", code, se)
	}
	bare := t.TempDir()
	writeWikiPage(t, bare, "page.md", "no frontmatter at all\n")
	if code, _, se, _ := postEdit(t, bare, `{"tool_input":{"file_path":"docs/wiki/page.md"}}`, passing); code != ExitOK || se != "" {
		t.Errorf("no wiki declared: %d %q", code, se)
	}
}

// The production door, through real tools: go vet on a module that compiles.
// The lint lane is overridden so {loomux} does not name the test binary, which
// would run this suite again.
func TestPostToolUseRunsTheGoLaneThroughRealTools(t *testing.T) {
	root := goProject(t)
	writeManifest(t, root, "[verify.go]\nlint = \"go vet ./...\"\n")
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package sample\n\nfunc Sample() int { return 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var so, se bytes.Buffer
	if code := PostToolUse(strings.NewReader(`{"tool_name":"Edit","tool_input":{"file_path":"sample.go"}}`), &so, &se, root, DefaultBudget); code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, se.String())
	}
	// A skipped lane says so on stdout, so silence tells a lane that ran and
	// passed from one that never started.
	if so.Len() != 0 {
		t.Fatalf("the lane did not run: %s", so.String())
	}
}

// Without its own path the binary still names itself for {loomux}.
func TestPostToolUseFallsBackToTheCommandName(t *testing.T) {
	executable := editExecutable
	t.Cleanup(func() { editExecutable = executable })
	editExecutable = func() (string, error) { return "", errors.New("no path") }
	var so, se bytes.Buffer
	if code := PostToolUse(strings.NewReader(`{"tool_input":{"file_path":"data.json"}}`), &so, &se, t.TempDir(), DefaultBudget); code != ExitOK {
		t.Fatalf("%d %q", code, se.String())
	}
}

func TestIsWikiPath(t *testing.T) {
	tests := []struct {
		rawPath  string
		root     string
		wikiDir  string
		expected bool
	}{
		{"wiki/index.md", "", "wiki/", true},
		{"docs/wiki/page.md", "", "docs/wiki", true},
		{"mybundle/page.md", "", "mybundle", true},
		{"sub/mybundle/page.md", "", "mybundle", true},
		{"README.md", "", "wiki/", false},
		{"docs/README.md", "", "wiki/", false},
		{"wiki", "", "", true},
		{"/root/docs/wiki/page.md", "/root", "docs/wiki", true},
		{"/root/other/page.md", "/root", "docs/wiki", false},
	}

	for _, tt := range tests {
		got := isWikiPath(tt.rawPath, tt.root, tt.wikiDir)
		if got != tt.expected {
			t.Errorf("isWikiPath(%q, %q, %q) = %v, want %v", tt.rawPath, tt.root, tt.wikiDir, got, tt.expected)
		}
	}
}

// The wiki directory itself, named absolutely: the path carries no separator
// after the directory's name, so only the form relative to the root answers.
func TestIsWikiPathMatchesTheWikiDirectoryItself(t *testing.T) {
	root := t.TempDir()
	if !isWikiPath(filepath.Join(root, "notes"), root, "notes") {
		t.Fatal("the wiki directory itself is a wiki path")
	}
}

func writeWikiPage(t *testing.T, root, name, body string) string {
	t.Helper()
	dir := filepath.Join(root, "docs", "wiki")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(dir, name)
	if err := os.WriteFile(page, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return page
}

// asJSON quotes a path the way a hook payload carries it, backslashes and all.
func asJSON(t *testing.T, value string) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func writeManifest(t *testing.T, root, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The manifest is the one place a loomux project names its wiki.
func TestTheWikiDirectoryComesFromTheManifest(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"notes\"\n")
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := wikiDirFor(root); got != "notes" {
		t.Fatalf("got %q", got)
	}
}

// Nothing declared and nothing on disk: the default stands.
func TestTheWikiDirectoryFallsBackToTheDefault(t *testing.T) {
	if got := wikiDirFor(t.TempDir()); got != "wiki/" {
		t.Fatalf("got %q", got)
	}
}

// A declared wiki with no directory behind it: wiki.Root answers nothing, and
// detection has the say.
func TestTheWikiDirectoryFallsBackToTheDetectedWiki(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[wiki]\n")

	if got := wikiDirFor(root); got != "wiki/" {
		t.Fatalf("got %q, want the detected wiki", got)
	}
}

// A neighbour wiki lies beside the project, so the answer leaves the root --
// which is the place wiki.Root found and the one the lane has to read.
func TestTheWikiDirectoryCanNameANeighbour(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "iam_backend")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(parent, "iam_wiki"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := wikiDirFor(root); got != "../iam_wiki" {
		t.Fatalf("got %q", got)
	}
}

// A loomux project declares its wiki as `[layout] wiki`, and detection does
// not read that key: it knows `[wiki]`, `wiki = true` and `okf_version` only.
// Without this the lane never fired in the pilot's own repository.
func TestTheLayoutWikiAddsTheWikiStack(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"notes\"\n")
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(root, "notes", "page.md")
	if err := os.WriteFile(page, []byte("no frontmatter at all\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	input := `{"tool_name":"Edit","tool_input":{"file_path":` + asJSON(t, page) + `}}`
	if code := PostToolUse(strings.NewReader(input), &stdout, &stderr, root, DefaultBudget); code != ExitDenied {
		t.Fatalf("code %d, err %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "page.md") {
		t.Fatalf("err %q", stderr.String())
	}
}

// The precedence wikiDirFor documents has to be the one the caller uses.
//
// A manifest carrying both a `[wiki]` table and a `[layout] wiki` elsewhere
// made detection answer first -- `wiki/`, which nobody created -- and the lane
// then judged the edited page against a directory the manifest does not mean.
// The page lies in the declared bundle and is broken, so the lane has to
// refuse.
func TestTheDeclaredWikiOutranksTheDetectedOne(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n[wiki]\n[layout]\nwiki = \"notes\"\n")
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(root, "notes", "page.md")
	if err := os.WriteFile(page, []byte("no frontmatter at all\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	input := `{"tool_name":"Edit","tool_input":{"file_path":` + asJSON(t, page) + `}}`
	if code := PostToolUse(strings.NewReader(input), &stdout, &stderr, root, DefaultBudget); code != ExitDenied {
		t.Fatalf("code %d, err %q: the lane read the detected wiki, not the declared one", code, stderr.String())
	}
}

// A manifest without that key, and a directory without a manifest, declare
// nothing.
func TestAProjectWithoutALayoutWikiDeclaresNothing(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n")

	if declaresWikiLayout(root) {
		t.Fatal("a manifest without [layout] wiki declares no wiki")
	}
	if declaresWikiLayout(filepath.Join(root, "nowhere")) {
		t.Fatal("a directory without a manifest declares no wiki")
	}
}

// A declared layout that names no directory is no declaration either: wiki.Root
// passes it by, and the lane would read a place the manifest does not mean.
func TestALayoutWikiWithoutADirectoryDeclaresNothing(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"notes\"\n")

	if declaresWikiLayout(root) {
		t.Fatal("a layout nobody created declares no wiki")
	}
}

// A layout that leaves the repository is refused by WikiLayout, and the
// refusal arrives here as "nothing declared".
func TestAnInvalidLayoutWikiDeclaresNothing(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"../elsewhere\"\n")

	if declaresWikiLayout(root) {
		t.Fatal("a layout outside the repository declares no wiki")
	}
}

// The bench corpus asks which stack an ending belongs to and which commands
// post-edit runs; both answers come from the presets, and presets that fail
// to load leave neither an answer.
func TestPresetAnswersForTheBenchCorpus(t *testing.T) {
	got := EditLaneCommands([]string{"python", "go", "nothing"})
	want := []string{
		"uvx ruff check . --output-format=concise",
		"uv run mypy --no-error-summary --no-pretty",
		"go vet ./...",
		"loomux check gofmt .",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("EditLaneCommands = %q, want %q", got, want)
	}

	loadPresets := editPresets
	t.Cleanup(func() { editPresets = loadPresets })
	editPresets = func() (*verify.Presets, error) { return nil, errors.New("boom") }
	if stack, ok := StackForExtension(".go"); ok || stack != "" {
		t.Errorf("StackForExtension without presets = %q, %v", stack, ok)
	}
	if got := EditLaneCommands([]string{"go"}); got != nil {
		t.Errorf("EditLaneCommands without presets = %q", got)
	}
}
