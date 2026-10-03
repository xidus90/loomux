package search_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
)

type mockSearchPort struct {
	searchFunc func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error)
	calls      int
}

func (m *mockSearchPort) Search(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
	m.calls++
	if m.searchFunc != nil {
		return m.searchFunc(query, collections, profile, n)
	}
	return nil, nil
}

func (m *mockSearchPort) Indexed(collection string) ([]string, error) { return nil, nil }
func (m *mockSearchPort) Refresh(collections []string) error          { return nil }
func (m *mockSearchPort) NotYetSearchable() (int, error)              { return 0, nil }
func (m *mockSearchPort) Embed(collections []string) error            { return nil }

func writeRegistry(t *testing.T, dir string, content string) {
	t.Helper()
	path := filepath.Join(dir, "registry.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write registry: %v", err)
	}
}

// searchNow is the clock of every search test. A world carries a stamp only where a test
// writes one, and then it is judged against this time.
var searchNow = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

// Measured against core._ask and core._assemble on 2026-09-15.
const (
	twiceEmptyFast = "the search engine answered empty twice in a row on profile fast; an empty answer to a meaning search is practically unreachable, so this is more likely a silent failure of the engine than an absence of matches (spec 16.14)"
	withheldOne    = "1 of the engine's hits were withheld -- excluded by [privacy] never, or from a collection this channel has no area for; the list is that many places shorter than it could have been"
	withheldTwo    = "2 of the engine's hits were withheld -- excluded by [privacy] never, or from a collection this channel has no area for; the list is that many places shorter than it could have been"
	agedStamp      = "the last full reconciliation was 2000-01-01T00:00:00+00:00, more than 24 hours ago: a source may have changed without this answer knowing (run `brain reconcile`)"
)

// areaEntry is one registry table for an area at path.
func areaEntry(scope, path string) string {
	return "[[area]]\nscope = \"" + scope + "\"\npath = \"" + filepath.ToSlash(path) + "\"\n\n"
}

