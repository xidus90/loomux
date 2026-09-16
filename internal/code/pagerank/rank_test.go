package pagerank_test

import (
	"math"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/pagerank"
)

func scoreOf(got []pagerank.Scored, id model.NodeID) float64 {
	for _, s := range got {
		if s.ID == id {
			return s.Score
		}
	}
	return 0
}

func has(got []pagerank.Scored, id model.NodeID) bool {
	for _, s := range got {
		if s.ID == id {
			return true
		}
	}
	return false
}

func TestRankConnectedSeedBeatsIsolatedSeed(t *testing.T) {
	g := graphOf(
		[]model.NodeID{"hub", "a", "b", "c", "lone"},
		[][3]string{{"hub", "a", ""}, {"hub", "b", ""}, {"hub", "c", ""}},
	)
	got := pagerank.Rank(pagerank.Prepare(g, nil), map[model.NodeID]float64{"hub": 1, "lone": 1}, pagerank.Options{})

	if scoreOf(got, "hub") <= scoreOf(got, "lone") {
		t.Errorf("connected seed %v must beat isolated seed %v", scoreOf(got, "hub"), scoreOf(got, "lone"))
	}
	if got[0].Score != 1 {
		t.Errorf("got top score %v, want 1 -- the output is max-normalized", got[0].Score)
	}
}

func TestRankNeighboursAccrueMassWithoutRestartWeight(t *testing.T) {
	g := graphOf([]model.NodeID{"hub", "a", "b"}, [][3]string{{"hub", "a", ""}, {"hub", "b", ""}})
	got := pagerank.Rank(pagerank.Prepare(g, nil), map[model.NodeID]float64{"hub": 1}, pagerank.Options{})

	if scoreOf(got, "a") <= 0 || scoreOf(got, "b") <= 0 {
		t.Errorf("a neighbour of the only seed gets walk mass, got a=%v b=%v", scoreOf(got, "a"), scoreOf(got, "b"))
	}
}

func TestRankEmptyOrZeroSeeds(t *testing.T) {
	g := graphOf([]model.NodeID{"a", "b"}, [][3]string{{"a", "b", ""}})
	topo := pagerank.Prepare(g, nil)

	if got := pagerank.Rank(topo, nil, pagerank.Options{}); got != nil {
		t.Errorf("got %v, want nil for no seeds", got)
	}
	if got := pagerank.Rank(topo, map[model.NodeID]float64{"a": 0}, pagerank.Options{}); got != nil {
		t.Errorf("got %v, want nil for an all-zero seed", got)
	}
	if got := pagerank.Rank(topo, map[model.NodeID]float64{"a": -3}, pagerank.Options{}); got != nil {
		t.Errorf("got %v, want nil for a negative seed", got)
	}
	if got := pagerank.Rank(topo, map[model.NodeID]float64{"ghost": 1}, pagerank.Options{}); got != nil {
		t.Errorf("got %v, want nil for a seed naming no node", got)
	}
}

func TestRankSeedWeightBeyondTheFloatRange(t *testing.T) {
	// An infinite weight makes the normalization Inf/Inf, and every rank a NaN.
	// NaN fails every comparison, so nothing would be filtered and the sort
	// order would be undefined -- the result must be nothing at all instead.
	g := graphOf([]model.NodeID{"a", "b"}, [][3]string{{"a", "b", ""}})
	got := pagerank.Rank(pagerank.Prepare(g, nil), map[model.NodeID]float64{"a": math.Inf(1)}, pagerank.Options{})

	if got != nil {
		t.Errorf("got %v, want nil -- no NaN score may reach a caller", got)
	}
}

func TestRankUnresolvedImportTargetNeverRanks(t *testing.T) {
	g := graphOf([]model.NodeID{"a"}, [][3]string{{"a", "npm:lodash", "imports"}})
	got := pagerank.Rank(pagerank.Prepare(g, nil), map[model.NodeID]float64{"a": 1}, pagerank.Options{})

	if scoreOf(got, "a") != 1 {
		t.Errorf("got %v, want 1 -- the sole seed without neighbours stays on top", scoreOf(got, "a"))
	}
	if has(got, "npm:lodash") {
		t.Error("a module string must never become a ranked node")
	}
}

