package linecov

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type goProfileFormat struct{}

func (goProfileFormat) Name() string         { return "goprofile" }
func (goProfileFormat) Detect(h []byte) bool { return bytes.HasPrefix(h, []byte("mode: ")) }

// goBlock is <import path>:<line>.<col>,<line>.<col> <statements> <count>.
var goBlock = regexp.MustCompile(`^(.+):(\d+)\.\d+,(\d+)\.\d+ \d+ (\d+)$`)

// Read gives every line of a block the block's count; a line under several
// blocks keeps the highest, so one run block marks it as hit.
func (goProfileFormat) Read(content []byte, ctx ReadContext) ([]Record, []string) {
	if ctx.Module == "" {
		return nil, []string{"no go.mod under the root: Go profile paths cannot be resolved"}
	}
	var recs []Record
	var problems []string
	index := map[string]int{}
	outside := map[string]bool{}
	// Blocks follow the mode line; blank lines may precede it.
	_, body, _ := strings.Cut(text(content), "mode:")
	for _, raw := range strings.Split(body, "\n")[1:] {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		file, from, to, count, ok := goFields(line)
		if !ok {
			problems = append(problems, fmt.Sprintf("unreadable record %q", line))
			continue
		}
		rel, inside := strings.CutPrefix(file, ctx.Module+"/")
		if !inside {
			if !outside[file] {
				outside[file] = true
				problems = append(problems, fmt.Sprintf("%s: outside module %s", file, ctx.Module))
			}
			continue
		}
		i, seen := index[rel]
		if !seen {
			i = len(recs)
			index[rel] = i
			recs = append(recs, Record{Path: rel, Lines: map[int]uint64{}})
		}
		for n := from; n <= to; n++ {
			if cur, ok := recs[i].Lines[n]; !ok || count > cur {
				recs[i].Lines[n] = count
			}
		}
	}
	return recs, problems
}

func goFields(line string) (file string, from, to int, count uint64, ok bool) {
	m := goBlock.FindStringSubmatch(line)
	if m == nil {
		return "", 0, 0, 0, false
	}
	from, err1 := strconv.Atoi(m[2])
	to, err2 := strconv.Atoi(m[3])
	count, err3 := strconv.ParseUint(m[4], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || from < 1 || to < from {
		return "", 0, 0, 0, false
	}
	return m[1], from, to, count, true
}
