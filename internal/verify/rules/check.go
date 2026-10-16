package rules

import (
	"github.com/xidus90/loomux/internal/config"
)

// Options say what Check looks at: the project at Root, and with Cached the
// index instead of the working tree.
type Options struct {
	Root   string
	Cached bool
}

// Result is what Check found. MapErrors mean the layer map itself is wrong;
// then only the findings that do not depend on the map (leftover
// instrumentation) are reported, the rest would be judged by a broken map.
type Result struct {
	Findings  []Finding
	MapErrors []string
	Notes     []string
}

// Check applies the rules of the project: instrumentation (only for the
// index), then edges, pairing and files outside every layer.
func Check(o Options) (Result, error) {
	rs, err := config.ReadRules(o.Root)
	if err != nil {
		return Result{MapErrors: []string{err.Error()}}, nil
	}
	if !rs.Present {
		return Result{Notes: []string{"no [rules] in .loomux/config.toml"}}, nil
	}
	files, err := Files(o.Root, o.Cached)
	if err != nil {
		return Result{}, err
	}
	rels := make([]string, len(files))
	for i, f := range files {
		rels[i] = f.Rel
	}
	var res Result
	if o.Cached {
		if res.Findings, err = Instrumented(files, o.Root); err != nil {
			return Result{}, err
		}
	}
	m, mapErrs := Assign(rs, rels)
	if len(mapErrs) > 0 {
		res.MapErrors = mapErrs
		return res, nil
	}
	edges, err := Edges(files, m)
	if err != nil {
		return Result{}, err
	}
	paired, mapErrs, err := Pairing(files, m)
	if err != nil {
		return Result{}, err
	}
	if len(mapErrs) > 0 {
		res.MapErrors = mapErrs
		return res, nil
	}
	res.Findings = append(res.Findings, edges...)
	res.Findings = append(res.Findings, paired...)
	res.Findings = append(res.Findings, Unlayered(rels, m)...)
	return res, nil
}
