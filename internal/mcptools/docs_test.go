package mcptools_test

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/mcptools"
)

// withoutValues drops what follows an argument name in the reference's table:
// the allowed values in braces and the default after the arrow.
var withoutValues = regexp.MustCompile("\\{[^}]*\\}|→ (`[^`]*`|\\d+)")

var backticked = regexp.MustCompile("`([^`]+)`")

// The "twelve tools" table of docs/<lang>/cli-reference.md is what a host's
// author reads to build a call, so every row must name exactly the arguments
// the tool's schema takes: a call as documented, with a name the handler does
// not read, is refused.
func TestTheReferenceNamesTheArgumentsOfEveryTool(t *testing.T) {
	for _, lang := range []string{"en", "de"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "docs", lang, "cli-reference.md"))
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(body), "\n")
		for _, tool := range mcptools.Tools() {
			prefix := "| `" + tool.Name + "` | "
			var rows []string
			for _, line := range lines {
				if cell, ok := strings.CutPrefix(line, prefix); ok {
					rows = append(rows, cell)
				}
			}
			if len(rows) != 1 {
				t.Errorf("%s/cli-reference.md: %d table rows for %s, want 1", lang, len(rows), tool.Name)
				continue
			}
			var documented []string
			for _, m := range backticked.FindAllStringSubmatch(withoutValues.ReplaceAllString(rows[0], ""), -1) {
				documented = append(documented, m[1])
			}
			var taken []string
			for name := range schemaOf(t, tool.Name)["properties"].(map[string]any) {
				taken = append(taken, name)
			}
			sort.Strings(documented)
			sort.Strings(taken)
			if !reflect.DeepEqual(documented, taken) {
				t.Errorf("%s/cli-reference.md: %s is documented with %v, its schema takes %v", lang, tool.Name, documented, taken)
			}
		}
	}
}
