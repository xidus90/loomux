package linecov

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type lcovFormat struct{}

func (lcovFormat) Name() string { return "lcov" }

func (lcovFormat) Detect(h []byte) bool {
	first, _, _ := bytes.Cut(h, []byte("\n"))
	return bytes.HasPrefix(first, []byte("TN:")) || bytes.HasPrefix(first, []byte("SF:"))
}

// daRecord reads DA:<line>,<hits>[,<checksum>] by position: splitting on the
// first comma would read "0,<checksum>" as the hits and lose an uncovered line.
var daRecord = regexp.MustCompile(`^(\d+),(\d+)(?:,|$)`)

// Read keeps only SF and DA; a DA before the first SF or after end_of_record
// belongs to no file and is passed over.
func (lcovFormat) Read(content []byte, _ ReadContext) ([]Record, []string) {
	var recs []Record
	var problems []string
	cur := -1
	for _, raw := range strings.Split(text(content), "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "SF:"):
			recs = append(recs, Record{Path: strings.TrimSpace(line[3:]), Lines: map[int]uint64{}})
			cur = len(recs) - 1
		case line == "end_of_record":
			cur = -1
		case strings.HasPrefix(line, "DA:") && cur >= 0:
			n, hits, ok := daFields(line[3:])
			if !ok {
				problems = append(problems, fmt.Sprintf("%s: unreadable record %q", recs[cur].Path, line))
				continue
			}
			recs[cur].Lines[n] = addSat(recs[cur].Lines[n], hits)
		}
	}
	return recs, problems
}

func daFields(s string) (int, uint64, bool) {
	m := daRecord.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 1 {
		return 0, 0, false
	}
	hits, err := strconv.ParseUint(m[2], 10, 64)
	return n, hits, err == nil
}
