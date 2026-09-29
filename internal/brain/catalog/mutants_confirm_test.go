package catalog_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/brain/catalog"
)

// Three spellings reach the rules of Withhold that the main test does not
// separate: a `=` inside an angle destination that also holds a blank, so
// only the angle form can read it; a scheme whose path cleans into the
// hidden tree, which leaves the vault all the same; and a destination that
// is only a fragment, which names the catalog's own page whatever follows
// the `#`.
func TestWithholdReadsEqualsSchemesAndBareFragmentsAsWritten(t *testing.T) {
	text := "* [Equals](<inner space/a=b.md>)\n" +
		"* [Drive](c:/../inner%20space/x.md)\n" +
		"* [Fragment](<#/../inner space/x.md>)\n"
	want := "* [Drive](c:/../inner%20space/x.md)\n" +
		"* [Fragment](<#/../inner space/x.md>)\n"
	if got := catalog.Withhold(text, under); got != want {
		t.Errorf("Withhold:\n%s\nwant:\n%s", got, want)
	}
}
