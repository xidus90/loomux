package index

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

func readRegistry(registryPath string) ([]config.Area, error) {
	dir := registryPath
	if filepath.Base(registryPath) == "registry.toml" {
		dir = filepath.Dir(registryPath)
	}
	return config.ReadRegistry(dir)
}

func nestedAreas(area config.Area, areas []config.Area) []string {
	here := filepath.Clean(area.Path)
	var res []string
	for _, other := range areas {
		if other.Scope == area.Scope {
			continue
		}
		for _, root := range []string{other.Path, other.WikiPath} {
			if root == "" {
				continue
			}
			cleanRoot := filepath.Clean(root)
			rel, err := filepath.Rel(here, cleanRoot)
			if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "." {
				res = append(res, cleanRoot)
			}
		}
	}
	return res
}

func nestedGlobs(area config.Area, areas []config.Area) []string {
	here := filepath.Clean(area.Path)
	var globs []string
	for _, child := range nestedAreas(area, areas) {
		rel, _ := filepath.Rel(here, child)
		globs = append(globs, filepath.ToSlash(rel)+"/**")
	}
	return globs
}

func toGraphDocs(documents []Document) []graph.Document {
	res := make([]graph.Document, len(documents))
	for i, d := range documents {
		res[i] = graph.Document{
			Relative: d.Relative,
			Title:    d.Title,
			Tags:     d.Tags,
			Links:    d.Links,
		}
	}
	return res
}

type indexedArea struct {
	area     config.Area
	manifest *config.Manifest
}

// The seams of this package: the calls a test replaces to reach an arm the
// filesystem will not produce on demand. Three came with the port; the
// recovery and the swap were added here, and for the same reason -- both fail
// only when the operating system declines a rename, which no test can ask for
// on both platforms. The swap is the one a run may be killed in the middle
// of, so it is the one the guarantee of this package is about.
var (
	readDocFn          = ReadDocument
	contentHashFn      = identity.ContentHash
	pruneCollectionsFn = PruneCollections
	recoverStockFn     = recoverStock
	replaceDirFn       = lock.ReplaceDir
)

// Reindex rebuilds directory catalogs, the link graph, and the identity registry,
// and synchronizes configured collections with qmd.
//
// stateDir is the one place anything is read from and written to; it is an
// argument for the reason `internal/serve` gives: everything hangs off the
// state directory it was handed, and a lookup that asked StateDir() itself
// would break that promise.
func Reindex(registryPath, stateDir string, port search.SearchPort) (int, error) {
	return ReindexWithOutput(registryPath, stateDir, port, os.Stderr)
}

// ReindexWithOutput is Reindex writing progress and diagnostic messages to custom writer stderr.
func ReindexWithOutput(registryPath, stateDir string, port search.SearchPort, stderr io.Writer) (int, error) {
	if stderr == nil {
		stderr = io.Discard
	}
	if registryPath == "" {
		registryPath = filepath.Join(stateDir, "registry.toml")
	}

	areas, err := readRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1, err
	}

	var indexed []indexedArea
	uninspectable := false
	for _, area := range areas {
		item, skip, err := indexArea(area, areas, stateDir, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1, err
		}
		if skip == skipUninspectable {
			uninspectable = true
		}
		if skip != notSkipped {
			continue
		}
		indexed = append(indexed, item)
	}

	code, err := syncSearch(areas, indexed, config.ArtifactLookup{Primary: stateDir}, port, stderr)
	if err == nil && uninspectable {
		code = 1
	}
	return code, err
}

// skipReason tells why indexArea left an area out.
type skipReason int

const (
	notSkipped skipReason = iota
	// skipAbsent is a path or declaration that is not there; the run is
	// unaffected.
	skipAbsent
	// skipUninspectable is a path the system refused to inspect; the other
	// areas still run, but the run fails.
	skipUninspectable
)

