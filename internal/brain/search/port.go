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

// Unavailable is a port that answers every question with err: the engine that
// cannot be built, a machine setting that does not read among the reasons. It
// fails where the engine is asked, so what needs no engine keeps working.
func Unavailable(err error) SearchPort { return unavailable{err} }

type unavailable struct{ err error }

func (u unavailable) Search(string, []string, Profile, int) ([]SearchHit, error) { return nil, u.err }
func (u unavailable) Indexed(string) ([]string, error)                           { return nil, u.err }
func (u unavailable) Refresh([]string) error                                     { return u.err }
func (u unavailable) NotYetSearchable() (int, error)                             { return 0, u.err }
func (u unavailable) Embed([]string) error                                       { return u.err }
