package verify

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func unset(string) string { return "" }

func TestGoWorkPointsUnsetDirsUnderHome(t *testing.T) {
	home := t.TempDir()
	env, pre, note := goWork(home, unset)
	if pre != "" {
		t.Fatalf("pre %q: %s", pre, note)
	}
	tmp := filepath.Join(home, ".loomux", "tmp")
	cache := filepath.Join(home, ".loomux", "go-cache")
	if !slices.Equal(env, []string{"GOTMPDIR=" + tmp, "GOCACHE=" + cache}) {
		t.Fatalf("env %q", env)
	}
	for _, dir := range []string{tmp, cache} {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Fatalf("%s: %v", dir, err)
		}
	}
}

func TestGoWorkKeepsWhatTheCallerSet(t *testing.T) {
	env, pre, _ := goWork(t.TempDir(), func(string) string { return "set" })
	if env != nil || pre != "" {
		t.Fatalf("env %q pre %q, want both left alone", env, pre)
	}
}

func TestGoWorkFillsOnlyTheUnsetOne(t *testing.T) {
	home := t.TempDir()
	getenv := func(key string) string {
		if key == "GOCACHE" {
			return "set"
		}
		return ""
	}
	env, pre, _ := goWork(home, getenv)
	want := []string{"GOTMPDIR=" + filepath.Join(home, ".loomux", "tmp")}
	if pre != "" || !slices.Equal(env, want) {
		t.Fatalf("env %q pre %q, want %q", env, pre, want)
	}
}

func TestGoWorkIsUnreadyWhenTheDirCannotBeMade(t *testing.T) {
	home := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(home, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	env, pre, note := goWork(home, unset)
	if env != nil || pre != StateUnready || !strings.Contains(note, filepath.Join(home, ".loomux", "tmp")) {
		t.Fatalf("env %q pre %q note %q", env, pre, note)
	}
}

func TestGoWorkIsUnreadyWithoutAHome(t *testing.T) {
	if env, pre, _ := goWork("", unset); env != nil || pre != StateUnready {
		t.Fatalf("env %q pre %q", env, pre)
	}
}

func TestPlanHandsGoLanesTheWorkEnv(t *testing.T) {
	e := env(t.TempDir())
	e.GoWork = func() ([]string, State, string) { return []string{"GOTMPDIR=w"}, "", "" }
	jobs, err := Plan(effFor(t, "", goOnly), Request{Kinds: []string{"lint"}}, e)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || !slices.Contains(jobs[0].Env, "GOTMPDIR=w") {
		t.Fatalf("%+v", jobs)
	}
	e.GoWork = func() ([]string, State, string) { return nil, StateUnready, "make it" }
	jobs, _ = Plan(effFor(t, "", goOnly), Request{Kinds: []string{"lint"}}, e)
	if len(jobs) != 1 || jobs[0].Pre != StateUnready || jobs[0].Note != "make it" {
		t.Fatalf("%+v", jobs)
	}
}