// indexArea rebuilds one area's stock and reports whether and why it was
// skipped. An area whose path or declaration is missing is skipped rather
// than refused: the registry outlives the checkouts on a machine, and one
// absent clone would otherwise stop the run for every other area too. A path
// that cannot be inspected for another reason (access denied, a malformed
// name) is not absent: it is skipped the same way, so the other areas still
// run, but it is reported so that the run can fail.
func indexArea(
	area config.Area,
	areas []config.Area,
	stateDir string,
	stderr io.Writer,
) (indexedArea, skipReason, error) {
	if _, err := os.Stat(area.Path); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(stderr, "skipping %s: %s cannot be inspected: %v\n", area.Scope, area.Path, err)
			return indexedArea{}, skipUninspectable, nil
		}
		fmt.Fprintf(stderr, "skipping %s: %s does not exist\n", area.Scope, area.Path)
		return indexedArea{}, skipAbsent, nil
	}

	// Held from the register's reading in collect to its writing in publish:
	// an approve advancing it in between would otherwise lose its row. It
	// covers the recovery of a half-done swap too, as approve's does.
	release, err := config.LockArea(area, stateDir)
	if err != nil {
		return indexedArea{}, notSkipped, err
	}
	defer release()

	if err := recoverStockFn(area, stateDir); err != nil {
		return indexedArea{}, notSkipped, err
	}

	// The declaration is read where the stock lies: the area's tree, or for a
	// read-only area the state directory.
	source := config.ManifestDir(area, stateDir)
	manifest, err := config.ReadAreaDeclaration(source)
	if err != nil {
		fmt.Fprintf(stderr, "skipping %s: %v\n", area.Scope, err)
		return indexedArea{}, skipAbsent, nil
	}

	documents, identities, err := collect(area, areas, source, manifest)
	if err != nil {
		return indexedArea{}, notSkipped, err
	}
	if err := publish(area, documents, identities, source, stateDir); err != nil {
		return indexedArea{}, notSkipped, err
	}
	return indexedArea{area: area, manifest: manifest}, notSkipped, nil
}

// collect reads every file of the area and carries the identity register
// forward: a file that did not change keeps its number, a changed one counts
// up, and a renamed one is recognised by its digest instead of being reborn.
func collect(
	area config.Area,
	areas []config.Area,
	source string,
	manifest *config.Manifest,
) ([]Document, map[string]identity.Identity, error) {
	files, err := FindFiles(area, manifest, nestedAreas(area, areas))
	if err != nil {
		return nil, nil, err
	}

	documents := make([]Document, len(files))
	for i, f := range files {
		doc, err := readDocFn(f, area.Path)
		if err != nil {
			return nil, nil, err
		}
		documents[i] = *doc
	}

	// An unreadable register is an empty one: it is this run's output, and a
	// run that refused to start over it could never repair it.
	previous, _ := identity.ReadIdentities(filepath.Join(source, identitiesName))
	digests := make(map[string]string, len(files))
	for i, f := range files {
		h, err := contentHashFn(f)
		if err != nil {
			return nil, nil, err
		}
		digests[documents[i].Relative] = h
	}

	return documents, carryForward(documents, digests, previous), nil
}

// carryForward decides each document's identity from the previous register
// and this run's digests.
func carryForward(
	documents []Document,
	digests map[string]string,
	previous map[string]identity.Identity,
) map[string]identity.Identity {
	renamed := identity.MatchRenames(previous, digests)
	updated := make(map[string]identity.Identity, len(documents))
	for _, doc := range documents {
		digest := digests[doc.Relative]
		prev, exists := previous[doc.Relative]
		switch {
		case exists && prev.ContentHash == digest:
			updated[doc.Relative] = prev
		case exists:
			updated[doc.Relative] = identity.Identity{
				DocID:       prev.DocID,
				Relative:    doc.Relative,
				ContentHash: digest,
				Revision:    prev.Revision + 1,
			}
		default:
			if re, ok := renamed[doc.Relative]; ok {
				updated[doc.Relative] = re
				continue
			}
			updated[doc.Relative] = identity.Identity{
				DocID:       identity.NewDocID(),
				Relative:    doc.Relative,
				ContentHash: digest,
				Revision:    1,
			}
		}
	}
	return updated
}

// ownedCollectionsName is the record of the collections this program made, in
// the state directory.
const ownedCollectionsName = "qmd-collections.json"

