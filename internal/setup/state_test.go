package setup

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestMissingAnswersAreTheZeroValue(t *testing.T) {
	a, err := ReadAnswers(t.TempDir())
	if err != nil || a.Hosts != nil || a.Parts != nil {
		t.Errorf("answers = %+v, err = %v", a, err)
	}
}

func TestAnswersRoundTrip(t *testing.T) {
	root := t.TempDir()
	want := Answers{Hosts: []string{"claude", "antigravity"}, Parts: map[string]bool{"git-hooks": false, "binary": true}}
	if err := writeState(root, answersPath, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadAnswers(root)
	if err != nil || !slices.Equal(got.Hosts, want.Hosts) || !maps.Equal(got.Parts, want.Parts) {
		t.Errorf("answers = %+v, err = %v", got, err)
	}
}

func TestInstalledReadsAsTheBriefShowsIt(t *testing.T) {
	root := t.TempDir()
	i := installed{
		Version: "2.12.1",
		At:      time.Date(2026, 9, 24, 18, 0, 0, 0, time.UTC),
		Files:   []string{".claude/settings.json", ".githooks/pre-commit"},
		Actions: []string{"binary-install", "hooks-path"},
	}
	if err := writeState(root, installedPath, i); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(installedPath)))
	want := "version = \"2.12.1\"\nat = 2026-09-24T18:00:00Z\n" +
		"files = [\".claude/settings.json\", \".githooks/pre-commit\"]\n" +
		"actions = [\"binary-install\", \"hooks-path\"]\n"
	if err != nil || string(data) != want {
		t.Errorf("installed.toml = %q, err = %v\nwant %q", data, err, want)
	}
}

func TestReadAnswersReadsHostsAndParts(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, answersPath, "hosts = [\"claude\"]\n\n[parts]\ngit-hooks = false\n")
	a, err := ReadAnswers(root)
	if err != nil || !slices.Equal(a.Hosts, []string{"claude"}) || a.Parts["git-hooks"] || len(a.Parts) != 1 {
		t.Errorf("answers = %+v, err = %v", a, err)
	}
}

func TestBrokenAnswersNameTheFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, answersPath, "hosts = [")
	if _, err := ReadAnswers(root); err == nil || !strings.Contains(err.Error(), answersPath) {
		t.Errorf("err = %v", err)
	}
	root = t.TempDir()
	writeFile(t, root, answersPath+"/x", "")
	if _, err := ReadAnswers(root); err == nil || !strings.Contains(err.Error(), answersPath) {
		t.Errorf("err = %v", err)
	}
}
