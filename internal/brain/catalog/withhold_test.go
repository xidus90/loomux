package catalog_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/brain/catalog"
)

// under is a concealment that hides the tree "inner space" and nothing else,
// as the cleaned relative paths privacy.Contained hands on spell it.
func under(relative string) bool {
	return relative == "inner space" || len(relative) > len("inner space/") && relative[:len("inner space/")] == "inner space/"
}

// A generated catalog names a nested wiki as a subdirectory line and may list
// its files; every line whose link lands in the hidden tree goes, in each
// spelling the writer and a hand can give it, and everything else stays as
// it was, byte for byte.
func TestWithholdDropsTheLinesThatLinkIntoAHiddenTree(t *testing.T) {
	text := "# hub\n\nIntro naming [the rule](rules.md).\n\n## Bereiche\n\n" +
		"* [inner space](<inner space/>)\n" +
		"* [inner-neu](inner-neu/)\n" +
		"* [other](other/)\n\n## Dateien\n\n" +
		"* [Deep](<inner space/deep page.md>) - a page\n" +
		"* [Escaped](inner%20space/x.md)\n" +
		"* [Anchored](<inner space/y.md#part>)\n" +
		"* [Dotted](<./inner space/../inner space/z.md>)\n" +
		"* [Top](top.md) - kept\n" +
		"* [Two](top.md), [links](<inner space/q.md>)\n" +
		"* [Web](https://example.org/inner%20space/)\n" +
		"* [Rooted](</inner space/r.md>)\n" +
		"* [RootedUp](</../inner space/v.md>)\n" +
		"* [RootedTwice](<//../inner space/w.md>)\n" +
		"* [Up](<../inner space/u.md>)\n" +
		"* [Broken](inner%zzspace/x.md)\n" +
		"* [Angled](<a%3Cb%3E.md>)\n\n" +
		"> Jeder neue Bereich bekommt sofort eine Zeile in diesem Katalog.\n"
	want := "# hub\n\nIntro naming [the rule](rules.md).\n\n## Bereiche\n\n" +
		"* [inner-neu](inner-neu/)\n" +
		"* [other](other/)\n\n## Dateien\n\n" +
		"* [Top](top.md) - kept\n" +
		"* [Web](https://example.org/inner%20space/)\n" +
		"* [Up](<../inner space/u.md>)\n" +
		"* [Broken](inner%zzspace/x.md)\n" +
		"* [Angled](<a%3Cb%3E.md>)\n\n" +
		"> Jeder neue Bereich bekommt sofort eine Zeile in diesem Katalog.\n"
	if got := catalog.Withhold(text, under); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// Nothing concealed, nothing changed: the local channel reads the catalog as
// the file holds it, CRLF and a missing final newline included.
func TestWithholdLeavesAnUnconcealedCatalogAlone(t *testing.T) {
	text := "# hub\r\n\r\n* [inner space](<inner space/>)"
	if got := catalog.Withhold(text, func(string) bool { return false }); got != text {
		t.Errorf("got %q, want %q", got, text)
	}
}
