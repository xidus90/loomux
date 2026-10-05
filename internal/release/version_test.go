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

func TestNextBeta(t *testing.T) {
	for _, c := range []struct {
		tags []string
		bump string
		want string
	}{
		{nil, "minor", "1.0.0-beta.1"},
		{[]string{"v1.0.0"}, "minor", "1.1.0-beta.1"},
		{[]string{"v1.0.0", "v1.1.0-beta.1", "v1.1.0-beta.2"}, "minor", "1.1.0-beta.3"},
		{[]string{"v1.0.0", "v1.1.0-beta.3", "v1.1.0-beta.1"}, "minor", "1.1.0-beta.4"},
		{[]string{"v1.0.0", "v1.1.0-beta.9", "v1.1.0-beta.10"}, "minor", "1.1.0-beta.11"},
		{[]string{"v1.0.0", "v1.0.1-beta.4"}, "minor", "1.1.0-beta.1"},
		{[]string{"v1.0.0", "v1.1.0-beta.2", "v1.1.0-beta.99999999999999999999", "v1.1.0-beta.7x", "xv1.1.0-beta.8"}, "minor", "1.1.0-beta.3"},
		{[]string{"v1.0.0", "v1.1.0-beta.02", "v1.1.0-rc.5", "archive/parity-recordings"}, "minor", "1.1.0-beta.1"},
	} {
		got, err := NextBeta(c.tags, c.bump)
		if err != nil || got != c.want {
			t.Errorf("NextBeta(%v, %s) = %q, %v; want %q", c.tags, c.bump, got, err, c.want)
		}
	}
	if _, err := NextBeta(nil, "huge"); !errors.Is(err, ErrBump) {
		t.Fatalf("err = %v", err)
	}
}
