package verify

import "strings"

// Line is the Godot line a project asks for.
//
// Major is always there, Minor only where the project names it: a Godot 4
// project writes `config/features=PackedStringArray("4.7")` and says both,
// while a Godot 3 project has no such setting at all and leaves only
// `config_version=4`, which separates the 3 line from the 4 line and nothing
// finer. A line without a minor is therefore not an incomplete reading but
// everything the project said, and Matches compares no further than that.
type Line struct {
	Major, Minor string
}

// Known says whether the project named a minor version.
func (l Line) Known() bool { return l.Minor != "" }

// String is the line as a message names it: "4.7" or, for a project that
// named no minor, "3".
func (l Line) String() string {
	if !l.Known() {
		return l.Major
	}
	return l.Major + "." + l.Minor
}

// Matches says whether a binary that reported this major.minor can serve the
// line. It compares no further than the project spoke: where the project
// named a minor, both halves must agree, and where it named none, 3.5 and
// 3.6 are equally its line.
func (l Line) Matches(reported string) bool {
	if l.Known() {
		return reported == l.String()
	}
	major, _, _ := strings.Cut(reported, ".")
	return major == l.Major
}
