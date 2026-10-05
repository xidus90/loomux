package selfupdate

import "testing"

func TestParseVersion(t *testing.T) {
	for _, c := range []struct {
		in   string
		want version
		ok   bool
	}{
		{"1.2.3", version{num: [3]int{1, 2, 3}}, true},
		{"v1.2.3", version{num: [3]int{1, 2, 3}}, true},
		{"1.2.3-beta.4", version{num: [3]int{1, 2, 3}, beta: 4}, true},
		{"1.2.3-beta.10", version{num: [3]int{1, 2, 3}, beta: 10}, true},
		{"1.2.3-beta.0", version{}, false},
		{"1.2.3-beta.-1", version{}, false},
		{"1.2.3-4", version{}, false},
		{"1.2.3-beta.04", version{}, false},
		{"1.2.3-beta", version{}, false},
		{"1.2.3-rc1", version{}, false},
		{"0.0.0-dev", version{}, false},
		{"1.02.3", version{}, false},
		{"1.2", version{}, false},
		{"", version{}, false},
	} {
		got, ok := parseVersion(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parseVersion(%q) = %+v, %v; want %+v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestParseReportedKnowsTheOldCount(t *testing.T) {
	for _, c := range []struct {
		in   string
		want version
		ok   bool
	}{
		{"7.2.0 (beta)", version{old: true, num: [3]int{7, 2, 0}}, true},
		{"7.2.0", version{num: [3]int{7, 2, 0}}, true},
		{"1.1.0-beta.1", version{num: [3]int{1, 1, 0}, beta: 1}, true},
		{"1.1.0-beta.1 (beta)", version{num: [3]int{1, 1, 0}, beta: 1}, true},
		{"7.2.0 (nightly)", version{}, false},
		{"7.2.0 (beta) x", version{}, false},
		{"0.0.0-dev", version{}, false},
	} {
		got, ok := parseReported(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parseReported(%q) = %+v, %v; want %+v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestReleaseVersionReadsTheCountFromThePrereleaseFlag(t *testing.T) {
	for _, c := range []struct {
		r    Release
		want version
	}{
		{Release{Tag: "v7.1.0", Prerelease: true}, version{old: true, num: [3]int{7, 1, 0}}},
		{Release{Tag: "v1.0.0"}, version{num: [3]int{1, 0, 0}}},
		{Release{Tag: "v1.1.0-beta.2", Prerelease: true}, version{num: [3]int{1, 1, 0}, beta: 2}},
	} {
		if got, ok := releaseVersion(c.r); !ok || got != c.want {
			t.Errorf("releaseVersion(%+v) = %+v, %v; want %+v", c.r, got, ok, c.want)
		}
	}
	if _, ok := releaseVersion(Release{Tag: "nightly"}); ok {
		t.Error("nightly read as a version")
	}
}

func TestLessOrdersBothCounts(t *testing.T) {
	ordered := []string{"7.1.0 (beta)", "7.2.0 (beta)", "1.0.0", "1.1.0-beta.2", "1.1.0-beta.10", "1.1.0", "1.1.1", "2.0.0"}
	for i := range ordered {
		for j := range ordered {
			a, _ := parseReported(ordered[i])
			b, _ := parseReported(ordered[j])
			if got := a.less(b); got != (i < j) {
				t.Errorf("%s < %s = %v, want %v", ordered[i], ordered[j], got, i < j)
			}
		}
	}
}

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		r       Release
		running string
		want    bool
	}{
		{Release{Tag: "v2.8.0", Prerelease: true}, "2.7.0 (beta)", true},
		{Release{Tag: "v2.7.0", Prerelease: true}, "2.7.0 (beta)", false},
		{Release{Tag: "v1.0.0"}, "7.2.0 (beta)", true},
		{Release{Tag: "v7.1.0", Prerelease: true}, "1.0.0", false},
		{Release{Tag: "v1.1.0"}, "1.1.0-beta.3", true},
		{Release{Tag: "v1.1.0-beta.3", Prerelease: true}, "1.1.0", false},
		{Release{Tag: "v2.8.0-rc1"}, "2.7.0", false},
		{Release{Tag: "v2.8.0"}, "0.0.0-dev", false},
		{Release{Tag: "nightly"}, "7.2.0 (beta)", false},
	} {
		if got := Newer(c.r, c.running); got != c.want {
			t.Errorf("Newer(%+v, %q) = %v, want %v", c.r, c.running, got, c.want)
		}
	}
}

func TestAtLeast(t *testing.T) {
	for _, c := range []struct {
		have, want string
		ok         bool
	}{
		{"2.13.0 (beta)", "2.13.0 (beta)", true},
		{"2.11.1 (beta)", "2.13.0 (beta)", false},
		{"7.2.0 (beta)", "1.0.0", false},
		{"1.0.0", "7.2.0 (beta)", true},
		{"1.1.0-beta.1", "1.1.0", false},
		{"", "2.13.0", false},
		{"", "7.2.0 (beta)", false},
		{"2.13.0", "0.0.0-dev", false},
	} {
		if got := AtLeast(c.have, c.want); got != c.ok {
			t.Errorf("AtLeast(%q, %q) = %v, want %v", c.have, c.want, got, c.ok)
		}
	}
	for s, want := range map[string]bool{"2.13.0": true, "7.2.0 (beta)": true, "1.1.0-beta.1": true, DevVersion: false, "": false} {
		if got := IsVersion(s); got != want {
			t.Errorf("IsVersion(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestReported(t *testing.T) {
	for _, c := range [][3]string{{"7.2.0", "beta", "7.2.0 (beta)"}, {"1.0.0", "", "1.0.0"}, {"1.0.0", "stable", "1.0.0"}} {
		if got := Reported(c[0], c[1]); got != c[2] {
			t.Errorf("Reported(%q, %q) = %q, want %q", c[0], c[1], got, c[2])
		}
	}
}

// Every release so far is a pre-release: release.sh marks each one so while
// RELEASE_CHANNEL is unset. A beta binary must still find them.
func TestPickFollowsTheChannel(t *testing.T) {
	all := []Release{
		{Tag: "v2.6.0", Prerelease: true},
		{Tag: "v2.10.0", Prerelease: true},
		{Tag: "v2.7.0", Prerelease: false},
		{Tag: "nightly", Prerelease: true},
	}
	onlyPre := []Release{{Tag: "v2.7.0", Prerelease: true}, {Tag: "v2.6.0", Prerelease: true}}
	for _, c := range []struct {
		name     string
		releases []Release
		channel  string
		want     string
		found    bool
	}{
		{"beta takes the highest of all", all, "beta", "v2.10.0", true},
		{"stable skips pre-releases", all, "stable", "v2.7.0", true},
		{"no channel reads as stable", all, "", "v2.7.0", true},
		{"beta over today's releases", onlyPre, "beta", "v2.7.0", true},
		{"stable over today's releases", onlyPre, "stable", "", false},
		{"nothing listed", nil, "beta", "", false},
		{"no tag is a version", []Release{{Tag: "nightly", Prerelease: true}, {Tag: "v2.8"}}, "beta", "", false},
	} {
		got, found := pick(c.releases, c.channel)
		if found != c.found || got.Tag != c.want {
			t.Errorf("%s: pick = %q, %v; want %q, %v", c.name, got.Tag, found, c.want, c.found)
		}
	}
}
