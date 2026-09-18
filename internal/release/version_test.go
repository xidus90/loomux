package release

import (
	"errors"
	"testing"
)

func TestNextVersion(t *testing.T) {
	tags := []string{"v1.2.3", "v1.10.0", "v1.10.0", "v1.9.9", "v2.0.0-rc.1", "latest", "v01.0.0", "g2a-before-rebase"}
	for bump, want := range map[string]string{"major": "2.0.0", "minor": "1.11.0", "patch": "1.10.1"} {
		got, err := NextVersion(tags, bump)
		if err != nil || got != want {
			t.Fatalf("%s: got %q, %v; want %q", bump, got, err, want)
		}
	}
}

func TestNextVersionStartsAtOneWithoutATag(t *testing.T) {
	for _, bump := range []string{"major", "minor", "patch"} {
		if got, err := NextVersion([]string{"nightly", ""}, bump); err != nil || got != "1.0.0" {
			t.Fatalf("%s: got %q, %v", bump, got, err)
		}
	}
}

func TestNextVersionRefusesAnUnknownBump(t *testing.T) {
	if _, err := NextVersion(nil, "none"); !errors.Is(err, ErrBump) {
		t.Fatalf("err %v", err)
	}
}

func TestNextVersionSkipsAnOverflowingTag(t *testing.T) {
	if got, err := NextVersion([]string{"v1.0.0", "v99999999999999999999.0.0"}, "patch"); err != nil || got != "1.0.1" {
		t.Fatalf("got %q, %v", got, err)
	}
}
