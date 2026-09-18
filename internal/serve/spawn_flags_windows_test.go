package serve

import "testing"

func TestDetachAttrsAskToLeaveTheJobOnlyOnTheFirstAttempt(t *testing.T) {
	with := detachAttrs(true)
	if with.CreationFlags != detachedProcess|createBreakawayFromJob {
		t.Errorf("creation flags are %#x, want DETACHED_PROCESS|CREATE_BREAKAWAY_FROM_JOB", with.CreationFlags)
	}
	// The fallback keeps the detachment and gives up only the breakaway: a
	// host that put us in a job object without BREAKAWAY_OK refuses the flag,
	// and a service inside that job is better than no service at all.
	without := detachAttrs(false)
	if without.CreationFlags != detachedProcess {
		t.Errorf("creation flags are %#x, want DETACHED_PROCESS alone", without.CreationFlags)
	}
}
