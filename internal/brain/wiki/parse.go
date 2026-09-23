package wiki

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	// Two different anchors, and the difference is the whole point. `\A` ties
	// the opening fence to the start of the file; `^` under `(?m)` ties the
	// closing one to the start of a line. Anchoring the opening fence with
	// `^` as well would match the leftmost `---` line anywhere in the file,
	// so any document with no frontmatter that uses `---` twice -- a
	// horizontal rule, a setext underline, a fenced YAML example -- would be
	// read as a block spanning the two, losing everything above the second
	// one out of the body and taking a `type` out of whatever stood between.
	// Measured over the 232 tracked `.md` files here, that cost 28 of them.
	// `(?s)` lets the captured block span lines.
	//
	// The newline before the closing fence belongs to the fence pattern
	// rather than to the capture, which is what lets `---\n---\n` match at
	// all. It has to: OKF §8 says index files "contain no frontmatter"
	// without regard to what stands inside one, so an empty block is a
	// block, and reading it as none left `okf/index-frontmatter-misplaced`
	// -- an error rule -- with a way around it. The capture then keeps the
	// trailing newline the older pattern ate, which YAML ignores.
	//
	// Measured against that older pattern over all 232 files and over
	// thirteen hand-built inputs -- empty, blank-line, normal, CRLF,
	// inline-`---`, body rule, unclosed, absent, no trailing newline, list
	// block, two rule lines, setext underlines, fenced YAML: the two agree
	// everywhere except the two empty forms, and every body is byte for
	// byte the one before.
	//
	// The Python `_FRONTMATTER` of `brain/document.py:10` reads
	// `\A---\r?\n(.*?)\r?\n---\r?\n` with DOTALL and no multiline. It anchors
	// the same way; the one input the two readers still answer differently
	// is the empty block, which Python reads as no frontmatter.
	frontmatterRe = regexp.MustCompile(`(?sm)\A---\r?\n(.*?)^---\r?\n?(.*)$`)
	mdLinkRe      = regexp.MustCompile(`\[(?:[^\]]*)\]\(([^)#\s]+)(?:#[^\)]*)?\)`)
	conflictBoxRe = regexp.MustCompile(`(?i)(?:^|\n)(?:>\s*\[!CONFLICT\]|` + "```conflict)")
)

func ReadPage(filePath, wikiRoot string) (*WikiPage, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return pageFrom(content, filePath, wikiRoot), nil
}

// pageFrom is ReadPage on bytes already read, for a caller that needs the
// file's own text as well: reading it twice would let the two disagree.
func pageFrom(content []byte, filePath, wikiRoot string) *WikiPage {
	rel, err := filepath.Rel(wikiRoot, filePath)
	if err != nil {
		rel = filePath
	}
	rel = filepath.ToSlash(rel)

	page := &WikiPage{
		Path:     filePath,
		Relative: rel,
		RawBody:  string(content),
	}

	// A page whose modification time cannot be read still has everything the
	// other rules judge it by, so this is no reason to refuse the page. Only
	// the untouched-days rules go by ModTime, and they see the zero time --
	// which is older than any deadline, so they would fire rather than stay
	// silent. The stat follows a successful ReadFile, so a failure here means
	// the file went away between the two calls.
	if info, err := os.Stat(filePath); err == nil {
		page.ModTime = info.ModTime()
	}

	matches := frontmatterRe.FindSubmatch(content)
	if len(matches) == 3 {
		yamlBytes := matches[1]
		page.RawBody = string(matches[2])
		// Set before the block is decoded, not after: an `index.md` whose
		// block is empty or unreadable has carried one all the same, and OKF
		// §8 forbids that outside the bundle root regardless of what stands
		// inside. Both states reach here -- the empty block because the
		// pattern above matches it, the unreadable one because the decode
		// below carries its error rather than rejecting the page.
		page.HasFrontmatter = true

		var fm Frontmatter
		if err := yaml.Unmarshal(yamlBytes, &fm); err != nil {
			// Carried, not swallowed. `okf/frontmatter-unparsable` is an OKF
			// §11.1 error, and a page whose frontmatter did not parse has no
			// type either -- reporting it twice would send the reader to the
			// wrong repair.
			msg := err.Error()
			page.BrokenFrontmatter = &msg
		} else {
			page.Frontmatter = fm
			page.PageType = fm.Type
			page.Title = fm.Title
			page.Description = fm.Description
			page.DeclaredConflicts = fm.OpenConflicts
			page.Sources = fm.Sources
			page.Status = fm.Status
			page.StaleAfter = fm.StaleAfter
			page.Realization = fm.Realization
			page.ImplementedIn = fm.ImplementedIn
			page.Generated = fm.Generated
			page.Runtime = fm.Runtime

			// The block is read a second time, as a node tree, because the
			// struct above keeps only the keys it names and OKF §12 judges a
			// bundle-root `index.md` by the key set itself. This decode
			// cannot fail where the one above succeeded: both parse the same
			// bytes, and a node tree puts no type on anything it holds, so
			// the value that would break a typed field is a plain node here.
			var doc yaml.Node
			_ = yaml.Unmarshal(yamlBytes, &doc)
			for _, node := range doc.Content {
				if node.Kind != yaml.MappingNode {
					continue
				}
				// A mapping node alternates key and value, so the keys sit at
				// the even positions.
				for i := 0; i+1 < len(node.Content); i += 2 {
					page.FrontmatterKeys = append(page.FrontmatterKeys, node.Content[i].Value)
				}
			}
		}
	}

	// Extract Markdown Links
	for _, target := range markdownLinks(page.RawBody) {
		if target != "" && !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") && !strings.HasPrefix(target, "mailto:") {
			page.Links = append(page.Links, target)
		}
	}

	// Count Conflict Boxes
	cBoxes := conflictBoxRe.FindAllString(string(content), -1)
	page.FoundConflicts = len(cBoxes)

	return page
}

