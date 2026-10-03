package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/schema"
)

// readManifest is a variable so that a test can make the read fail.
var readManifest = os.ReadFile

// manifestNames are the files this command reports on: loomux's one name,
// then ultra-brain's two in the order its reader asked them.
var manifestNames = append([]string{filepath.Join(".loomux", "config.toml")}, config.OldManifestNames()...)

// chosenManifest is the name a reader that still knew the old names took:
// the first regular file that declares an area. A `.loomux/config.toml`
// without [area] gives way to the next name; an old name without [area], or
// any name that does not read, chooses none. loomux reads only the first
// name; this line tells which file carried the declaration before.
func chosenManifest(root string) string {
	for _, name := range manifestNames {
		path := filepath.Join(root, name)
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			continue
		}
		_, err := config.ReadDeclaration(path)
		if errors.Is(err, config.ErrNoArea) && name == manifestNames[0] {
			continue
		}
		if err != nil {
			return ""
		}
		return name
	}
	return ""
}

const areaCheckUsage = "usage: loomux area check <path>"

// areaCheck is `loomux area check`: for the manifests of one area root, what
// the readers do with each key. It reads and writes nothing else.
func areaCheck(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, areaCheckUsage)
		return 2
	}
	root := args[0]
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		fmt.Fprintln(stderr, areaCheckUsage)
		return 2
	}
	chosen := chosenManifest(root)
	needsHand := false
	found := false
	for _, name := range manifestNames {
		path := filepath.Join(root, name)
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			continue
		}
		found = true
		if checkOneManifest(stdout, root, name, name == manifestNames[0]) {
			needsHand = true
		}
		if chosen != "" && name != chosen && fileDeclaresArea(path) {
			fmt.Fprintf(stdout, "%s: shadowed\n", name)
		}
	}
	if chosen == "" {
		fmt.Fprintln(stdout, "chosen: none")
	} else {
		fmt.Fprintf(stdout, "chosen: %s\n", chosen)
	}
	if !found || needsHand {
		return 1
	}
	return 0
}

// fileDeclaresArea says whether the file carries an [area] table.
func fileDeclaresArea(path string) bool {
	_, err := config.ReadDeclaration(path)
	return !errors.Is(err, config.ErrNoArea)
}

// checkOneManifest prints the keys of one manifest and says whether it needs
// a hand: a key no reader reads, or a manifest a reader refuses.
func checkOneManifest(stdout io.Writer, root, name string, policyOnlyIsFine bool) bool {
	path := filepath.Join(root, name)
	data, err := readManifest(path)
	if err != nil {
		fmt.Fprintf(stdout, "%s: refused: %v\n", name, err)
		return true
	}
	document := map[string]any{}
	if err := toml.Unmarshal(data, &document); err != nil {
		fmt.Fprintf(stdout, "%s: refused: not valid TOML: %v\n", name, err)
		return true
	}
	needsHand := false
	for _, report := range classifyKeys(document) {
		fmt.Fprintf(stdout, "%s: %s  %s  %s\n", name, report.ID, report.Class, report.Hint)
		if report.Class == classIgnored {
			needsHand = true
		}
	}
	if _, err := config.ReadDeclaration(path); errors.Is(err, config.ErrNoArea) && policyOnlyIsFine {
		fmt.Fprintf(stdout, "%s: no [area], policy only\n", name)
	} else if err != nil {
		fmt.Fprintf(stdout, "%s: refused: %v\n", name, err)
		needsHand = true
	}
	return needsHand
}

// keyClass says what the readers do with one key of an area declaration.
type keyClass string

const (
	// classRead: config.ReadDeclaration reads the key.
	classRead keyClass = "read"
	// classElsewhere: another reader of the same file reads it (`[commit]`,
	// `[verify]`), so moving the file loses nothing.
	classElsewhere keyClass = "elsewhere"
	// classIgnored: no reader reads it, and a move by hand would drop it
	// without a word.
	classIgnored keyClass = "ignored"
)

// keyReport is one key of a manifest and what becomes of it.
type keyReport struct {
	ID    string
	Class keyClass
	Hint  string
}

// legacyHints name where an old key belongs in today's schema. A hint only:
// nothing here translates a value.
var legacyHints = map[string]string{
	"llm.local":                    "the local model is [model] enabled and roles",
	"layout.sources":               "written by `area add`, read by no reader",
	"check.lanes":                  "read by nothing; lanes live under [verify]",
	"area.wiki":                    "written by `area add`, read by no reader; the wiki place is [layout] wiki",
	"area.readonly":                "a registry property, not a manifest key; the registry flag readonly carries it",
	"maintenance.merge_branch":     "written by ultra-brain, read by no reader; loomux reads [maintenance] branch",
	"maintenance.watch":            "named in ultra-brain's spec, read by no reader; loomux has no counterpart",
	"maintenance.stale_after_days": "named in ultra-brain's spec, read by no reader; loomux has no counterpart",
}

// flattenKeys lists the keys of a decoded manifest as dotted IDs, sorted:
// `section.key` for a table's entries, the table itself for anything deeper
// or a list of tables, the bare name for a top-level value.
func flattenKeys(document map[string]any) []string {
	var ids []string
	for name, value := range document {
		table, isTable := value.(map[string]any)
		if !isTable || len(table) == 0 {
			ids = append(ids, name)
			continue
		}
		for key := range table {
			ids = append(ids, name+"."+key)
		}
	}
	slices.Sort(ids)
	return ids
}

// classifyKeys sorts every key of a manifest into read, elsewhere or ignored,
// with the tables the readers already name: config.DeclarationKeys for the
// declaration and the schema for the rest of `.loomux/config.toml`.
//
// The depth is two, as flattenKeys cuts it: a key below a table the schema
// names (`policy.paths.bogus`, `agent.models.foo.provider`) reads as that
// table (`policy.paths`, `agent.models`) and is elsewhere. Those deeper keys
// are checked by schema.Validate, not here; a deeper pass would be a second
// reading of the same file.
func classifyKeys(document map[string]any) []keyReport {
	read := map[string]bool{}
	for section, keys := range config.DeclarationKeys() {
		for _, key := range keys {
			read[section+"."+key] = true
		}
	}
	var reports []keyReport
	for _, id := range flattenKeys(document) {
		switch {
		case read[id]:
			reports = append(reports, keyReport{ID: id, Class: classRead})
		case schemaKnows(id):
			reports = append(reports, keyReport{ID: id, Class: classElsewhere})
		default:
			reports = append(reports, keyReport{ID: id, Class: classIgnored, Hint: legacyHints[id]})
		}
	}
	return reports
}

// schemaKnows says whether some key of the schema is id or lies below it, or
// whether a reader outside the schema reads it: internal/verify decodes every
// `[verify.<stack>.<kind>]` table and `[verify.gdscript] import_check`, which
// the schema does not list by name.
//
// The boundary: every `verify.*` counts as elsewhere, although parseVerify in
// internal/verify refuses an unknown stack, so `[verify.bogus]` is refused by
// the real reader and reads as elsewhere here. Like the depth of two, that
// check belongs to the reader, not to a second reading in this command.
func schemaKnows(id string) bool {
	if strings.HasPrefix(id, "verify.") {
		return true
	}
	if _, ok := schema.Lookup(id); ok {
		return true
	}
	for _, key := range schema.Keys() {
		if strings.HasPrefix(key.ID(), id+".") {
			return true
		}
	}
	return false
}