// writeArea lays down an area's directory, its manifest and, when registered paths are
// given, its identity register: a registry entry alone says nothing about the area's
// privacy mode, and a hit outside the register is a finding.
func writeArea(t *testing.T, root, scope, manifest string, registered ...string) string {
	t.Helper()
	dir := filepath.Join(root, search.CollectionName(scope))
	if err := os.MkdirAll(filepath.Join(dir, ".loomux"), 0o750); err != nil {
		t.Fatalf("failed to create area dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".loomux", "config.toml"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if len(registered) > 0 {
		register := "doc_id\tpfad\tcontent_hash\trevision\n"
		for i, relative := range registered {
			register += "id-" + string(rune('a'+i)) + "\t" + relative + "\tsha256:0\t1\n"
		}
		if err := os.WriteFile(filepath.Join(dir, "_identities.tsv"), []byte(register), 0o600); err != nil {
			t.Fatalf("failed to write register: %v", err)
		}
	}
	return dir
}

// writeStamp puts a reconcile stamp into a state directory.
func writeStamp(t *testing.T, stateDir, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(stateDir, "maintenance"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "maintenance", "last-run.txt"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func manifestOf(scope string) string {
	return "[area]\nscope = \"" + scope + "\"\n"
}

func TestCollectionName(t *testing.T) {
	cases := []struct {
		scope    string
		expected string
	}{
		{"knowledge", "knowledge"},
		{"project/ultra-brain", "project-ultra-brain"},
		{"project/deep/nested", "project-deep-nested"},
		{"-unsafe@scope-", "unsafe-scope"},
		{"foo--bar", "foo--bar"},
		{"a/b/c", "a-b-c"},
	}

	for _, tc := range cases {
		t.Run(tc.scope, func(t *testing.T) {
			got := search.CollectionName(tc.scope)
			if got != tc.expected {
				t.Errorf("CollectionName(%q) = %q; want %q", tc.scope, got, tc.expected)
			}
		})
	}
}

func TestExecuteSearch_RegistryNotFound(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "nonexistent")
	port := &mockSearchPort{}
	_, err := search.ExecuteSearch("query", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, nonexistent, searchNow)
	if err == nil {
		t.Fatal("expected error for nonexistent registry, got nil")
	}
	if port.calls != 0 {
		t.Errorf("the engine was asked without a registry: %d calls", port.calls)
	}
}

func TestExecuteSearch_EmptyRegistry(t *testing.T) {
	dir := t.TempDir()
	writeRegistry(t, dir, "# empty\n")
	writeStamp(t, dir, "2000-01-01T00:00:00+00:00\n")

	port := &mockSearchPort{}
	answer, err := search.ExecuteSearch("query", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(answer.Hits) != 0 || len(answer.Findings) != 0 {
		t.Errorf("expected empty answer for empty registry, got hits=%d findings=%v", len(answer.Hits), answer.Findings)
	}
	if port.calls != 0 {
		t.Errorf("expected port not to be called for empty registry, got %d calls", port.calls)
	}
}

// core._visible_areas reads the manifest of every registered area and lets the error out:
// an area without a declaration fails the search instead of being asked about blind.
func TestExecuteSearch_AnAreaWithoutADeclarationFailsTheSearch(t *testing.T) {
	dir := t.TempDir()
	bare := filepath.Join(dir, "bare")
	if err := os.MkdirAll(bare, 0o750); err != nil {
		t.Fatal(err)
	}
	writeRegistry(t, dir, areaEntry("bare", bare))

	port := &mockSearchPort{}
	if _, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow); err == nil {
		t.Fatal("expected an error for an area without a manifest")
	}
	if port.calls != 0 {
		t.Errorf("the engine was asked: %d calls", port.calls)
	}
}

func TestExecuteSearch_UnknownScope(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	ub := writeArea(t, dir, "project/ultra-brain", manifestOf("project/ultra-brain"))
	writeRegistry(t, dir, areaEntry("knowledge", knowledge)+areaEntry("project/ultra-brain", ub))

	port := &mockSearchPort{}
	_, err := search.ExecuteSearch("query", "project/missing", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow)
	want := "unknown scope 'project/missing'; known scopes are: knowledge, project/ultra-brain"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
	if port.calls != 0 {
		t.Errorf("expected port not to be called on unknown scope error, got %d calls", port.calls)
	}
}

func TestExecuteSearch_AllScope(t *testing.T) {
	dir := t.TempDir()
	ub := writeArea(t, dir, "project/ultra-brain", manifestOf("project/ultra-brain"), "docs/intro.md")
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"), "note.md")
	writeRegistry(t, dir, areaEntry("project/ultra-brain", ub)+areaEntry("knowledge", knowledge))

	var requestedCols []string
	port := &mockSearchPort{
		searchFunc: func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
			requestedCols = collections
			return []search.SearchHit{
				{Collection: "project-ultra-brain", Relative: "docs/intro.md", Line: 1, Title: "Introduction", Snippet: "First line\nSecond line", Score: 0.85},
				{Collection: "knowledge", Relative: "note.md", Line: 5, Title: "A Note", Snippet: "Note content", Score: 0.72},
			}, nil
		},
	}

	answer, err := search.ExecuteSearch("arch", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(requestedCols, []string{"knowledge", "project-ultra-brain"}) {
		t.Errorf("expected sorted collections [knowledge project-ultra-brain], got %v", requestedCols)
	}
	if len(answer.Hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(answer.Hits))
	}
	if answer.Hits[0].Scope != "project/ultra-brain" || answer.Hits[1].Scope != "knowledge" {
		t.Errorf("scopes %q and %q", answer.Hits[0].Scope, answer.Hits[1].Scope)
	}
	if len(answer.Findings) != 0 {
		t.Errorf("expected no findings, got %v", answer.Findings)
	}
}

// A named scope asks, and reads the register of, that area alone: the broken register of
// the other one does not fail the search.
func TestExecuteSearch_SpecificScope(t *testing.T) {
	dir := t.TempDir()
	ub := writeArea(t, dir, "project/ultra-brain", manifestOf("project/ultra-brain"), "docs/intro.md")
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	if err := os.WriteFile(filepath.Join(knowledge, "_identities.tsv"), []byte("header\nbroken\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeRegistry(t, dir, areaEntry("project/ultra-brain", ub)+areaEntry("knowledge", knowledge))

	var requestedCols []string
	port := &mockSearchPort{
		searchFunc: func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
			requestedCols = collections
			return []search.SearchHit{{Collection: "project-ultra-brain", Relative: "docs/intro.md", Line: 1, Title: "Introduction", Snippet: "Hello", Score: 0.9}}, nil
		},
	}

	answer, err := search.ExecuteSearch("hello", "project/ultra-brain", search.ProfileFull, 3, privacy.ChannelLocal, port, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(requestedCols, []string{"project-ultra-brain"}) {
		t.Errorf("expected [project-ultra-brain], got %v", requestedCols)
	}
	if len(answer.Hits) != 1 || answer.Hits[0].Scope != "project/ultra-brain" || len(answer.Findings) != 0 {
		t.Errorf("expected 1 hit with scope project/ultra-brain and no findings, got %+v %v", answer.Hits, answer.Findings)
	}
}

func TestExecuteSearch_PortError(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))

	expectedErr := errors.New("backend timeout")
	port := &mockSearchPort{
		searchFunc: func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
			return nil, expectedErr
		},
	}

	_, err := search.ExecuteSearch("query", "all", search.ProfileKeyword, 5, privacy.ChannelLocal, port, dir, searchNow)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if port.calls != 1 {
		t.Errorf("a failed search is not retried, got %d calls", port.calls)
	}
}

