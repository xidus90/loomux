package treesitter

import gts "github.com/odvcencio/gotreesitter"

// ParseWith is Parse on a parser the test configured: a limit or a
// cancellation flag is how a parse stops early on purpose.
func ParseWith(p *gts.Parser, lang *gts.Language, rel string, src []byte) (*Doc, error) {
	return parse(p, lang, rel, src)
}
