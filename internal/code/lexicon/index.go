package lexicon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
)

// indexVersion is the sidecar's shape. Another version is a shape this code
// cannot read, which is the same as no sidecar.
const indexVersion = 1

// Doc is one node's three token bags.
type Doc struct {
	ID   model.NodeID   `json:"id"`
	Name map[string]int `json:"name"`
	Path map[string]int `json:"path"`
	Body map[string]int `json:"body"`
}

// Index is the build-time sidecar.
//
// It exists because tokenizing every node's name, path and body on every query
// was ~45% of query time at 32k nodes, profiled. It is derived data and lives
// under cache/ next to the graph, not in it.
type Index struct {
	Version    int            `json:"version"`
	AvgBodyLen float64        `json:"avg_body_len"`
	DocCount   int            `json:"doc_count"`
	DF         map[string]int `json:"df"`
	Docs       []Doc          `json:"docs"`
}

// Path is where the sidecar lives.
func Path(root string) string { return store.CachePath(root, "ask-index.json") }

// Build tokenizes a whole graph.
//
// It must be handed the graph as the build holds it in memory, WITH BodyText.
// Reconstructing this from the written wiring.json is impossible: the body is
// stripped there, which is the point of stripping it.
func Build(g *model.Graph) *Index {
	ix := &Index{Version: indexVersion, DF: map[string]int{}}
	bodyTotal := 0
	for _, n := range g.Nodes {
		doc := Doc{
			ID:   n.ID,
			Name: Counts(Tokenize(n.Name)),
			Path: Counts(Tokenize(n.Path)),
			Body: Counts(Tokenize(n.BodyText)),
		}
		ix.Docs = append(ix.Docs, doc)
		for _, n := range doc.Body {
			bodyTotal += n
		}
		// A term counts once per document, however many of its fields hold it:
		// df is a document frequency, not a term frequency.
		for term := range terms(doc) {
			ix.DF[term]++
		}
	}
	ix.DocCount = len(ix.Docs)
	if ix.DocCount > 0 {
		ix.AvgBodyLen = float64(bodyTotal) / float64(ix.DocCount)
	}
	sortDocs(ix.Docs)
	return ix
}

// terms is the set of tokens a document holds, across its three fields.
func terms(d Doc) map[string]bool {
	out := map[string]bool{}
	for _, bag := range []map[string]int{d.Name, d.Path, d.Body} {
		for term := range bag {
			out[term] = true
		}
	}
	return out
}

// Where keeps the documents whose id the predicate admits, and recomputes
// the statistics over what is left.
//
// Recomputing is the whole difference between a filter and a post-filter: a
// word that is common across the repository and rare inside one subtree has to
// discriminate inside that subtree. The original pins this with a test of its
// own -- a term's rank flips relative to another between filtered and
// unfiltered. The same holds for an area's never globs: a filter applied after
// scoring would let a refused document shape df and take a place in the limit.
//
// The predicate gets the id and not a path, because a path cannot be cut out
// of an id: a file may be named "#gen.go" or sit under "dir#1/", and the first
// '#' then falls inside the path. A caller that judges by path looks the id up
// in its graph, where the node carries its Path; ask.Run does, for a subtree
// prefix and never globs alike.
func (ix *Index) Where(keep func(id model.NodeID) bool) *Index {
	out := &Index{Version: ix.Version, DF: map[string]int{}}
	bodyTotal := 0
	for _, d := range ix.Docs {
		if !keep(d.ID) {
			continue
		}
		out.Docs = append(out.Docs, d)
		for _, n := range d.Body {
			bodyTotal += n
		}
		for term := range terms(d) {
			out.DF[term]++
		}
	}
	out.DocCount = len(out.Docs)
	if out.DocCount > 0 {
		out.AvgBodyLen = float64(bodyTotal) / float64(out.DocCount)
	}
	return out
}

// NormalizePrefix brings a repo-relative prefix into the shape node ids carry:
// forward slashes, no leading or trailing separator. An all-separator prefix
// normalizes to the empty one, which names the whole repository. The form a
// shell completes is the form a caller passes: "lib/" and, on Windows, "lib\"
// name the same subtree as "lib".
//
// It is exported because the prefix is interpreted outside this package:
// ask.Run narrows both the index and the graph walk to the same subtree, and
// both have to reach the same verdict about what "lib/" means. Two
// normalizers that agree today are two normalizers.
func NormalizePrefix(prefix string) string {
	return strings.Trim(strings.ReplaceAll(prefix, `\`, "/"), "/")
}

// UnderPrefix reports whether a repo-relative path lies at or under prefix,
// comparing whole segments -- "widgets" never matches "widgets-extra". The
// prefix has been through NormalizePrefix; the path is slash-separated by
// model.Node's own invariant.
//
// Exported for the same reason as NormalizePrefix, and it is the other half of
// it: normalizing a prefix identically is worth nothing if the index and the
// graph walk then compare it differently. One of them would keep a subtree the
// other dropped, and a walk that keeps nothing is silent -- no error, only a
// missing graph axis.
func UnderPrefix(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

// Write serializes the sidecar. Maps are serialized by encoding/json in sorted
// key order, so two writes of one index are byte-identical.
//
// The body goes to a temporary file first and replaces the sidecar by rename,
// as store.Write does for the graph: a run that lost the rebuild lock reads
// the sidecar while the winner writes it, and a file rewritten in place would
// hand that reader half a document -- which it could only take for no sidecar
// and answer without the body text.
//
//coverage:exempt MarshalIndent's error arm needs a value json cannot encode (a channel, a func, a cyclic pointer, or a NaN/Inf float); an Index is an int, a float64 the arithmetic here keeps finite, a map[string]int and a slice of the same -- no Index this program builds makes it fail
func Write(root string, ix *Index) error {
	body, err := json.MarshalIndent(ix, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(Path(root)), 0o755); err != nil {
		return err
	}
	final := Path(root)
	tmp := final + "." + strconv.Itoa(os.Getpid()) + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, final); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// Usable reports whether the sidecar on disk is one this binary can read.
//
// This is the freshness probe's question and not a query's, so it must not
// cost what Read costs: it stops at the version field and never materializes a
// token bag. The version is the first field Write emits, so in practice this
// is three tokens and a few hundred bytes; the loop over the remaining keys is
// there so a reordered struct stays readable rather than counted as drift.
func Usable(root string) bool {
	f, err := os.Open(Path(root))
	if err != nil {
		return false
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return false
	}
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return false
		}
		if key != "version" {
			var skip json.RawMessage
			if err := dec.Decode(&skip); err != nil {
				return false
			}
			continue
		}
		var version int
		if err := dec.Decode(&version); err != nil {
			return false
		}
		return version == indexVersion
	}
	return false
}

// Read loads the sidecar.
func Read(root string) (*Index, error) {
	b, err := os.ReadFile(Path(root))
	if err != nil {
		return nil, fmt.Errorf("read ask index: %w", err)
	}
	var ix Index
	if err := json.Unmarshal(b, &ix); err != nil {
		return nil, fmt.Errorf("read ask index: %w", err)
	}
	if ix.Version != indexVersion {
		return nil, fmt.Errorf("ask index version %d, want %d", ix.Version, indexVersion)
	}
	return &ix, nil
}

// sortDocs keeps the document order deterministic, so a rebuilt sidecar is
// byte-identical to the one it replaces.
func sortDocs(docs []Doc) {
	sort.Slice(docs, func(i, j int) bool { return docs[i].ID < docs[j].ID })
}
