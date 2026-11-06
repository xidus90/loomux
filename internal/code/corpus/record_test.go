package corpus

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// buildCorpus loads the ignore state, walks the tree and reads every search
// file, the way the build does, and returns the record it would write.
func buildCorpus(t *testing.T, root string, maxFilesize int64) Record {
	t.Helper()
	rules, _, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	listing, err := Walk(root, rules)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	files, _, _ := SearchCorpus(listing, maxFilesize)
	prints := make(map[string]Print, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f.Abs)
		if err != nil {
			prints[f.Rel] = Print{Size: f.Size, MTime: f.MTime}
			continue
		}
		sum := sha256.Sum256(data)
		prints[f.Rel] = Print{Size: f.Size, MTime: f.MTime, Hash: hex.EncodeToString(sum[:]), Binary: IsBinary(data)}
	}
	return NewRecord(rules, prints, maxFilesize)
}

func TestCanonicalSettings(t *testing.T) {
	got := string(CanonicalSettings(1024, true))
	want := "max_filesize=1024;binary=nul-v1;ignore=git-v1;ignorecase=true"
	if got != want {
		t.Fatalf("CanonicalSettings = %q, want %q", got, want)
	}
}

func TestDigestDeterministic(t *testing.T) {
	rec := Record{
		Settings: CanonicalSettings(1, false),
		Sources: []Source{
			{Rel: "b/.gitignore", Hash: [32]byte{2}},
			{Rel: ".gitignore", Hash: [32]byte{1}},
		},
		Files: map[string]Print{"b.go": {Hash: "bb"}, "a.go": {Hash: "aa"}},
	}
	first := Digest(rec)
	if second := Digest(rec); first != second {
		t.Fatal("Digest is not deterministic")
	}
	shuffled := rec
	shuffled.Sources = []Source{rec.Sources[1], rec.Sources[0]}
	if Digest(shuffled) != first {
		t.Fatal("Digest depends on the source order")
	}
	if Digest(rec) == Digest(Record{Settings: rec.Settings, Files: rec.Files}) {
		t.Fatal("Digest ignores the sources")
	}
}

func TestDigestChangesWithSettings(t *testing.T) {
	files := map[string]Print{"a.go": {Hash: "aa"}}
	small := Digest(Record{Settings: CanonicalSettings(1, false), Files: files})
	large := Digest(Record{Settings: CanonicalSettings(2, false), Files: files})
	if small == large {
		t.Fatal("Digest ignores max_filesize")
	}
	ci := Digest(Record{Settings: CanonicalSettings(1, true), Files: files})
	if small == ci {
		t.Fatal("Digest ignores core.ignorecase")
	}
}

func TestDigestChangesWithIgnoreSource(t *testing.T) {
	files := map[string]Print{"a.go": {Hash: "aa"}}
	one := Digest(Record{Settings: CanonicalSettings(1, false), Files: files, Sources: []Source{{Rel: ".gitignore", Hash: [32]byte{1}}}})
	two := Digest(Record{Settings: CanonicalSettings(1, false), Files: files, Sources: []Source{{Rel: ".gitignore", Hash: [32]byte{2}}}})
	if one == two {
		t.Fatal("Digest ignores an ignore source's hash")
	}
}

func TestDigestChangesWithTrackedFile(t *testing.T) {
	settings := CanonicalSettings(1, false)
	plain := Digest(Record{Settings: settings, Files: map[string]Print{"a.go": {Hash: "aa"}}})
	// git add -f a previously ignored file: the file joins the corpus.
	added := Digest(Record{Settings: settings, Files: map[string]Print{
		"a.go": {Hash: "aa"}, "b.log": {Hash: "bb"},
	}})
	if plain == added {
		t.Fatal("Digest ignores an added file, which is what git add -f produces")
	}
	// git rm --cached a tracked file: it leaves the corpus.
	removed := Digest(Record{Settings: settings, Files: map[string]Print{}})
	if plain == removed {
		t.Fatal("Digest ignores a removed file, which is what git rm --cached produces")
	}
}

