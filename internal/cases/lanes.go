package cases

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"sort"
)

// redStates fail a check on either side.
var redStates = []string{"failed", "timed-out", "blocked", "missing-tool", "unready", "error"}

// KindVerdicts judges every kind a check report names: red when one of its
// lines is red, else ok when one is ok, else neutral. loomux's note that a
// requested kind had nothing to check fails the check, so it is red too.
// Everything else is tool output and judges nothing.
func KindVerdicts(stdout []byte) map[string]string {
	// The head of a kind in the old chain (`test: failed [preset]`) or of a
	// lane in loomux (`test/go: failed [preset] 1.0s`). The old chain wrote
	// `unavailable` and `blocked` as the source in brackets, so its state word
	// there is `failed`, which is how Python judged them. Compiled here, not
	// at start: the binary carries this package for `dev import-cases`.
	verdictLine := regexp.MustCompile(`^(lint|types|test|coverage)(/[^:]*)?: ([a-z-]+)`)
	nothingLine := regexp.MustCompile("^nothing to check for `(lint|types|test|coverage)`$")
	out := map[string]string{}
	for _, line := range bytes.Split(stdout, []byte("\n")) {
		if n := nothingLine.FindSubmatch(line); n != nil {
			out[string(n[1])] = "red"
			continue
		}
		m := verdictLine.FindSubmatch(line)
		if m == nil {
			continue
		}
		kind, state := string(m[1]), string(m[3])
		switch {
		case slices.Contains(redStates, state):
			out[kind] = "red"
		case state == "ok" && out[kind] != "red":
			out[kind] = "ok"
		case out[kind] == "":
			out[kind] = "neutral"
		}
	}
	return out
}

// compareLanes names every kind whose verdict differs, expected first; a
// kind one side does not name at all is absent there.
func compareLanes(expected, actual []byte) []string {
	want, got := KindVerdicts(expected), KindVerdicts(actual)
	kinds := map[string]bool{}
	for k := range want {
		kinds[k] = true
	}
	for k := range got {
		kinds[k] = true
	}
	names := make([]string, 0, len(kinds))
	for k := range kinds {
		names = append(names, k)
	}
	sort.Strings(names)
	var mismatches []string
	for _, k := range names {
		w, g := or(want[k], "absent"), or(got[k], "absent")
		if w != g {
			mismatches = append(mismatches, fmt.Sprintf("lanes: %s %s != %s", k, w, g))
		}
	}
	return mismatches
}

func or(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
