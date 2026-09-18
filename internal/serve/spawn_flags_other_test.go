//go:build !windows

package serve

import "testing"

func TestDetachAttrsAskForASessionOfItsOwn(t *testing.T) {
	// There is no job object here and nothing to break away from, so both
	// attempts ask for the same thing: a session in which no terminal signal
	// of the parent's reaches the service.
	for _, breakaway := range []bool{true, false} {
		if attrs := detachAttrs(breakaway); !attrs.Setsid {
			t.Errorf("detachAttrs(%v) does not ask for setsid", breakaway)
		}
	}
}
