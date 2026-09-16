package privacy_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/testlock"
)

func TestParseChannel(t *testing.T) {
	ch, err := privacy.ParseChannel("local")
	if err != nil || ch != privacy.ChannelLocal {
		t.Fatalf("expected ChannelLocal, got %v (err: %v)", ch, err)
	}

	ch, err = privacy.ParseChannel("cloud")
	if err != nil || ch != privacy.ChannelCloud {
		t.Fatalf("expected ChannelCloud, got %v (err: %v)", ch, err)
	}

	for _, invalid := range []string{"", "invalid", "LOCAL", "Cloud", "other"} {
		_, err := privacy.ParseChannel(invalid)
		if err == nil {
			t.Errorf("expected error for invalid channel %q, got nil", invalid)
		}
	}
}

func TestIsVisible(t *testing.T) {
	if !privacy.IsVisible(nil, privacy.ChannelLocal) {
		t.Error("nil manifest should be visible on ChannelLocal")
	}
	if !privacy.IsVisible(nil, privacy.ChannelCloud) {
		t.Error("nil manifest should be visible on ChannelCloud")
	}

	mManual := &config.Manifest{PrivacyMode: "manual_cloud"}
	if !privacy.IsVisible(mManual, privacy.ChannelLocal) {
		t.Error("manual_cloud should be visible on ChannelLocal")
	}
	if !privacy.IsVisible(mManual, privacy.ChannelCloud) {
		t.Error("manual_cloud should be visible on ChannelCloud")
	}

	mLocal := &config.Manifest{PrivacyMode: "local_only"}
	if !privacy.IsVisible(mLocal, privacy.ChannelLocal) {
		t.Error("local_only should be visible on ChannelLocal")
	}
	if privacy.IsVisible(mLocal, privacy.ChannelCloud) {
		t.Error("local_only must NOT be visible on ChannelCloud")
	}
}

// VisibleManifest is the one gate the visibility callers share. Every failure
// is an error, the absent declaration included: `_visible_areas` lets
// `read_manifest` raise for each registered area, so an area without a usable
// declaration stops the whole call instead of being served or hidden. The
// names are read in the order config.ReadAreaManifestUntilStage4 gives them:
// the first one that exists as a file is the declaration, readable or not,
// except a .loomux/config.toml without an [area] table, which declares nothing.
func TestVisibleManifest(t *testing.T) {
	const open = "[area]\nscope = \"k\"\n\n[privacy]\nmode = \"manual_cloud\"\n"
	const closed = "[area]\nscope = \"k\"\n\n[privacy]\nmode = \"local_only\"\n"
	tests := []struct {
		name         string
		build        func(t *testing.T, dir string)
		local, cloud bool
		wantManifest bool
		wantErr      bool
	}{
		{"no manifest", func(t *testing.T, dir string) {}, false, false, false, true},
		{"open", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, ".brain.toml"), open)
		}, true, true, true, false},
		{"local_only", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, ".brain.toml"), closed)
		}, true, false, true, false},
		{"not TOML", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, ".brain.toml"), "[area\n")
		}, false, false, false, true},
		{"misspelt mode", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, ".brain.toml"), "[area]\nscope = \"k\"\n\n[privacy]\nmode = \"lokal_only\"\n")
		}, false, false, false, true},
		{"closed config.toml that cannot be read beside an open .brain.toml", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, ".brain.toml"), open)
			path := filepath.Join(dir, ".ultra-brain", "config.toml")
			writeFile(t, path, closed)
			testlock.Lock(t, path)
		}, false, false, false, true},
		{"closed .loomux/config.toml that cannot be read beside an open .brain.toml", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, ".brain.toml"), open)
			path := filepath.Join(dir, ".loomux", "config.toml")
			writeFile(t, path, closed)
			testlock.Lock(t, path)
		}, false, false, false, true},
		{"the loomux name before an open .brain.toml", func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, ".brain.toml"), open)
			writeFile(t, filepath.Join(dir, ".loomux", "config.toml"), closed)
		}, true, false, true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.build(t, dir)
			for _, c := range []struct {
				ch   privacy.Channel
				want bool
			}{{privacy.ChannelLocal, tc.local}, {privacy.ChannelCloud, tc.cloud}} {
				m, visible, err := privacy.VisibleManifest(dir, c.ch)
				if (err != nil) != tc.wantErr {
					t.Errorf("%s: err = %v, want an error: %v", c.ch, err, tc.wantErr)
				}
				if visible != c.want {
					t.Errorf("%s: visible = %v, want %v", c.ch, visible, c.want)
				}
				if (m != nil) != tc.wantManifest {
					t.Errorf("%s: manifest = %+v, want one exactly for a readable declaration", c.ch, m)
				}
			}
		})
	}
}

