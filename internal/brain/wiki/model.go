package wiki

import (
	"time"
)

// Source is one entry of `sources[]`.
//
// The field set is the Python `SourceRef` of `brain/wiki/page.py`, copied
// rather than shared: `doc_id`, `content_hash` and `revision` are what the
// maintenance layer follows a page back to its origin with, and the earlier
// `title`/`type` pair here carried none of the three.
type Source struct {
	ID          string `yaml:"id,omitempty" json:"id,omitempty"`
	Resource    string `yaml:"resource" json:"resource"`
	DocID       string `yaml:"doc_id,omitempty" json:"doc_id,omitempty"`
	ContentHash string `yaml:"content_hash,omitempty" json:"content_hash,omitempty"`
	Revision    *int   `yaml:"revision,omitempty" json:"revision,omitempty"`
}

// Generated records who wrote a page.
//
// OKF §5.2 makes `by` mandatory inside `generated`, and `at` is not -- hence
// the pointer: a page that states no timestamp must stay distinguishable from
// one that states the zero time, or the rule would report a date nobody wrote.
type Generated struct {
	By string     `yaml:"by,omitempty" json:"by,omitempty"`
	At *time.Time `yaml:"at,omitempty" json:"at,omitempty"`
}

// Frontmatter is the YAML block as it decodes -- the reader's container, not
// the model. Everything a check reads is copied onto WikiPage below, so that
// no rule has to choose between two spellings of the same value.
//
// The optional scalars are pointers because the Python side reads them with
// `meta.get(...)` and keeps `None` apart from an empty string: the
// `missing-type` message of `wiki/lint.py` differs between a page that
// declares no `type` and a page whose `type` is empty.
type Frontmatter struct {
	Title         string     `yaml:"title" json:"title"`
	Type          *string    `yaml:"type" json:"type,omitempty"`
	Description   string     `yaml:"description,omitempty" json:"description,omitempty"`
	Tags          []string   `yaml:"tags,omitempty" json:"tags,omitempty"`
	Topics        []string   `yaml:"topics,omitempty" json:"topics,omitempty"`
	Created       time.Time  `yaml:"created,omitempty" json:"created,omitempty"`
	Updated       time.Time  `yaml:"updated,omitempty" json:"updated,omitempty"`
	Sources       []Source   `yaml:"sources,omitempty" json:"sources,omitempty"`
	OpenConflicts *int       `yaml:"open_conflicts,omitempty" json:"open_conflicts,omitempty"`
	Status        *string    `yaml:"status,omitempty" json:"status,omitempty"`
	StaleAfter    *time.Time `yaml:"stale_after,omitempty" json:"stale_after,omitempty"`
	Realization   *string    `yaml:"realization,omitempty" json:"realization,omitempty"`
	ImplementedIn *string    `yaml:"implemented_in,omitempty" json:"implemented_in,omitempty"`
	Generated     *Generated `yaml:"generated,omitempty" json:"generated,omitempty"`
	// Runtime is no pointer, unlike the scalars above: OKF §10.2 makes it
	// REQUIRED for an Attested Computation and says nothing about an empty
	// one, so an absent key and an empty value ask for the same repair. A
	// rule that told them apart would give one fix two names.
	Runtime string `yaml:"runtime,omitempty" json:"runtime,omitempty"`
}

// WikiPage is one page reduced to what the checks judge it by. The field set
// follows the Python `WikiPage`; `Path`, `Frontmatter` and `RawBody` are the
// reader's own additions, which the Python side takes from the `Path` object
// and from the text it still holds while it lints.
type WikiPage struct {
	Path              string
	Relative          string
	Frontmatter       Frontmatter
	PageType          *string
	Title             string
	Description       string
	Sources           []Source
	Links             []string
	DeclaredConflicts *int
	FoundConflicts    int
	Status            *string
	StaleAfter        *time.Time
	ModTime           time.Time
	// The message, not a bool: `okf/frontmatter-unparsable` (OKF §11.1) has to
	// name what YAML choked on, or the reader has nowhere to start.
	BrokenFrontmatter *string
	Realization       *string
	ImplementedIn     *string
	Generated         *Generated
	RawBody           string
	Runtime           string
	// HasFrontmatter says whether the file opened with a `---` block at all.
	// No other field answers that: an empty block and no block leave every
	// value zero alike, and OKF §8 permits a block in exactly one `index.md`
	// of a bundle, so the two states decide a rule.
	HasFrontmatter bool
	// FrontmatterKeys are the block's top-level keys in the order they were
	// written. `Frontmatter` above drops every key it does not name, and OKF
	// §12 judges a bundle-root `index.md` by the key set itself rather than
	// by any value. The written order is kept because the check run has to
	// render byte for byte the same output twice, and a map would not.
	FrontmatterKeys []string
}

type BundleContext struct {
	Root          string
	UntouchedDays int
	Now           time.Time
	DeclaredTypes map[string]bool
	IsProject     bool
}
