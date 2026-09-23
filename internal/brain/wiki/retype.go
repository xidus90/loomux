package wiki

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/lock"
)

// Retype is `retype`: it renames the page type source to target in every
// page below wikiRoot and answers the pages it wrote, in walk order.
//
// Only the frontmatter's `type:` line changes. A YAML round trip would
// normalise field order, quoting and comments on every page it touched, so
// ReadPage decides which pages qualify and the line is replaced in the raw
// text. A page qualifies when it parses and its type equals source whole: a
// longer type that merely starts with source never matches, and a second run
// finds nothing left to do.
//
// Skipped rather than guessed at: a scaffold file, broken frontmatter, bytes
// that are not UTF-8 (a lenient read would write replacement characters
// back), and a type that parses to source but is not a plain scalar on its own
// line -- quoted, folded, or in a header with CR-only line endings. A page
// whose text would come out the same is not written and not reported.
//
// A written page is LF throughout: every file brain writes is, so a CRLF page
// has all its line endings moved in the one run that renames it. It is written
// through lock.ReplaceText, alongside and swapped in, where the reference
// wrote in place.
func Retype(wikiRoot, source, target string) ([]string, error) {
	root := filepath.Clean(wikiRoot)
	var changed []string
	for _, path := range markdownBelow(root) {
		if IsScaffoldFile(path) {
			continue
		}
		// Read once and parsed from the same bytes: the frontmatter is where
		// the line to change stands, and ReadPage keeps only the body.
		data, err := os.ReadFile(path)
		if err != nil {
			return changed, err
		}
		page := pageFrom(data, path, root)
		if page.BrokenFrontmatter != nil || page.PageType == nil || *page.PageType != source {
			continue
		}
		text := string(data)
		if !utf8.ValidString(text) {
			continue
		}
		renamed := renameTypeField(text, source, target)
		if renamed == text {
			continue
		}
		if err := lock.ReplaceText(path, strings.ReplaceAll(renamed, "\r\n", "\n")); err != nil {
			return changed, err
		}
		changed = append(changed, path)
	}
	return changed, nil
}

// retypeHeader is the frontmatter at the very start of a file, its group
// ending after the block's own trailing line break so every line in it ends
// in one.
func retypeHeader() *regexp.Regexp {
	return regexp.MustCompile(`\A---\r?\n((?s:.*?)\r?\n)---\r?\n`)
}

// renameTypeField replaces the top-level `type: <source>` line of the
// frontmatter and leaves every other byte in place. The reference replaces
// every such line because PyYAML reads a duplicate key as its last value; the
// reader here refuses a duplicate key as broken frontmatter, so a page that
// reaches this function carries one.
func renameTypeField(text, source, target string) string {
	header := retypeHeader().FindStringSubmatchIndex(text)
	if header == nil {
		return text
	}
	line := regexp.MustCompile(`(?m)^type:([ \t]*)` + regexp.QuoteMeta(source) + `([ \t]*\r?\n)`)
	block := text[header[2]:header[3]]
	renamed := line.ReplaceAllStringFunc(block, func(match string) string {
		parts := line.FindStringSubmatch(match)
		return "type:" + parts[1] + target + parts[2]
	})
	return text[:header[2]] + renamed + text[header[3]:]
}
