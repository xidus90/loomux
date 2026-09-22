package cli

import (
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/cases"
)

// wantCases3b is pinned, not merely non-zero: a partial import must not pass
// as parity. Raise it with the corpus when a case is added.
const wantCases3b = 24

// expectation3b is how the replay of one case may differ from its recording.
// The lists are exact, as in expectation3a: a listed case that starts to pass
// fails as loudly as one that grows a new difference, and a case that is not
// listed must pass.
type expectation3b struct {
	// why names the rows of docs/.superpowers/parity/stufe-3b.md that the
	// differences belong to.
	why string
	// differ are mismatch lines as the runner prints them.
	differ []string
	// formatOnly are files whose bytes differ while their decoded value does
	// not, as in expectation3a.
	formatOnly []string
	// rehashed are registers whose rows hash a page the approval stamped with
	// its own time: the bytes cannot agree, so each side's content_hash is
	// held against the file it names on that side, and the rows are then
	// compared without it. Each one also stands in differ.
	rehashed []string
}

// The scratch index both sides leave at `<state>/maintenance/index`: a git
// index, whose entries carry the stat data of the run that staged them. What
// it staged is pinned by git.after.
const scratchIndex = "content mismatch: maintenance/index"

// The register the technical update rewrites: it hashes the page and audit.md
// the approval stamped with the run's time and reviewer.
const stampedRegister = "repo-a/_identities.tsv"

// wroteAndIndexed is the expectation of an approval that wrote the page and
// ran the technical update, with the scratch index its commit leaves or, when
// the commit was refused before the index was made, without.
func wroteAndIndexed(committed bool) expectation3b {
	why := "Register über gestempelte Seiten; format of graph.json, index.yml, qmd-collections.json"
	differ := []string{"content mismatch: " + stampedRegister}
	if committed {
		why = "Scratch-Index im Fallsatz; " + why
		differ = append(differ, scratchIndex)
	}
	return expectation3b{
		why:        why,
		differ:     differ,
		formatOnly: indexedFormats("repo-a/graph.json"),
		rehashed:   []string{stampedRegister},
	}
}

var expected3b = map[string]expectation3b{
	"approve/success": wroteAndIndexed(true),
	"approve/amend":   wroteAndIndexed(true),
	// Both sides refuse the commit before the scratch index is made.
	"approve/rebase":  wroteAndIndexed(false),
	"approve/no-repo": wroteAndIndexed(false),
	// A rejection runs no technical update and rewrites no register.
	"approve/reject": {why: "Scratch-Index im Fallsatz", differ: []string{scratchIndex}},
}

// TestCases3b replays the recordings of brain-mcp's cases, case and approve
// against loomux, over the normalization of NormalizeState: the commit an
// approval makes is held through git.after, the files it writes through
// world_after.
func TestCases3b(t *testing.T) {
	corpus, err := filepath.Abs(filepath.Join("..", "..", "testdata", "cases", "3b"))
	if err != nil {
		t.Fatal(err)
	}
	all, err := cases.DiscoverCases(corpus, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != wantCases3b {
		t.Fatalf("found %d cases, want %d", len(all), wantCases3b)
	}
	for _, c := range all {
		t.Run(c.Verb+"/"+c.Name, func(t *testing.T) {
			want := expected3b[c.Verb+"/"+c.Name]
			written := map[string][]byte{}
			rows := map[string]map[string]string{}
			outcome, err := cases.RunCaseWith(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Chdir(dir)
				t.Setenv("LOOMUX_STATE_DIR", dir)
				t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", dir)
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
				for _, name := range want.rehashed {
					rows[name] = registerRows(t, c, dir, name)
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
			for _, name := range want.rehashed {
				recorded := registerRows(t, c, filepath.Join(c.Path, "world_after"), name)
				if !maps.Equal(recorded, rows[name]) {
					t.Errorf("%s differs beyond the hashes of stamped pages:\nrecorded %v\nwritten  %v", name, recorded, rows[name])
				}
			}
		})
	}
}

// registerRows reads the register name below root, holds every row's
// content_hash against the file the row names below root, and returns the
// rows keyed by path without the hash. A doc id the case's world does not
// hold was minted by the run and stands as "minted".
func registerRows(t *testing.T, c *cases.Case, root, name string) map[string]string {
	t.Helper()
	known := map[string]bool{}
	if old, err := os.ReadFile(filepath.Join(c.Path, "world", filepath.FromSlash(name))); err == nil {
		for _, line := range strings.Split(string(old), "\n") {
			id, _, _ := strings.Cut(line, "\t")
			known[id] = true
		}
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	// An id minted twice stays spelled out, as in NormalizeState's
	// mintedOnce: the fold must not hide exactly the duplicate.
	uses := map[string]int{}
	for _, line := range lines[1:] {
		id, _, _ := strings.Cut(line, "\t")
		uses[id]++
	}
	rows := map[string]string{"": lines[0]}
	for _, line := range lines[1:] {
		fields := strings.Split(line, "\t")
		if len(fields) != 4 {
			t.Fatalf("%s: row %q has %d fields", name, line, len(fields))
		}
		id, rel, hash, revision := fields[0], fields[1], fields[2], fields[3]
		if _, seen := rows[rel]; seen {
			t.Fatalf("%s: %s stands on more than one row", name, rel)
		}
		file := filepath.Join(root, filepath.FromSlash(path.Dir(name)), filepath.FromSlash(rel))
		if actual, err := identity.ContentHash(file); err != nil || actual != hash {
			t.Errorf("%s: %s is registered as %s, the file hashes to %s (%v)", name, rel, hash, actual, err)
		}
		if !known[id] && uses[id] == 1 {
			id = "minted"
		}
		rows[rel] = id + "\t" + revision
	}
	return rows
}
