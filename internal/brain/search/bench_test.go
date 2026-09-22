package search_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
)

// The three benchmarks below split what `loomux brain search` spends before
// and after the engine answers: the registry and every area's manifest, the
// identity registers of every visible area, and the whole of ExecuteSearch
// with the engine replaced. The spec asks for the register share on its own,
// because it grows with the areas a machine registers and not with the
// query. They read the real state directories and never write them; point
// LOOMUX_BENCH_REGISTRY at loomux's and LOOMUX_BENCH_LEGACY at ultra-brain's,
// and they skip themselves when nobody did.
func benchDirs(b *testing.B) (string, string) {
	registryDir := os.Getenv("LOOMUX_BENCH_REGISTRY")
	legacyDir := os.Getenv("LOOMUX_BENCH_LEGACY")
	if registryDir == "" || legacyDir == "" {
		b.Skip("set LOOMUX_BENCH_REGISTRY to loomux's state directory and LOOMUX_BENCH_LEGACY to ultra-brain's")
	}
	return registryDir, legacyDir
}

func BenchmarkVisibleAreasOfTheRealRegistry(b *testing.B) {
	registryDir, legacyDir := benchDirs(b)
	// A registry that fails would measure the error path, which returns
	// after the first broken area and says nothing about the others.
	if _, err := privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelLocal); err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		_, _ = privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelLocal)
	}
}

func BenchmarkRegistersOfTheRealRegistry(b *testing.B) {
	registryDir, legacyDir := benchDirs(b)
	areas, err := privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelLocal)
	if err != nil {
		b.Fatal(err)
	}
	paths := make([]string, len(areas))
	for i, visible := range areas {
		paths[i] = filepath.Join(config.ResolvedAreaDir(visible.Area, registryDir, legacyDir), "_identities.tsv")
	}
	for _, path := range paths {
		if _, err := identity.ReadIdentities(path); err != nil {
			b.Fatal(err)
		}
	}
	for b.Loop() {
		for _, path := range paths {
			_, _ = identity.ReadIdentities(path)
		}
	}
	// After the loop: b.Loop() resets the timer on its first call, and a reset
	// deletes every metric reported before it.
	b.ReportMetric(float64(len(paths)), "registers")
}

func BenchmarkExecuteSearchWithoutTheEngine(b *testing.B) {
	registryDir, legacyDir := benchDirs(b)
	now := time.Now()
	// One hit from an area the real registry holds, so the register lookup
	// runs once as it does on a real answer; the engine itself is scripted.
	hit := search.SearchHit{
		Collection: search.CollectionName("engineering/python"),
		Relative:   "index.md",
		Line:       1,
		Title:      "engineering/python",
		Score:      0.5,
	}
	answer := func() error {
		port := search.NewFakePort()
		port.Results = []search.ScriptedSearch{{Hits: []search.SearchHit{hit}}}
		_, err := search.ExecuteSearch("latenz", "all", search.ProfileFast, 5,
			privacy.ChannelLocal, port, registryDir, legacyDir, now)
		return err
	}
	if err := answer(); err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		_ = answer()
	}
}
