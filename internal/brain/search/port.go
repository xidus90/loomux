package search

// Profile names the search chain mode offered by the backend.
type Profile string

const (
	// ProfileKeyword performs BM25 keyword search.
	ProfileKeyword Profile = "keyword"
	// ProfileFast performs vector search without reranking or query expansion.
	ProfileFast Profile = "fast"
	// ProfileFull performs the full hybrid search chain with expansion and reranking.
	ProfileFull Profile = "full"
)

// SearchHit represents a single search result matching a query.
type SearchHit struct {
	Collection string
	Scope      string
	Relative   string
	Line       int
	Title      string
	Snippet    string
	Score      float64
	ContentKey string
}

// SearchPort abstracts backend search execution and maintenance.
type SearchPort interface {
	Search(query string, collections []string, profile Profile, n int) ([]SearchHit, error)
	Indexed(collection string) ([]string, error)
	Refresh(collections []string) error
	NotYetSearchable() (int, error)
	Embed(collections []string) error
}
