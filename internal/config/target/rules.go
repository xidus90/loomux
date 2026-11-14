package target

import "github.com/xidus90/loomux/internal/config/edit"

// AppendRule adds one entry to the rule list at the dotted section in the
// form the file already uses for that list (see edit.AppendRule). The result
// is checked the way Change checks one.
func (t Target) AppendRule(text, section string, pairs [][2]string) (string, error) {
	next, err := edit.AppendRule(text, section, pairs)
	if err != nil {
		return "", err
	}
	return t.Validated(next)
}

// RemoveRule takes the index-th entry out of the rule list at the dotted
// section (see edit.RemoveItem). The result is checked the way Change checks
// one.
func (t Target) RemoveRule(text, section string, index int) (string, error) {
	next, err := edit.RemoveItem(text, section, index)
	if err != nil {
		return "", err
	}
	return t.Validated(next)
}
