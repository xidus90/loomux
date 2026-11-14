package web

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPrefsOfAMissingFileAreEmpty(t *testing.T) {
	p, err := ReadPrefs(t.TempDir())
	if err != nil || !reflect.DeepEqual(p, Prefs{}) {
		t.Fatalf("ReadPrefs = %+v, %v, want the zero value", p, err)
	}
}

func TestPrefsComeBackAsWritten(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	// A path with a quote and a backslash holds only if the writer escapes.
	want := Prefs{Projects: []string{`C:/a "b"/c`, `D:\x`}, Tiles: []string{"projects", "version"}, TilesSet: true}
	if err := WritePrefs(dir, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPrefs(dir)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadPrefs = %+v, %v, want %+v", got, err, want)
	}
}

func TestAnEmptyTileListIsNotAnAbsentOne(t *testing.T) {
	dir := t.TempDir()
	if err := WritePrefs(dir, Prefs{TilesSet: true}); err != nil {
		t.Fatal(err)
	}
	if p, err := ReadPrefs(dir); err != nil || !p.TilesSet || len(p.Tiles) != 0 {
		t.Fatalf("ReadPrefs = %+v, %v, want tiles set and empty", p, err)
	}
	if err := WritePrefs(dir, Prefs{Projects: []string{"C:/p"}}); err != nil {
		t.Fatal(err)
	}
	text, _ := os.ReadFile(prefsPath(dir))
	if strings.Contains(string(text), "dashboard") {
		t.Errorf("tiles never set, yet written: %s", text)
	}
	if p, err := ReadPrefs(dir); err != nil || p.TilesSet {
		t.Fatalf("ReadPrefs = %+v, %v, want tiles unset", p, err)
	}
}

func TestPrefsThatDoNotParseNameTheFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(prefsPath(dir), []byte("projects = ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPrefs(dir); err == nil || !strings.Contains(err.Error(), "ui.toml") {
		t.Fatalf("ReadPrefs error = %v, want one naming ui.toml", err)
	}
}

func TestPrefsThatCannotBeReadAreAnError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(prefsPath(dir), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPrefs(dir); err == nil {
		t.Fatal("a directory in place of ui.toml read as prefs")
	}
	// The same obstacle on the way out: the swap onto a directory fails.
	if err := WritePrefs(dir, Prefs{}); err == nil {
		t.Fatal("WritePrefs replaced a directory")
	}
}

func TestPrefsAreNotWrittenWhereNoStateDirectoryCanBe(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WritePrefs(filepath.Join(blocker, "state"), Prefs{}); err == nil {
		t.Fatal("WritePrefs made a directory below a file")
	}
}