func TestExecuteSearch_EmptyRetrySuccess(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"), "retry.md")
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))

	var port *mockSearchPort
	port = &mockSearchPort{
		searchFunc: func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
			if port.calls == 1 {
				return nil, nil // first call empty
			}
			return []search.SearchHit{{Collection: "knowledge", Relative: "retry.md", Line: 1, Title: "Found on retry", Score: 0.5}}, nil
		},
	}

	answer, err := search.ExecuteSearch("retry query", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port.calls != 2 {
		t.Errorf("expected 2 port calls, got %d", port.calls)
	}
	if len(answer.Hits) != 1 {
		t.Fatalf("expected 1 hit from second attempt, got %d", len(answer.Hits))
	}
	if len(answer.Findings) != 0 {
		t.Errorf("expected 0 findings when retry succeeds, got %v", answer.Findings)
	}
}

func TestExecuteSearch_EmptyRetryError(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))

	retryErr := errors.New("backend failed on retry")
	var port *mockSearchPort
	port = &mockSearchPort{
		searchFunc: func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
			if port.calls == 1 {
				return nil, nil // first call empty
			}
			return nil, retryErr
		},
	}

	_, err := search.ExecuteSearch("retry err", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow)
	if !errors.Is(err, retryErr) {
		t.Fatalf("expected error %v, got %v", retryErr, err)
	}
}

func TestExecuteSearch_EmptyRetryTwiceGivesFinding(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))

	port := &mockSearchPort{}
	answer, err := search.ExecuteSearch("nothing", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port.calls != 2 {
		t.Errorf("expected 2 port calls on double empty, got %d", port.calls)
	}
	if len(answer.Hits) != 0 {
		t.Errorf("expected 0 hits, got %d", len(answer.Hits))
	}
	if !reflect.DeepEqual(answer.Findings, []string{twiceEmptyFast}) {
		t.Fatalf("got %q", answer.Findings)
	}
}

