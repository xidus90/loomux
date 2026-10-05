package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/dev/fakeqmd"
)

// wantCases3a is pinned, not merely non-zero: a partial import must not pass
// as parity. Raise it with the corpus when a case is added.
const wantCases3a = 28

// expectation3a is how the replay of one case may differ from its recording.
//
// Both lists are exact: the runner has to report precisely these mismatches,
// no more and no fewer, so a listed case that starts to pass fails as loudly
// as one that grows a new difference. A case that is not listed must pass.
type expectation3a struct {
	// why names the rows of docs/.superpowers/parity/stufe-3a.md that the
	// differences belong to.
	why string
	// differ are mismatch lines as the runner prints them.
	differ []string
	// formatOnly are files whose bytes differ while their decoded value does
	// not: the replay decodes both sides -- JSON, or YAML for `.yml` -- and
	// holds them equal, so a listed file cannot hide a change of content.
	formatOnly []string
}

// The two machine-read files beside graph.json that every index run writes,
// each in another format than the reference writes it: the rows on qmd's
// `index.yml` and on `qmd-collections.json` in the parity list.
var indexFormats = []string{"qmd-collections.json", "xdg/qmd/index.yml"}

// indexedFormats is indexFormats and the link graphs the run wrote, whose key
// order is a row of its own.
func indexedFormats(graphs ...string) []string {
	return append(append([]string{}, indexFormats...), graphs...)
}

// The index run `area add` appends, as the recording of `brain init` has none.
var areaAddIndexRun = []string{
	"unexpected extra file in actual: qmd-calls.log",
	"unexpected extra file in actual: qmd-collections.json",
	"unexpected extra file in actual: repo-new/_identities.tsv",
	"unexpected extra file in actual: repo-new/docs/index.md",
	"unexpected extra file in actual: repo-new/graph.json",
	"unexpected extra file in actual: repo-new/index.md",
	"unexpected extra file in actual: xdg/qmd/index.yml",
}

// The host half `brain init` writes and `area add` leaves to `loomux init`.
const noMCPJSON = "missing file in actual: repo-new/.mcp.json"

// The lock file loomux keeps where the reference removes its own.
const registryLock = "unexpected extra file in actual: registry.lock"

// areaLock is the lock reindex and approve share per area, which the
// reference does not take (Heilung #4).
func areaLock(scope string) string {
	return "unexpected extra file in actual: areas/" + scope + ".lock"
}

