package verify

import (
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/detect"
)

// GUT gets the test lane where its addon is found, gdUnit4 beside it or not;
// coverage stays the gdUnit4 lane, since GUT measures nothing here.
func TestTheGutVariantRunsGutSuite(t *testing.T) {
	for _, stacks := range [][]string{{"gdscript", "gut"}, {"gdscript", "gut", "godot"}} {
		eff, err := Resolve(Config{Stacks: map[string]map[string]Override{}}, presetsFor(t), detect.Facts{Stacks: stacks})
		if err != nil {
			t.Fatal(err)
		}
		test := eff.Stacks["gdscript"]["test"].Lane
		if !slices.Equal(test.Commands, []string{"{loomux} check gut-suite --godot {godot}"}) || !slices.Equal(test.WhenFound, []string{`plugin.cfg:name="Gut"`}) || !test.Lock || test.Measuring != "" {
			t.Errorf("%v: test %+v", stacks, test)
		}
		cov := eff.Stacks["gdscript"]["coverage"].Lane
		if !slices.Equal(cov.Commands, []string{"{loomux} check linecov --report {coverdata}"}) {
			t.Errorf("%v: coverage %+v", stacks, cov)
		}
	}
}
