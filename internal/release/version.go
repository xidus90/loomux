// Package release holds the platform-neutral rules of a loomux release: the
// next version, the checks on a pull request body, the changelog entry and
// the cross build. CI glue for a particular forge calls it through
// `loomux dev release`.
package release

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"sync"
)

// ErrBump names a bump that is not one of the three SemVer positions.
var ErrBump = errors.New("bump must be major, minor or patch")

// tagPattern accepts only plain release tags; leading zeros and pre-release
// suffixes are no releases of this scheme and must not win the maximum. It is
// compiled on first use, since this package loads on every loomux start.
var tagPattern = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
})

// NextVersion returns the version after the highest release tag. Without any
// release tag the first version is 1.0.0, whatever the bump says.
func NextVersion(tags []string, bump string) (string, error) {
	if bump != "major" && bump != "minor" && bump != "patch" {
		return "", fmt.Errorf("%w, got %q", ErrBump, bump)
	}
	var best [3]int
	found := false
	for _, tag := range tags {
		m := tagPattern().FindStringSubmatch(tag)
		if m == nil {
			continue
		}
		v, ok := parseTriple(m[1:])
		if !ok {
			continue
		}
		if !found || less(best, v) {
			best, found = v, true
		}
	}
	if !found {
		return "1.0.0", nil
	}
	switch bump {
	case "major":
		best = [3]int{best[0] + 1, 0, 0}
	case "minor":
		best = [3]int{best[0], best[1] + 1, 0}
	default:
		best[2]++
	}
	return fmt.Sprintf("%d.%d.%d", best[0], best[1], best[2]), nil
}

func less(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// parseTriple reads the three numbers of a matched tag; a number too large
// for int makes the tag no release rather than a saturated one.
func parseTriple(parts []string) ([3]int, bool) {
	var v [3]int
	for i := range v {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return v, false
		}
		v[i] = n
	}
	return v, true
}
