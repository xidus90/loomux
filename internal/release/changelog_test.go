package release

import (
	"errors"
	"testing"
)

func TestInsertChangelogCreatesTheFile(t *testing.T) {
	got, err := InsertChangelog(nil, "1.0.0", "2026-09-18", "https://x/pull/1", "### Added\n- a\n")
	want := ChangelogHeader + "\n## [1.0.0] - 2026-09-18\n\n<https://x/pull/1>\n\n### Added\n- a\n"
	if err != nil || string(got) != want {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestInsertChangelogPutsTheNewestFirst(t *testing.T) {
	old := ChangelogHeader + "\n## [1.0.0] - 2026-09-18\n\n<l1>\n\n### Added\n- a\n"
	got, err := InsertChangelog([]byte(old), "1.0.1", "2026-09-19", "l2", "### Fixed\n- b\n")
	want := ChangelogHeader + "\n## [1.0.1] - 2026-09-19\n\n<l2>\n\n### Fixed\n- b\n" +
		"\n## [1.0.0] - 2026-09-18\n\n<l1>\n\n### Added\n- a\n"
	if err != nil || string(got) != want {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestInsertChangelogAppendsBelowAHeaderWithoutEntries(t *testing.T) {
	got, err := InsertChangelog([]byte("# Changelog\n"), "1.0.0", "d", "l", "### Added\n- a\n")
	if err != nil || string(got) != "# Changelog\n\n## [1.0.0] - d\n\n<l>\n\n### Added\n- a\n" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestInsertChangelogRefusesADuplicate(t *testing.T) {
	old := ChangelogHeader + "\n## [1.0.0] - 2026-09-18\n"
	if _, err := InsertChangelog([]byte(old), "1.0.0", "d", "l", "n"); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("err %v", err)
	}
}

func TestInsertChangelogRefusesADuplicateAtTheStartOfTheFile(t *testing.T) {
	old := "## [1.0.0] - 2026-09-18\n\n<l1>\n\n### Added\n- a\n"
	if _, err := InsertChangelog([]byte(old), "1.0.0", "d", "l", "n"); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("err %v", err)
	}
}

func TestInsertChangelogPutsTheNewestFirstInAFileWithoutHeader(t *testing.T) {
	old := "## [1.0.0] - 2026-09-18\n\n<l1>\n\n### Added\n- a\n"
	got, err := InsertChangelog([]byte(old), "1.0.1", "2026-09-19", "l2", "### Fixed\n- b\n")
	want := "## [1.0.1] - 2026-09-19\n\n<l2>\n\n### Fixed\n- b\n\n" + old
	if err != nil || string(got) != want {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestInsertChangelogPutsTheNewestFirstAboveTwoHeaderlessEntries(t *testing.T) {
	old := "## [1.0.1] - b\n\n<l2>\n\nx\n\n## [1.0.0] - a\n\n<l1>\n\ny\n"
	got, err := InsertChangelog([]byte(old), "1.0.2", "c", "l3", "z\n")
	want := "## [1.0.2] - c\n\n<l3>\n\nz\n\n" + old
	if err != nil || string(got) != want {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestInsertChangelogTreatsAnEmptyFileAsMissing(t *testing.T) {
	got, err := InsertChangelog([]byte{}, "1.0.0", "d", "l", "### Added\n- a\n")
	if err != nil || string(got) != ChangelogHeader+"\n## [1.0.0] - d\n\n<l>\n\n### Added\n- a\n" {
		t.Fatalf("got %q, %v", got, err)
	}
}
