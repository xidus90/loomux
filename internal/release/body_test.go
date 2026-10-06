package release

import (
	"reflect"
	"testing"
)

const goodBody = "Release: minor — adds export\r\n\r\n## Changelog\r\n### Added\r\n- `loomux graph export`\r\n\r\n### Fixed\r\n- a crash\r\n\r\n## Notes\r\nnot part of it\r\n"

func TestParseBodyTakesTheChangelogBlock(t *testing.T) {
	got, problems := ParseBody([]string{"bug", "release:minor"}, goodBody)
	want := Parsed{Bump: "minor", Changelog: "### Added\n- `loomux graph export`\n\n### Fixed\n- a crash\n"}
	if len(problems) != 0 || got != want {
		t.Fatalf("got %#v, %v", got, problems)
	}
}

func TestParseBodyNoneNeedsNoChangelog(t *testing.T) {
	got, problems := ParseBody([]string{"release:none"}, "")
	if len(problems) != 0 || got != (Parsed{Bump: "none"}) {
		t.Fatalf("got %#v, %v", got, problems)
	}
}

func TestParseBodyFindings(t *testing.T) {
	for name, tc := range map[string]struct {
		labels []string
		body   string
		want   []string
	}{
		"no label":      {nil, goodBody, []string{"exactly one release:* label is required, found 0"}},
		"two labels":    {[]string{"release:minor", "release:patch"}, goodBody, []string{"exactly one release:* label is required, found 2"}},
		"unknown label": {[]string{"release:huge"}, goodBody, []string{`unknown release label "release:huge"`}},
		"no block":      {[]string{"release:patch"}, "text", []string{"the body has no \"## Changelog\" section"}},
		"empty block":   {[]string{"release:patch"}, "## Changelog\n### Fixed\n", []string{"the changelog has no entry"}},
		"bad heading":   {[]string{"release:patch"}, "## Changelog\n### Misc\n- x\n", []string{`changelog heading "### Misc" is not one of Added, Changed, Deprecated, Removed, Fixed, Security`}},
		"orphan entry":  {[]string{"release:patch"}, "## Changelog\n- x\n", []string{"changelog entry \"- x\" stands before any ### heading", "the changelog has no entry"}},
	} {
		if _, got := ParseBody(tc.labels, tc.body); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}

func TestParseBodyRefusesStrayLines(t *testing.T) {
	body := "## Changelog\n# Title\n### Added\n- a\n#### x\nprose\n<br>\n\n"
	want := []string{
		`changelog line "# Title" is neither a ### heading nor a "- " entry`,
		`changelog line "#### x" is neither a ### heading nor a "- " entry`,
		`changelog line "prose" is neither a ### heading nor a "- " entry`,
		`changelog line "<br>" is neither a ### heading nor a "- " entry`,
	}
	if _, got := ParseBody([]string{"release:patch"}, body); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}

func TestParseBodyEndsTheBlockAtAnIndentedHeading(t *testing.T) {
	body := "## Changelog\n### Added\n- x\n  ## Notes\nsee below\n"
	got, problems := ParseBody([]string{"release:patch"}, body)
	want := Parsed{Bump: "patch", Changelog: "### Added\n- x\n"}
	if len(problems) != 0 || got != want {
		t.Fatalf("got %#v, %v", got, problems)
	}
}

func TestParseBodyKeepsAForgedHeadingOutOfTheChangelog(t *testing.T) {
	body := "## Changelog\n### Added\n- x\n  ## [9.9.9] - forged\n- y\n"
	got, problems := ParseBody([]string{"release:patch"}, body)
	want := Parsed{Bump: "patch", Changelog: "### Added\n- x\n"}
	if len(problems) != 0 || got != want {
		t.Fatalf("got %#v, %v", got, problems)
	}
}