var expected3a = map[string]expectation3a{
	"area-add/new-area": {
		why:    "Wirtsteil; Indexlauf; Sperre; Sperre je Bereich",
		differ: append([]string{noMCPJSON, registryLock, areaLock("project-repo-new")}, areaAddIndexRun...),
	},
	"area-add/without-yes": {
		why:    "Wirtsteil; Indexlauf; Sperre; Sperre je Bereich",
		differ: append([]string{noMCPJSON, registryLock, areaLock("project-repo-new")}, areaAddIndexRun...),
	},
	"area-add/no-reindex": {
		why:    "Wirtsteil; Sperre",
		differ: []string{noMCPJSON, registryLock},
	},
	"area-add/known-scope": {
		why: "Reihenfolge und doppelter Scope: refused before the repository is written; Sperre",
		differ: []string{
			"exit code: expected 0, got 1",
			"missing file in actual: repo-new/.mcp.json",
			"missing file in actual: repo-new/.ultra-brain/config.toml",
			"missing file in actual: repo-new/AGENTS.md",
			"missing file in actual: repo-new/docs/wiki/_identities.tsv",
			"missing file in actual: repo-new/docs/wiki/_schema.md",
			"missing file in actual: repo-new/docs/wiki/audit.md",
			"missing file in actual: repo-new/docs/wiki/index.md",
			"missing file in actual: repo-new/docs/wiki/log.md",
			registryLock,
		},
	},
	"area-add/no-registry": {
		why: "area add ohne vorher angelegte registry.toml: the reference fails, loomux creates it; Sperre; Sperre je Bereich",
		differ: []string{
			"exit code: expected 1, got 0",
			areaLock("project-repo-new"),
			"unexpected extra file in actual: qmd-calls.log",
			"unexpected extra file in actual: qmd-collections.json",
			"unexpected extra file in actual: registry.lock",
			"unexpected extra file in actual: registry.toml",
			"unexpected extra file in actual: repo-new/.loomux/config.toml",
			"unexpected extra file in actual: repo-new/AGENTS.md",
			"unexpected extra file in actual: repo-new/_identities.tsv",
			"unexpected extra file in actual: repo-new/docs/index.md",
			"unexpected extra file in actual: repo-new/docs/wiki/_identities.tsv",
			"unexpected extra file in actual: repo-new/docs/wiki/_schema.md",
			"unexpected extra file in actual: repo-new/docs/wiki/audit.md",
			"unexpected extra file in actual: repo-new/docs/wiki/index.md",
			"unexpected extra file in actual: repo-new/docs/wiki/log.md",
			"unexpected extra file in actual: repo-new/graph.json",
			"unexpected extra file in actual: repo-new/index.md",
			"unexpected extra file in actual: xdg/qmd/index.yml",
		},
	},
	"reconcile/area-without-include": {
		why: "leeres [index] include: the reference walks no file and counts none, loomux walks **/*.md",
		differ: []string{
			"content mismatch: maintenance/notes/stats.tsv",
			// `2 Quellen geprüft, 2 davon gehasht` where the reference counts one.
			"stdout mismatch: expected 44 bytes, got 45 bytes",
		},
	},
	"reindex/area-without-include": {
		why: "leeres [index] include; graph.json, index.yml, qmd-collections.json; Sperre je Bereich",
		differ: []string{
			areaLock("notes"),
			areaLock("project-a"),
			"content mismatch: areas/notes/_identities.tsv",
			"content mismatch: areas/notes/graph.json",
			"content mismatch: areas/notes/index.md",
			"content mismatch: maintenance/notes/stats.tsv",
		},
		formatOnly: indexedFormats("repo-a/graph.json"),
	},
	"reindex/nothing-open": {
		why:        "format of graph.json, qmd's index.yml and qmd-collections.json; Sperre je Bereich",
		differ:     []string{areaLock("notes"), areaLock("project-a")},
		formatOnly: indexedFormats("repo-a/graph.json", "areas/notes/graph.json"),
	},
	"reindex/cases-opened": {
		why:        "format of graph.json, index.yml, qmd-collections.json; Sperre je Bereich",
		differ:     []string{areaLock("project-a")},
		formatOnly: indexedFormats("repo-a/graph.json"),
	},
	"reindex/no-review-centre": {
		why:        "format of graph.json, index.yml, qmd-collections.json; Sperre je Bereich",
		differ:     []string{areaLock("project-a")},
		formatOnly: indexedFormats("repo-a/graph.json"),
	},
	// The case that holds S1 and S2 of the self-use: a review centre named
	// `95 Prüfzentrum` stays out, and `docs/**/*.md` and `docs/**/draft.md`
	// match a file directly below docs. Everything but the format and the
	// area locks agrees.
	"reindex/globs": {
		why:        "format of graph.json, index.yml, qmd-collections.json; Sperre je Bereich",
		differ:     []string{areaLock("project-a"), areaLock("project-b")},
		formatOnly: indexedFormats("repo-a/graph.json", "repo-b/graph.json"),
	},
	// project/b has no manifest and is skipped, but only after its lock was
	// taken: the lock file stays behind for it too.
	"reindex/missing-manifest": {
		why:        "format of graph.json, index.yml, qmd-collections.json; Sperre je Bereich",
		differ:     []string{areaLock("project-a"), areaLock("project-b")},
		formatOnly: indexedFormats("repo-a/graph.json"),
	},
}

