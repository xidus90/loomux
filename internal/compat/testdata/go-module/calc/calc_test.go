package calc

import "testing"

func TestChildAddsOne(t *testing.T) {
	if got := Child(); got != 2 {
		t.Fatalf("Child() = %d, want 2", got)
	}
}

func TestDouble(t *testing.T) {
	if got := Double(3); got != 6 {
		t.Fatalf("Double(3) = %d, want 6", got)
	}
}
