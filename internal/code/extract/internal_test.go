package extract

import "testing"

func TestItoaZero(t *testing.T) {
	// MintID only ever calls itoa with k starting at 2, so the n == 0 arm is
	// unreachable through it. Direct call is the only way to it.
	if got := itoa(0); got != "0" {
		t.Errorf("itoa(0) = %q, want %q", got, "0")
	}
}