// markdownLinks is `mdLinkRe` plus the one thing RE2 cannot express: the
// negative lookbehind of Python's `_MARKDOWN_LINK`
// (`src/brain/document.py:11`, `(?<!!)`). An embedded image is not a
// link, and `house/dead-link` is an error -- so without this a page
// holding a picture that is not a wiki page fails a run Python lets
// through.
//
// The scan is a loop and not a filter over `FindAllStringSubmatchIndex`,
// and the difference is measurable. A lookbehind that refuses a position
// makes the engine try the *next* one; a filter over finished matches
// drops the whole line. Measured against Python, `![a[b](c.md)` answers
// `c.md` there, because the second `[` is preceded by an `a` -- a filter
// would answer nothing. Hence `from = start + 1` on a refusal and
// `from = end` on a match.
//
// What it looks at is the byte, not the syntax: Python's lookbehind asks
// whether the character before the `[` is `!`, so `Wow![x](y.md)` is
// refused on both sides although no Markdown reader would call it an
// image. Copying the rule rather than the intent is what keeps the two
// sides comparable.
//
// One blind spot stays, and it is shared: in `[![alt](i.png)](y.md)` the
// outer `[` is unpreceded, and the text class runs to the *first* `]`,
// which is the image's -- so both sides record `i.png` and neither
// records `y.md`. A repair here would be a divergence from Python, not a
// fix, and the rule it feeds would then disagree over one page.
func markdownLinks(body string) []string {
	var out []string
	for from := 0; from < len(body); {
		at := mdLinkRe.FindStringSubmatchIndex(body[from:])
		if at == nil {
			break
		}
		start, end := from+at[0], from+at[1]
		if start > 0 && body[start-1] == '!' {
			from = start + 1
			continue
		}
		out = append(out, strings.TrimSpace(body[from+at[2]:from+at[3]]))
		from = end
	}
	return out
}

// ScaffoldFiles names the files that carry structure rather than knowledge.
// The spec of Scheibe 3, §4, calls exactly these five scaffold and no OKF
// pages; linted as concept pages they would report `missing-type` in rows.
// The set is the Python `SCAFFOLD_FILES` of `brain/wiki/page.py`, copied
// rather than shared: whoever changes one changes the other.
//
// `index.md` is scaffold but still gets read -- it is a valid source of links
// for the orphan rule.
var ScaffoldFiles = map[string]bool{
	"_schema.md":      true,
	"index.md":        true,
	"log.md":          true,
	"audit.md":        true,
	"_identities.tsv": true,
}

// IsScaffoldFile answers for a path what ScaffoldFiles answers for a name.
// The Python side asks `path.name in SCAFFOLD_FILES`, so the base name
// decides here as well: a nested `sub/index.md` is scaffold like the root one.
func IsScaffoldFile(filename string) bool {
	return ScaffoldFiles[filepath.Base(filename)]
}
