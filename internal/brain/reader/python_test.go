package reader_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/reader"
	"github.com/xidus90/loomux/internal/config"
)

// Every expected value in this file was measured at the Python reference
// (brain.core._section and Path.read_text, Python 3.14.7, ultra-brain tag
// loomux-1a-source) on 2026-09-15.

func TestExtractSectionFollowsPython(t *testing.T) {
	for _, c := range []struct {
		name, text, heading, want string
	}{
		{"carriage return", "# A\rbody\r# B\rrest", "A", "# A\nbody\n"},
		{"crlf", "# A\r\nbody\r\n# B\r\nrest\r\n", "A", "# A\nbody\n"},
		{"form feed", "# A\fbody\f# B", "A", "# A\nbody\n"},
		{"vertical tab", "# A\vbody\v# B", "A", "# A\nbody\n"},
		{"file separator", "# A\x1cbody\x1c# B", "A", "# A\nbody\n"},
		{"group separator", "# A\x1dbody\x1d# B", "A", "# A\nbody\n"},
		{"record separator", "# A\x1ebody\x1e# B", "A", "# A\nbody\n"},
		{"next line", "# A\xc2\x85body\xc2\x85# B", "A", "# A\nbody\n"},
		{"line separator", "# A\xe2\x80\xa8body\xe2\x80\xa8# B", "A", "# A\nbody\n"},
		{"paragraph separator", "# A\xe2\x80\xa9body\xe2\x80\xa9# B", "A", "# A\nbody\n"},
		{"unit separator around the title", "#\x1fA\nbody\x1f\n", "A", "#\x1fA\nbody\n"},
		{"ideographic space after the title", "# A\xe3\x80\x80\nbody\n", "A", "# A\xe3\x80\x80\nbody\n"},
		{"no-break space before the title", "#\xc2\xa0A\nbody\n", "A", "#\xc2\xa0A\nbody\n"},
		{"unicode whitespace at the end", "# A\nbody\xe3\x80\x80\xc2\xa0\n\n", "A", "# A\nbody\n"},
		{"a hashtag line ends the section", "## A\nbody\n#tag\nafter\n", "A", "## A\nbody\n"},
		{"a deeper heading stays inside", "## A\nbody\n### B\nmore\n", "A", "## A\nbody\n### B\nmore\n"},
		{"closing hashes belong to the title", "## A ##\nbody\n", "A ##", "## A ##\nbody\n"},
		{"empty heading", "# \nx", "", "# \nx\n"},
		{"hashes only", "###\nx\n## y\n", "", "###\nx\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := reader.ExtractSection(c.text, c.heading)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("expected %q, got %q", c.want, got)
			}
		})
	}
}

func TestExtractSectionNamesTheMissingHeadingLikeRepr(t *testing.T) {
	doc := "# Title\n\nIntroduction paragraph.\n\n## First\n\nContent of first.\n"
	for _, c := range []struct {
		name, text, heading, want string
	}{
		{"plain", doc, "NonExistent", "no section titled 'NonExistent'"},
		{"apostrophe", doc, "it's", `no section titled "it's"`},
		{"both quotes", doc, `both ' and "`, `no section titled 'both \' and "'`},
		{"tab", doc, "a\tb", `no section titled 'a\tb'`},
		{"empty text", "", "A", "no section titled 'A'"},
		{"byte order mark before the hash", "\xef\xbb\xbf# A\nbody\n", "A", "no section titled 'A'"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := reader.ExtractSection(c.text, c.heading)
			if err == nil || err.Error() != c.want {
				t.Fatalf("expected %q, got %v", c.want, err)
			}
		})
	}
}

// area writes files below a fresh directory and returns a writable area on it.
func area(t *testing.T, files map[string]string) config.Area {
	t.Helper()
	dir := t.TempDir()
	for name, data := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return config.Area{Scope: "project/test", Path: dir}
}

