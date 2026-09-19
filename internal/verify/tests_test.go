package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasTests(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	write("internal/cli/a_test.go", "package cli")
	write("node_modules/x/b_test.go", "")
	write("CMakeLists.txt", "project(x)\nenable_testing()\n")
	if !HasTests(root, []string{"*_test.go"}) {
		t.Error("two levels deep must count")
	}
	if !HasTests(root, []string{"CMakeLists.txt:enable_testing("}) {
		t.Error("content marker")
	}
	if HasTests(root, []string{"tests/", "test_*.py"}) {
		t.Error("no python tests here")
	}
	only := t.TempDir()
	os.MkdirAll(filepath.Join(only, "node_modules", "y"), 0o755)
	os.WriteFile(filepath.Join(only, "node_modules", "y", "c_test.go"), nil, 0o644)
	if HasTests(only, []string{"*_test.go"}) {
		t.Error("node_modules is skipped")
	}
}

// A virtual environment carries the tests of the packages it installed; a
// project without tests of its own must not look tested because of them.
// Every dot directory is skipped, like detection does, and so are the usual
// build and environment directories.
func TestHasTestsSkipsDotAndBuildDirectories(t *testing.T) {
	for _, dir := range []string{".venv/Lib/site-packages/pkg", ".tox/py", "venv/lib", "build/x", "target/debug", "dist/y"} {
		root := t.TempDir()
		os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir), "tests"), 0o755)
		os.WriteFile(filepath.Join(root, filepath.FromSlash(dir), "test_x.py"), nil, 0o644)
		if HasTests(root, []string{"tests/", "test_*.py"}) {
			t.Errorf("%s is skipped", dir)
		}
	}
}

func TestHasTestsPatternForms(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "app", "tests"), 0o755)
	os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[project]\n"), 0o644)
	if !HasTests(root, []string{"tests/"}) {
		t.Error("a nested tests directory counts")
	}
	if HasTests(root, []string{"pyproject.toml:[tool.pytest"}) {
		t.Error("the file alone is not the marker")
	}
	if HasTests(root, []string{"app:tests"}) {
		t.Error("a directory never carries a content marker")
	}
	if HasTests(root, []string{"[", "app"}) {
		t.Error("a bad glob and a directory name match nothing")
	}
	if HasTests(filepath.Join(root, "missing"), []string{"*"}) {
		t.Error("a root that is not there has no tests")
	}
	nested := filepath.Join(root, "vendor")
	os.MkdirAll(nested, 0o755)
	os.WriteFile(filepath.Join(nested, "x_test.go"), nil, 0o644)
	if !HasTests(nested, []string{"*_test.go"}) {
		t.Error("the root itself is never skipped")
	}
}
