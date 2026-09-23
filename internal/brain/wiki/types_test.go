package wiki

import (
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

func typ(s string) *string { return &s }

func TestTheCoreHoldsTheFourMeasuredTypes(t *testing.T) {
	want := []string{"Architecture", "Decision", "Open Question", "Reference"}
	if len(coreTypes()) != len(want) {
		t.Fatalf("core = %v, want %v", coreTypes(), want)
	}
	for _, name := range want {
		if !coreTypes()[name] {
			t.Errorf("%q is not in the core", name)
		}
	}
}

func TestEveryTypeRanksWhereTypesPyPutsIt(t *testing.T) {
	for _, c := range []struct {
		name     string
		pageType *string
		declared map[string]bool
		want     Rank
	}{
		{"core", typ("Architecture"), nil, RankCore},
		{"catalogue without a declaration", typ("API Endpoint"), nil, RankCatalogue},
		{"origin", typ("Topic"), nil, RankOrigin},
		{"declared", typ("Balancing Rule"), map[string]bool{"Balancing Rule": true}, RankDeclared},
		{"undeclared stranger", typ("Balancing Rule"), nil, RankUnknown},
		{"missing", nil, nil, RankUnknown},
		// Letter for letter: a case fold would take back what the catalogue
		// is for.
		{"lower case", typ("topic"), nil, RankUnknown},
	} {
		if got := RankOf(c.pageType, c.declared); got != c.want {
			t.Errorf("%s: rank %q, want %q", c.name, got, c.want)
		}
	}
}

func TestAKnownAliasNamesItsTarget(t *testing.T) {
	for _, alias := range []string{"Design Decision", "Architecture Decision"} {
		if target, ok := AliasTarget(alias); !ok || target != "Decision" {
			t.Errorf("%q: target %q, %v", alias, target, ok)
		}
	}
	if target, ok := AliasTarget("Decision"); ok {
		t.Errorf("a catalogue name is an alias of %q", target)
	}
}

func TestTheThreeSetsDoNotOverlap(t *testing.T) {
	// A type with two ranks would make RankOf depend on the order it asks in.
	sets := []map[string]bool{coreTypes(), catalogueTypes(), originTypes()}
	for i, a := range sets {
		for _, b := range sets[i+1:] {
			for name := range a {
				if b[name] {
					t.Errorf("%q stands in two sets", name)
				}
			}
		}
	}
}

func TestTheConfigVocabularyIsTheUnionOfTheThreeSets(t *testing.T) {
	// config keeps its own copy for `KnowsType`, because config may not
	// import this package. This holds one direction of the two: every ranked
	// type is known there. The other direction -- config knowing a type no
	// set ranks -- would need config to hand out its map for a test.
	union := map[string]bool{}
	for _, set := range []map[string]bool{coreTypes(), catalogueTypes(), originTypes()} {
		for name := range set {
			union[name] = true
		}
	}
	bare := config.Manifest{}
	for name := range union {
		if !bare.KnowsType(name) {
			t.Errorf("config does not know %q", name)
		}
	}
}