// The reference values of the dangling fixture, from Graft's own test file and
// recomputed independently on 2026-09-16.
func TestRankDanglingMassFixture(t *testing.T) {
	g := graphOf(
		[]model.NodeID{"a", "b", "c", "d", "e"},
		[][3]string{{"a", "b", ""}, {"b", "c", ""}},
	)
	got := pagerank.Rank(pagerank.Prepare(g, nil), map[model.NodeID]float64{"a": 2, "d": 1}, pagerank.Options{})

	want := map[model.NodeID]float64{
		"a": 0.9577162737326514,
		"b": 1,
		"c": 0.37462976423958994,
		"d": 0.29154325474653076,
	}
	for id, w := range want {
		if diff := math.Abs(scoreOf(got, id) - w); diff > 1e-9 {
			t.Errorf("%s: got %v, want %v (diff %v)", id, scoreOf(got, id), w, diff)
		}
	}
	if got[0].ID != "b" {
		t.Errorf("got top node %q, want b -- the hub linking both chain ends", got[0].ID)
	}
	if has(got, "e") {
		t.Error("an unreached node must be absent from the result")
	}
}

func TestRankTieOrderIsAlphabetical(t *testing.T) {
	// Two seeds of equal weight in mirror-image positions score identically;
	// only the id can break the tie, and it must do so the same way every run.
	g := graphOf([]model.NodeID{"zeta", "alpha"}, nil)
	got := pagerank.Rank(pagerank.Prepare(g, nil), map[model.NodeID]float64{"zeta": 1, "alpha": 1}, pagerank.Options{})

	if len(got) != 2 || got[0].ID != "alpha" || got[1].ID != "zeta" {
		t.Errorf("got %v, want alpha before zeta at equal score", got)
	}
}

func TestRankOptionsOverrideTheDefaults(t *testing.T) {
	g := graphOf([]model.NodeID{"hub", "a"}, [][3]string{{"hub", "a", ""}})
	topo := pagerank.Prepare(g, nil)

	loose := pagerank.Rank(topo, map[model.NodeID]float64{"hub": 1}, pagerank.Options{Alpha: 0.9, Iterations: 1})
	tight := pagerank.Rank(topo, map[model.NodeID]float64{"hub": 1}, pagerank.Options{})
	if scoreOf(loose, "a") >= scoreOf(tight, "a") {
		t.Errorf("a higher restart probability starves the neighbour: got a=%v with alpha 0.9, a=%v with the default",
			scoreOf(loose, "a"), scoreOf(tight, "a"))
	}
}

func TestRankKeepFilterMatchesRankingTheComponentAlone(t *testing.T) {
	componentA := []model.NodeID{"hub", "a", "b", "c"}
	edgesA := [][3]string{{"hub", "a", ""}, {"hub", "b", ""}, {"hub", "c", ""}}
	full := graphOf(append(append([]model.NodeID{}, componentA...), "x", "y", "z"),
		append(append([][3]string{}, edgesA...), [3]string{"x", "y", ""}, [3]string{"y", "z", ""}))
	only := graphOf(componentA, edgesA)

	seeds := map[model.NodeID]float64{"hub": 1, "a": 0.5}
	inA := map[model.NodeID]bool{"hub": true, "a": true, "b": true, "c": true}
	filtered := pagerank.Rank(pagerank.Prepare(full, func(id model.NodeID) bool { return inA[id] }), seeds, pagerank.Options{})
	reference := pagerank.Rank(pagerank.Prepare(only, nil), seeds, pagerank.Options{})

	if len(filtered) != len(reference) {
		t.Fatalf("got %d scores, want %d", len(filtered), len(reference))
	}
	for i := range reference {
		if filtered[i] != reference[i] {
			t.Errorf("at %d: got %v, want %v", i, filtered[i], reference[i])
		}
	}
	if has(filtered, "x") {
		t.Error("a filtered-out component must not appear")
	}
}

func TestRankSeedsOutsideTheFilterAreIgnored(t *testing.T) {
	g := graphOf([]model.NodeID{"hub", "a", "outside"}, [][3]string{{"hub", "a", ""}})
	got := pagerank.Rank(
		pagerank.Prepare(g, func(id model.NodeID) bool { return id != "outside" }),
		map[model.NodeID]float64{"hub": 1, "outside": 5},
		pagerank.Options{},
	)

	if has(got, "outside") {
		t.Error("a filtered-out seed must never appear in the output")
	}
	if scoreOf(got, "hub") <= 0 {
		t.Error("the seed inside the filter is still ranked")
	}
}
