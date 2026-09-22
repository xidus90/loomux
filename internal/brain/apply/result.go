package apply

import "github.com/xidus90/loomux/internal/brain/maintenance"

// Result is what one decision did, in the terms a caller reports on: it is
// `ApplyResult` (apply.py:263-277).
//
// Commit is empty both where the vault has no repository and where the
// commit did not land; Warning then says which. A caller that treats the two
// alike would report a lost commit as a clean run.
type Result struct {
	Case     maintenance.Case
	Decision string
	Written  bool
	Commit   string // "" when nothing was committed
	Warning  string
	Dropped  []string
}