func TestExecuteSearch_WithheldHits(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"), "public.md")
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))

	port := &mockSearchPort{
		searchFunc: func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
			return []search.SearchHit{
				{Collection: "unregistered-collection", Relative: "secret.md", Line: 1, Title: "Secret", Score: 0.9},
				{Collection: "knowledge", Relative: "public.md", Line: 1, Title: "Public", Score: 0.8},
			}, nil
		},
	}

	answer, err := search.ExecuteSearch("mix", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(answer.Hits) != 1 || answer.Hits[0].Relative != "public.md" {
		t.Fatalf("expected only public hit, got %v", answer.Hits)
	}
	if !reflect.DeepEqual(answer.Findings, []string{withheldOne}) {
		t.Fatalf("got %q", answer.Findings)
	}
}

// Measured against core._assemble with n = 2: every kept hit is checked against its register
// -- the third one too, which the cut to n removes afterwards -- and both drops are counted.
func TestExecuteSearch_RegisterFindingsCoverEveryKeptHitBeforeTheCut(t *testing.T) {
	dir := t.TempDir()
	x := writeArea(t, dir, "project/x", "[area]\nscope = \"project/x\"\n\n[privacy]\nnever = [\"secret/**\"]\n")
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	writeRegistry(t, dir, areaEntry("project/x", x)+areaEntry("knowledge", knowledge))

	port := &mockSearchPort{searchFunc: func(string, []string, search.Profile, int) ([]search.SearchHit, error) {
		return []search.SearchHit{
			{Collection: "project-x", Relative: "secret/a.md"},
			{Collection: "stranger", Relative: "b.md"},
			{Collection: "project-x", Relative: "notes/open.md"},
			{Collection: "knowledge", Relative: "n1.md"},
			{Collection: "knowledge", Relative: "n2.md"},
		}, nil
	}}

	answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 2, privacy.ChannelLocal, port, dir, searchNow)
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, hit := range answer.Hits {
		kept = append(kept, hit.Scope+"/"+hit.Relative)
	}
	if !reflect.DeepEqual(kept, []string{"project/x/notes/open.md", "knowledge/n1.md"}) {
		t.Errorf("kept %q", kept)
	}
	want := []string{
		"project/x/notes/open.md: hit is not in the register; reindex to catch up",
		"knowledge/n1.md: hit is not in the register; reindex to catch up",
		"knowledge/n2.md: hit is not in the register; reindex to catch up",
		withheldTwo,
	}
	if !reflect.DeepEqual(answer.Findings, want) {
		t.Errorf("findings %q", answer.Findings)
	}
}

func TestExecuteSearch_LimitsHitsToN(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"), "file.md")
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))

	port := &mockSearchPort{
		searchFunc: func(query string, collections []string, profile search.Profile, n int) ([]search.SearchHit, error) {
			var hits []search.SearchHit
			for i := 1; i <= 5; i++ {
				hits = append(hits, search.SearchHit{Collection: "knowledge", Relative: "file.md", Line: i, Title: "Title", Score: 0.5})
			}
			return hits, nil
		},
	}

	answer, err := search.ExecuteSearch("limit", "all", search.ProfileFast, 2, privacy.ChannelLocal, port, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(answer.Hits) != 2 || answer.Hits[1].Line != 2 {
		t.Errorf("expected the first 2 hits, got %+v", answer.Hits)
	}
}

