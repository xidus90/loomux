package repomap

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
)

// Hub is a prominent symbol within a directory or repository.
type Hub struct {
	Name     string     `json:"name"`
	Kind     model.Kind `json:"kind"`
	Path     string     `json:"path"`
	Span     model.Span `json:"span"`
	InDegree int        `json:"in_degree"`
}

// DirEntry represents a top-level or refined directory cluster.
type DirEntry struct {
	Path      string   `json:"path"`
	Files     int      `json:"files"`
	Symbols   int      `json:"symbols"`
	Languages []string `json:"languages"`
	Hubs      []Hub    `json:"hubs"`
	IsFile    bool     `json:"is_file,omitempty"`
}

// Totals contains repository-wide aggregate statistics.
type Totals struct {
	Files     int      `json:"files"`
	Symbols   int      `json:"symbols"`
	Edges     int      `json:"edges"`
	Languages []string `json:"languages"`
}

// RepoMap is the token-budgeted orientation map of the codebase.
type RepoMap struct {
	Totals   Totals     `json:"totals"`
	Dirs     []DirEntry `json:"dirs"`
	Hotspots []Hub      `json:"hotspots"`
	Dropped  int        `json:"dropped"`
}

// The limits Build applies when an Options field is not positive (map.ts).
const (
	DefaultMaxDirs    = 16
	DefaultHubsPerDir = 3
	DefaultHotspots   = 12
)

// Options configure the repo map calculation limits.
type Options struct {
	MaxDirs    int // DefaultMaxDirs when not positive
	HubsPerDir int // DefaultHubsPerDir when not positive
	Hotspots   int // DefaultHotspots when not positive
}

func langFromPath(p string) string {
	ext := strings.ToLower(path.Ext(p))
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".py":
		return "python"
	default:
		clean := strings.TrimPrefix(ext, ".")
		if clean == "" {
			return "unknown"
		}
		return clean
	}
}