func TestVisibleManifestWithoutDeclarationIsErrNoManifest(t *testing.T) {
	_, _, err := privacy.VisibleManifest(t.TempDir(), privacy.ChannelLocal)
	if !errors.Is(err, config.ErrNoManifest) {
		t.Fatalf("err = %v, want config.ErrNoManifest", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Contained is `_contained` (src/brain/core.py:649-668). Every answer below was
// measured against it on 2026-09-15 under Python 3.14.7.
func TestContained(t *testing.T) {
	good := []struct {
		relative string
		expected string
	}{
		{"doc.md", "doc.md"},
		{"sub/doc.md", "sub/doc.md"},
		{"sub\\doc.md", "sub/doc.md"},
		{"a/b/c.md", "a/b/c.md"},
		{"./a.md", "a.md"},
		{"a//b.md", "a/b.md"},
		{"a\\\\b.md", "a/b.md"},
		{"a/./b.md", "a/b.md"},
		{"a/", "a"},
		{"a/b/", "a/b"},
		{"x\\", "x"},
		{".", "."},
		{"", "."},
		{"./", "."},
		{":x.md", ":x.md"},
		{"ab:c.md", "ab:c.md"},
		{"a/1:b.md", "a/1:b.md"},
		{"a//C:x", "a/C:x"},
		{"a..b.md", "a..b.md"},
		{"...", "..."},
	}

	for _, tc := range good {
		res, err := privacy.Contained("area", tc.relative)
		if err != nil {
			t.Errorf("expected %q to be contained, got error: %v", tc.relative, err)
		}
		if res != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, res)
		}
	}

	bad := []string{
		"../escape.md",
		"..\\escape.md",
		"sub/../../escape.md",
		"a/../b.md",
		"..",
		"/abs/doc.md",
		"\\abs\\doc.md",
		"C:/abs/doc.md",
		"C:\\abs\\doc.md",
		"C:",
		"d:relative.md",
		"1:foo.md",
		"é:x.md",
		"./1:foo.md",
		".\\1:x.md",
		".//d:x",
		"//server/share/doc.md",
		"\\\\server\\share\\doc.md",
		"\\",
		"\\\\",
	}

	for _, relative := range bad {
		_, err := privacy.Contained("test-scope", relative)
		if err == nil {
			t.Errorf("expected %q to be rejected, got nil", relative)
			continue
		}
		if want := "test-scope/" + relative + " leaves the area"; err.Error() != want {
			t.Errorf("expected %q, got %q", want, err.Error())
		}
	}
}

func TestIsReadable(t *testing.T) {
	if !privacy.IsReadable(nil, "any/path.md") {
		t.Error("nil manifest should be readable")
	}

	mEmpty := &config.Manifest{}
	if !privacy.IsReadable(mEmpty, "any/path.md") {
		t.Error("manifest without NeverGlobs should be readable")
	}

	m := &config.Manifest{
		NeverGlobs: []string{"secrets/**", "*.pem", "**/private/**", "test?.txt"},
	}

	cases := []struct {
		path     string
		readable bool
	}{
		{"doc.md", true},
		{"notes/doc.md", true},
		{"secrets/key.txt", false},
		{"SECRETS/key.txt", false},
		{"Secrets/sub/key.txt", false},
		{"key.pem", false},
		{"sub/key.pem", true},
		{"sub/private/file.txt", false},
		{"private/file.txt", false},
		{"test1.txt", false},
		{"test12.txt", true},
	}

	for _, tc := range cases {
		got := privacy.IsReadable(m, tc.path)
		if got != tc.readable {
			t.Errorf("path %q: expected readable=%v, got %v", tc.path, tc.readable, got)
		}
	}
}

func TestReviewExcludes(t *testing.T) {
	excludes, err := privacy.ReviewExcludes(nil)
	if err != nil || len(excludes) != 0 {
		t.Errorf("expected empty excludes for nil manifest, got %v, err: %v", excludes, err)
	}

	mNoReview := &config.Manifest{}
	excludes, err = privacy.ReviewExcludes(mNoReview)
	if err != nil || len(excludes) != 0 {
		t.Errorf("expected empty excludes for manifest without LayoutReview, got %v, err: %v", excludes, err)
	}

	mReview := &config.Manifest{LayoutReview: "95 Prüfzentrum"}
	excludes, err = privacy.ReviewExcludes(mReview)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(excludes) != 2 || excludes[0] != "95 Prüfzentrum/**" || excludes[1] != "95 Prüfzentrum" {
		t.Errorf("unexpected excludes: %v", excludes)
	}

	// `review_excludes` (src/brain/walk.py:96-113) refuses a value only when
	// `PurePosixPath(review).parts` is empty, and uses every other value as
	// written. Measured on 2026-09-15 under Python 3.14.7.
	for _, invalid := range []string{".", "./.", ".//."} {
		mInvalid := &config.Manifest{LayoutReview: invalid}
		_, err := privacy.ReviewExcludes(mInvalid)
		if err == nil {
			t.Errorf("expected error for invalid LayoutReview %q, got nil", invalid)
		}
	}
	_, err = privacy.ReviewExcludes(&config.Manifest{LayoutReview: "./"})
	if want := "[layout] review must not resolve to the area root, found './'"; err == nil || err.Error() != want {
		t.Errorf("expected %q, got %v", want, err)
	}
	for _, tc := range []struct {
		review string
		want   []string
	}{
		{"/", []string{"//**", "/"}},
		{"\\", []string{"\\/**", "\\"}},
		{"///", []string{"////**", "///"}},
		{" review ", []string{" review /**", " review "}},
		{"a/..", []string{"a/../**", "a/.."}},
		{"it's/.", []string{"it's/./**", "it's/."}},
	} {
		excludes, err := privacy.ReviewExcludes(&config.Manifest{LayoutReview: tc.review})
		if err != nil || !slices.Equal(excludes, tc.want) {
			t.Errorf("LayoutReview %q: got %q, err %v; want %q", tc.review, excludes, err, tc.want)
		}
	}
}
