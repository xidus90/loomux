package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
)

// benchRoot is this repository, whose graph the benchmarks read.
const benchRoot = "../.."

// benchFile is the file whose edit the benchmarks replay.
const benchFile = "internal/code/blast/reach.go"

// benchSources are the file as it is on disk, and with the body of Reach
// changed, so the walk and the formatting run as well.
func benchSources(b *testing.B) map[string]string {
	b.Helper()
	if _, err := os.Stat(store.WiringPath(benchRoot)); err != nil {
		b.Skip("no wiring.json in this checkout; run `loomux graph build` first")
	}
	src, err := os.ReadFile(filepath.Join(benchRoot, filepath.FromSlash(benchFile)))
	if err != nil {
		b.Fatal(err)
	}
	changed := strings.Replace(string(src), "var hits []Hit", "var hits []Hit // edited", 1)
	return map[string]string{"unchanged": string(src), "changed": changed}
}

// BenchmarkBlastAside is the monitor on this repository's graph: read and
// decode wiring.json, parse the edited file, compare, walk.
func BenchmarkBlastAside(b *testing.B) {
	for _, name := range []string{"unchanged", "changed"} {
		source := benchSources(b)[name]
		b.Run(name, func(b *testing.B) {
			aside := ""
			for b.Loop() {
				aside = blastAside(benchRoot, benchFile, fixed(source))
			}
			b.ReportMetric(float64(len(aside)), "aside-bytes")
		})
	}
}

// BenchmarkBlastAsideScaled is the monitor on this repository's graph copied
// k times, to show how the decode grows with the graph.
func BenchmarkBlastAsideScaled(b *testing.B) {
	sources := benchSources(b)
	g, err := store.Read(benchRoot)
	if err != nil {
		b.Fatal(err)
	}
	for _, k := range []int{1, 2, 5, 10, 20} {
		root := b.TempDir()
		if err := store.Write(root, scaled(g, k)); err != nil {
			b.Fatal(err)
		}
		info, err := os.Stat(store.WiringPath(root))
		if err != nil {
			b.Fatal(err)
		}
		rel := "copy0/" + benchFile
		source := sources["changed"]
		b.Run(fmt.Sprintf("k=%d", k), func(b *testing.B) {
			for b.Loop() {
				blastAside(root, rel, fixed(source))
			}
			b.ReportMetric(float64(info.Size())/(1<<20), "wiring-MiB")
		})
	}
}

// scaled is g k times over, every id and path under copyN/.
func scaled(g *model.Graph, k int) *model.Graph {
	out := &model.Graph{Meta: g.Meta}
	for c := range k {
		prefix := fmt.Sprintf("copy%d/", c)
		for _, n := range g.Nodes {
			n.ID = model.NodeID(prefix) + n.ID
			n.Path = prefix + n.Path
			out.Nodes = append(out.Nodes, n)
		}
		for _, e := range g.Edges {
			e.Source = model.NodeID(prefix) + e.Source
			e.Target = model.NodeID(prefix) + e.Target
			out.Edges = append(out.Edges, e)
		}
	}
	out.Meta.NodeCount = len(out.Nodes)
	out.Meta.EdgeCount = len(out.Edges)
	return out
}
