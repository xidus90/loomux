package benchsearch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// The rules of a corpus stand are few and blunt on purpose: fixed counts, a
// partition into themes, provenance for every file and a checksum per note.
// What they buy is that two numbers measured a year apart are numbers about
// the same thing.
const (
	notesWanted    = 100
	themesWanted   = 10
	notesPerTheme  = notesWanted / themesWanted
	reverseMinimum = 5
)

// An English question against a German source is told apart by counting
// function words across the whole question rather than by its first word:
// English questions begin with conditions, places and subclauses nobody
// foresaw, while function words appear in every sentence and the two
// languages barely collide on them. "in" and "is" are both languages, so they
// are in neither list. This stays a heuristic for a hand-written set.
func englishWords() map[string]bool {
	return wordSet("the", "of", "and", "to", "does", "what", "how", "when", "which", "with", "for", "that", "a", "an")
}

func germanWords() map[string]bool {
	return wordSet("der", "die", "das", "und", "von", "zu", "wie", "was", "wann", "welche", "mit", "für", "dass", "ein", "eine", "den", "dem")
}

func wordSet(words ...string) map[string]bool {
	set := make(map[string]bool, len(words))
	for _, w := range words {
		set[w] = true
	}
	return set
}

// Python's \w is Unicode, Go's is ASCII: the letter and number classes keep
// a note name like grüße.md whole.
var (
	named = regexp.MustCompile(`[\p{L}\p{N}_-]+\.md`)
	word  = regexp.MustCompile(`[a-zA-Zäöüß]+`)
)

type theme struct {
	name, neighbour string
	notes           []string
}

// CheckCorpus holds the stand at the rules and returns every violation as
// Problems, or nil. A stand that does not pass does not exist.
func CheckCorpus(stand string) error {
	// Only the files the checks below read stop the run. Everything else
	// that is missing is a finding like any other: an absent HERKUNFT.md must
	// not hide what is wrong with the rest of the stand.
	var absent Problems
	for _, name := range []string{"notes", "themes.yaml", "questions.yaml", "manifest.json"} {
		if _, err := os.Stat(filepath.Join(stand, name)); err != nil {
			absent = append(absent, name+" is missing")
		}
	}
	if len(absent) > 0 {
		return absent
	}
	var problems Problems
	notes := noteNames(stand)
	if len(notes) != notesWanted {
		problems = append(problems, fmt.Sprintf("%d notes, expected %d", len(notes), notesWanted))
	}
	problems = append(problems, herkunftProblems(stand, notes)...)
	problems = append(problems, manifestProblems(stand, notes)...)
	problems = append(problems, themeProblems(stand, notes)...)
	problems = append(problems, questionProblems(stand)...)
	if len(problems) > 0 {
		return problems
	}
	return nil
}

// noteNames lists the notes by name, sorted. A notes entry that is not a
// directory holds none.
func noteNames(stand string) []string {
	entries, _ := os.ReadDir(filepath.Join(stand, "notes"))
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	return names
}

// herkunftProblems holds the provenance file against the notes, in both
// directions. The licence asks for the source of every note by name, and
// HERKUNFT.md is where those names stand; its mere existence would let a
// note be added without one.
func herkunftProblems(stand string, notes []string) []string {
	data, err := os.ReadFile(filepath.Join(stand, "HERKUNFT.md"))
	if err != nil {
		return []string{"HERKUNFT.md is missing"}
	}
	found := map[string]bool{}
	for _, name := range named.FindAllString(string(data), -1) {
		found[name] = true
	}
	var problems []string
	for _, name := range notes {
		if !found[name] {
			problems = append(problems, name+": not named in HERKUNFT.md")
		}
	}
	for _, name := range sortedKeys(found) {
		if !slices.Contains(notes, name) {
			problems = append(problems, name+": named in HERKUNFT.md but not among the notes")
		}
	}
	return problems
}

func manifestProblems(stand string, notes []string) []string {
	manifest, broken := loadedManifest(stand)
	if broken != "" {
		return []string{broken}
	}
	var problems []string
	for _, name := range notes {
		raw, ok := manifest[name]
		if !ok {
			problems = append(problems, name+": not in the manifest")
			continue
		}
		entry, ok := raw.(map[string]any)
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: entry must be a mapping, found %s", name, typeName(raw)))
			continue
		}
		problems = append(problems, provenanceProblems(name, entry)...)
		want, ok := entry["sha256"]
		if !ok {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(stand, "notes", name))
		sum := sha256.Sum256(data)
		if digest := hex.EncodeToString(sum[:]); want != digest {
			problems = append(problems, fmt.Sprintf("%s: checksum %s does not match the manifest", name, digest))
		}
	}
	// The loop above walks the notes, so a manifest key no note points at
	// would never be looked at; a renamed note leaves exactly that behind.
	for _, name := range sortedKeys(manifest) {
		if !slices.Contains(notes, name) {
			problems = append(problems, name+": in the manifest but not among the notes")
		}
	}
	return problems
}

// provenanceProblems names the missing fields of one manifest entry, then
// the empty ones: presence alone would pass an empty source, and where a
// check carries the licence condition that is the likelier fault.
func provenanceProblems(name string, entry map[string]any) []string {
	keys := []string{"sha256", "quelle", "abgerufen", "lizenz"}
	var problems []string
	for _, key := range keys {
		if _, ok := entry[key]; !ok {
			problems = append(problems, fmt.Sprintf("%s: missing %q", name, key))
		}
	}
	for _, key := range keys {
		if value, ok := entry[key]; ok && strings.TrimSpace(fmt.Sprint(value)) == "" {
			problems = append(problems, fmt.Sprintf("%s: empty %q", name, key))
		}
	}
	return problems
}

