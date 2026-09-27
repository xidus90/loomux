package config

import "testing"

func TestIsIdentifier(t *testing.T) {
	for name, want := range map[string]bool{
		"reviewer": true, "_x": true, "Gemini_Pro2": true,
		"": false, "2fast": false, "dev-cycle": false, "a.b": false, "ü": false,
	} {
		if got := IsIdentifier(name); got != want {
			t.Errorf("IsIdentifier(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestIsFlowName(t *testing.T) {
	for name, want := range map[string]bool{
		"example": true, "dev-cycle": true, "strict-security-review2": true, "a": true,
		"": false, "Review": false, "_x": false, ".x": false, "-x": false, "9x": false, "dev_cycle": false, "a/b": false,
	} {
		if got := IsFlowName(name); got != want {
			t.Errorf("IsFlowName(%q) = %v, want %v", name, got, want)
		}
	}
}
