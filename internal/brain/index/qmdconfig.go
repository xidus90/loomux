package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	backupSuffix = ".brain-backup"
	tempSuffix   = ".brain-tmp"
)

// CollectionSpec defines the configuration of a single qmd collection.
type CollectionSpec struct {
	Path    string
	Pattern string
	Ignore  []string
}

// OwnershipRecord is where the list of collections this program made is read
// from and written to. The two differ while the reference's state directory
// is the fallback: its qmd-collections.json names the collections qmd already
// carries, and a run that read only the new place would take every one of
// them for someone else's and refuse it. Writing goes to the new place alone.
type OwnershipRecord struct {
	Read  string
	Write string
}

// SyncOutcome reports the collection names that were updated and those refused.
type SyncOutcome struct {
	Changed []string
	Refused []string
}

// keptSpelling preserves the user's path format (e.g. backslashes on Windows)
// if it already resolves to the same directory, preventing unnecessary diffs.
func keptSpelling(present any, wanted string) string {
	if p, ok := present.(string); ok {
		if filepath.Clean(p) == filepath.Clean(wanted) {
			return p
		}
	}
	return filepath.ToSlash(filepath.Clean(wanted))
}

// entry overlays our managed keys on whatever keys the entry already carried.
func (s CollectionSpec) entry(present any) map[string]any {
	merged := make(map[string]any)
	if m, ok := present.(map[string]any); ok {
		for k, v := range m {
			merged[k] = v
		}
	}
	merged["path"] = keptSpelling(merged["path"], s.Path)
	merged["pattern"] = s.Pattern

	// Deduplicate ignore patterns while preserving order
	seen := make(map[string]bool)
	var uniqueIgnore []string
	for _, pattern := range s.Ignore {
		if !seen[pattern] {
			seen[pattern] = true
			uniqueIgnore = append(uniqueIgnore, pattern)
		}
	}
	if uniqueIgnore == nil {
		uniqueIgnore = []string{}
	}
	merged["ignore"] = uniqueIgnore
	return merged
}

func toStringSlice(val any) []string {
	switch v := val.(type) {
	case []string:
		return v
	case []any:
		res := make([]string, len(v))
		for i, item := range v {
			if s, ok := item.(string); ok {
				res[i] = s
			} else {
				res[i] = fmt.Sprint(item)
			}
		}
		return res
	default:
		return nil
	}
}

func sliceStringEqual(a, b any) bool {
	sa := toStringSlice(a)
	sb := toStringSlice(b)
	if len(sa) != len(sb) {
		return false
	}
	for i := range sa {
		if sa[i] != sb[i] {
			return false
		}
	}
	return true
}

func entriesEqual(existing any, next map[string]any) bool {
	em, ok := existing.(map[string]any)
	if !ok {
		return false
	}
	if len(em) != len(next) {
		return false
	}
	for k, nextVal := range next {
		existVal, found := em[k]
		if !found {
			return false
		}
		if k == "ignore" {
			if !sliceStringEqual(existVal, nextVal) {
				return false
			}
		} else if !reflect.DeepEqual(existVal, nextVal) {
			return false
		}
	}
	return true
}

var userHomeDir = os.UserHomeDir

// QmdConfigPath returns the path to qmd's index.yml, respecting XDG on all platforms.
func QmdConfigPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := userHomeDir()
		if err != nil {
			home = ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "qmd", "index.yml")
}

func loadDocument(configPath, raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return make(map[string]any), nil
	}
	var rawDoc any
	if err := yaml.Unmarshal([]byte(raw), &rawDoc); err != nil {
		return nil, fmt.Errorf("%s: not valid YAML: %w", configPath, err)
	}
	switch m := rawDoc.(type) {
	case map[string]any:
		return m, nil
	case map[any]any:
		converted := make(map[string]any, len(m))
		for k, v := range m {
			converted[fmt.Sprint(k)] = v
		}
		return converted, nil
	default:
		// Replace unexpected root shapes (e.g. lists) rather than aborting reindex
		return make(map[string]any), nil
	}
}

func readOwned(recordPath string) (map[string]bool, error) {
	data, err := os.ReadFile(recordPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]bool), nil
		}
		return nil, fmt.Errorf("%s: unreadable (%w)", recordPath, err)
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: unreadable (%w)", recordPath, err)
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s: expected a list of collection names", recordPath)
	}
	res := make(map[string]bool, len(list))
	for _, item := range list {
		res[fmt.Sprint(item)] = true
	}
	return res, nil
}

func remember(recordPath string, names map[string]bool) error {
	sortedNames := make([]string, 0, len(names))
	for name := range names {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)
	data, _ := json.MarshalIndent(sortedNames, "", " ")
	return replaceWith(recordPath, append(data, '\n'))
}

func replaceWith(destination string, data []byte) error {
	dir := filepath.Dir(destination)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	base := filepath.Base(destination)
	tempPath := filepath.Join(dir, base+tempSuffix)
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tempPath, destination); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}