// Python reads every register before it looks at a hit: a broken one fails the search even
// when no hit comes from its area.
func TestExecuteSearch_ABrokenRegisterFailsEvenWithoutItsHits(t *testing.T) {
	dir := t.TempDir()
	a := writeArea(t, dir, "a", manifestOf("a"), "x.md")
	b := writeArea(t, dir, "b", manifestOf("b"))
	if err := os.WriteFile(filepath.Join(b, "_identities.tsv"), []byte("doc_id\tpfad\tcontent_hash\trevision\nid\tb.md\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeRegistry(t, dir, areaEntry("a", a)+areaEntry("b", b))

	port := &mockSearchPort{searchFunc: func(string, []string, search.Profile, int) ([]search.SearchHit, error) {
		return []search.SearchHit{{Collection: "a", Relative: "x.md"}}, nil
	}}
	_, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow)
	want := filepath.Join(b, "_identities.tsv") + ": line 2: expected 4 tab-separated fields, found 2"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
}

func TestExecuteSearch_FindingsComeInPythonsOrder(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))
	writeStamp(t, dir, "2000-01-01T00:00:00+00:00\n")

	t.Run("twice empty, then the stamp", func(t *testing.T) {
		answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, &mockSearchPort{}, dir, searchNow)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(answer.Findings, []string{twiceEmptyFast, agedStamp}) {
			t.Fatalf("got %q", answer.Findings)
		}
	})
	t.Run("register, withheld, then the stamp", func(t *testing.T) {
		port := &mockSearchPort{searchFunc: func(string, []string, search.Profile, int) ([]search.SearchHit, error) {
			return []search.SearchHit{{Collection: "stranger", Relative: "b.md"}, {Collection: "knowledge", Relative: "n1.md"}}, nil
		}}
		answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"knowledge/n1.md: hit is not in the register; reindex to catch up", withheldOne, agedStamp}
		if !reflect.DeepEqual(answer.Findings, want) {
			t.Fatalf("got %q", answer.Findings)
		}
	})
}

// core.search returns before the stamp is read when no area is visible.
func TestExecuteSearch_NoVisibleAreaCarriesNoStaleFinding(t *testing.T) {
	dir := t.TempDir()
	closed := writeArea(t, dir, "project/secret", "[area]\nscope = \"project/secret\"\n\n[privacy]\nmode = \"local_only\"\n")
	writeRegistry(t, dir, areaEntry("project/secret", closed))
	writeStamp(t, dir, "2000-01-01T00:00:00+00:00\n")

	answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelCloud, &mockSearchPort{}, dir, searchNow)
	if err != nil {
		t.Fatal(err)
	}
	if answer.Findings != nil || answer.Hits != nil {
		t.Fatalf("got %+v", answer)
	}
}

func TestExecuteSearch_AStampThatCannotBeReadFailsTheSearch(t *testing.T) {
	dir := t.TempDir()
	knowledge := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	writeRegistry(t, dir, areaEntry("knowledge", knowledge))
	if err := os.MkdirAll(filepath.Join(dir, "maintenance", "last-run.txt"), 0o750); err != nil {
		t.Fatal(err)
	}
	port := &mockSearchPort{}
	if _, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow); err == nil {
		t.Fatal("expected the unreadable stamp to fail the search")
	}
	if port.calls != 2 {
		t.Errorf("the stamp is read after the engine answered, got %d calls", port.calls)
	}
}

