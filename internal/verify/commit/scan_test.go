package commit_test

import (
	"regexp"
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/verify/commit"
)

func TestScanEnglishMessageIsClean(t *testing.T) {
	text := "Let the stop gate run one profile instead of the whole chain"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanGermanProseIsFound(t *testing.T) {
	text := "Das Gate laeuft jetzt mit dem Profil und nicht mehr ueber die ganze Kette"
	findings := commit.Scan(text, "en", 2, nil)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].LineNumber != 1 {
		t.Errorf("expected line 1, got %d", findings[0].LineNumber)
	}
	if len(findings[0].Hits) < 2 {
		t.Errorf("expected at least 2 hits, got %v", findings[0].Hits)
	}
}

func TestScanThresholdCountsPerLine(t *testing.T) {
	text := "Add a page\n\nSee der Titel\nSee das Andere"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanOneHitInALineIsNotEnough(t *testing.T) {
	text := "Rename the file to konzept-der-woche.md"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanQuotedSentenceDoesNotCount(t *testing.T) {
	text := `The page says "der Bericht ist nicht vollstaendig" and it is right`
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanCodeSpanDoesNotCount(t *testing.T) {
	text := "Rename `der_alte_name` to `the_new_name` and nicht more"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanPathDoesNotCount(t *testing.T) {
	text := "Move wiki/decisions/das-und-der-fall.md into the archive"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanTrailerDoesNotCount(t *testing.T) {
	text := "Fix the gate\n\nCo-Authored-By: Der Name <von@example.org>"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanNameParticleDoesNotCount(t *testing.T) {
	text := "The paper by von Neumann and von Braun describes the algorithm"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanVonWithoutCapitalizedNameCounts(t *testing.T) {
	text := "Das Ergebnis von dem Bericht und von der Pruefung"
	findings := commit.Scan(text, "en", 2, nil)
	if len(findings) != 1 || findings[0].LineNumber != 1 {
		t.Errorf("expected finding on line 1, got %v", findings)
	}
}

func TestScanDiffBelowScissorsIsIgnored(t *testing.T) {
	text := "Add the page\n# ------------------------ >8 ------------------------\ndiff --git a/wiki/x.md b/wiki/x.md\n+der Bericht und das Ergebnis sind nicht vollstaendig\n"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanCommentLinesAreIgnored(t *testing.T) {
	text := "Add the page\n# Bitte gib eine Commit-Beschreibung fuer die Aenderungen ein\n"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanOtherDirectionFindsEnglishInGerman(t *testing.T) {
	text := "The gate now runs with the profile and not with the whole chain"
	findings := commit.Scan(text, "de", 2, nil)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

func TestScanGermanMessageCleanUnderDE(t *testing.T) {
	text := "Das Gate laeuft jetzt mit dem Profil statt ueber die ganze Kette"
	if findings := commit.Scan(text, "de", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanAllowPatternDropsLine(t *testing.T) {
	text := "Add the page\nQuelle: der Bericht und das Ergebnis"
	allow := []*regexp.Regexp{regexp.MustCompile(`^Quelle:`)}
	if findings := commit.Scan(text, "en", 2, allow); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanFindingCarriesLineAndHits(t *testing.T) {
	text := "Add a page\nDas Ergebnis und der Bericht fehlen"
	findings := commit.Scan(text, "en", 2, nil)
	if len(findings) != 1 || findings[0].LineNumber != 2 {
		t.Fatalf("expected finding on line 2, got %v", findings)
	}
	if !slices.Contains(findings[0].Hits, "und") {
		t.Errorf("expected 'und' in hits, got %v", findings[0].Hits)
	}
}

func TestScanUmlautsFoundAlthoughListIsASCII(t *testing.T) {
	for _, text := range []string{
		"Für alles über allem",
		"Über die Katze, für sich genommen",
	} {
		findings := commit.Scan(text, "en", 2, nil)
		if len(findings) != 1 || findings[0].LineNumber != 1 {
			t.Errorf("expected finding on line 1 for %q, got %v", text, findings)
		}
	}
}

func TestScanHyphenatedTrailerDoesNotCount(t *testing.T) {
	text := "Fix the gate\n\nSigned-off-by: Der Name und der Andere"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanListedUnhyphenatedTrailerDoesNotCount(t *testing.T) {
	for _, key := range []string{"Fixes", "Closes", "Refs", "Ref", "Cc", "Link", "Bug", "BREAKING CHANGE"} {
		text := "Fix the gate\n\n" + key + ": das und der Bericht"
		if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
			t.Errorf("expected 0 findings for %q, got %v", key, findings)
		}
	}
}

func TestScanConventionalCommitSubjectIsNotTrailer(t *testing.T) {
	for _, subject := range []string{"fix: ", "Fix: ", "docs: ", "chore: ", "Note: "} {
		text := subject + "behebt den Fehler und das Problem"
		findings := commit.Scan(text, "en", 2, nil)
		if len(findings) != 1 || findings[0].LineNumber != 1 {
			t.Errorf("expected finding on line 1 for %q, got %v", subject, findings)
		}
	}
}

func TestScanEnglishFestIsNotFinding(t *testing.T) {
	text := "Add the fest and the beer fest to the calendar"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanNoTrailerExemptOnFirstLine(t *testing.T) {
	for _, key := range []string{"Ref", "Fixes", "Co-Authored-By", "Auto-merge", "Feature-flag", "BREAKING CHANGE"} {
		text := key + ": behebt den Fehler und das Problem"
		findings := commit.Scan(text, "en", 2, nil)
		if len(findings) != 1 || findings[0].LineNumber != 1 {
			t.Errorf("expected finding on line 1 for %q, got %v", key, findings)
		}
	}
}

func TestScanBreakingChangeFooterDoesNotCount(t *testing.T) {
	text := "Change the gate\n\nBREAKING CHANGE: das Verhalten und der Vertrag aendern sich"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanGermanStillIsNotFindingUnderDE(t *testing.T) {
	text := "Lasse den Prozess still laufen und still beenden"
	if findings := commit.Scan(text, "de", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanWrappedCodeSpanIsExempt(t *testing.T) {
	text := "Widen the gate\n\nreal trailer key and a perfectly good subject: `Ref: behebt den Fehler und das\nProblem` and any capitalised hyphenated first word"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanClosingBacktickWithNoOpenerLeavesTextScored(t *testing.T) {
	text := "Widen the gate\n\nund der Bericht` shows what the gate printed"
	findings := commit.Scan(text, "en", 2, nil)
	if len(findings) != 1 || findings[0].LineNumber != 3 {
		t.Errorf("expected finding on line 3, got %v", findings)
	}
}

func TestScanBalancedSpanIsLeftAlone(t *testing.T) {
	if findings := commit.Scan("Report `das und der` in the output", "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
	findings := commit.Scan("Report `x` und der Bericht das", "en", 2, nil)
	if len(findings) != 1 || findings[0].LineNumber != 1 {
		t.Errorf("expected finding on line 1, got %v", findings)
	}
}

func TestScanLoneTrailingBacktickStripsNothing(t *testing.T) {
	findings := commit.Scan("Der Bericht und das Ergebnis `", "en", 2, nil)
	if len(findings) != 1 || findings[0].LineNumber != 1 {
		t.Errorf("expected finding on line 1, got %v", findings)
	}
}

func TestScanTailOfWrappedCodeSpanIsExempt(t *testing.T) {
	text := "Widen the gate\n\nThe subject was `Ref: behebt den Fehler\nund das Problem` and the gate said nothing"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanCodeSpanWrappingThreeLinesIsExempt(t *testing.T) {
	text := "Widen the gate\n\nThe subject was `Ref: behebt den Fehler\nund das Problem und der Bericht\nund die Pruefung` and the gate said nothing"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanQuotedSpanDoesNotWrapAcrossLines(t *testing.T) {
	text := "Widen the gate\n\nHe said \"es behebt den Fehler und das\nProblem und der Bericht\" and left"
	findings := commit.Scan(text, "en", 2, nil)
	if len(findings) != 2 || findings[0].LineNumber != 3 || findings[1].LineNumber != 4 {
		t.Errorf("expected findings on lines 3 and 4, got %v", findings)
	}
}

func TestScanQuotedSpanWithinOneLineIsStillExempt(t *testing.T) {
	text := `He said "das und der" and left`
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanLoneQuoteIsPunctuation(t *testing.T) {
	text := `Set the width to 80" und der Bericht das`
	findings := commit.Scan(text, "en", 2, nil)
	if len(findings) != 1 || findings[0].LineNumber != 1 {
		t.Errorf("expected finding on line 1, got %v", findings)
	}
}

func TestScanApostropheIsNotQuoteDelimiter(t *testing.T) {
	text := "The gate don't und der Bericht das care"
	findings := commit.Scan(text, "en", 2, nil)
	if len(findings) != 1 || findings[0].LineNumber != 1 {
		t.Errorf("expected finding on line 1, got %v", findings)
	}
}

func TestScanGitHintLineDoesNotMoveSpanFlags(t *testing.T) {
	noise := "Widen the gate\n\n# On branch feat/x -- use `git add` to stage\n# Changes not staged for commit: `\nDer Bericht und das Ergebnis fehlen"
	findings := commit.Scan(noise, "en", 2, nil)
	if len(findings) != 1 || findings[0].LineNumber != 5 {
		t.Errorf("expected finding on line 5, got %v", findings)
	}

	spanning := "Widen the gate\n\nHe wrote `Ref: behebt den Fehler\n# a git hint with a stray ` backtick\nund das Problem` and stopped"
	if findings := commit.Scan(spanning, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanExemptedLineCarriesSpanOnward(t *testing.T) {
	allow := []*regexp.Regexp{regexp.MustCompile(`^WIP`)}
	text := "Widen the gate\n\nWIP `Ref: behebt den Fehler\nund das Problem` and stopped"
	if findings := commit.Scan(text, "en", 2, allow); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanUnpairedQuoteDoesNotOutliveParagraph(t *testing.T) {
	text := "Add a 80\" wide banner\n\nDas Ergebnis und der Bericht fehlen\nDas Verhalten und der Vertrag aendern sich"
	findings := commit.Scan(text, "en", 2, nil)
	if len(findings) != 2 || findings[0].LineNumber != 3 || findings[1].LineNumber != 4 {
		t.Errorf("expected findings on lines 3 and 4, got %v", findings)
	}
}

func TestScanUnpairedBacktickDoesNotOutliveParagraph(t *testing.T) {
	text := "Add a 80` wide banner\n\nDas Ergebnis und der Bericht fehlen\nDas Verhalten und der Vertrag aendern sich"
	findings := commit.Scan(text, "en", 2, nil)
	if len(findings) != 2 || findings[0].LineNumber != 3 || findings[1].LineNumber != 4 {
		t.Errorf("expected findings on lines 3 and 4, got %v", findings)
	}
}

func TestScanSpanDoesNotWrapAcrossBlankLine(t *testing.T) {
	code := "Widen the gate\n\nHe wrote `Ref: behebt den Fehler\n\nund das Problem` and stopped"
	findings := commit.Scan(code, "en", 2, nil)
	if len(findings) != 1 || findings[0].LineNumber != 5 {
		t.Errorf("expected finding on line 5, got %v", findings)
	}
}

func TestScanRemovedSpanLeavesSeparator(t *testing.T) {
	text := "Fix un`x`d der parser"
	if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

func TestScanTailOfCarriedSpanCannotPassAsTrailer(t *testing.T) {
	text := "Widen the gate\n\nHe wrote `something\nx`Ref: das und der Bericht"
	findings := commit.Scan(text, "en", 2, nil)
	if len(findings) != 1 || findings[0].LineNumber != 4 {
		t.Errorf("expected finding on line 4, got %v", findings)
	}
}

func TestScanStrayBacktickInSubjectDoesNotSilenceBody(t *testing.T) {
	text := "Add a 80` wide banner\nDas Ergebnis und der Bericht fehlen"
	findings := commit.Scan(text, "en", 2, nil)
	if len(findings) != 1 || findings[0].LineNumber != 2 {
		t.Errorf("expected finding on line 2, got %v", findings)
	}
}

func TestScanNonLatinScripts(t *testing.T) {
	// Two foreign script words clear threshold 2
	findings := commit.Scan("Fix the parser\n\n修复解析 器错误", "en", 2, nil)
	if len(findings) != 1 || findings[0].LineNumber != 3 {
		t.Errorf("expected finding on line 3, got %v", findings)
	}

	// One foreign word stays under threshold 2
	if findings := commit.Scan("Rename the 北京 constant", "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}

	// Single script run counts once however long
	if findings := commit.Scan("Fix\n\n修复解析器错误修复", "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}

	// Latin with diacritics is not a script hit
	if findings := commit.Scan("Add a café fixture and a naïve retry", "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}

	// Covered scripts produce hits
	samples := map[string]string{
		"Han":        "修复 解析",
		"Hiragana":   "これは それは",
		"Katakana":   "パーサ エラー",
		"Hangul":     "파서 오류",
		"Cyrillic":   "исправление ошибки",
		"Arabic":     "إصلاح الخطأ",
		"Hebrew":     "תיקון שגיאה",
		"Greek":      "διόρθωση σφάλματος",
		"Devanagari": "त्रुटि सुधार",
		"Thai":       "แก้ไข ข้อผิด",
	}
	for name, sample := range samples {
		f := commit.Scan("Fix\n\n"+sample, "en", 2, nil)
		if len(f) == 0 {
			t.Errorf("script %s produced no findings", name)
		}
	}

	// Japanese loanword with prolonged sound mark \u30FC is 1 CJK run
	if findings := commit.Scan("Rename the \u30d1\u30fc\u30b5 constant", "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings for single Japanese word, got %v", findings)
	}
	// Japanese sentence refused
	if findings := commit.Scan("Fix\n\n\u3053\u308c\u306f \u30d1\u30fc\u30b5\u306e \u30a8\u30e9\u30fc", "en", 2, nil); len(findings) == 0 {
		t.Errorf("expected findings for Japanese sentence, got 0")
	}

	// Fullwidth and mathematical Latin fold to Latin
	if findings := commit.Scan("Ｆｉｘ ｔｈｅ ｐａｒｓｅｒ", "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings for fullwidth Latin, got %v", findings)
	}
}

func TestScanRomanceProse(t *testing.T) {
	for name, text := range map[string]string{
		"Spanish":    "Fix\n\nCorrige el error que aparece con la entrada",
		"Portuguese": "Fix\n\nAjusta o tratamento de erros para que a leitura nao falhe com ficheiros grandes",
		"French":     "Fix\n\nCorrige les erreurs qui apparaissent avec cette entree",
	} {
		findings := commit.Scan(text, "en", 2, nil)
		if len(findings) != 1 || findings[0].LineNumber != 3 {
			t.Errorf("%s: expected finding on line 3, got %v", name, findings)
		}
	}
}

func TestScanOrdinaryEnglishSurvives(t *testing.T) {
	line := "Do not care as no car is in the lot"
	if findings := commit.Scan(line, "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings for ordinary English, got %v", findings)
	}
}

func TestScanJoinersExemptWord(t *testing.T) {
	if findings := commit.Scan("Add de-duplication and de-serialization helpers", "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings for hyphen-joined words, got %v", findings)
	}
	if findings := commit.Scan("Rename fill_na_values to drop_na_rows", "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings for underscore-joined words, got %v", findings)
	}
	if findings := commit.Scan("Handle the de- prefix and the -de suffix", "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings for prefix/suffix, got %v", findings)
	}
	// Standing alone still counts
	if findings := commit.Scan("Fix\n\nAjusta de novo o de sempre", "en", 2, nil); len(findings) == 0 {
		t.Errorf("expected findings for unjoined 'de', got 0")
	}
}

func TestScanVariantBDeviations(t *testing.T) {
	// The German probes from language_test.go:
	// 1. "feat: füge neue sprachprüfung hinzu" -> 4 hits -> refused
	f1 := commit.Scan("feat: füge neue sprachprüfung hinzu", "en", 2, nil)
	if len(f1) != 1 {
		t.Errorf("expected probe 1 refused, got %v", f1)
	}

	// 2. "korrigiere fehler in der verifikation" -> 3 hits -> refused
	f2 := commit.Scan("korrigiere fehler in der verifikation", "en", 2, nil)
	if len(f2) != 1 {
		t.Errorf("expected probe 2 refused, got %v", f2)
	}

	// 3. "aktualisiere dokumentation und beispiele" -> 4 hits -> refused
	f3 := commit.Scan("aktualisiere dokumentation und beispiele", "en", 2, nil)
	if len(f3) != 1 {
		t.Errorf("expected probe 3 refused, got %v", f3)
	}

	// 4. "WIP: ändere dateien" -> 2 hits -> refused
	f4 := commit.Scan("WIP: ändere dateien", "en", 2, nil)
	if len(f4) != 1 {
		t.Errorf("expected probe 4 refused, got %v", f4)
	}

	// 5. "Verbessere Performance für Windows" -> 2 hits -> refused
	f5 := commit.Scan("Verbessere Performance für Windows", "en", 2, nil)
	if len(f5) != 1 {
		t.Errorf("expected probe 5 refused, got %v", f5)
	}

	// 6. "entferne ungenutzte importe" -> 1 hit ('entferne') -> NOT refused at threshold 2
	f6 := commit.Scan("entferne ungenutzte importe", "en", 2, nil)
	if len(f6) != 0 {
		t.Errorf("expected probe 6 not refused at threshold 2, got %v", f6)
	}

	// 7. Pure Umlaut words (proper nouns / German words without stopword list match)
	f7 := commit.Scan("Müller und Zürich", "en", 2, nil)
	if len(f7) != 1 {
		t.Errorf("expected pure umlaut line refused at threshold 2, got %v", f7)
	}
	if !slices.Contains(f7[0].Hits, "Müller") || !slices.Contains(f7[0].Hits, "Zürich") {
		t.Errorf("expected untranslated casing in hits, got %v", f7[0].Hits)
	}

	// 8. Empty commit messages
	if findings := commit.Scan("", "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings for empty message, got %v", findings)
	}
	if findings := commit.Scan("   \n\t  ", "en", 2, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings for whitespace message, got %v", findings)
	}
}

func TestScanCoverageCornerCases(t *testing.T) {
	// Exercise foldGerman with Ä, Ö, ß
	commit.Scan("Änderung Öffnung Straße", "en", 2, nil)

	// Exercise isAsciiAlphanumeric with non-alphanumeric character in dot suffix
	commit.Scan("file.a#b", "en", 2, nil)

	// Exercise scriptOf unassigned / non-script letter
	commit.Scan("Fix\n\n\u02BC \u02BC", "en", 2, nil)

	// Exercise formatRun with runes > 12
	commit.Scan("Fix\n\nисправлениеошибокисправление", "en", 2, nil)
}

// Go's \w and \b are ASCII; Python's are Unicode. A surname with an umlaut
// behind a particle must be stripped exactly like an ASCII one.
func TestScanNameParticleWithNonASCIISurname(t *testing.T) {
	for _, text := range []string{
		"Credit to von Müller for the report",
		"Thank Martin von Löwis and the von Müller team",
	} {
		if findings := commit.Scan(text, "en", 2, nil); len(findings) != 0 {
			t.Errorf("%q: expected 0 findings, got %v", text, findings)
		}
	}
}

// Python's [A-Z] is ASCII, so a particle before an umlaut capital stays.
func TestScanNameParticleNeedsASCIICapital(t *testing.T) {
	findings := commit.Scan("Ask von Österreich", "en", 1, nil)
	if len(findings) != 1 || !slices.Contains(findings[0].Hits, "von") {
		t.Errorf("expected von as a hit, got %v", findings)
	}
}

// \b in front of the particle: inside a word it is no particle.
func TestScanNameParticleNeedsWordBoundary(t *testing.T) {
	findings := commit.Scan("Ersatzvon Müller", "en", 1, nil)
	if len(findings) != 1 || !slices.Contains(findings[0].Hits, "Müller") {
		t.Errorf("expected Müller as a hit, got %v", findings)
	}
}

// str.splitlines breaks on more than \n and \r.
func TestScanSplitsLinesLikePython(t *testing.T) {
	for _, sep := range []string{"\f", "\v", "\x1c", "\x1d", "\x1e", "\u0085", "\u2028", "\u2029"} {
		findings := commit.Scan("fix: x"+sep+"body"+sep+"der und das", "en", 2, nil)
		if len(findings) != 1 || findings[0].LineNumber != 3 {
			t.Errorf("sep %q: expected a finding on line 3, got %v", sep, findings)
		}
	}
}

// [^\W\d_] takes letters and the non-decimal numbers, so und² is one word.
func TestScanWordIncludesNonDecimalNumbers(t *testing.T) {
	if findings := commit.Scan("x²und und²", "en", 1, nil); len(findings) != 0 {
		t.Errorf("expected 0 findings, got %v", findings)
	}
}

// Python lowers ẞ to ß and folds it to ss.
func TestScanFoldsCapitalSharpS(t *testing.T) {
	findings := commit.Scan("GROẞ und", "en", 1, nil)
	if len(findings) != 1 || !slices.Contains(findings[0].Hits, "GROẞ") {
		t.Errorf("expected GROẞ as an umlaut hit, got %v", findings)
	}
}

func TestScanSplitsCarriageReturns(t *testing.T) {
	findings := commit.Scan("fix: x\r\nbody\rder und das\r\n", "en", 2, nil)
	if len(findings) != 1 || findings[0].LineNumber != 3 || findings[0].Line != "der und das" {
		t.Errorf("expected line 3 without CR, got %v", findings)
	}
}
