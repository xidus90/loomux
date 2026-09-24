package selfupdate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var stamp = time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)

func fetchOptions(f *fakeGH) Options {
	return Options{GOOS: "windows", GOARCH: "amd64", Run: f.run, Now: func() time.Time { return stamp }}
}

func TestFetchStagesAVerifiedBinary(t *testing.T) {
	dir := t.TempDir()
	if err := fetch(context.Background(), fetchOptions(release("2.8.0")), "v2.8.0", dir); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, "loomux.new.exe")
	data, err := os.ReadFile(staged)
	if err != nil || string(data) != "binary 2.8.0" {
		t.Fatalf("staged = %q, %v", data, err)
	}
	// The activation is the modification time: OlderThan compares nothing
	// else, so the pass sets it rather than trusting the download's.
	info, _ := os.Stat(staged)
	if !info.ModTime().Equal(stamp) {
		t.Fatalf("mtime = %v, want %v", info.ModTime(), stamp)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("the download directory was left behind: %v", entries)
	}
}

func TestFetchRefuses(t *testing.T) {
	asset := AssetName("2.8.0", "windows", "amd64")
	for _, c := range []struct {
		name  string
		spoil func(f *fakeGH)
		want  string
	}{
		{"no SHA256SUMS", func(f *fakeGH) { delete(f.files, "SHA256SUMS") }, "release v2.8.0 has no SHA256SUMS"},
		{"no line for the asset", func(f *fakeGH) { f.files["SHA256SUMS"] = "abc  other.exe\n" }, "SHA256SUMS of v2.8.0 has no line for " + asset},
		{"no asset", func(f *fakeGH) { delete(f.files, asset) }, "release v2.8.0 has no asset " + asset},
		{"a changed asset", func(f *fakeGH) { f.files[asset] = "tampered" }, "checksum mismatch for " + asset},
		{"another version inside", func(f *fakeGH) { f.version = "loomux 2.7.0 (beta)\n" }, `downloaded binary reports "loomux 2.7.0 (beta)", not loomux 2.8.0`},
		{"a binary that does not run", func(f *fakeGH) { f.fail["--version"] = errors.New("exec format error") }, "run downloaded binary: exec format error"},
		{"a failed download", func(f *fakeGH) { f.fail["download"] = errors.New("gh: no assets match") }, "gh: no assets match"},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			f := release("2.8.0")
			c.spoil(f)
			err := fetch(context.Background(), fetchOptions(f), "v2.8.0", dir)
			if err == nil || err.Error() != c.want {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
			if _, err := os.Stat(filepath.Join(dir, "loomux.new.exe")); !os.IsNotExist(err) {
				t.Fatal("a refused download was staged")
			}
		})
	}
}

func TestFetchAcceptsTheBinaryModeMarker(t *testing.T) {
	f := release("2.8.0")
	asset := AssetName("2.8.0", "windows", "amd64")
	f.files["SHA256SUMS"] = strings.Replace(f.files["SHA256SUMS"], "  "+asset, " *"+asset, 1)
	if err := fetch(context.Background(), fetchOptions(f), "v2.8.0", t.TempDir()); err != nil {
		t.Fatal(err)
	}
}

func TestFetchFailsWithoutItsDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	err := fetch(context.Background(), fetchOptions(release("2.8.0")), "v2.8.0", missing)
	if err == nil || !strings.Contains(err.Error(), "create download directory") {
		t.Fatalf("err = %v", err)
	}
}

func TestFetchFailsWhenItCannotStage(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "loomux.new.exe")
	if err := os.MkdirAll(filepath.Join(blocker, "inside"), 0o700); err != nil {
		t.Fatal(err)
	}
	err := fetch(context.Background(), fetchOptions(release("2.8.0")), "v2.8.0", dir)
	if err == nil || !strings.Contains(err.Error(), "stage") {
		t.Fatalf("err = %v", err)
	}
}

func TestStageFailsWhenItCannotStamp(t *testing.T) {
	err := stage(filepath.Join(t.TempDir(), "missing"), "x", stamp)
	if err == nil || !strings.Contains(err.Error(), "stamp") {
		t.Fatalf("err = %v", err)
	}
}

func TestReportsVersion(t *testing.T) {
	for out, want := range map[string]bool{
		"loomux 2.8.0\n":        true,
		"loomux 2.8.0 (beta)\n": true,
		"loomux 2.8.01\n":       false,
		"loomux 2.8\n":          false,
		"":                      false,
	} {
		if got := reportsVersion(out, "2.8.0"); got != want {
			t.Errorf("reportsVersion(%q) = %v, want %v", out, got, want)
		}
	}
}

func TestAssetName(t *testing.T) {
	if got := AssetName("2.7.0", "windows", "amd64"); got != "loomux_2.7.0_windows_amd64.exe" {
		t.Errorf("windows: %s", got)
	}
	if got := AssetName("2.7.0", "linux", "arm64"); got != "loomux_2.7.0_linux_arm64" {
		t.Errorf("linux: %s", got)
	}
}