// TestCases3a replays the recordings of the reference's reconcile, reindex, embed
// and init against loomux, over the normalization of NormalizeState.
func TestCases3a(t *testing.T) {
	// Absolute, because every run changes into its staged world and the
	// world_after the runner compares against is read after that.
	corpus, err := filepath.Abs(filepath.Join("..", "..", "testdata", "cases", "3a"))
	if err != nil {
		t.Fatal(err)
	}
	all, err := cases.DiscoverCases(corpus, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != wantCases3a {
		t.Fatalf("found %d cases, want %d", len(all), wantCases3a)
	}
	for _, c := range all {
		t.Run(c.Verb+"/"+c.Name, func(t *testing.T) {
			want := expected3a[c.Verb+"/"+c.Name]
			written := map[string][]byte{}
			outcome, err := cases.RunCaseWith(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Chdir(dir)
				t.Setenv("LOOMUX_STATE_DIR", dir)
				t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
				// The git environment the recorder ran the reference under.
				for _, entry := range cases.GitEnv(dir) {
					key, value, _ := strings.Cut(entry, "=")
					t.Setenv(key, value)
				}
				useRecordedEngine(t, dir)
				code := Run(args, stdin, stdout, stderr)
				for _, name := range want.formatOnly {
					data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
					if err != nil {
						t.Fatal(err)
					}
					written[name] = cases.Normalize(data, dir)
				}
				return code
			}, cases.NormalizeState)
			if err != nil {
				t.Fatal(err)
			}
			got := slices.DeleteFunc(slices.Clone(outcome.Mismatches), func(m string) bool {
				return strings.HasPrefix(m, "stderr:")
			})
			wanted := append([]string{}, want.differ...)
			for _, name := range want.formatOnly {
				wanted = append(wanted, "content mismatch: "+name)
			}
			slices.Sort(got)
			slices.Sort(wanted)
			if !slices.Equal(got, wanted) {
				why := want.why
				if why == "" {
					why = "none expected"
				}
				t.Fatalf("mismatches differ from the expected ones (%s)\ngot:\n%s\nwant:\n%s\nstdout:\n%s\n%s",
					why, strings.Join(got, "\n"), strings.Join(wanted, "\n"),
					outcome.ActualStdout, strings.Join(outcome.Mismatches, "\n"))
			}
			for _, name := range want.formatOnly {
				recorded, err := os.ReadFile(filepath.Join(c.Path, "world_after", filepath.FromSlash(name)))
				if err != nil {
					t.Fatal(err)
				}
				if a, b := decoded(t, name, recorded), decoded(t, name, written[name]); !reflect.DeepEqual(a, b) {
					t.Errorf("%s differs in content, not only in format:\nrecorded %v\nwritten  %v", name, a, b)
				}
			}
		})
	}
}

// decoded is a machine-read file's value, whatever its bytes look like.
func decoded(t *testing.T, name string, data []byte) any {
	t.Helper()
	// The token stands unquoted in YAML, where `{{` opens a flow mapping.
	data = bytes.ReplaceAll(data, []byte(cases.WorldToken), []byte("WORLD"))
	var value any
	var err error
	if strings.HasSuffix(name, ".yml") {
		err = yaml.Unmarshal(data, &value)
	} else {
		err = json.Unmarshal(data, &value)
	}
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}

// useRecordedEngine puts the world's fake qmd behind both seams the index
// commands reach the engine through: the PATH lookup of `embed` and the
// command line port. A world without a fixture is a machine without qmd, as
// its recording ran with no qmd on PATH.
func useRecordedEngine(t *testing.T, dir string) {
	t.Helper()
	fixture := filepath.Join(dir, fakeqmd.FixtureName)
	saved := indexLook
	t.Cleanup(func() { indexLook = saved })
	indexLook = func(string) (string, error) {
		if _, err := os.Stat(fixture); err != nil {
			return "", errors.New("qmd: executable file not found in %PATH%")
		}
		return "qmd", nil
	}
	runner := func(argv []string) ([]byte, []byte, int, error) {
		var out, errOut bytes.Buffer
		code := fakeqmd.Run(fixture, argv[1:], &out, &errOut)
		return out.Bytes(), errOut.Bytes(), code, nil
	}
	stubBrainStatusPort(t, &search.QmdPort{Executable: "qmd", Runner: runner})
}
