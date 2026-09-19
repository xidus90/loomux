package detect

import (
	"slices"
	"sort"
	"testing"
)

func TestSignalNamesAreSortedAndUnique(t *testing.T) {
	names := SignalNames()
	if !sort.StringsAreSorted(names) || !slices.Contains(names, "pyright") || !slices.Contains(names, "biome") {
		t.Fatalf("%v", names)
	}
	for i := 1; i < len(names); i++ {
		if names[i] == names[i-1] {
			t.Fatalf("duplicate %q", names[i])
		}
	}
}
