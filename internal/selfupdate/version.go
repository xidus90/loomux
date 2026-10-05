// Package selfupdate keeps the machine-wide loomux binary at the newest
// release of its channel: it asks gh for the releases, downloads one, checks
// it against SHA256SUMS and its own --version, and swaps it in. What happened
// is written to update.json, which session start reads.
package selfupdate

import (
	"strconv"
	"strings"
)

// version is a release number of either count: old marks one from before the
// restart at 1.0.0, which ranks below every number after it; beta is N of an
// -beta.N suffix, 0 for a release without one.
type version struct {
	old  bool
	num  [3]int
	beta int
}

// parseVersion reads "1.2.3", "v1.2.3" or "1.2.3-beta.4". Any other suffix,
// and a leading zero anywhere, is no version the updater acts on.
func parseVersion(s string) (version, bool) {
	core, suffix, hasSuffix := strings.Cut(strings.TrimPrefix(s, "v"), "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return version{}, false
	}
	var v version
	for i, part := range parts {
		n, ok := plainNumber(part)
		if !ok {
			return version{}, false
		}
		v.num[i] = n
	}
	if !hasSuffix {
		return v, true
	}
	n, ok := plainNumber(strings.TrimPrefix(suffix, "beta."))
	if !ok || n == 0 || !strings.HasPrefix(suffix, "beta.") {
		return version{}, false
	}
	v.beta = n
	return v, true
}

// plainNumber is a non-negative decimal without a leading zero.
func plainNumber(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	return n, err == nil && n >= 0 && s == strconv.Itoa(n)
}

// parseReported reads what a binary says it is after "loomux ": the version,
// optionally followed by " (beta)". A plain number with that channel is the
// old count; a beta of the new count carries its suffix and may say so too.
func parseReported(s string) (version, bool) {
	ver, channel, hasChannel := strings.Cut(s, " ")
	v, ok := parseVersion(ver)
	if !ok || (hasChannel && channel != "(beta)") {
		return version{}, false
	}
	v.old = hasChannel && v.beta == 0
	return v, true
}

// releaseVersion places a release in its count. A tag alone cannot: v8.0.0
// of the old count and v1.0.0 of the new look alike, but every release of
// the old count was published as a pre-release and every new pre-release
// carries -beta.N.
func releaseVersion(r Release) (version, bool) {
	v, ok := parseVersion(r.Tag)
	v.old = ok && r.Prerelease && v.beta == 0
	return v, ok
}

// less orders the old count below the new, then by number, then a beta
// below the release it leads up to and betas by their N.
func (v version) less(w version) bool {
	if v.old != w.old {
		return v.old
	}
	for i := range v.num {
		if v.num[i] != w.num[i] {
			return v.num[i] < w.num[i]
		}
	}
	switch {
	case v.beta == w.beta:
		return false
	case v.beta == 0:
		return false
	case w.beta == 0:
		return true
	}
	return v.beta < w.beta
}

// Reported is what a binary built with ver and channel says after "loomux ",
// the form Newer, AtLeast and IsVersion read.
func Reported(ver, channel string) string {
	if channel == "" || channel == "stable" {
		return ver
	}
	return ver + " (" + channel + ")"
}

// Newer reports whether r is a later release than the running binary, given
// as Reported. A side that does not parse is never newer, so a guess never
// replaces a binary and a development build never asks for one.
func Newer(r Release, running string) bool {
	t, ok := releaseVersion(r)
	if !ok {
		return false
	}
	have, ok := parseReported(running)
	return ok && have.less(t)
}

// AtLeast reports whether have is no older than want, both as Reported.
// Unlike !Newer it is false when either side does not parse: a binary that
// cannot say what it is does not pass for one new enough.
func AtLeast(have, want string) bool {
	h, ok := parseReported(have)
	if !ok {
		return false
	}
	w, ok := parseReported(want)
	return ok && !h.less(w)
}

// IsVersion reports whether s, as Reported, is a release version, as a
// development build's DevVersion is not.
func IsVersion(s string) bool {
	_, ok := parseReported(s)
	return ok
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