func TestReadDocumentFoldsNewlinesLikeReadText(t *testing.T) {
	a := area(t, map[string]string{"crlf.md": "# A\r\nb\rc\n"})
	got, err := reader.ReadDocument(a, &config.Manifest{}, "crlf.md", "", privacy.ChannelLocal, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "# A\nb\nc\n" {
		t.Errorf("expected %q, got %q", "# A\nb\nc\n", got)
	}
}

func TestReadDocumentFindsASectionBehindLoneCarriageReturns(t *testing.T) {
	a := area(t, map[string]string{"cr.md": "# A\rbody\r# B\rrest"})
	got, err := reader.ReadDocument(a, &config.Manifest{}, "cr.md", "A", privacy.ChannelLocal, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "# A\nbody\n" {
		t.Errorf("expected %q, got %q", "# A\nbody\n", got)
	}
}

func TestReadDocumentRefusesInvalidUTF8(t *testing.T) {
	a := area(t, map[string]string{"bad.md": "# A\n\xff\n"})
	_, err := reader.ReadDocument(a, &config.Manifest{}, "bad.md", "", privacy.ChannelLocal, "")
	want := filepath.Join(a.Path, "bad.md") + ": not valid UTF-8"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestReadDocumentKeepsTheByteOrderMark(t *testing.T) {
	a := area(t, map[string]string{"bom.md": "\xef\xbb\xbf# A\n"})
	got, err := reader.ReadDocument(a, &config.Manifest{}, "bom.md", "", privacy.ChannelLocal, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "\xef\xbb\xbf# A\n" {
		t.Errorf("expected %q, got %q", "\xef\xbb\xbf# A\n", got)
	}
	_, err = reader.ReadDocument(a, &config.Manifest{}, "bom.md", "A", privacy.ChannelLocal, "")
	if err == nil || err.Error() != "no section titled 'A'" {
		t.Fatalf("expected %q, got %v", "no section titled 'A'", err)
	}
}

func TestReadDocumentHandsOnAMissingFile(t *testing.T) {
	a := area(t, nil)
	_, err := reader.ReadDocument(a, &config.Manifest{}, "missing.md", "", privacy.ChannelLocal, "")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected a not-exist error, got %v", err)
	}
}

func TestReadDocumentChecksNeverBeforeTheReviewCentre(t *testing.T) {
	a := area(t, map[string]string{"review/secret.md": "x"})
	manifest := &config.Manifest{LayoutReview: "review", NeverGlobs: []string{"review/secret.md"}}
	_, err := reader.ReadDocument(a, manifest, "review/secret.md", "", privacy.ChannelCloud, "")
	want := "project/test/review/secret.md is excluded by [privacy] never"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestReadDocumentChecksContainmentBeforeNever(t *testing.T) {
	a := area(t, nil)
	manifest := &config.Manifest{NeverGlobs: []string{"**"}}
	_, err := reader.ReadDocument(a, manifest, "../secrets/key.txt", "", privacy.ChannelLocal, "")
	want := "project/test/../secrets/key.txt leaves the area"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestReadDocumentNamesTheContainedPathInRefusals(t *testing.T) {
	a := area(t, map[string]string{"secrets/key.txt": "k", "review/cases/c1.md": "c"})
	manifest := &config.Manifest{LayoutReview: "review", NeverGlobs: []string{"secrets/**"}}
	_, err := reader.ReadDocument(a, manifest, "./secrets//key.txt", "", privacy.ChannelLocal, "")
	want := "project/test/secrets/key.txt is excluded by [privacy] never"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
	_, err = reader.ReadDocument(a, manifest, "review/./cases/c1.md", "", privacy.ChannelCloud, "")
	want = "project/test/review/cases/c1.md is the review centre; refused on the cloud channel"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestReadDocumentAsksForTheReviewCentreOnlyOnTheCloud(t *testing.T) {
	a := area(t, map[string]string{"doc.md": "text\n"})
	got, err := reader.ReadDocument(a, &config.Manifest{LayoutReview: "."}, "doc.md", "", privacy.ChannelLocal, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "text\n" {
		t.Errorf("expected %q, got %q", "text\n", got)
	}
}

func TestReadDocumentWithoutAPathReadsTheAreaItself(t *testing.T) {
	a := area(t, nil)
	for _, relative := range []string{"", "."} {
		_, err := reader.ReadDocument(a, &config.Manifest{}, relative, "", privacy.ChannelLocal, "")
		if err == nil || strings.Contains(err.Error(), "leaves the area") {
			t.Fatalf("%q: expected a read error on the area directory, got %v", relative, err)
		}
	}
}
