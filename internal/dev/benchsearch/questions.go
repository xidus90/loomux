// Package benchsearch measures how well the search finds a note: a
// versioned corpus of notes, a question set against it, and the rules both
// have to keep before a single measurement runs.
//
// Every violation is collected and reported together. Stopping at the first
// one would turn fixing a set into a queue of runs, and the person fixing it
// cannot see how much is wrong until the last problem is gone.
package benchsearch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Kind is the sort of a question. The values are the file format, hence
// German.
type Kind string

const (
	Exact        Kind = "exakt"
	Paraphrase   Kind = "umschreibung"
	Mixed        Kind = "gemischt"
	CrossLingual Kind = "sprachuebergreifend"
)

// kinds is the order of every table and every listing of kinds.
var kinds = [...]Kind{Exact, Paraphrase, Mixed, CrossLingual}

// Shape is how many questions of each kind a set must hold.
type Shape map[Kind]int

// DefaultShape is the shape of the fifty questions of a corpus stand.
func DefaultShape() Shape {
	return Shape{Exact: 13, Paraphrase: 13, Mixed: 10, CrossLingual: 14}
}

// Question is one entry of a question set.
type Question struct {
	ID, Query, Evidence, Note string
	Kind                      Kind
	Expect                    string // absolute, cleaned
}

// Problems is everything wrong with a set or a stand, one finding per line.
type Problems []string

func (p Problems) Error() string { return strings.Join(p, "\n") }

// LoadQuestions reads the set at path and checks it against shape. The error
// is Problems for anything wrong with the set, or the I/O error that kept it
// from being read.
func LoadQuestions(path string, shape Shape) ([]Question, error) {
	abs, err := filepath.Abs(path)
	var data []byte
	if err == nil {
		data, err = os.ReadFile(abs)
	}
	if err != nil {
		return nil, err
	}
	// A hand-written file with a stray bracket is the likeliest fault of
	// all, and this is the gate: it has to name that fault, not fail on it.
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, Problems{fmt.Sprintf("%s is not valid YAML: %v", filepath.Base(path), err)}
	}
	entries, ok := raw.([]any)
	if !ok {
		return nil, Problems{fmt.Sprintf("the question set must be a list of entries, found %s", typeName(raw))}
	}
	var problems Problems
	var questions []Question
	seen := map[string]bool{}
	base := filepath.Dir(abs)
	for i, entry := range entries {
		if q, ok := readQuestion(entry, i+1, seen, &problems, base); ok {
			questions = append(questions, q)
		}
	}
	problems = append(problems, shapeProblems(questions, shape)...)
	if len(problems) > 0 {
		return nil, problems
	}
	return questions, nil
}

func readQuestion(entry any, position int, seen map[string]bool, problems *Problems, base string) (Question, bool) {
	fields, ok := mapping(entry)
	if !ok {
		*problems = append(*problems, fmt.Sprintf("entry %d: must be a mapping, found %s", position, typeName(entry)))
		return Question{}, false
	}
	name := fmt.Sprintf("entry %d", position)
	if id, ok := fields["id"]; ok {
		name = fmt.Sprint(id)
	}
	missing := false
	for _, key := range []string{"id", "sort", "query", "expect", "beleg"} {
		if _, ok := fields[key]; !ok {
			*problems = append(*problems, fmt.Sprintf("%s: missing field %q", name, key))
			missing = true
		}
	}
	if missing {
		return Question{}, false
	}
	if seen[name] {
		*problems = append(*problems, fmt.Sprintf("%s: duplicate id %q", name, name))
	}
	seen[name] = true
	kind, known := kindOf(fmt.Sprint(fields["sort"]), name, problems)
	// A relative expect is meant relative to the question set, not to
	// whoever happens to run the bench: a corpus ships its notes beside its
	// questions.yaml. An absolute one names its file wherever that lies.
	expect := filepath.Clean(fmt.Sprint(fields["expect"]))
	if !filepath.IsAbs(expect) {
		expect = filepath.Join(base, expect)
	}
	evidence := fmt.Sprint(fields["beleg"])
	targetProblems(expect, evidence, name, problems)
	if !known {
		return Question{}, false
	}
	q := Question{ID: name, Kind: kind, Query: fmt.Sprint(fields["query"]), Expect: expect, Evidence: evidence}
	if note, ok := fields["hinweis"]; ok {
		q.Note = fmt.Sprint(note)
	}
	return q, true
}

func kindOf(value, name string, problems *Problems) (Kind, bool) {
	for _, k := range kinds {
		if string(k) == value {
			return k, true
		}
	}
	*problems = append(*problems, fmt.Sprintf("%s: unknown sort %q", name, value))
	return "", false
}

func targetProblems(expect, evidence, name string, problems *Problems) {
	data, err := os.ReadFile(expect)
	if err != nil {
		*problems = append(*problems, fmt.Sprintf("%s: expect does not exist: %s", name, expect))
		return
	}
	// Evidence quotes prose. Where the target wraps that prose, the break is
	// the file's layout and says nothing about the content: same words in
	// the same order is what the check is for, not where the newlines fall.
	if !strings.Contains(collapsed(string(data)), collapsed(evidence)) {
		*problems = append(*problems, fmt.Sprintf("%s: evidence not found verbatim in %s", name, expect))
	}
}

// collapsed joins the words with single spaces. strings.Fields splits at
// every Unicode space, the no-break space of a Wikipedia excerpt included,
// which is what Python's \s does in a str pattern and Go's regexp \s does not.
func collapsed(s string) string { return strings.Join(strings.Fields(s), " ") }

func shapeProblems(questions []Question, shape Shape) []string {
	counted := map[Kind]int{}
	for _, q := range questions {
		counted[q.Kind]++
	}
	var problems []string
	for _, k := range kinds {
		if wanted, ok := shape[k]; ok && counted[k] != wanted {
			problems = append(problems, fmt.Sprintf("%s: %d questions, expected %d", k, counted[k], wanted))
		}
	}
	return problems
}

// mapping is a decoded YAML mapping by its string keys. yaml.v3 decodes a
// mapping with a key of another type into map[any]any; such a key can never
// be a field name, so it is dropped.
func mapping(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		fields := map[string]any{}
		for key, value := range m {
			if name, ok := key.(string); ok {
				fields[name] = value
			}
		}
		return fields, true
	}
	return nil, false
}

// typeName names the type of a decoded value in a finding.
func typeName(v any) string {
	switch v.(type) {
	case map[string]any, map[any]any:
		return "map"
	case []any:
		return "list"
	}
	return fmt.Sprintf("%T", v)
}