// ownershipRecord names the record anew on every call, so that the prune
// after the first sync reads what that sync has just written.
func ownershipRecord(state config.ArtifactLookup) OwnershipRecord {
	return OwnershipRecord{Read: state.Resolve(ownedCollectionsName), Write: state.WritePath(ownedCollectionsName)}
}

// syncSearch tells qmd which collections exist, drops the ones no registered
// area claims any more, and refreshes the port.
func syncSearch(
	areas []config.Area,
	indexed []indexedArea,
	state config.ArtifactLookup,
	port search.SearchPort,
	stderr io.Writer,
) (int, error) {
	specs := make(map[string]CollectionSpec, len(indexed))
	for _, item := range indexed {
		specs[search.CollectionName(item.area.Scope)] = collectionSpec(item, areas)
	}

	qmdConfig := QmdConfigPath()
	outcome, err := SyncCollections(qmdConfig, specs, ownershipRecord(state))
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1, err
	}

	regScopes := make([]string, len(areas))
	for i, a := range areas {
		regScopes[i] = search.CollectionName(a.Scope)
	}
	dropped, err := pruneCollectionsFn(qmdConfig, regScopes, ownershipRecord(state))
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1, err
	}

	report(outcome, dropped, stderr)

	if port != nil {
		specNames := make([]string, 0, len(specs))
		for name := range specs {
			specNames = append(specNames, name)
		}
		sort.Strings(specNames)
		if err := port.Refresh(specNames); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1, err
		}
	}

	if len(outcome.Refused) > 0 {
		return 1, nil
	}
	return 0, nil
}

// collectionSpec is what one indexed area asks qmd to index of it.
func collectionSpec(item indexedArea, areas []config.Area) CollectionSpec {
	m := item.manifest
	pattern := "**/*.md"
	if len(m.IndexInclude) > 0 {
		pattern = m.IndexInclude[0]
	}
	var ignores []string
	ignores = append(ignores, m.NeverGlobs...)
	ignores = append(ignores, m.IndexExclude...)
	ignores = append(ignores, m.IndexUnsearched...)
	ignores = append(ignores, nestedGlobs(item.area, areas)...)
	ignores = append(ignores, ArtifactExcludes(item.area, false)...)
	ignores = append(ignores, AlwaysExcludes...)
	if revExcludes, err := privacy.ReviewExcludes(m); err == nil {
		ignores = append(ignores, revExcludes...)
	}
	return CollectionSpec{Path: item.area.Path, Pattern: pattern, Ignore: ignores}
}

// report says what the run changed in the search engine's configuration.
func report(outcome SyncOutcome, dropped []string, stderr io.Writer) {
	if len(outcome.Changed) > 0 {
		fmt.Fprintf(stderr, "updated qmd collections: %s\n", strings.Join(outcome.Changed, ", "))
	}
	if len(dropped) > 0 {
		fmt.Fprintf(stderr, "dropped qmd collections: %s\n", strings.Join(dropped, ", "))
	}
	for _, stranger := range outcome.Refused {
		fmt.Fprintf(
			stderr,
			"error: %s already exists in the search engine and was not created by brain; skipped -- rename the area or remove the collection yourself\n",
			stranger,
		)
	}
}

// Embed generates vector representations for all registered areas.
func Embed(registryPath, stateDir string, port search.SearchPort) (int, error) {
	return EmbedWithOutput(registryPath, stateDir, port, os.Stderr)
}

// EmbedWithOutput is Embed writing status and errors to custom writer stderr.
func EmbedWithOutput(registryPath, stateDir string, port search.SearchPort, stderr io.Writer) (int, error) {
	if stderr == nil {
		stderr = io.Discard
	}
	if registryPath == "" {
		registryPath = filepath.Join(stateDir, "registry.toml")
	}

	areas, err := readRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1, err
	}

	scopes := make([]string, len(areas))
	for i, a := range areas {
		scopes[i] = a.Scope
	}
	sort.Strings(scopes)

	if port != nil {
		if err := port.Embed(scopes); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1, err
		}
	}

	fmt.Fprintf(stderr, "embedded %d area(s)\n", len(scopes))
	return 0, nil
}
