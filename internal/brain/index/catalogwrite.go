package index

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/config"
)

// CatalogRule is the invariant human rule appended to every generated index.md.
//
// German, and staying German, as are the two headings RenderCatalog writes.
// They are not messages of this program but the content of a file that
// already lies in every indexed area on this machine, and a rerun has to
// produce it byte for byte (spec 14). Translating them would rewrite every
// index.md there is, and the rule they state is addressed to the person
// reading the catalog, not to a tool reading this package.
const CatalogRule = "> Jeder neue Bereich bekommt sofort eine Zeile in diesem Katalog."

// DestinationSpecials defines characters that require a Markdown link destination
// to be wrapped in angle brackets <...>.
const DestinationSpecials = " ()<>"

func formatDestination(target string) string {
	if !strings.ContainsAny(target, DestinationSpecials) {
		return target
	}
	escaped := strings.ReplaceAll(target, "<", "%3C")
	escaped = strings.ReplaceAll(escaped, ">", "%3E")
	return "<" + escaped + ">"
}

func escapeLinkText(text string) string {
	escaped := strings.ReplaceAll(text, "[", `\[`)
	escaped = strings.ReplaceAll(escaped, "]", `\]`)
	return escaped
}

// RenderCatalog generates the contents of an index.md catalog for one directory level.
// Output is sorted throughout so reruns produce byte-identical results (Spec 14).
func RenderCatalog(
	title string,
	entries []Document,
	subdirectories []string,
	intro string,
) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("# %s", title), "")

	if intro != "" {
		parts = append(parts, strings.TrimRight(intro, "\r\n"), "")
	}

	if len(subdirectories) > 0 {
		sortedSubdirs := append([]string{}, subdirectories...)
		sort.Strings(sortedSubdirs)

		parts = append(parts, "## Bereiche", "")
		for _, name := range sortedSubdirs {
			line := fmt.Sprintf("* [%s](%s)", escapeLinkText(name), formatDestination(name+"/"))
			parts = append(parts, line)
		}
		parts = append(parts, "")
	}

	if len(entries) > 0 {
		sortedEntries := append([]Document{}, entries...)
		sort.Slice(sortedEntries, func(i, j int) bool {
			return sortedEntries[i].Relative < sortedEntries[j].Relative
		})

		parts = append(parts, "## Dateien", "")
		for _, doc := range sortedEntries {
			line := fmt.Sprintf("* [%s](%s)", escapeLinkText(doc.Title), formatDestination(doc.Relative))
			if doc.Description != "" {
				line += " - " + doc.Description
			}
			parts = append(parts, line)
		}
		parts = append(parts, "")
	}

	parts = append(parts, CatalogRule)
	return strings.Join(parts, "\n") + "\n"
}

// ReadIntro reads a handwritten index.intro.md file if present.
// An empty or whitespace-only intro file is considered an error so authors
// are alerted rather than silently having their file ignored.
func ReadIntro(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	text := string(data)
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("%s: intro file is empty; delete it or write something in it", path)
	}
	return text, nil
}

func writeIfChanged(path string, content string) error {
	existing, err := os.ReadFile(path)
	if err == nil && string(existing) == content {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func parentOf(rel string) string {
	if idx := strings.LastIndex(rel, "/"); idx >= 0 {
		return rel[:idx]
	}
	return ""
}

// WriteCatalogs generates one index.md catalog per directory in the area tree.
// Directories belonging to the area's own wiki bundle are skipped to protect
// handwritten or signpost bundle catalogs (Decision 43).
func WriteCatalogs(area config.Area, documents []Document, targetDir string) error {
	byDirectory := map[string][]Document{"": {}}
	for _, doc := range documents {
		dir := parentOf(doc.Relative)
		byDirectory[dir] = append(byDirectory[dir], doc)
		for dir != "" {
			if _, ok := byDirectory[dir]; !ok {
				byDirectory[dir] = []Document{}
			}
			dir = parentOf(dir)
		}
	}

	prefix := OwnWikiPrefix(area)

	var allDirs []string
	for d := range byDirectory {
		allDirs = append(allDirs, d)
	}
	sort.Strings(allDirs)

	for _, dir := range allDirs {
		if InOwnWiki(dir, prefix) {
			continue
		}

		var children []string
		for other := range byDirectory {
			if other != "" && parentOf(other) == dir {
				childName := other[strings.LastIndex(other, "/")+1:]
				children = append(children, childName)
			}
		}
		sort.Strings(children)

		var entries []Document
		for _, doc := range byDirectory[dir] {
			entry := doc
			if idx := strings.LastIndex(entry.Relative, "/"); idx >= 0 {
				entry.Relative = entry.Relative[idx+1:]
			}
			entries = append(entries, entry)
		}

		title := dir
		if title == "" {
			title = area.Scope
		}

		var catalogPath, introPath string
		if dir != "" {
			catalogPath = filepath.Join(targetDir, filepath.FromSlash(dir), "index.md")
			introPath = filepath.Join(area.Path, filepath.FromSlash(dir), "index.intro.md")
		} else {
			catalogPath = filepath.Join(targetDir, "index.md")
			introPath = filepath.Join(area.Path, "index.intro.md")
		}

		intro, err := ReadIntro(introPath)
		if err != nil {
			return err
		}

		content := RenderCatalog(title, entries, children, intro)
		if err := writeIfChanged(catalogPath, content); err != nil {
			return err
		}
	}

	return nil
}