func TestNewRecordCarriesGitState(t *testing.T) {
	root := newRepo(t, map[string]string{
		".gitignore":     "*.log\n",
		"keep.go":        "package x\n",
		"sub/.gitignore": "drop.md\n",
		"sub/drop.md":    "x\n",
		"sub/keep.md":    "x\n",
	}, []string{"keep.go"}, nil)
	rec := buildCorpus(t, root, DefaultMaxFilesize)
	if rec.Version != corpusRecordVersion {
		t.Fatalf("version = %d", rec.Version)
	}
	if rec.GitPaths.Root == "" || rec.GitPaths.Index == "" || rec.GitPaths.Config == "" {
		t.Fatalf("git paths = %+v", rec.GitPaths)
	}
	if len(rec.Sources) == 0 || rec.Tracked == nil || len(rec.Files) == 0 {
		t.Fatalf("record = %+v", rec)
	}
	kinds := map[string]bool{}
	for _, src := range rec.Sources {
		kinds[src.Kind] = true
		if len(src.Body) == 0 {
			t.Errorf("source %s carries no body", src.Rel)
		}
	}
	if !kinds["gitignore"] {
		t.Fatalf("kinds = %v", kinds)
	}
	info, err := os.Stat(rec.GitPaths.Index)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Stamp.Size != info.Size() || rec.Stamp.MTime != info.ModTime().UnixNano() {
		t.Fatalf("stamp = %+v, file = %+v", rec.Stamp, info)
	}
	if Digest(rec) == [32]byte{} {
		t.Fatal("Digest is zero")
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	root := newRepo(t, map[string]string{".gitignore": "*.log\n", "keep.go": "package x\n"}, nil, nil)
	rec := buildCorpus(t, root, DefaultMaxFilesize)
	if err := Write(root, rec); err != nil {
		t.Fatalf("Write: %v", err)
	}
	back, ok := Read(root)
	if !ok {
		t.Fatal("Read refused the record just written")
	}
	if back.Version != rec.Version || string(back.Settings) != string(rec.Settings) || len(back.Files) != len(rec.Files) {
		t.Fatalf("round trip differs: %+v", back)
	}
	if len(back.Sources) != len(rec.Sources) {
		t.Fatalf("sources = %d, want %d", len(back.Sources), len(rec.Sources))
	}
}

func TestReadRefusesWhatItCannotUse(t *testing.T) {
	root := t.TempDir()
	if _, ok := Read(root); ok {
		t.Fatal("Read accepted a missing record")
	}
	if err := os.MkdirAll(filepath.Dir(RecordPath(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(RecordPath(root), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("{not json")
	if _, ok := Read(root); ok {
		t.Fatal("Read accepted broken JSON")
	}
	write(`{"version":99,"settings":"eA==","files":{}}`)
	if _, ok := Read(root); ok {
		t.Fatal("Read accepted another version")
	}
	write(`{"version":1,"settings":"eA=="}`)
	if _, ok := Read(root); ok {
		t.Fatal("Read accepted a record without files")
	}
	write(`{"version":1,"files":{}}`)
	if _, ok := Read(root); ok {
		t.Fatal("Read accepted a record without settings")
	}
}

func TestNewRecordOutsideRepository(t *testing.T) {
	root := t.TempDir()
	rules, _, err := Load(root)
	if err != nil || rules != nil {
		t.Fatalf("Load outside a repository = %v, %v", rules, err)
	}
	rec := NewRecord(nil, map[string]Print{}, 42)
	if rec.GitPaths.Root != "" || len(rec.Sources) != 0 || len(rec.Tracked) != 0 {
		t.Fatalf("record = %+v", rec)
	}
	if string(rec.Settings) != string(CanonicalSettings(42, false)) {
		t.Fatalf("settings = %q", rec.Settings)
	}
	if err := Write(root, rec); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, ok := Read(root); !ok {
		t.Fatal("Read refused the record just written")
	}
}
