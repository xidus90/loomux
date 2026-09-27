package runs

// Baseline is what a run starts from: the HEAD commit and the paths that
// were already changed then, relative to the project root. It lives here,
// with the marker that carries it, so that reading a marker links no part
// of the runtime.
type Baseline struct {
	Commit string
	Dirty  []string
}
