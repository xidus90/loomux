package selfupdate

import "testing"

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		tag, running string
		want         bool
	}{
		{"v2.8.0", "2.7.0", true},
		{"2.8.0", "2.7.0", true},
		{"v10.0.0", "9.9.9", true},
		{"v2.7.0", "2.7.0", false},
		{"v2.6.9", "2.7.0", false},
		{"v2.8.0-rc1", "2.7.0", false},
		{"v2.8", "2.7.0", false},
		{"v2.08.0", "2.7.0", false},
		{"v-1.0.0", "0.0.0", false},
		{"v2.8.0", "0.0.0-dev", false},
	} {
		if got := Newer(c.tag, c.running); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.tag, c.running, got, c.want)
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
