package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The loomux graph commands read the graph and the files as they stand and
// pass no filter for the [privacy] never globs, so the docs that bound never
// to loomux's own channels must name them as the exception. This test holds
// the premise and the wording together: a command that starts filtering has
// to change the docs with it.
func TestTheDocsNameTheGraphCommandsAsTheExceptionOfNever(t *testing.T) {
	root := repo(t, map[string]string{
		"go.mod":              "module example.com/repo\n",
		".loomux/config.toml": "[privacy]\nmode = \"manual_cloud\"\nnever = [\"secret/**\"]\n",
		"secret/key.go":       "package secret\n\nconst Key = \"sk-live-TOPSECRET-4711\"\n",
		"main.go":             "package main\n\nfunc main() {}\n",
	})
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build: exit %d, stderr %q", code, errOut.String())
	}
	for _, args := range [][]string{
		{"grep", "TOPSECRET", "--root", root},
		{"ask", "Key", "--source", "--root", root},
	} {
		out.Reset()
		errOut.Reset()
		if code := graphCommand(args, nil, &out, &errOut); code != 0 {
			t.Fatalf("%v: exit %d, stderr %q", args, code, errOut.String())
		}
		if !strings.Contains(out.String(), "sk-live-TOPSECRET-4711") {
			t.Errorf("%v prints no line of the never file (%q); the docs say it does", args, out.String())
		}
	}

	read := func(path ...string) string {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(append([]string{"..", ".."}, path...)...))
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	const graphCommands = "`loomux graph`"

	for _, lang := range []string{"en", "de"} {
		_, rest, _ := strings.Cut(read("docs", lang, "configuration.md"), "### `[privacy]`")
		section, _, _ := strings.Cut(rest, "\n---\n")
		if !strings.Contains(section, graphCommands) {
			t.Errorf("%s/configuration.md: the [privacy] section does not name %s as the exception of never", lang, graphCommands)
		}

		reference := read("docs", lang, "cli-reference.md")
		_, graphSection, _ := strings.Cut(reference, "(`loomux graph`)\n")
		graphSection, _, _ = strings.Cut(graphSection, "\n## 7.")
		if !strings.Contains(graphSection, "`never`") {
			t.Errorf("%s/cli-reference.md: the loomux graph section does not say that it does not filter `never` paths", lang)
		}
		for _, line := range strings.Split(reference, "\n") {
			if (strings.HasPrefix(line, "- **Channel**:") || strings.HasPrefix(line, "- **Kanal**:")) && !strings.Contains(line, graphCommands) {
				t.Errorf("%s/cli-reference.md: the channel bullet says never globs apply on every channel and leaves out %s", lang, graphCommands)
			}
		}
	}

	if wiki := read("docs", "wiki", "topics", "datenschutz-und-kanaele.md"); !strings.Contains(wiki, "loomux graph") {
		t.Error("the wiki page on privacy does not name loomux graph as the exception of never")
	}
}
