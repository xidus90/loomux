package commit_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/verify/commit"
)

func TestStopwordsEN(t *testing.T) {
	en := commit.Stopwords("en")
	if en == nil {
		t.Fatal("expected non-nil stopwords for en")
	}

	// Must contain German source words
	for _, w := range []string{"auf", "aus", "nicht", "und", "fuer", "ueber"} {
		if _, ok := en[w]; !ok {
			t.Errorf("expected %q in Stopwords(en)", w)
		}
	}

	// Must contain Romance source words
	for _, w := range []string{"para", "que", "avec", "cette"} {
		if _, ok := en[w]; !ok {
			t.Errorf("expected %q in Stopwords(en)", w)
		}
	}

	// Must contain Go list words (Variant B)
	for _, w := range []string{"aktualisiere", "schnittstelle", "behebe", "korrigiere", "hinzu"} {
		if _, ok := en[w]; !ok {
			t.Errorf("expected %q in Stopwords(en)", w)
		}
	}

	// Must NOT contain ordinary English words
	for _, w := range []string{"die", "war", "man", "den", "hat", "in", "so", "an", "do", "care", "as", "no", "car", "fest"} {
		if _, ok := en[w]; ok {
			t.Errorf("ordinary English word %q must not be in Stopwords(en)", w)
		}
	}
}

func TestStopwordsDE(t *testing.T) {
	de := commit.Stopwords("de")
	if de == nil {
		t.Fatal("expected non-nil stopwords for de")
	}

	// Must contain English source words
	for _, w := range []string{"about", "against", "because", "between", "without"} {
		if _, ok := de[w]; !ok {
			t.Errorf("expected %q in Stopwords(de)", w)
		}
	}

	// Must contain Romance source words
	for _, w := range []string{"para", "que", "avec", "cette"} {
		if _, ok := de[w]; !ok {
			t.Errorf("expected %q in Stopwords(de)", w)
		}
	}

	// Must NOT contain ordinary German words
	for _, w := range []string{"in", "so", "am", "da", "im", "man", "ist", "war", "still"} {
		if _, ok := de[w]; ok {
			t.Errorf("ordinary German word %q must not be in Stopwords(de)", w)
		}
	}
}

func TestStopwordsUnknown(t *testing.T) {
	if got := commit.Stopwords("fr"); got != nil {
		t.Errorf("expected nil for unknown language, got %v", got)
	}
}
