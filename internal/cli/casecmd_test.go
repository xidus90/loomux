package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// standingCase is a case file as the review centre holds it, the fields
// spelt the way `WriteCase` renders them. extra lands between the stamp and
// the sources, where the optional keys stand.
func standingCase(id, target, state, extra string) string {
	return "id = \"" + id + "\"\narea = \"project/a\"\ntarget = \"" + target +
		"\"\ntarget_hash = \"12345678\"\nstate = \"" + state +
		"\"\ntrigger = \"source_change\"\nweight = \"normal\"\ncreated = 2026-09-03T12:00:00Z\n" +
		extra + "\n[[sources]]\ndoc_id = \"01DOC1\"\nrevision = 1\ncontent_hash = \"abcdef12\"\n"
}

// placeCase writes one case directory beneath the review centre, with the
// files given beside `case.toml`, and answers the directory.
func placeCase(t *testing.T, w reconcileWorld, relative, caseText string, files map[string]string) string {
	t.Helper()
	directory := filepath.Join(w.Review, filepath.FromSlash(relative))
	writeFile(t, filepath.Join(directory, "case.toml"), caseText)
	for name, body := range files {
		writeFile(t, filepath.Join(directory, name), body)
	}
	return directory
}

func TestCasesListsEveryCaseInTheReferenceOrder(t *testing.T) {
	w := newReconcileWorld(t, reconcileOptions{})
	placeCase(t, w, "project-a/Zeta-case", standingCase("Zeta-case", "z.md", "due", ""), nil)
	placeCase(t, w, "project-a/alpha-case", standingCase("alpha-case", "a.md", "source_changed", "manual = true\n"), nil)
	placeCase(t, w, "project-a/moved/deeper", standingCase("deeper", "d.md", "due", ""), nil)
	code, out, errOut := run("cases")
	if code != 0 || errOut != "" {
		t.Fatalf("exit = %d, stderr = %q", code, errOut)
	}
	want := "alpha-case\tproject/a\ta.md\tsource_changed\tmanuell\n" +
		"deeper\tproject/a\td.md\tdue\n" +
		"Zeta-case\tproject/a\tz.md\tdue\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

// An empty review centre and one that does not exist yet both answer the same
// sentence: nothing is waiting.
func TestCasesSaysSoWhenNothingWaits(t *testing.T) {
	newReconcileWorld(t, reconcileOptions{})
	code, out, _ := run("cases")
	if code != 0 || out != "keine offenen Fälle\n" {
		t.Fatalf("exit = %d, stdout = %q", code, out)
	}
}

// The directory addresses the case; a field that parted ways with it is
// named on stderr, and the listing prints the directory.
func TestCasesWarnsAboutARenamedCase(t *testing.T) {
	w := newReconcileWorld(t, reconcileOptions{})
	directory := placeCase(t, w, "project-a/renamed-dir", standingCase("it's-old", "a.md", "due", ""), nil)
	code, out, errOut := run("cases")
	if code != 0 || out != "renamed-dir\tproject/a\ta.md\tdue\n" {
		t.Fatalf("exit = %d, stdout = %q", code, out)
	}
	want := "warning: " + filepath.Join(directory, "case.toml") +
		" calls itself \"it's-old\", but the case is addressed as 'renamed-dir'; the directory name is what counts\n"
	if errOut != want {
		t.Fatalf("stderr = %q, want %q", errOut, want)
	}
}

// A broken case file is named and scores the run, while the good one beside it
// is still listed; with nothing but broken files the queue is still empty.
func TestCasesReportsAnUnreadableCase(t *testing.T) {
	w := newReconcileWorld(t, reconcileOptions{})
	placeCase(t, w, "project-a/broken", "not toml at all [", nil)
	code, out, errOut := run("cases")
	if code != 1 || out != "keine offenen Fälle\n" || !strings.HasPrefix(errOut, "unreadable case: ") {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
	placeCase(t, w, "project-a/good", standingCase("good", "a.md", "due", ""), nil)
	code, out, _ = run("cases")
	if code != 1 || out != "good\tproject/a\ta.md\tdue\n" {
		t.Fatalf("exit = %d, stdout = %q", code, out)
	}
}

func TestCasesRefusesWithoutAReviewCentre(t *testing.T) {
	newReconcileWorld(t, reconcileOptions{NoReview: true})
	code, out, errOut := run("cases")
	if code != 1 || out != "" || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
}

func TestCasesRefusesABrokenRegistry(t *testing.T) {
	w := newReconcileWorld(t, reconcileOptions{})
	writeFile(t, filepath.Join(w.State, "registry.toml"), "[[area]\n")
	if code, _, errOut := run("cases"); code != 1 || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("exit = %d, stderr = %q", code, errOut)
	}
}

func TestCasesRefusesArguments(t *testing.T) {
	newReconcileWorld(t, reconcileOptions{})
	for _, args := range [][]string{{"cases", "extra"}, {"cases", "--nope"}} {
		if code, _, _ := run(args...); code != 2 {
			t.Fatalf("%q: exit = %d, want 2", args, code)
		}
	}
}

// caseHead is the part of `case` every open case in these tests prints before
// its files.
const caseHead = "Fall open-case (source_changed, ausgelöst durch source_change, Gewicht normal)\n" +
	"Bereich: project/a\nZiel: a.md\nQuelle: 01DOC1 (Revision 1, abcdef12)\n"

func TestCaseShowsAnOpenCaseWithItsFiles(t *testing.T) {
	w := newReconcileWorld(t, reconcileOptions{})
	placeCase(t, w, "project-a/open-case",
		standingCase("open-case", "a.md", "source_changed",
			"note = \"rebuilt\"\nsuperseded_proposal = \"2026-09-02\"\nmanual = true\n"),
		map[string]string{
			"package.md":             "# Paket\r\n",
			"proposal.md":            "# Vorschlag\n",
			"superseded-proposal.md": "# Alt\n",
		})
	code, out, errOut := run("case", "open-case")
	if code != 0 || errOut != "" {
		t.Fatalf("exit = %d, stderr = %q", code, errOut)
	}
	want := caseHead +
		"manuell: für diesen Fall wird kein Skill-Pfad angeboten\n" +
		"Vermerk: rebuilt\n" +
		"\n===== package.md =====\n# Paket\n" +
		"\n===== proposal.md =====\n# Vorschlag\n" +
		"\n===== superseded-proposal.md =====\n# Alt\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

// A file the case lacks is named, with its full path, rather than skipped.
func TestCaseNamesAMissingFile(t *testing.T) {
	w := newReconcileWorld(t, reconcileOptions{})
	directory := placeCase(t, w, "project-a/open-case", standingCase("open-case", "a.md", "source_changed", ""),
		map[string]string{"package.md": "# Paket\n"})
	code, out, _ := run("case", "open-case")
	want := caseHead + "\n===== package.md =====\n# Paket\n" +
		"\n===== proposal.md =====\n(nicht vorhanden: " + filepath.Join(directory, "proposal.md") + ")\n"
	if code != 0 || out != want {
		t.Fatalf("exit = %d, stdout = %q, want %q", code, out, want)
	}
}

// The three ways a case is withheld: its own mark, the area's current mode,
// and no declaration to ask at all.
func TestCaseWithholdsTheFilesOfAClosedCase(t *testing.T) {
	withheld := "local_only: der Quelldiff dieses Bereichs darf kein Cloud-Modell erreichen — " +
		"kein Skill-Pfad in einer Wolkensitzung\n"
	unknown := "Datenschutzmodus unbekannt: Der Bereich ist nicht registriert oder hat kein " +
		"lesbares Manifest und gilt deshalb als local_only\n"
	block := "\n===== zurückgehalten =====\n" +
		"Die Falldateien bleiben ungedruckt: Der Quelldiff dieses Bereichs darf kein Cloud-Modell " +
		"erreichen, und ein Vorschlag zitiert ihn wörtlich.\n" +
		"Im Prüfzentrum unter: project-a/moved/open-case\n" +
		"Bewusst ausgeben: loomux case --package open-case\n"
	for _, tc := range []struct {
		name, mode, extra, area, lines string
	}{
		{name: "marked", extra: "local_only = true\n", lines: withheld},
		{name: "area closed", mode: "local_only", lines: withheld},
		{name: "unregistered area", area: "project/gone", lines: withheld + unknown},
		{name: "manual too", extra: "local_only = true\nmanual = true\n",
			lines: withheld + "manuell: für diesen Fall wird kein Skill-Pfad angeboten\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newReconcileWorld(t, reconcileOptions{PrivacyMode: tc.mode})
			text := standingCase("open-case", "a.md", "source_changed", tc.extra)
			head := caseHead
			if tc.area != "" {
				text = strings.Replace(text, "project/a", tc.area, 1)
				head = strings.Replace(head, "project/a", tc.area, 1)
			}
			placeCase(t, w, "project-a/moved/open-case", text,
				map[string]string{"package.md": "# Paket\n", "proposal.md": "# Vorschlag\n"})
			code, out, _ := run("case", "open-case")
			if want := head + tc.lines + block; code != 0 || out != want {
				t.Fatalf("exit = %d, stdout = %q, want %q", code, out, want)
			}
		})
	}
}

// `--package` lifts the barrier, before or after the id, as argparse reads
// it; the halt line stays above the files.
func TestCasePackagePrintsTheWithheldFiles(t *testing.T) {
	w := newReconcileWorld(t, reconcileOptions{PrivacyMode: "local_only"})
	placeCase(t, w, "project-a/open-case", standingCase("open-case", "a.md", "source_changed", ""),
		map[string]string{"package.md": "# Paket\n", "proposal.md": "# Vorschlag\n"})
	want := caseHead +
		"local_only: der Quelldiff dieses Bereichs darf kein Cloud-Modell erreichen — kein Skill-Pfad in einer Wolkensitzung\n" +
		"\n===== package.md =====\n# Paket\n" +
		"\n===== proposal.md =====\n# Vorschlag\n"
	for _, args := range [][]string{
		{"case", "--package", "open-case"},
		{"case", "open-case", "--package"},
		{"case", "-package", "open-case"},
	} {
		code, out, errOut := run(args...)
		if code != 0 || out != want {
			t.Fatalf("%q: exit = %d, stdout = %q, stderr = %q", args, code, out, errOut)
		}
	}
}

// The rename warning of the listing stands before `case` prints anything.
func TestCaseWarnsAboutARenamedCase(t *testing.T) {
	w := newReconcileWorld(t, reconcileOptions{})
	placeCase(t, w, "project-a/open-case", standingCase("other", "a.md", "source_changed", ""), nil)
	code, _, errOut := run("case", "open-case")
	if code != 0 || !strings.Contains(errOut, "calls itself 'other', but the case is addressed as 'open-case'") {
		t.Fatalf("exit = %d, stderr = %q", code, errOut)
	}
}

func TestCaseRefusesAnUnknownOrAmbiguousID(t *testing.T) {
	w := newReconcileWorld(t, reconcileOptions{})
	first := placeCase(t, w, "project-a/dup", standingCase("dup", "a.md", "due", ""), nil)
	second := placeCase(t, w, "project-b/dup", standingCase("dup", "b.md", "due", ""), nil)

	code, out, errOut := run("case", "nonexistent-id")
	if want := "error: no case named 'nonexistent-id' in the review centre at " + w.Review + "\n"; code != 1 || out != "" || errOut != want {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q, want %q", code, out, errOut, want)
	}
	code, _, errOut = run("case", "dup")
	if want := "error: no case named 'dup' unambiguously; " + first + " and " + second + " both carry it\n"; code != 1 || errOut != want {
		t.Fatalf("exit = %d, stderr = %q, want %q", code, errOut, want)
	}
}

func TestCaseRefusesWhatItCannotRead(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, w reconcileWorld)
	}{
		{"no review centre", func(t *testing.T, w reconcileWorld) {
			writeFile(t, filepath.Join(w.Area, ".loomux", "config.toml"),
				"[area]\nscope = \"project/a\"\n")
		}},
		{"broken registry", func(t *testing.T, w reconcileWorld) {
			writeFile(t, filepath.Join(w.State, "registry.toml"), "[[area]\n")
		}},
		{"broken case file", func(t *testing.T, w reconcileWorld) {
			placeCase(t, w, "project-a/open-case", "state = 1\n", nil)
		}},
		{"package not UTF-8", func(t *testing.T, w reconcileWorld) {
			placeCase(t, w, "project-a/open-case", standingCase("open-case", "a.md", "due", ""),
				map[string]string{"package.md": "\xff\n"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newReconcileWorld(t, reconcileOptions{})
			placeCase(t, w, "project-a/other", standingCase("other", "a.md", "due", ""), nil)
			tc.setup(t, w)
			if code, _, errOut := run("case", "open-case"); code != 1 || !strings.HasPrefix(errOut, "error: ") {
				t.Fatalf("exit = %d, stderr = %q", code, errOut)
			}
		})
	}
}

// A declaration that does not read is refused before any case is looked up:
// ReviewRoot reads every area's declaration, the case's own among them.
func TestCaseRefusesABrokenDeclaration(t *testing.T) {
	w := newReconcileWorld(t, reconcileOptions{})
	placeCase(t, w, "project-a/open-case", standingCase("open-case", "a.md", "due", ""), nil)
	writeFile(t, filepath.Join(w.Area, ".loomux", "config.toml"), "[area\n")
	if code, _, errOut := run("case", "open-case"); code != 1 || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("exit = %d, stderr = %q", code, errOut)
	}
}

func TestCaseRefusesBadArguments(t *testing.T) {
	newReconcileWorld(t, reconcileOptions{})
	for _, args := range [][]string{
		{"case"},
		{"case", "a", "b"},
		{"case", "--nope", "a"},
		{"case", "a", "--package=maybe"},
	} {
		code, out, errOut := run(args...)
		if code != 2 || out != "" || errOut == "" {
			t.Fatalf("%q: exit = %d, stdout = %q, stderr = %q", args, code, out, errOut)
		}
	}
}

// After `--` everything is the id, however it looks.
func TestCaseReadsEverythingAfterTheTerminatorAsTheID(t *testing.T) {
	w := newReconcileWorld(t, reconcileOptions{})
	placeCase(t, w, "project-a/--package", standingCase("--package", "a.md", "due", ""), nil)
	code, out, errOut := run("case", "--", "--package")
	if code != 0 || !strings.HasPrefix(out, "Fall --package (due,") {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
}
