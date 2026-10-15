package linecov

import (
	"fmt"
	"maps"
	"slices"
	"time"
)

// Input is everything the rules need; Check builds it.
type Input struct {
	Report  Report // resolved paths only
	Tree    *Tree
	Active  []Active
	Natives []Marker
	Newest  time.Time // modification time of the newest report read
	Read    func(rel string) ([]byte, error)
	ModTime func(rel string) (time.Time, error)
}

// mtimeSlack absorbs FAT's two-second mtime: a report written moments after
// a source can carry a time slightly before it.
const mtimeSlack = 2 * time.Second

// Gate applies the rules to every reported file and every expected one.
func Gate(in Input) []Finding {
	var out []Finding
	graded := map[string]bool{}
	for _, rel := range slices.Sorted(maps.Keys(in.Report)) {
		if excluded(in.Active, rel) {
			continue
		}
		graded[rel] = true
		src, err := in.Read(rel)
		if err != nil {
			out = append(out, Finding{rel, 0, "unreadable: " + err.Error()})
			continue
		}
		s := ScanSource(rel, src, in.Natives)
		out = append(out, s.Findings...)
		out = append(out, grade(rel, in.Report[rel], s)...)
		if mt, err := in.ModTime(rel); err == nil && mt.After(in.Newest.Add(mtimeSlack)) {
			out = append(out, Finding{rel, 0, "is newer than the report"})
		}
	}
	for _, rel := range in.Tree.Files {
		if graded[rel] || !slices.ContainsFunc(in.Active, func(a Active) bool { return a.Expects(rel) }) {
			continue
		}
		src, err := in.Read(rel)
		if err != nil {
			out = append(out, Finding{rel, 0, "unreadable: " + err.Error()})
			continue
		}
		s := ScanSource(rel, src, in.Natives)
		out = append(out, s.Findings...)
		if !s.File && !s.NativeFile {
			out = append(out, Finding{rel, 0, "is absent from the report"})
		}
	}
	return SortFindings(out)
}

// excluded: a record for a file the tool's settings exclude describes
// nothing that belongs to the scope.
func excluded(act []Active, rel string) bool {
	return slices.ContainsFunc(act, func(a Active) bool { return a.Owns(rel) && a.Excludes(rel) })
}

func grade(rel string, lines map[int]uint64, s Scan) []Finding {
	if s.NativeFile {
		return nil
	}
	if s.File {
		hit := 0
		for _, h := range lines {
			if h > 0 {
				hit++
			}
		}
		if hit > 0 {
			return []Finding{{rel, 0, fmt.Sprintf("coverage:exempt-file but %d lines hit", hit)}}
		}
		return nil
	}
	var out []Finding
	for _, n := range slices.Sorted(maps.Keys(lines)) {
		e, exempt := s.Lines[n]
		switch hits := lines[n]; {
		case hits > 0 && exempt && e.own:
			out = append(out, Finding{rel, n, fmt.Sprintf("coverage:exempt but hit %d times", hits)})
		case hits == 0 && !exempt:
			out = append(out, Finding{rel, n, "not covered"})
		case hits == 0 && e.own && !e.reason:
			out = append(out, Finding{rel, n, "coverage:exempt without a reason"})
		}
	}
	return out
}
