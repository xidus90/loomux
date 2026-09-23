package okf

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/brain/wiki"
)

// read is the bundle reader the rules expect: one walk, every `.md` file, the
// page set they then run over. The design puts a single read pass in front of
// the rules, so a test that let a rule reach for the disk itself would prove
// something the real run never does.
func read(t *testing.T, dir string) []wiki.WikiPage {
	t.Helper()
	var pages []wiki.WikiPage
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		page, err := wiki.ReadPage(p, dir)
		if err != nil {
			return err
		}
		pages = append(pages, *page)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return pages
}

func hasRule(findings []check.Finding, rule string) bool {
	for _, f := range findings {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

// hasRuleOn asks the same question for one page. The two rules that share
// `index.md` cannot be told apart by rule name alone once a bundle holds more
// than one reserved file.
func hasRuleOn(findings []check.Finding, relative, rule string) bool {
	for _, f := range findings {
		if f.Relative == relative && f.Rule == rule {
			return true
		}
	}
	return false
}

func messageOf(t *testing.T, findings []check.Finding, relative, rule string) string {
	t.Helper()
	for _, f := range findings {
		if f.Relative == relative && f.Rule == rule {
			return f.Message
		}
	}
	t.Fatalf("no %s finding on %s; got %v", rule, relative, findings)
	return ""
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFrontmatterUnparsableNamesWhatYAMLChokedOn(t *testing.T) {
	// The message has to carry the parser's own words: a reader told only
	// that a block is unreadable has nowhere to start looking.
	dir := t.TempDir()
	write(t, dir, "a.md", "---\ntype: [unclosed\n---\nbody\n")
	got := Errors(read(t, dir), Context{Root: dir})
	msg := messageOf(t, got, "a.md", "frontmatter-unparsable")
	if !strings.HasPrefix(msg, "frontmatter is unreadable: ") {
		t.Fatalf("message does not carry the parser error: %q", msg)
	}
	if len(msg) <= len("frontmatter is unreadable: ") {
		t.Fatalf("message names no cause: %q", msg)
	}
}

func TestBrokenFrontmatterIsNotAlsoReportedAsMissingType(t *testing.T) {
	// A page whose frontmatter did not parse has no type either. Reporting
	// both would give one defect two names and send the reader to the wrong
	// repair -- the same reason `wiki/lint.py` keeps the two apart.
	dir := t.TempDir()
	write(t, dir, "a.md", "---\ntype: [unclosed\n---\nbody\n")
	if hasRule(Errors(read(t, dir), Context{Root: dir}), "type-missing") {
		t.Fatal("an unreadable frontmatter was also reported as a missing type")
	}
}

func TestTypeMissingTellsAnAbsentTypeFromAnEmptyOne(t *testing.T) {
	// Two states, two repairs: one page has to gain a `type` key, the other
	// has to fill the one it has. A single message for both would let the two
	// branches collapse into one without any test noticing.
	dir := t.TempDir()
	write(t, dir, "absent.md", "---\ntitle: a\n---\nbody\n")
	write(t, dir, "empty.md", "---\ntype: \"\"\n---\nbody\n")
	got := Errors(read(t, dir), Context{Root: dir})
	absent := messageOf(t, got, "absent.md", "type-missing")
	empty := messageOf(t, got, "empty.md", "type-missing")
	if absent == empty {
		t.Fatalf("absent and empty type share one message: %q", absent)
	}
}

func TestScaffoldFilesNeedNoType(t *testing.T) {
	// OKF §11.1 and §11.2 bind the non-reserved files; the design widens that
	// to the repository's scaffold set. These files carry structure, not
	// knowledge, and linting them as concepts reports in rows.
	dir := t.TempDir()
	write(t, dir, "index.md", "# c\n")
	write(t, dir, "log.md", "# log\n")
	write(t, dir, "_schema.md", "# schema\n")
	write(t, dir, "audit.md", "# audit\n")
	if got := Errors(read(t, dir), Context{Root: dir}); len(got) != 0 {
		t.Fatalf("scaffold files were reported: %v", got)
	}
}

func TestSourceResourceMissingNamesTheEntry(t *testing.T) {
	// OKF §5.1 makes `resource` required within an entry. The message points
	// at the entry, because a page may carry many and only one be broken.
	dir := t.TempDir()
	write(t, dir, "a.md", "---\ntype: Topic\nsources:\n  - resource: https://example.org/one\n  - id: two\n---\nbody\n")
	got := Errors(read(t, dir), Context{Root: dir})
	msg := messageOf(t, got, "a.md", "source-resource-missing")
	if !strings.Contains(msg, "1") {
		t.Fatalf("message does not name the offending entry: %q", msg)
	}
	if strings.Contains(msg, "sources[0]") {
		t.Fatalf("the complete first entry was reported too: %q", msg)
	}
}

func TestGeneratedByMissing(t *testing.T) {
	// OKF §5.2: `by` is required within `generated`. A `generated` block that
	// names no actor records nothing.
	dir := t.TempDir()
	write(t, dir, "bare.md", "---\ntype: Topic\ngenerated:\n  at: 2026-06-20T22:53:05Z\n---\nbody\n")
	write(t, dir, "named.md", "---\ntype: Topic\ngenerated:\n  by: human:x\n---\nbody\n")
	got := Errors(read(t, dir), Context{Root: dir})
	if !hasRuleOn(got, "bare.md", "generated-by-missing") {
		t.Fatalf("a generated block without `by` was not reported: %v", got)
	}
	if hasRuleOn(got, "named.md", "generated-by-missing") {
		t.Fatal("a generated block with `by` was reported")
	}
}

func TestReservedNameAsConceptCoversIndexAndLog(t *testing.T) {
	// OKF §3.1 reserves exactly these two names, at any level, and forbids
	// using them for concept documents. A `type` is what makes a file one.
	dir := t.TempDir()
	write(t, dir, "sub/log.md", "---\ntype: Topic\n---\nbody\n")
	write(t, dir, "sub/index.md", "---\ntype: Topic\n---\nbody\n")
	got := Errors(read(t, dir), Context{Root: dir})
	if !hasRuleOn(got, "sub/log.md", "reserved-name-as-concept") {
		t.Fatalf("a typed log.md was not reported: %v", got)
	}
	if !hasRuleOn(got, "sub/index.md", "reserved-name-as-concept") {
		t.Fatalf("a typed index.md was not reported: %v", got)
	}
}

func TestReservedNameAndMisplacedFrontmatterDoNotHideEachOther(t *testing.T) {
	// Both rules judge `index.md`, and each has a case the other must not
	// answer: a typed `log.md` is no index question at all, and a non-root
	// `index.md` carrying only `okf_version` is misplaced without being a
	// concept document.
	dir := t.TempDir()
	write(t, dir, "log.md", "---\ntype: Topic\n---\nbody\n")
	write(t, dir, "sub/index.md", "---\nokf_version: \"0.2\"\n---\n# c\n")
	got := Errors(read(t, dir), Context{Root: dir})
	if !hasRuleOn(got, "log.md", "reserved-name-as-concept") {
		t.Fatalf("the typed log.md was not reported: %v", got)
	}
	if hasRuleOn(got, "log.md", "index-frontmatter-misplaced") {
		t.Fatal("a log.md was judged by the index rule")
	}
	if !hasRuleOn(got, "sub/index.md", "index-frontmatter-misplaced") {
		t.Fatalf("frontmatter below the bundle root was not reported: %v", got)
	}
	if hasRuleOn(got, "sub/index.md", "reserved-name-as-concept") {
		t.Fatal("an index.md without a type was called a concept document")
	}
}

func TestOnlyOKFVersionMayStandInTheRootCatalog(t *testing.T) {
	// OKF §12 permits frontmatter in a bundle-root `index.md` and nowhere
	// else, and only for `okf_version`. The message names the key that has to
	// go, since a block may carry several.
	dir := t.TempDir()
	write(t, dir, "index.md", "---\nokf_version: \"0.2\"\ntitle: c\n---\n* [a](a.md)\n")
	write(t, dir, "a.md", "---\ntype: Topic\n---\nbody\n")
	got := Errors(read(t, dir), Context{Root: dir})
	msg := messageOf(t, got, "index.md", "index-frontmatter-misplaced")
	if !strings.Contains(msg, "title") {
		t.Fatalf("message does not name the offending key: %q", msg)
	}
}

func TestAnUnreadableBlockIsStillFrontmatter(t *testing.T) {
	// A block that did not parse has been carried all the same, and §8 forbids
	// one below the bundle root whatever stands inside it. Reading the flag
	// off the decoded values would let a catalog escape the rule by breaking
	// its own YAML.
	dir := t.TempDir()
	write(t, dir, "sub/index.md", "---\nokf_version: [unclosed\n---\n# c\n")
	got := Errors(read(t, dir), Context{Root: dir})
	if !hasRuleOn(got, "sub/index.md", "index-frontmatter-misplaced") {
		t.Fatalf("an unreadable block below the root was not reported as misplaced: %v", got)
	}
}

func TestARootCatalogWithAnUnreadableBlockIsStillNamed(t *testing.T) {
	// §8 lets the bundle-root index.md carry a block, so the misplacement rule
	// has nothing to say here -- and an unreadable block yields no key for it
	// to name either. `frontmatter-unparsable` is the only rule left, which is
	// why it judges the reserved names although §11.1 does not bind them.
	dir := t.TempDir()
	write(t, dir, "index.md", "---\nokf_version: [unclosed\n---\n# c\n")
	got := Errors(read(t, dir), Context{Root: dir})
	if !hasRuleOn(got, "index.md", "frontmatter-unparsable") {
		t.Fatalf("nothing named the unreadable root catalog: %v", got)
	}
	if hasRuleOn(got, "index.md", "index-frontmatter-misplaced") {
		t.Fatal("the root catalog was reported for carrying a block §8 permits")
	}
}

func TestMisplacedKeysAreReportedInTheOrderTheyWereWritten(t *testing.T) {
	// Two runs over an unchanged bundle have to render the same output, or no
	// two runs can be compared. Keys read into a map would come back in a
	// different order on almost every run.
	dir := t.TempDir()
	write(t, dir, "index.md", "---\nzulu: 1\nalpha: 2\n---\n# c\n")
	got := Errors(read(t, dir), Context{Root: dir})
	var order []string
	for _, f := range got {
		if f.Rule == "index-frontmatter-misplaced" {
			order = append(order, f.Message)
		}
	}
	if len(order) != 2 {
		t.Fatalf("expected both keys reported, got %v", got)
	}
	if !strings.Contains(order[0], "zulu") || !strings.Contains(order[1], "alpha") {
		t.Fatalf("keys were not reported in written order: %v", order)
	}
}

func TestTheRootCatalogMayDeclareTheVersion(t *testing.T) {
	// The counter-probe to the rule above: a bundle-root `index.md` carrying
	// nothing but `okf_version` is exactly what §12 asks for.
	dir := t.TempDir()
	write(t, dir, "index.md", "---\nokf_version: \"0.2\"\n---\n* [a](a.md)\n")
	write(t, dir, "a.md", "---\ntype: Topic\n---\nbody\n")
	if got := Errors(read(t, dir), Context{Root: dir}); len(got) != 0 {
		t.Fatalf("a conformant root catalog was reported: %v", got)
	}
}

func TestLogDateFormWantsISOHeadings(t *testing.T) {
	// OKF §9: date headings MUST use `YYYY-MM-DD`. Anything else breaks the
	// one thing a consumer can do with a log mechanically -- order it.
	dir := t.TempDir()
	write(t, dir, "log.md", "# Directory Update Log\n\n## 2026-05-22\n* a\n\n## May 15\n* b\n")
	got := Errors(read(t, dir), Context{Root: dir})
	msg := messageOf(t, got, "log.md", "log-date-form")
	if !strings.Contains(msg, "May 15") {
		t.Fatalf("message does not name the heading: %q", msg)
	}
	if len(got) != 1 {
		t.Fatalf("the ISO heading was reported too: %v", got)
	}
}

func TestComputationRuntimeMissing(t *testing.T) {
	// OKF §10.2 makes `runtime` required for this type: it is the field that
	// says how the computation is to be run at all. The type is matched
	// case-insensitively, because §4.1 leaves type spelling to the producer.
	dir := t.TempDir()
	write(t, dir, "bare.md", "---\ntype: Attested Computation\n---\nbody\n")
	write(t, dir, "run.md", "---\ntype: attested computation\nruntime: bigquery\n---\nbody\n")
	got := Errors(read(t, dir), Context{Root: dir})
	if !hasRuleOn(got, "bare.md", "computation-runtime-missing") {
		t.Fatalf("a computation without a runtime was not reported: %v", got)
	}
	if hasRuleOn(got, "run.md", "computation-runtime-missing") {
		t.Fatal("a computation with a runtime was reported")
	}
}

func TestCatalogMalformedFindsTheOmittedPage(t *testing.T) {
	// The blind spot that let a generated catalog overwrite a bundle catalog
	// in three areas for weeks: nothing checked that a catalog enumerates its
	// directory.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.md"), []byte("# c\n\n* [a](a.md)\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("---\ntype: Topic\n---\nb\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.md"), []byte("---\ntype: Topic\n---\nb\n"), 0o644)
	got := Errors(read(t, dir), Context{Root: dir})
	if !hasRule(got, "catalog-malformed") {
		t.Fatal("b.md is missing from the catalog and was not reported")
	}
}

func TestAMissingCatalogIsNotAnError(t *testing.T) {
	// OKF §8 makes index.md optional and §11 forbids rejecting a bundle for
	// its absence. The rule fires on incomplete-when-present, never on absent.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("---\ntype: Topic\n---\nb\n"), 0o644)
	for _, f := range Errors(read(t, dir), Context{Root: dir}) {
		if f.Rule == "catalog-malformed" {
			t.Fatal("a missing catalog was reported as an error")
		}
	}
}

func TestCatalogMalformedNamesTheOmittedPageAndJudgesEachDirectory(t *testing.T) {
	// A catalog answers for its own directory, not for the tree below it: the
	// subdirectory has a catalog of its own. Naming the omitted page is what
	// turns the finding into a repair.
	dir := t.TempDir()
	write(t, dir, "index.md", "# c\n\n* [a](a.md)\n* [sub](sub/)\n")
	write(t, dir, "a.md", "---\ntype: Topic\n---\nbody\n")
	write(t, dir, "sub/index.md", "# c\n\n* [x](x.md)\n")
	write(t, dir, "sub/x.md", "---\ntype: Topic\n---\nbody\n")
	write(t, dir, "sub/y.md", "---\ntype: Topic\n---\nbody\n")
	got := Errors(read(t, dir), Context{Root: dir})
	msg := messageOf(t, got, "sub/index.md", "catalog-malformed")
	if !strings.Contains(msg, "y.md") {
		t.Fatalf("message does not name the omitted page: %q", msg)
	}
	if hasRuleOn(got, "index.md", "catalog-malformed") {
		t.Fatal("the root catalog was asked to list the subdirectory's pages")
	}
}

func TestCatalogLinksResolveAgainstTheBundleRoot(t *testing.T) {
	// The design decides the root question for absolute targets: a leading
	// slash is bundle-relative, as OKF §6.1 defines it. Percent escapes are
	// undone first, because the link pattern refuses a raw space, so a page
	// name carrying one can only be linked encoded. A broken escape is left
	// standing rather than dropped -- it then matches no page, which is the
	// safe direction for a rule of this severity.
	dir := t.TempDir()
	write(t, dir, "sub/index.md", "# c\n\n* [a](/sub/a.md)\n* [b](b%20b.md)\n* [c](%zz.md)\n")
	write(t, dir, "sub/a.md", "---\ntype: Topic\n---\nbody\n")
	write(t, dir, "sub/b b.md", "---\ntype: Topic\n---\nbody\n")
	if got := Errors(read(t, dir), Context{Root: dir}); hasRule(got, "catalog-malformed") {
		t.Fatalf("a catalog listing every page was reported: %v", got)
	}
}

func TestACleanBundleIsNotReportedAtAll(t *testing.T) {
	// The counter-probe for the whole axis: a bundle that satisfies every
	// must has to come back silent, or none of the nine rules above proves
	// anything about the bundles this repository actually produces.
	dir := t.TempDir()
	write(t, dir, "index.md", "---\nokf_version: \"0.2\"\n---\n\n# Topics\n\n* [A](a.md) - the one page\n")
	write(t, dir, "log.md", "# Log\n\n## 2026-05-22\n* **Creation**: a\n")
	write(t, dir, "a.md", "---\ntype: Topic\ntitle: A\nsources:\n  - id: s\n    resource: https://example.org/s\ngenerated:\n  by: human:x\n---\nbody\n")
	if got := Errors(read(t, dir), Context{Root: dir}); len(got) != 0 {
		t.Fatalf("a conformant bundle was reported: %v", got)
	}
}

func TestEveryFindingIsAnOKFError(t *testing.T) {
	// The axis and the severity are what a caller routes on. A rule that
	// forgot either would still satisfy every test above.
	dir := t.TempDir()
	write(t, dir, "index.md", "---\ntitle: c\n---\n# c\n")
	write(t, dir, "log.md", "---\ntype: Topic\n---\n## nope\n")
	write(t, dir, "a.md", "---\ntype: [unclosed\n---\nbody\n")
	write(t, dir, "b.md", "---\ntype: Attested Computation\nsources:\n  - id: s\ngenerated:\n  at: 2026-06-20T22:53:05Z\n---\nbody\n")
	write(t, dir, "c.md", "---\ntitle: c\n---\nbody\n")
	got := Errors(read(t, dir), Context{Root: dir})
	seen := map[string]bool{}
	for _, f := range got {
		if f.Axis != check.AxisOKF {
			t.Fatalf("finding is not on the okf axis: %v", f)
		}
		if f.Severity != check.Error {
			t.Fatalf("finding is not an error: %v", f)
		}
		seen[f.Rule] = true
	}
	for _, rule := range []string{
		"frontmatter-unparsable", "type-missing", "source-resource-missing",
		"generated-by-missing", "reserved-name-as-concept", "catalog-malformed",
		"log-date-form", "computation-runtime-missing", "index-frontmatter-misplaced",
	} {
		if !seen[rule] {
			t.Errorf("rule %s never fired on a bundle that breaks all nine musts", rule)
		}
	}
}

func TestLogDateFormJudgesOnlyLogFiles(t *testing.T) {
	// OKF §9 puts date headings in a log.md. A concept page's `## ` headings
	// are ordinary section headings and mean nothing to this rule -- without
	// the filename guard, every second-level heading in the bundle would be
	// read as a broken date.
	dir := t.TempDir()
	write(t, dir, "a.md", "---\ntype: Topic\n---\n\n## Overview\n\ntext\n")
	if got := Errors(read(t, dir), Context{Root: dir}); hasRule(got, "log-date-form") {
		t.Fatalf("a section heading on a concept page was judged as a date: %v", got)
	}
}

func TestLogDateFormJudgesOnlySecondLevelHeadings(t *testing.T) {
	// §9 shows the date one level below the log's title, and nothing forbids
	// a log from structuring an entry further down. A guard that matched `##`
	// as a prefix rather than `## ` would read every deeper heading as a
	// malformed date.
	dir := t.TempDir()
	write(t, dir, "log.md", "# Log\n\n## 2026-05-22\n\n### Detail\n* a\n")
	if got := Errors(read(t, dir), Context{Root: dir}); hasRule(got, "log-date-form") {
		t.Fatalf("a third-level heading was judged as a date: %v", got)
	}
}

func TestLogDateFormWantsTheWholeHeadingToBeTheDate(t *testing.T) {
	// §9 says the heading MUST use the form, not merely contain it. An
	// unanchored pattern would pass anything with a date somewhere inside,
	// which is exactly the heading a consumer cannot order by.
	dir := t.TempDir()
	write(t, dir, "log.md", "# Log\n\n## Stand 2026-05-22\n* a\n")
	got := Errors(read(t, dir), Context{Root: dir})
	msg := messageOf(t, got, "log.md", "log-date-form")
	if !strings.Contains(msg, "Stand 2026-05-22") {
		t.Fatalf("message does not name the heading: %q", msg)
	}
}

func TestLogDateFormReadsACRLFLog(t *testing.T) {
	// A file checked out with CRLF line endings leaves a `\r` at the end of
	// every heading. Judged with it still attached, an ISO date matches
	// nothing and every entry of the log is reported.
	dir := t.TempDir()
	write(t, dir, "log.md", "# Log\r\n\r\n## 2026-05-22\r\n* a\r\n")
	if got := Errors(read(t, dir), Context{Root: dir}); hasRule(got, "log-date-form") {
		t.Fatalf("a CRLF log with ISO headings was reported: %v", got)
	}
}

func TestAnEmptyFrontmatterBlockIsStillABlock(t *testing.T) {
	// §8 says index files "contain no frontmatter" without regard to what
	// stands inside, so an empty block is a block. Reading it as none would
	// leave a way past an error rule: a catalog below the bundle root that
	// opens and closes the fence with nothing between escapes §8 and §12.
	dir := t.TempDir()
	write(t, dir, "sub/index.md", "---\n---\n# c\n")
	got := Errors(read(t, dir), Context{Root: dir})
	if !hasRuleOn(got, "sub/index.md", "index-frontmatter-misplaced") {
		t.Fatalf("an empty block below the bundle root was not reported: %v", got)
	}
}

func TestAWhitespaceOnlyValueCountsAsAbsent(t *testing.T) {
	// YAML keeps a quoted run of spaces as a value, and the three fields
	// below are all required by OKF. A page naming its type as "   " has
	// named none, and a rule going by the raw string would pass it.
	dir := t.TempDir()
	write(t, dir, "a.md", "---\ntype: \"   \"\n---\nbody\n")
	write(t, dir, "b.md", "---\ntype: \"  Attested Computation  \"\nruntime: \"   \"\nsources:\n  - id: s\n    resource: \"   \"\ngenerated:\n  by: \"   \"\n---\nbody\n")
	got := Errors(read(t, dir), Context{Root: dir})
	if messageOf(t, got, "a.md", "type-missing") != "type is empty; every OKF page needs a non-empty one" {
		t.Fatalf("a whitespace-only type was not read as empty: %v", got)
	}
	for _, rule := range []string{"source-resource-missing", "generated-by-missing", "computation-runtime-missing"} {
		if !hasRuleOn(got, "b.md", rule) {
			t.Errorf("%s passed a whitespace-only value: %v", rule, got)
		}
	}
}

func TestAReservedFileWithAnEmptyTypeIsStillAConcept(t *testing.T) {
	// §3.1 turns on the key, not on its value: writing `type:` at all is what
	// makes a file a concept document, and an empty one is no way out. The
	// scaffold exemption keeps `type-missing` off the same page, so this rule
	// is the only one that speaks here.
	dir := t.TempDir()
	write(t, dir, "log.md", "---\ntype: \"\"\n---\nbody\n")
	got := Errors(read(t, dir), Context{Root: dir})
	if !hasRuleOn(got, "log.md", "reserved-name-as-concept") {
		t.Fatalf("an empty type on a reserved name was not reported: %v", got)
	}
	if hasRuleOn(got, "log.md", "type-missing") {
		t.Fatal("a scaffold file was also reported for its type")
	}
}

func TestASeparatorHeavyBundleIsClean(t *testing.T) {
	// The two files of a bundle that carry no frontmatter are the catalog and
	// the log, and they are also the two most likely to use `---` as a section
	// separator. Read with the opening fence anchored to any line start rather
	// than to the start of the file, both are taken for frontmatter: measured
	// on exactly this bundle, that produced four errors, two of them
	// `catalog-malformed` against a catalog that does list every page.
	dir := t.TempDir()
	write(t, dir, "index.md", "# Catalog\n\n* [A](a.md) - one\n\n---\n\n# More\n\n* [B](b.md) - two\n\n---\n\n# Even more\n\n* [C](c.md) - three\n")
	write(t, dir, "log.md", "# Log\n\n## 2026-05-22\n* a\n\n---\n\n## 2026-05-15\n* b\n\n---\n\n## 2026-05-01\n* c\n")
	for _, name := range []string{"a.md", "b.md", "c.md"} {
		write(t, dir, name, "---\ntype: Topic\nsources:\n  - id: s\n    resource: https://example.org/s\n---\nbody\n")
	}
	if got := Errors(read(t, dir), Context{Root: dir}); len(got) != 0 {
		t.Fatalf("a conformant bundle using `---` separators was reported: %v", got)
	}
}

func TestACatalogEntryWithAQueryStringIsNotMalformed(t *testing.T) {
	// Measured in an earlier review and carried here: `resolveLink`
	// keeps a query in the path and answers `topics/b.md?x=1`, while the
	// reader hands the edge on as `b.md` (`internal/brain/wiki/parse.go:44`). The
	// catalog lists the page and is reported all the same, at the grade
	// `error`.
	//
	// The anchor case is *not* the same defect and is here to say so:
	// `mdLinkRe` consumes the fragment in a group outside the capture, so
	// `a.md#top` reaches the rules as `a.md` and never got this far.
	dir := t.TempDir()
	write(t, dir, "topics/index.md",
		"# c\n\n* [b](b.md?x=1)\n* [a](a.md#top)\n")
	write(t, dir, "topics/a.md", "---\ntype: Topic\n---\nbody\n")
	write(t, dir, "topics/b.md", "---\ntype: Topic\n---\nbody\n")
	got := Errors(read(t, dir), Context{Root: dir})
	if hasRule(got, "catalog-malformed") {
		t.Fatalf("a catalog listing every page was reported: %v", got)
	}
}

func TestAnEncodedQuestionMarkStaysPartOfTheName(t *testing.T) {
	// The order the query has to be cut in, and the reason it is cut
	// before the escapes are undone, as Python does it
	// (`src/brain/wiki/lint.py:265-268` splits and only then unquotes).
	// Cutting afterwards would turn `b%3F.md` into `b` and report a page
	// its catalog does list.
	//
	// The pages are built by hand: a file called `b?.md` cannot be
	// written on this host, and the defect is in the resolution rather
	// than on the disk.
	pages := []wiki.WikiPage{
		{Relative: "index.md", Links: []string{"b%3F.md"}},
		{Relative: "b?.md"},
	}
	got := Errors(pages, Context{Root: t.TempDir()})
	if hasRule(got, "catalog-malformed") {
		t.Fatalf("an encoded question mark was cut off: %v", got)
	}
}

func TestACatalogLinkCannotClimbOutOfTheBundle(t *testing.T) {
	// The second finding of that review: `resolveLink` strips
	// the leading slash before normalising, so `/../escape.md` becomes
	// the string `../escape.md`. Today that only means the entry matches
	// no page -- `errors.go` imports no `os` and compares strings alone
	// -- but the house resolver next door already normalises first
	// (`internal/brain/check/house/bundle.go:147-151`), and a caller that put this
	// answer on the filesystem would inherit a climb out of the bundle.
	dir := t.TempDir()
	write(t, dir, "index.md", "# c\n\n* [e](/../escape.md)\n")
	write(t, dir, "escape.md", "---\ntype: Topic\n---\nbody\n")
	got := Errors(read(t, dir), Context{Root: dir})
	if hasRule(got, "catalog-malformed") {
		t.Fatalf("an absolute target climbed out of the bundle: %v", got)
	}
}

func TestAQueryAtTheStartOfATargetLeavesTheDirectory(t *testing.T) {
	// A catalog link that is nothing but a query resolves to the catalog's
	// own directory, as it would once the query is cut anywhere else. Found
	// by the mutation round of 2026-09-23.
	if got := resolveLink("topics", "?x"); got != "topics" {
		t.Fatalf("resolveLink = %q, want topics", got)
	}
}