// loadedManifest is the manifest, or the one finding that made reading it
// pointless. A hand-edited JSON file with a stray comma is the likeliest
// fault of all; the gate names it rather than failing on it.
func loadedManifest(stand string) (map[string]any, string) {
	data, _ := os.ReadFile(filepath.Join(stand, "manifest.json"))
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Sprintf("manifest.json is not valid JSON: %v", err)
	}
	manifest, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Sprintf("manifest.json must be a mapping, found %s", typeName(raw))
	}
	return manifest, ""
}

func themeProblems(stand string, notes []string) []string {
	themes, problems := readThemes(stand)
	if len(themes) != themesWanted {
		problems = append(problems, fmt.Sprintf("%d themes, expected %d", len(themes), themesWanted))
	}
	names := map[string]bool{}
	for _, t := range themes {
		names[t.name] = true
	}
	claimed := map[string]int{}
	var order []string
	for _, t := range themes {
		if len(t.notes) != notesPerTheme {
			problems = append(problems, fmt.Sprintf("%s: %d notes, expected %d", t.name, len(t.notes), notesPerTheme))
		}
		if t.neighbour == t.name {
			problems = append(problems, t.name+": is its own neighbour")
		} else if !names[t.neighbour] {
			problems = append(problems, fmt.Sprintf("%s: neighbour %q is not a theme", t.name, t.neighbour))
		}
		for _, note := range t.notes {
			if claimed[note] == 0 {
				order = append(order, note)
			}
			claimed[note]++
			// The partition has two directions. Without this one a theme may
			// name a note that does not exist, and the count still adds up
			// whenever some real note is left unclaimed in exchange.
			if !slices.Contains(notes, note) {
				problems = append(problems, fmt.Sprintf("%s: %s is not among the notes", t.name, note))
			}
		}
	}
	for _, note := range order {
		if claimed[note] > 1 {
			problems = append(problems, note+": claimed by two themes")
		}
	}
	for _, note := range notes {
		if claimed[note] == 0 {
			problems = append(problems, note+": claimed by no theme")
		}
	}
	return problems
}

// readThemes returns the themes and whatever made an entry unreadable. As
// with the question set, a typo in a hand-written file is named, and the
// entries around it are still checked.
func readThemes(stand string) ([]theme, []string) {
	data, _ := os.ReadFile(filepath.Join(stand, "themes.yaml"))
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, []string{fmt.Sprintf("themes.yaml is not valid YAML: %v", err)}
	}
	entries, ok := raw.([]any)
	if !ok {
		return nil, []string{fmt.Sprintf("themes.yaml must be a list of entries, found %s", typeName(raw))}
	}
	var themes []theme
	var problems []string
	for i, entry := range entries {
		if t, ok := readTheme(entry, i+1, &problems); ok {
			themes = append(themes, t)
		}
	}
	return themes, problems
}

func readTheme(entry any, position int, problems *[]string) (theme, bool) {
	fields, ok := mapping(entry)
	if !ok {
		*problems = append(*problems, fmt.Sprintf("theme %d: must be a mapping, found %s", position, typeName(entry)))
		return theme{}, false
	}
	name := fmt.Sprintf("theme %d", position)
	if n, ok := fields["name"]; ok {
		name = fmt.Sprint(n)
	}
	missing := false
	for _, key := range []string{"name", "nachbar", "notizen"} {
		if _, ok := fields[key]; !ok {
			*problems = append(*problems, fmt.Sprintf("%s: missing field %q", name, key))
			missing = true
		}
	}
	if missing {
		return theme{}, false
	}
	listed, ok := fields["notizen"].([]any)
	if !ok {
		*problems = append(*problems, fmt.Sprintf("%s: %q must be a list, found %s", name, "notizen", typeName(fields["notizen"])))
		return theme{}, false
	}
	t := theme{name: name, neighbour: fmt.Sprint(fields["nachbar"])}
	for _, note := range listed {
		t.notes = append(t.notes, fmt.Sprint(note))
	}
	return t, true
}

func questionProblems(stand string) []string {
	questions, err := LoadQuestions(filepath.Join(stand, "questions.yaml"), DefaultShape())
	var p Problems
	if errors.As(err, &p) {
		return p
	}
	if err != nil {
		return []string{err.Error()}
	}
	reverse := 0
	for _, q := range questions {
		if isReverse(q) {
			reverse++
		}
	}
	if reverse < reverseMinimum {
		return []string{fmt.Sprintf("%d questions in the reverse direction, expected at least %d", reverse, reverseMinimum)}
	}
	return nil
}

// isReverse tells an English question against the German notes: a
// cross-lingual question with more English than German function words.
func isReverse(q Question) bool {
	if q.Kind != CrossLingual {
		return false
	}
	english, german := englishWords(), germanWords()
	var e, g int
	for _, w := range word.FindAllString(q.Query, -1) {
		w = strings.ToLower(w)
		if english[w] {
			e++
		}
		if german[w] {
			g++
		}
	}
	return e > g
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
