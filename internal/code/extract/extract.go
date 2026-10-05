// Package extract is what every language's extractor hands the resolver: one
// file's nodes, its raw edges and its imports, and the interface a language
// implements to produce them.
//
// A package of its own and tied to no language, because two sides depend on
// it that must not pull a parser along: the hook path, which extracts one Go
// file per edit and has a budget, and extract/all, which lists the languages.
// A language's dependencies stay in that language's package; nothing here
// may import one.
package extract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"strings"

	"github.com/xidus90/loomux/internal/code/model"
)

// MaxBodyChars caps the searchable body. The original's figure; a definition
// longer than this is findable by its first 5000 characters or not at all.
const MaxBodyChars = 5000

// MintID returns base, or base with the lowest free ordinal appended.
//
// A loop and not a single "~2" guess: a qualified source name may itself end
// in "~2", and only the loop is tight against that.
func MintID(base string, minted map[string]bool) string {
	id := base
	for k := 2; minted[id]; k++ {
		id = base + "~" + itoa(k)
	}
	minted[id] = true
	return id
}

// itoa is strconv.Itoa without the import.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// Collapse turns a definition's text into one searchable line: every run of
// whitespace becomes a single space, and the result is capped.
func Collapse(text string) string {
	out := strings.Join(strings.Fields(text), " ")
	if len(out) > MaxBodyChars {
		return out[:MaxBodyChars]
	}
	return out
}

// Hash is sha256 as full hex. Truncating would save bytes in wiring.json and
// buy a collision nobody would debug.
func Hash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// FileNode is the node that stands for a whole file. covered holds the
// 1-based lines some symbol's span covers; they stay out of its body text.
//
// Its id is the path itself, with no '#'. That is not cosmetic: an import edge
// has the file as its source, and model.Validate requires an edge's source to
// be a node of the graph.
//
// Its span is L1-L<newlines+1>: a trailing newline counts the empty line after
// it, where a symbol's span ends on the last line that holds any of the
// symbol. Every Go graph was built with this count, and changing it would
// change every file node.
func FileNode(rel, source string, covered map[int]bool) model.Node {
	return model.Node{
		ID:       model.NodeID(rel),
		Name:     path.Base(rel),
		Kind:     model.KindFile,
		Path:     rel,
		Span:     model.Span(fmt.Sprintf("L1-L%d", strings.Count(source, "\n")+1)),
		Exported: true,
		BodyHash: Hash(source),
		BodyText: Collapse(residual(source, covered)),
	}
}

// residual is the file's text outside every symbol span: the import header,
// module and package constants, the package comment or module docstring.
//
// Symbol bodies are indexed on their own nodes, so repeating them here would
// store most of the repository twice. What is left is exactly what makes a file
// findable by a word that lives in no definition.
func residual(source string, covered map[int]bool) string {
	var keep []string
	for i, line := range strings.Split(source, "\n") {
		if !covered[i+1] {
			keep = append(keep, line)
		}
	}
	return strings.Join(keep, "\n")
}

// Language is one extractor: the files it claims and what it reads out of
// one of them.
type Language interface {
	// Name is the language as the graph's meta lists it: "go".
	Name() string

	// Version is this extractor's identity, bumped by hand whenever its nodes
	// or edges change shape or meaning: "go/1".
	Version() string

	// Extensions are the file extensions it claims, dot included, and matched
	// exactly -- a file's extension in another case is not claimed: [".go"].
	Extensions() []string

	// File extracts one file. rel is its repo-relative, slash-separated path;
	// source is its contents, which the caller has already read in order to
	// hash it.
	File(rel, source string) (Result, error)
}

// Import is one import of a file, with the alias exactly as written.
//
// The alias is kept raw and not resolved to a name here, because resolving it
// needs the TARGET package's clause -- Go's `import "gopkg.in/yaml.v3"` binds
// `yaml` and not `v3` -- and an extractor parses one file at a time. Guessing
// the last path segment there would put the guess where nothing can correct
// it.
//
// Name is the imported name of Python's `from Path import Name`, where the
// file binds a name out of a module rather than the module itself. Go has no
// such form and leaves it empty.
type Import struct {
	Alias string `json:"alias,omitempty"`
	Path  string `json:"path"`
	Name  string `json:"name,omitempty"`
}

// RawEdge is an edge whose target is not resolved yet.
//
// Three shapes of call reach the resolver, and the difference is exactly what
// the resolver is allowed to assume:
//
//   - Name alone      -- a bare call; the same file first, then a unique match
//   - Name and Owner  -- a member call on a local whose type is known
//   - Name and Receiver -- a selector whose receiver is declared nowhere in the
//     file; the resolver decides whether that receiver names a package, because
//     only it can see the target's package clause
type RawEdge struct {
	Source    model.NodeID   `json:"source"`
	Relation  model.Relation `json:"relation"`
	TargetID  model.NodeID   `json:"target_id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Owner     string         `json:"owner,omitempty"`
	Receiver  string         `json:"receiver,omitempty"`
	Specifier string         `json:"specifier,omitempty"`
	File      string         `json:"file"`
}

// Result is what one file contributes to the graph.
//
// Language is the Name of the extractor that read the file. The resolver
// groups files by it: a call is resolved only against files of its own
// language, and a language without resolution rules keeps its nodes and its
// containment and nothing else.
//
// ParseErrors counts what a tolerant parser skipped over and still extracted
// around. go/parser refuses a broken file outright, so the Go extractor
// leaves it 0.
type Result struct {
	Path        string       `json:"path"`
	Language    string       `json:"language"`
	Package     string       `json:"package,omitempty"`
	Imports     []Import     `json:"imports,omitempty"`
	Nodes       []model.Node `json:"nodes"`
	Edges       []RawEdge    `json:"edges,omitempty"`
	ParseErrors int          `json:"parse_errors,omitempty"`
}
