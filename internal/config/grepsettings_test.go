package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeGrepManifest puts text into the project's manifest at root.
func writeGrepManifest(t *testing.T, root, text string) {
	t.Helper()
	path := ManifestPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGrepMaxFilesizeAbsentMeansNothingDeclared(t *testing.T) {
	limit, err := ReadGrepMaxFilesize(t.TempDir())
	if err != nil || limit != nil {
		t.Fatalf("limit = %v, err = %v; want nil and no error", limit, err)
	}
}

func TestGrepMaxFilesizeTakesTheLimit(t *testing.T) {
	root := t.TempDir()
	writeGrepManifest(t, root, "[grep]\nmax_filesize = 1234\n")
	limit, err := ReadGrepMaxFilesize(root)
	if err != nil || limit == nil || *limit != 1234 {
		t.Fatalf("limit = %v, err = %v; want 1234", limit, err)
	}
}

func TestGrepMaxFilesizeZeroIsUnlimited(t *testing.T) {
	root := t.TempDir()
	writeGrepManifest(t, root, "[grep]\nmax_filesize = 0\n")
	limit, err := ReadGrepMaxFilesize(root)
	if err != nil || limit == nil || *limit != 0 {
		t.Fatalf("limit = %v, err = %v; want an explicit 0", limit, err)
	}
}

func TestGrepMaxFilesizeRefusesWhatIsNoLimit(t *testing.T) {
	cases := map[string]string{
		"negative":    "[grep]\nmax_filesize = -1\n",
		"text":        "[grep]\nmax_filesize = \"big\"\n",
		"fraction":    "[grep]\nmax_filesize = 1.5\n",
		"not a table": "grep = 3\n",
		"unknown key": "[grep]\nmax_size = 1\n",
		"not toml":    "[grep\n",
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeGrepManifest(t, root, text)
			if _, err := ReadGrepMaxFilesize(root); err == nil || !strings.Contains(err.Error(), ManifestPath(root)) {
				t.Fatalf("err = %v; want a refusal that names the manifest", err)
			}
		})
	}
}
