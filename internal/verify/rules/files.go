// Package rules applies the layer, pairing and instrumentation rules of
// check rules to the files git knows.
package rules

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/gitenv"
)

// File is one file git knows, Rel its slash-separated path below the root.
// Content is read only for the files the rules parse (see wantsContent) and
// nil for every other one.
type File struct {
	Rel     string
	Content []byte
}

// Files lists the files git knows under root: with cached the index
// (contents from the index), else tracked plus untracked-not-ignored
// (contents from disk; a tracked file missing on disk is left out).
func Files(root string, cached bool) ([]File, error) {
	if cached {
		return indexFiles(root)
	}
	// --stage prefixes tracked entries with their mode, which tells a
	// submodule (a directory on disk) from a file; untracked ones come bare.
	out, err := query.Git(root, "ls-files", "-z", "--stage", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	var files []File
	seen := map[string]bool{}
	for _, entry := range strings.Split(string(out), "\x00") {
		meta, rel, tracked := strings.Cut(entry, "\t")
		if !tracked {
			rel = entry
		}
		// A nested repository is listed as a directory entry: not a file.
		if rel == "" || strings.HasSuffix(rel, "/") || seen[rel] || strings.HasPrefix(meta, "160000 ") {
			continue
		}
		seen[rel] = true
		abs := filepath.Join(root, filepath.FromSlash(rel))
		var data []byte
		var err error
		if wantsContent(rel) {
			data, err = os.ReadFile(abs)
		} else {
			_, err = os.Stat(abs)
		}
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		files = append(files, File{rel, data})
	}
	return files, nil
}

func indexFiles(root string) ([]File, error) {
	out, err := query.Git(root, "ls-files", "-z", "--stage")
	if err != nil {
		return nil, err
	}
	rels, ids := parseStage(out)
	var wanted []string
	for i, rel := range rels {
		if wantsContent(rel) {
			wanted = append(wanted, ids[i])
		}
	}
	contents, err := catBlobs(root, wanted)
	if err != nil {
		return nil, err
	}
	files := make([]File, len(rels))
	for i, rel := range rels {
		files[i] = File{Rel: rel}
		if wantsContent(rel) {
			files[i].Content, contents = contents[0], contents[1:]
		}
	}
	return files, nil
}

// contentExts are the extensions of the files the rules parse: what the
// extractors read. Instrumented reads the first line of the files of a tool
// with a marker; the gdUnit4 tool is the only one, with .gd.
var contentExts = []string{".go", ".py", ".gd", ".tscn", ".tres", ".godot"}

// wantsContent says whether a rule reads the content of rel: an extractor's
// file, go.mod (module paths) and plugin.cfg (test tool detection). The other
// rules need paths only.
func wantsContent(rel string) bool {
	base := path.Base(rel)
	return base == "go.mod" || base == "plugin.cfg" || slices.Contains(contentExts, path.Ext(rel))
}

// parseStage turns `ls-files -z --stage` into paths and blob ids. Submodule
// entries (mode 160000) are no files; a path in conflict keeps its first stage.
func parseStage(out []byte) (rels, ids []string) {
	seen := map[string]bool{}
	for _, entry := range strings.Split(string(out), "\x00") {
		meta, rel, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 || fields[0] == "160000" || seen[rel] {
			continue
		}
		seen[rel] = true
		rels = append(rels, rel)
		ids = append(ids, fields[1])
	}
	return rels, ids
}

// catBlobs reads the blobs with one `cat-file --batch`. The blobs live in
// the object store, so the hook's index file is not needed here.
func catBlobs(root string, ids []string) ([][]byte, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	cmd := exec.Command("git", "cat-file", "--batch")
	cmd.Dir = root
	cmd.Env = gitenv.Environ()
	cmd.Stdin = strings.NewReader(strings.Join(ids, "\n") + "\n")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git cat-file: %w: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}
	return readBatch(bufio.NewReader(bytes.NewReader(out)), len(ids))
}

// readBatch parses n records of `<id> blob <size>\n<content>\n`.
func readBatch(r *bufio.Reader, n int) ([][]byte, error) {
	blobs := make([][]byte, 0, n)
	for range n {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("git cat-file: short output: %w", err)
		}
		f := strings.Fields(header)
		if len(f) != 3 || f[1] != "blob" {
			return nil, fmt.Errorf("git cat-file: unexpected header %q", strings.TrimSpace(header))
		}
		size, err := strconv.Atoi(f[2])
		if err != nil {
			return nil, fmt.Errorf("git cat-file: bad size in %q", strings.TrimSpace(header))
		}
		data := make([]byte, size+1) // the content and the newline after it
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, fmt.Errorf("git cat-file: short blob: %w", err)
		}
		blobs = append(blobs, data[:size])
	}
	return blobs, nil
}
