package setup

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/gitenv"
)

// world writes files under a fresh root; a key ending in "/" is a
// directory. It also points the state directory at an empty place, so no
// test reads the registry of the machine, LOCALAPPDATA at an empty place,
// so none finds the machine's binary, and makes every tool present.
func world(t *testing.T, files map[string]string) string {
	t.Helper()
	t.Setenv("LOOMUX_STATE_DIR", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	all := func(string) (string, error) { return "found", nil }
	old := lookPath
	lookPath = all
	t.Cleanup(func() { lookPath = old })
	root := filepath.Join(t.TempDir(), "demo")
	for rel, text := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if strings.HasSuffix(rel, "/") {
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

// fixture copies testdata/loomux into a fresh root, the configuration to
// the place the guard keeps agents from writing by name.
func fixture(t *testing.T) map[string]string {
	t.Helper()
	files := map[string]string{}
	base := filepath.Join("testdata", "loomux")
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(base, path)
		rel = filepath.ToSlash(rel)
		if rel == "loomux-config.toml" {
			rel = configPath
		}
		files[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// writeFile writes text to rel under root, creating its directory.
func writeFile(t *testing.T, root, rel, text string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// git answers core.hooksPath with hooksPath and names the directory hooks
// run from as a plain repository does: hooksPath, taken from dir when
// relative, or <dir>/.git/hooks when it is empty.
func git(hooksPath string) detect.Runner {
	return func(dir string, argv ...string) (string, error) {
		if slices.Contains(argv, "rev-parse") {
			hooks := filepath.Join(dir, ".git", "hooks")
			if hooksPath != "" {
				hooks = hooksPath
				if !filepath.IsAbs(filepath.FromSlash(hooks)) {
					hooks = filepath.Join(dir, hooks)
				}
			}
			return filepath.ToSlash(hooks) + "\n", nil
		}
		return hooksPath + "\n", nil
	}
}

// realGit runs git; an exit without output is an unset setting, as the
// detect.Runner contract asks.
func realGit(dir string, argv ...string) (string, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	// A test run from a git hook inherits GIT_DIR and its kin; they would
	// point these calls at the repository being committed.
	cmd.Env = gitenv.Environ()
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return "", nil
		}
	}
	return string(out), err
}

// run runs git in dir and fails the test when it fails.
func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	// A test run from a git hook inherits GIT_DIR and its kin; they would
	// point these calls at the repository being committed.
	cmd.Env = gitenv.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// reader reads files under root the way init does.
func reader(root string) func(string) ([]byte, bool, error) {
	return func(rel string) ([]byte, bool, error) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return data, err == nil, err
	}
}

// gather is Gather with an empty home, failing the test on an error.
func gather(t *testing.T, root, hooksPath string) Facts {
	t.Helper()
	f, err := Gather(root, t.TempDir(), git(hooksPath))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// plan builds the default plan of root.
func plan(t *testing.T, f Facts) Plan {
	t.Helper()
	p, err := Build(f, DefaultChoice(f, Answers{}), reader(f.Root))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func paths(p Plan) []string {
	var out []string
	for _, c := range p.Changes {
		out = append(out, c.Path)
	}
	return out
}

func actions(p Plan) []string {
	var out []string
	for _, a := range p.Actions {
		out = append(out, a.ID)
	}
	return out
}

func changeOf(p Plan, path string) (Change, bool) {
	for _, c := range p.Changes {
		if c.Path == path {
			return c, true
		}
	}
	return Change{}, false
}

func hasNote(p Plan, part string) bool {
	for _, n := range p.Notes {
		if strings.Contains(n, part) {
			return true
		}
	}
	return false
}

const goMod = "module example.com/demo\n\ngo 1.26\n"
const checkoutGoMod = "module github.com/xidus90/loomux\n\ngo 1.26.0\n"
