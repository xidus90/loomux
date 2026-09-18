package hooks

import "testing"

func TestStackForExtension(t *testing.T) {
	if stack, ok := StackForExtension(".go"); !ok || stack != "go" {
		t.Errorf("StackForExtension(.go) = %q, %v; want go, true", stack, ok)
	}
	if stack, ok := StackForExtension(".unknown"); ok || stack != "" {
		t.Errorf("StackForExtension(.unknown) = %q, %v; want \"\", false", stack, ok)
	}
}