// SyncCollections updates the collection entries in qmd's index.yml.
// Unowned collections already present in index.yml are refused to avoid accidental overwrite.
func SyncCollections(configPath string, wanted map[string]CollectionSpec, owned OwnershipRecord) (SyncOutcome, error) {
	var raw string
	data, err := os.ReadFile(configPath)
	if err == nil {
		raw = string(data)
	} else if !os.IsNotExist(err) {
		return SyncOutcome{}, err
	}

	doc, err := loadDocument(configPath, raw)
	if err != nil {
		return SyncOutcome{}, err
	}

	colsRaw := doc["collections"]
	collections := make(map[string]any)
	if m, ok := colsRaw.(map[string]any); ok {
		for k, v := range m {
			collections[k] = v
		}
	}

	ours, err := readOwned(owned.Read)
	if err != nil {
		return SyncOutcome{}, err
	}

	var refused []string
	for name := range wanted {
		if _, inCols := collections[name]; inCols && !ours[name] {
			refused = append(refused, name)
		}
	}
	sort.Strings(refused)
	refusedSet := make(map[string]bool, len(refused))
	for _, r := range refused {
		refusedSet[r] = true
	}

	mine := make(map[string]CollectionSpec)
	for name, spec := range wanted {
		if !refusedSet[name] {
			mine[name] = spec
		}
	}

	entries := make(map[string]map[string]any, len(mine))
	var changed []string
	for name, spec := range mine {
		entry := spec.entry(collections[name])
		entries[name] = entry
		if !entriesEqual(collections[name], entry) {
			changed = append(changed, name)
		}
	}
	sort.Strings(changed)

	if len(changed) == 0 {
		return SyncOutcome{Changed: nil, Refused: refused}, nil
	}

	// Preserve comments and formatting of the initial configuration by keeping a single backup
	backupPath := configPath + backupSuffix
	if raw != "" {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			if err := replaceWith(backupPath, []byte(raw)); err != nil {
				return SyncOutcome{}, err
			}
		}
	}

	for _, name := range changed {
		collections[name] = entries[name]
	}
	doc["collections"] = collections

	yamlData, _ := yaml.Marshal(doc)

	if err := replaceWith(configPath, yamlData); err != nil {
		return SyncOutcome{}, err
	}

	newOwned := make(map[string]bool, len(ours)+len(mine))
	for k, v := range ours {
		newOwned[k] = v
	}
	for k := range mine {
		newOwned[k] = true
	}
	if err := remember(owned.Write, newOwned); err != nil {
		return SyncOutcome{}, err
	}

	return SyncOutcome{Changed: changed, Refused: refused}, nil
}

// PruneCollections removes collections created by ultra-brain that are no longer registered.
func PruneCollections(configPath string, keep []string, owned OwnershipRecord) ([]string, error) {
	ours, err := readOwned(owned.Read)
	if err != nil {
		return nil, err
	}
	keepSet := make(map[string]bool, len(keep))
	for _, k := range keep {
		keepSet[k] = true
	}
	var orphaned []string
	for name := range ours {
		if !keepSet[name] {
			orphaned = append(orphaned, name)
		}
	}
	sort.Strings(orphaned)
	if len(orphaned) == 0 {
		return nil, nil
	}

	removed, err := DropCollections(configPath, orphaned)
	if err != nil {
		return nil, err
	}

	for _, r := range removed {
		delete(ours, r)
	}
	if err := remember(owned.Write, ours); err != nil {
		return nil, err
	}

	return removed, nil
}

// DropCollections removes specified collection names from qmd's index.yml.
func DropCollections(configPath string, names []string) ([]string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	doc, err := loadDocument(configPath, string(data))
	if err != nil {
		return nil, err
	}
	colsRaw, ok := doc["collections"]
	if !ok {
		return nil, nil
	}
	collections, ok := colsRaw.(map[string]any)
	if !ok {
		return nil, nil
	}

	var removed []string
	for _, name := range names {
		if _, exists := collections[name]; exists {
			removed = append(removed, name)
			delete(collections, name)
		}
	}
	sort.Strings(removed)
	if len(removed) == 0 {
		return nil, nil
	}

	doc["collections"] = collections
	yamlData, _ := yaml.Marshal(doc)
	if err := replaceWith(configPath, yamlData); err != nil {
		return nil, err
	}
	return removed, nil
}

var modelKeys = []struct {
	ours   string
	theirs string
}{
	{"embedding", "embed"},
	{"query_expansion", "generate"},
	{"rerank", "rerank"},
}

// Models extracts the model identifiers configured under qmd's models block.
func Models(configPath string) map[string]string {
	result := map[string]string{
		"embedding":       "unknown",
		"query_expansion": "unknown",
		"rerank":          "unknown",
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return result
	}
	doc, err := loadDocument(configPath, string(data))
	if err != nil {
		return result
	}
	modelsRaw, ok := doc["models"]
	if !ok {
		return result
	}
	modelsMap, ok := modelsRaw.(map[string]any)
	if !ok {
		return result
	}
	for _, pair := range modelKeys {
		if val, ok := modelsMap[pair.theirs]; ok {
			if s, ok := val.(string); ok && s != "" {
				result[pair.ours] = s
			}
		}
	}
	return result
}
