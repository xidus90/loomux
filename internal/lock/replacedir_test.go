package lock_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/lock"
)

// area writes a directory with the files an indexed area carries, so a test
// can tell a whole stock from half of one.
func area(t *testing.T, dir string, mark string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "notes"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	for name, text := range map[string]string{
		"index.md":        "# " + mark + "\n",
		"graph.json":      "{\"scope\":\"" + mark + "\"}\n",
		"notes/index.md":  "# notes " + mark + "\n",
		"_identities.tsv": "doc_id\tpfad\n",
	} {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}
	return dir
}

// stock reads a directory tree into a map of relative slash paths, so two
// stocks compare as values.
func stock(t *testing.T, dir string) map[string]string {
	t.Helper()
	got := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		got[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
	return got
}

func TestReplaceDirPutsTheStagingInTheEmptyPlace(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "areas", "scope")
	staging, err := lock.StagingDir(target)
	if err != nil {
		t.Fatalf("StagingDir: %v", err)
	}
	want := stock(t, area(t, staging, "new"))
	if err := lock.ReplaceDir(staging, target); err != nil {
		t.Fatalf("ReplaceDir: %v", err)
	}
	if got := stock(t, target); len(got) != len(want) || got["index.md"] != want["index.md"] {
		t.Fatalf("stock = %v, want %v", got, want)
	}
}

func TestReplaceDirLeavesNoSiblingBehind(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "areas")
	target := filepath.Join(parent, "scope")
	area(t, target, "old")
	staging, err := lock.StagingDir(target)
	if err != nil {
		t.Fatalf("StagingDir: %v", err)
	}
	area(t, staging, "new")
	if err := lock.ReplaceDir(staging, target); err != nil {
		t.Fatalf("ReplaceDir: %v", err)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "scope" {
		t.Fatalf("entries = %v, want only scope", entries)
	}
	if got := stock(t, target)["index.md"]; got != "# new\n" {
		t.Fatalf("index.md = %q, want the new stock", got)
	}
}

// The promise of this function is about the instant between the two renames,
// and a killed process is what reaches it. Recover is what the next run does
// with what that leaves.
func TestRecoverPutsTheOldStockBack(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "areas", "scope")
	area(t, target, "old")
	want := stock(t, target)
	aside := target + lock.AsideSuffix
	if err := os.Rename(target, aside); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if err := lock.Recover(target); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	got := stock(t, target)
	if len(got) != len(want) || got["index.md"] != want["index.md"] {
		t.Fatalf("stock = %v, want the old one whole", got)
	}
	if _, err := os.Stat(aside); !os.IsNotExist(err) {
		t.Fatalf("aside still there: %v", err)
	}
}

// A swap that got through and died before its cleanup leaves both. The target
// is the newer stock and outranks the aside, which goes.
func TestRecoverDropsAnAsideBesideATarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "areas", "scope")
	area(t, target, "new")
	area(t, target+lock.AsideSuffix, "old")
	if err := lock.Recover(target); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if got := stock(t, target)["index.md"]; got != "# new\n" {
		t.Fatalf("index.md = %q, want the new stock", got)
	}
	if _, err := os.Stat(target + lock.AsideSuffix); !os.IsNotExist(err) {
		t.Fatalf("aside still there: %v", err)
	}
}

func TestRecoverPassesWithoutAnAside(t *testing.T) {
	if err := lock.Recover(filepath.Join(t.TempDir(), "scope")); err != nil {
		t.Fatalf("Recover: %v", err)
	}
}

func TestStagingDirRefusesAParentItCannotMake(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := lock.StagingDir(filepath.Join(blocked, "areas", "scope")); err == nil {
		t.Fatal("expected an error below a plain file")
	}
}