// Build generates a RepoMap from graph g and index x.
//
// Refines monolith directories taking > 60% of files one segment deeper.
// Ranks hubs and hotspots by inDegree descending, ties by name then path.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/graph/map.ts.
func Build(g *model.Graph, x *blast.Index, opts Options) RepoMap {
	if g == nil {
		return RepoMap{}
	}
	if x == nil {
		x = blast.New(g)
	}

	maxDirs := opts.MaxDirs
	if maxDirs <= 0 {
		maxDirs = DefaultMaxDirs
	}
	hubsPerDir := opts.HubsPerDir
	if hubsPerDir <= 0 {
		hubsPerDir = DefaultHubsPerDir
	}
	maxHotspots := opts.Hotspots
	if maxHotspots <= 0 {
		maxHotspots = DefaultHotspots
	}

	res := RepoMap{}
	res.Totals.Edges = len(g.Edges)

	fileLangs := make(map[string]bool)
	var filePaths []string

	for i := range g.Nodes {
		n := &g.Nodes[i]
		if n.Kind == model.KindFile {
			res.Totals.Files++
			filePaths = append(filePaths, n.Path)
			lang := langFromPath(n.Path)
			fileLangs[lang] = true
		} else {
			res.Totals.Symbols++
			deg := x.InDegree(n.ID)
			if deg > 0 {
				res.Hotspots = append(res.Hotspots, Hub{
					Name:     n.Name,
					Kind:     n.Kind,
					Path:     n.Path,
					Span:     n.Span,
					InDegree: deg,
				})
			}
		}
	}

	for l := range fileLangs {
		res.Totals.Languages = append(res.Totals.Languages, l)
	}
	sort.Strings(res.Totals.Languages)

	// Sort hotspots: inDegree desc, name asc, path asc
	sort.SliceStable(res.Hotspots, func(i, j int) bool {
		if res.Hotspots[i].InDegree != res.Hotspots[j].InDegree {
			return res.Hotspots[i].InDegree > res.Hotspots[j].InDegree
		}
		if res.Hotspots[i].Name != res.Hotspots[j].Name {
			return res.Hotspots[i].Name < res.Hotspots[j].Name
		}
		return res.Hotspots[i].Path < res.Hotspots[j].Path
	})
	if len(res.Hotspots) > maxHotspots {
		res.Hotspots = res.Hotspots[:maxHotspots]
	}

	// 1st segment counts for monolith refinement
	topCounts := make(map[string]int)
	for _, p := range filePaths {
		normP := strings.ReplaceAll(p, `\`, "/")
		parts := strings.Split(normP, "/")
		if len(parts) > 1 {
			topCounts[parts[0]]++
		}
	}

	splitDirs := make(map[string]bool)
	if res.Totals.Files > 0 {
		threshold := 0.6 * float64(res.Totals.Files)
		for top, count := range topCounts {
			if float64(count) > threshold {
				splitDirs[top] = true
			}
		}
	}

	fileToBucket := make(map[string]string)
	bucketIsFile := make(map[string]bool)

	for _, p := range filePaths {
		normP := strings.ReplaceAll(p, `\`, "/")
		parts := strings.Split(normP, "/")
		if len(parts) == 1 {
			fileToBucket[p] = parts[0]
			bucketIsFile[parts[0]] = true
		} else if splitDirs[parts[0]] {
			if len(parts) == 2 {
				fileToBucket[p] = parts[0]
			} else {
				bucket := parts[0] + "/" + parts[1]
				fileToBucket[p] = bucket
			}
		} else {
			fileToBucket[p] = parts[0]
		}
	}

	type dirInfo struct {
		path      string
		isFile    bool
		files     int
		symbols   int
		languages map[string]bool
		hubs      []Hub
	}
	dirMap := make(map[string]*dirInfo)

	for _, p := range filePaths {
		b := fileToBucket[p]
		info := dirMap[b]
		if info == nil {
			info = &dirInfo{
				path:      b,
				isFile:    bucketIsFile[b],
				languages: make(map[string]bool),
			}
			dirMap[b] = info
		}
		info.files++
		info.languages[langFromPath(p)] = true
	}

	// Associate symbols with buckets
	for i := range g.Nodes {
		n := &g.Nodes[i]
		if n.Kind == model.KindFile {
			continue
		}
		b := fileToBucket[n.Path]
		info := dirMap[b]
		if info == nil {
			continue
		}
		info.symbols++
		deg := x.InDegree(n.ID)
		if deg > 0 {
			info.hubs = append(info.hubs, Hub{
				Name:     n.Name,
				Kind:     n.Kind,
				Path:     n.Path,
				Span:     n.Span,
				InDegree: deg,
			})
		}
	}

	var allDirs []DirEntry
	for _, info := range dirMap {
		var langs []string
		for l := range info.languages {
			langs = append(langs, l)
		}
		sort.Strings(langs)

		sort.SliceStable(info.hubs, func(i, j int) bool {
			if info.hubs[i].InDegree != info.hubs[j].InDegree {
				return info.hubs[i].InDegree > info.hubs[j].InDegree
			}
			if info.hubs[i].Name != info.hubs[j].Name {
				return info.hubs[i].Name < info.hubs[j].Name
			}
			return info.hubs[i].Path < info.hubs[j].Path
		})
		if len(info.hubs) > hubsPerDir {
			info.hubs = info.hubs[:hubsPerDir]
		}

		allDirs = append(allDirs, DirEntry{
			Path:      info.path,
			Files:     info.files,
			Symbols:   info.symbols,
			Languages: langs,
			Hubs:      info.hubs,
			IsFile:    info.isFile,
		})
	}

	if len(allDirs) > maxDirs {
		// Sort by files desc to pick top maxDirs
		sort.SliceStable(allDirs, func(i, j int) bool {
			if allDirs[i].Files != allDirs[j].Files {
				return allDirs[i].Files > allDirs[j].Files
			}
			if allDirs[i].Symbols != allDirs[j].Symbols {
				return allDirs[i].Symbols > allDirs[j].Symbols
			}
			return allDirs[i].Path < allDirs[j].Path
		})
		res.Dropped = len(allDirs) - maxDirs
		allDirs = allDirs[:maxDirs]
	}

	// Final sort: alphabetical by Path
	sort.SliceStable(allDirs, func(i, j int) bool {
		return allDirs[i].Path < allDirs[j].Path
	})
	res.Dirs = allDirs

	return res
}

// Format renders a RepoMap into a concise human-readable orientation overview.
func Format(m RepoMap) string {
	var b strings.Builder
	langs := strings.Join(m.Totals.Languages, ", ")
	if langs != "" {
		langs = " [" + langs + "]"
	}
	fmt.Fprintf(&b, "Repo Map: %d files, %d symbols, %d edges%s\n",
		m.Totals.Files, m.Totals.Symbols, m.Totals.Edges, langs)

	if len(m.Hotspots) > 0 {
		b.WriteString("\nHotspots:\n")
		for _, h := range m.Hotspots {
			fmt.Fprintf(&b, "  - %s (%s, inDegree=%d) in %s:%s\n",
				h.Name, h.Kind, h.InDegree, h.Path, h.Span)
		}
	}

	if len(m.Dirs) > 0 {
		b.WriteString("\nDirectories:\n")
		for _, d := range m.Dirs {
			typeLabel := "dir"
			if d.IsFile {
				typeLabel = "file"
			}
			fmt.Fprintf(&b, "  %s (%s: %d files, %d symbols)\n",
				d.Path, typeLabel, d.Files, d.Symbols)
			if len(d.Hubs) > 0 {
				var hubParts []string
				for _, h := range d.Hubs {
					hubParts = append(hubParts, fmt.Sprintf("%s (%d)", h.Name, h.InDegree))
				}
				fmt.Fprintf(&b, "    hubs: %s\n", strings.Join(hubParts, ", "))
			}
		}
	}

	if m.Dropped > 0 {
		fmt.Fprintf(&b, "  ... and %d more directories dropped\n", m.Dropped)
	}

	return b.String()
}
