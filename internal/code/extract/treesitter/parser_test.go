package treesitter_test

import (
	"os"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/treesitter"
)

// Parser reaches every tree-sitter language's version and through it the
// extract cache and the graph's meta. A raised pin with the constant left
// behind would keep answering from trees the old runtime built.
func TestParserMatchesGoMod(t *testing.T) {
	b, err := os.ReadFile("../../../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	const module = "github.com/odvcencio/gotreesitter"
	version := ""
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		for i := 0; i+1 < len(f); i++ {
			if f[i] == module && strings.HasPrefix(f[i+1], "v") {
				version = f[i+1]
			}
		}
	}
	if version == "" {
		t.Fatalf("go.mod pins no %s", module)
	}
	if want := "gotreesitter/" + version; treesitter.Parser != want {
		t.Errorf("Parser = %q, want %q to match the go.mod pin", treesitter.Parser, want)
	}
}