func TestFormatSearch(t *testing.T) {
	t.Run("empty hits", func(t *testing.T) {
		ans := &search.SearchAnswer{Hits: nil}
		out := search.FormatSearch(ans)
		if out != search.NoMatches+"\n" {
			t.Errorf("expected %q, got %q", search.NoMatches+"\n", out)
		}
	})

	t.Run("formatted results with snippet indentation", func(t *testing.T) {
		ans := &search.SearchAnswer{
			Hits: []search.SearchHit{
				{Scope: "project/ultra-brain", Relative: "docs/intro.md", Line: 10, Title: "Introduction", Score: 0.854, Snippet: "@@ -1,3 @@ (header)\n# Introduction\n\nSome detail"},
				{Collection: "knowledge", Scope: "knowledge", Relative: "note.md", Line: 1, Title: "A Note", Score: 0.5, Snippet: "Single line snippet"},
			},
		}

		out := search.FormatSearch(ans)
		expected := "brain://project/ultra-brain/docs/intro.md:10  85%  Introduction\n" +
			"    @@ -1,3 @@ (header)\n" +
			"    # Introduction\n" +
			"    \n" +
			"    Some detail\n\n" +
			"brain://knowledge/note.md:1  50%  A Note\n" +
			"    Single line snippet\n\n"

		if out != expected {
			t.Errorf("FormatSearch mismatch.\nGot:\n%s\nWant:\n%s", out, expected)
		}
	})

	// Measured against cli._print_search on 2026-09-15.
	head := "brain://project/ultra-brain/docs/intro.md:10  85%  Introduction\n"
	for _, tc := range []struct {
		name, snippet, want string
	}{
		{"an empty snippet prints no line", "", head + "\n"},
		{"a trailing newline prints no extra line", "a\n", head + "    a\n\n"},
		{"every separator of str.splitlines", "a\r\nb\rc\vd\fe\x1cf\x1dg\x1eh\xc2\x85i\xe2\x80\xa8j\xe2\x80\xa9k",
			head + "    a\n    b\n    c\n    d\n    e\n    f\n    g\n    h\n    i\n    j\n    k\n\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hit := search.SearchHit{Scope: "project/ultra-brain", Relative: "docs/intro.md", Line: 10, Title: "Introduction", Score: 0.854, Snippet: tc.snippet}
			if got := search.FormatSearch(&search.SearchAnswer{Hits: []search.SearchHit{hit}}); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("scores round like format(score, '.0%')", func(t *testing.T) {
		for score, want := range map[float64]string{0.005: "0%", 0.015: "2%", 0.125: "12%", 0.145: "14%", 0.995: "100%", 1.0: "100%", 0.0: "0%"} {
			hit := search.SearchHit{Scope: "s", Relative: "r.md", Line: 1, Title: "T", Score: score}
			if got := search.FormatSearch(&search.SearchAnswer{Hits: []search.SearchHit{hit}}); got != "brain://s/r.md:1  "+want+"  T\n\n" {
				t.Errorf("score %v: got %q", score, got)
			}
		}
	})

	t.Run("an empty scope stays empty", func(t *testing.T) {
		hit := search.SearchHit{Collection: "knowledge", Relative: "note.md", Line: 1, Title: "A Note", Score: 0.5}
		if got := search.FormatSearch(&search.SearchAnswer{Hits: []search.SearchHit{hit}}); got != "brain:///note.md:1  50%  A Note\n\n" {
			t.Fatalf("got %q", got)
		}
	})
}

// The promise slice 2a exists for: on the cloud channel a `local_only` area
// is not filtered out of the answer, it is never asked about. Filtering
// afterwards would mean the engine was handed the question, and the question
// is already the disclosure (arch spec 7.2.1).
func TestExecuteSearch_LocalOnlyAreaIsNotAskedOnCloud(t *testing.T) {
	dir := t.TempDir()
	open := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	closed := writeArea(t, dir, "project/secret",
		"[area]\nscope = \"project/secret\"\n\n[privacy]\nmode = \"local_only\"\n")
	writeRegistry(t, dir, areaEntry("knowledge", open)+areaEntry("project/secret", closed))

	var requested []string
	port := &mockSearchPort{
		searchFunc: func(_ string, collections []string, _ search.Profile, _ int) ([]search.SearchHit, error) {
			requested = collections
			return nil, nil
		},
	}

	_, err := search.ExecuteSearch(
		"q", "all", search.ProfileFast, 5, privacy.ChannelCloud, port, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, col := range requested {
		if col == "project-secret" {
			t.Fatalf("the engine was asked about a local_only collection on the cloud channel: %v", requested)
		}
	}
	if len(requested) != 1 || requested[0] != "knowledge" {
		t.Errorf("expected only [knowledge], got %v", requested)
	}
}

// The counterpart: the same area is ordinary on the local channel. Without
// this, a filter that simply dropped everything would pass the test above.
func TestExecuteSearch_LocalOnlyAreaIsAskedOnLocal(t *testing.T) {
	dir := t.TempDir()
	open := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	closed := writeArea(t, dir, "project/secret",
		"[area]\nscope = \"project/secret\"\n\n[privacy]\nmode = \"local_only\"\n")
	writeRegistry(t, dir, areaEntry("knowledge", open)+areaEntry("project/secret", closed))

	var requested []string
	port := &mockSearchPort{
		searchFunc: func(_ string, collections []string, _ search.Profile, _ int) ([]search.SearchHit, error) {
			requested = collections
			return nil, nil
		},
	}

	if _, err := search.ExecuteSearch(
		"q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(requested) != 2 {
		t.Errorf("expected both collections on the local channel, got %v", requested)
	}
}

// Naming the invisible area by scope must not be a way around the channel,
// and the refusal must be the *same* refusal an area that does not exist
// gets. Not because the name is a secret -- the caller just typed it -- but
// because two different messages would let a caller tell "no such area" from
// "there, but not for you", which is exactly what local_only hides. Python
// makes them one error for this reason (core.py ScopeError).
func TestExecuteSearch_InvisibleScopeIsRefusedLikeAnAbsentOne(t *testing.T) {
	dir := t.TempDir()
	open := writeArea(t, dir, "knowledge", manifestOf("knowledge"))
	closed := writeArea(t, dir, "project/secret",
		"[area]\nscope = \"project/secret\"\n\n[privacy]\nmode = \"local_only\"\n")
	writeRegistry(t, dir, areaEntry("knowledge", open)+areaEntry("project/secret", closed))

	invisible := &mockSearchPort{}
	_, errInvisible := search.ExecuteSearch(
		"q", "project/secret", search.ProfileFast, 5, privacy.ChannelCloud, invisible, dir, searchNow)
	absent := &mockSearchPort{}
	_, errAbsent := search.ExecuteSearch(
		"q", "project/does-not-exist", search.ProfileFast, 5, privacy.ChannelCloud, absent, dir, searchNow)

	if errInvisible == nil || errAbsent == nil {
		t.Fatalf("expected both to be refused; invisible=%v absent=%v", errInvisible, errAbsent)
	}
	// Same shape, differing only in the scope the caller themselves named.
	shape := func(err error, scope string) string {
		return strings.Replace(err.Error(), "'"+scope+"'", "<scope>", 1)
	}
	if shape(errInvisible, "project/secret") != shape(errAbsent, "project/does-not-exist") {
		t.Errorf("the two refusals differ, which tells them apart: invisible=%v absent=%v",
			errInvisible, errAbsent)
	}
	// And neither may list it among the scopes that do exist here.
	if strings.Contains(strings.SplitN(errInvisible.Error(), "known scopes are:", 2)[1], "secret") {
		t.Errorf("the invisible area is listed as known: %v", errInvisible)
	}
	if invisible.calls != 0 || absent.calls != 0 {
		t.Errorf("the engine was asked despite the refusal: %d and %d calls", invisible.calls, absent.calls)
	}
}

// With every area invisible the engine must not be asked at all: an empty
// collection list drops qmd's collection filter, so it would search
// everything -- local_only included.
func TestExecuteSearch_NoVisibleAreaAsksNothing(t *testing.T) {
	dir := t.TempDir()
	closed := writeArea(t, dir, "project/secret",
		"[area]\nscope = \"project/secret\"\n\n[privacy]\nmode = \"local_only\"\n")
	writeRegistry(t, dir, areaEntry("project/secret", closed))

	port := &mockSearchPort{}
	answer, err := search.ExecuteSearch(
		"q", "all", search.ProfileFast, 5, privacy.ChannelCloud, port, dir, searchNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port.calls != 0 {
		t.Errorf("the engine was asked with no visible area: %d calls", port.calls)
	}
	if len(answer.Hits) != 0 {
		t.Errorf("expected no hits, got %d", len(answer.Hits))
	}
}

// `[privacy] never` names a path no channel may reach. Python's _assemble
// drops such a hit before it ever looks the path up; the withholding line
// claims "excluded by [privacy] never", so the filter has to be there.
func TestExecuteSearch_NeverPathIsDroppedOnItsOwnChannel(t *testing.T) {
	dir := t.TempDir()
	area := writeArea(t, dir, "knowledge",
		"[area]\nscope = \"knowledge\"\n\n[privacy]\nnever = [\"secret/**\"]\n", "notes/open.md")
	writeRegistry(t, dir, areaEntry("knowledge", area))

	port := &mockSearchPort{searchFunc: func(_ string, _ []string, _ search.Profile, _ int) ([]search.SearchHit, error) {
		return []search.SearchHit{
			{Collection: "knowledge", Relative: "secret/passwords.md", Title: "Vault", Snippet: "hunter2"},
			{Collection: "knowledge", Relative: "notes/open.md", Title: "Open", Snippet: "public"},
		}, nil
	}}

	answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow)
	if err != nil {
		t.Fatalf("ExecuteSearch returned an error: %v", err)
	}
	if len(answer.Hits) != 1 || answer.Hits[0].Relative != "notes/open.md" {
		t.Fatalf("expected only the readable hit, got %+v", answer.Hits)
	}

	// The withheld path must not surface anywhere -- not in a hit, not in a
	// finding. A hit dropped by `never` is precisely the one whose name may
	// not leave the machine.
	rendered := search.FormatSearch(answer) + strings.Join(answer.Findings, "\n")
	for _, forbidden := range []string{"secret/passwords.md", "hunter2", "Vault"} {
		if strings.Contains(rendered, forbidden) {
			t.Errorf("the withheld hit leaked through the output: %q in %q", forbidden, rendered)
		}
	}
	if !reflect.DeepEqual(answer.Findings, []string{withheldOne}) {
		t.Errorf("expected exactly one withholding finding counting 1, got %v", answer.Findings)
	}
}

// The count is one line for both reasons together, so two wordings cannot be
// read back to tell which filter caught a hit.
func TestExecuteSearch_NeverAndUnknownCollectionShareOneCount(t *testing.T) {
	dir := t.TempDir()
	area := writeArea(t, dir, "knowledge",
		"[area]\nscope = \"knowledge\"\n\n[privacy]\nnever = [\"secret/**\"]\n", "open.md")
	writeRegistry(t, dir, areaEntry("knowledge", area))

	port := &mockSearchPort{searchFunc: func(_ string, _ []string, _ search.Profile, _ int) ([]search.SearchHit, error) {
		return []search.SearchHit{
			{Collection: "knowledge", Relative: "secret/a.md"},
			{Collection: "stranger", Relative: "b.md"},
			{Collection: "knowledge", Relative: "open.md"},
		}, nil
	}}

	answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow)
	if err != nil {
		t.Fatalf("ExecuteSearch returned an error: %v", err)
	}
	if len(answer.Hits) != 1 {
		t.Fatalf("expected one surviving hit, got %+v", answer.Hits)
	}
	if !reflect.DeepEqual(answer.Findings, []string{withheldTwo}) {
		t.Errorf("expected a single finding counting 2, got %v", answer.Findings)
	}
}

// An area without a `never` list keeps every hit: the filter must not turn a
// missing declaration into a blanket refusal.
func TestExecuteSearch_WithoutNeverEveryHitSurvives(t *testing.T) {
	dir := t.TempDir()
	area := writeArea(t, dir, "knowledge", manifestOf("knowledge"), "secret/a.md", "open.md")
	writeRegistry(t, dir, areaEntry("knowledge", area))

	port := &mockSearchPort{searchFunc: func(_ string, _ []string, _ search.Profile, _ int) ([]search.SearchHit, error) {
		return []search.SearchHit{
			{Collection: "knowledge", Relative: "secret/a.md"},
			{Collection: "knowledge", Relative: "open.md"},
		}, nil
	}}

	answer, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, privacy.ChannelLocal, port, dir, searchNow)
	if err != nil {
		t.Fatalf("ExecuteSearch returned an error: %v", err)
	}
	if len(answer.Hits) != 2 {
		t.Fatalf("expected both hits, got %+v", answer.Hits)
	}
	if len(answer.Findings) != 0 {
		t.Errorf("expected no withholding finding, got %v", answer.Findings)
	}
}
