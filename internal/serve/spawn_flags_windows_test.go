package serve

import "testing"

func TestDetachAttrsAskToLeaveTheJobOnlyOnTheFirstAttempt(t *testing.T) {
	with := detachAttrs(true, false)
	if with.CreationFlags != detachedProcess|createBreakawayFromJob {
		t.Errorf("creation flags are %#x, want DETACHED_PROCESS|CREATE_BREAKAWAY_FROM_JOB", with.CreationFlags)
	}
	// The fallback keeps the detachment and gives up only the breakaway: a
	// host that put us in a job object without BREAKAWAY_OK refuses the flag,
	// and a service inside that job is better than no service at all.
	without := detachAttrs(false, false)
	if without.CreationFlags != detachedProcess {
		t.Errorf("creation flags are %#x, want DETACHED_PROCESS alone", without.CreationFlags)
	}
}

// A child that starts console programs gets a console without a window: with
// none, Windows opens a visible one for every program it starts.
func TestDetachAttrsGiveAHiddenConsoleInsteadOfNone(t *testing.T) {
	if got := detachAttrs(true, true).CreationFlags; got != createNoWindow|createBreakawayFromJob {
		t.Errorf("creation flags are %#x, want CREATE_NO_WINDOW|CREATE_BREAKAWAY_FROM_JOB", got)
	}
	if got := detachAttrs(false, true).CreationFlags; got != createNoWindow {
		t.Errorf("creation flags are %#x, want CREATE_NO_WINDOW alone", got)
	}
}
