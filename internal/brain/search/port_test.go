package search_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
)

func TestProfileConstants(t *testing.T) {
	if search.ProfileKeyword != "keyword" {
		t.Errorf("expected ProfileKeyword to be 'keyword', got %q", search.ProfileKeyword)
	}
	if search.ProfileFast != "fast" {
		t.Errorf("expected ProfileFast to be 'fast', got %q", search.ProfileFast)
	}
	if search.ProfileFull != "full" {
		t.Errorf("expected ProfileFull to be 'full', got %q", search.ProfileFull)
	}
}
