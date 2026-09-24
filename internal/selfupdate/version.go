// Package selfupdate keeps the machine-wide loomux binary at the newest
// release of its channel: it asks gh for the releases, downloads one, checks
// it against SHA256SUMS and its own --version, and swaps it in. What happened
// is written to update.json, which session start reads.
package selfupdate

import (
	"strconv"
	"strings"
)

// version is a release number without a suffix; the tags this repository
// cuts carry none.
type version [3]int

// parseVersion reads "v1.2.3" or "1.2.3". Anything else, a suffix or a
// leading zero included, is not a version the updater acts on.
func parseVersion(s string) (version, bool) {
	parts := strings.Split(strings.TrimPrefix(s, "v"), ".")
	if len(parts) != 3 {
		return version{}, false
	}
	var v version
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || part != strconv.Itoa(n) {
			return version{}, false
		}
		v[i] = n
	}
	return v, true
}

func (v version) less(w version) bool {
	for i := range v {
		if v[i] != w[i] {
			return v[i] < w[i]
		}
	}
	return false
}

// Newer reports whether tag names a later release than running. A side that
// does not parse is never newer, so a guess never replaces a binary and a
// development build never asks for one.
func Newer(tag, running string) bool {
	t, ok := parseVersion(tag)
	if !ok {
		return false
	}
	r, ok := parseVersion(running)
	if !ok {
		return false
	}
	return r.less(t)
}

// Release is one entry of `gh release list --json tagName,isPrerelease`.
type Release struct {
	Tag        string `json:"tagName"`
	Prerelease bool   `json:"isPrerelease"`
}

// pick is the highest release the channel takes, by version rather than by
// publication. The stable channel, and a build that names none, takes no
// pre-release; every other channel takes both.
func pick(releases []Release, channel string) (Release, bool) {
	stable := channel == "" || channel == "stable"
	var best Release
	var bestVersion version
	found := false
	for _, r := range releases {
		if stable && r.Prerelease {
			continue
		}
		v, ok := parseVersion(r.Tag)
		if !ok {
			continue
		}
		if !found || bestVersion.less(v) {
			best, bestVersion, found = r, v, true
		}
	}
	return best, found
}
