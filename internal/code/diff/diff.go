// Package diff reads what git says about a change: which files, and which
// lines of their post-image. It parses and never runs git; the caller hands in
// the output of `git diff --name-status -z` and of `git diff --unified=0`.
package diff

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Status is the one-letter change kind git prints in --name-status.
type Status byte

const (
	Added       Status = 'A'
	Modified    Status = 'M'
	Deleted     Status = 'D'
	Renamed     Status = 'R'
	Copied      Status = 'C'
	TypeChanged Status = 'T'
)

// unmerged is the status of a path with a conflict still in the index. It
// never reaches a File: ParseNameStatus refuses it with ErrUnmerged.
const unmerged Status = 'U'

// ErrUnmerged is a path whose conflict is not resolved: it has no single
// post-image whose lines a blast radius could read.
var ErrUnmerged = errors.New("unresolved conflict")

// MarshalText makes a Status serialise as its letter instead of a number.
func (s Status) MarshalText() ([]byte, error) { return []byte{byte(s)}, nil }

// The caps keep a large change readable; the ranges of a hunk stay complete,
// only its lines are cut.
const (
	MaxHunkLines = 24
	MaxFileLines = 200
)

// Hunk is one change range in the post-image. A pure deletion (+c,0) is the
// one line before the gap, and line 1 when the gap is at the top.
type Hunk struct {
	From    int      `json:"from"`
	To      int      `json:"to"`
	Lines   []string `json:"lines,omitempty"`   // '+' and '-' lines, capped; context is left out
	Omitted int      `json:"omitted,omitempty"` // lines beyond the caps
}

// File is one changed path; OldPath is set for a rename or a copy.
type File struct {
	Status  Status `json:"status"`
	Path    string `json:"path"`
	OldPath string `json:"old_path,omitempty"`
	Hunks   []Hunk `json:"hunks,omitempty"`
}

// ParseNameStatus reads `git diff --name-status -z`: a status field, then one
// path, or two (old, new) for a rename or a copy, each ended by a NUL.
func ParseNameStatus(r io.Reader) ([]File, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	fields := strings.Split(string(raw), "\x00")
	var out []File
	for i := 0; i < len(fields); i++ {
		tok := fields[i]
		if tok == "" && i == len(fields)-1 {
			break
		}
		if tok == "" {
			return nil, fmt.Errorf("name-status: empty status at field %d", i)
		}
		st := Status(tok[0])
		paths := 1
		switch st {
		case Renamed, Copied:
			paths = 2
		case Added, Modified, Deleted, TypeChanged, unmerged:
		default:
			return nil, fmt.Errorf("name-status: unknown status %q", tok)
		}
		if i+paths >= len(fields) || fields[i+paths] == "" {
			return nil, fmt.Errorf("name-status: %q without its path", tok)
		}
		if st == unmerged {
			return nil, fmt.Errorf("%w in %s", ErrUnmerged, fields[i+1])
		}
		f := File{Status: st, Path: fields[i+paths]}
		if paths == 2 {
			f.OldPath = fields[i+1]
		}
		out = append(out, f)
		i += paths
	}
	return out, nil
}

// ApplyHunks reads `git diff --unified=0` and attaches each hunk to the file
// of files with the same post-image path. Files the patch names but files
// lacks are skipped. Hunk bodies are consumed by their header counts, so a
// removed line that reads like a "--- " header stays a line. A hunk git fused
// under diff.interHunkContext keeps its context lines inside From..To, so a
// caller that maps ranges to symbols pins --inter-hunk-context=0.
func ApplyHunks(files []File, r io.Reader) error {
	byPath := make(map[string]*File, len(files))
	for i := range files {
		byPath[files[i].Path] = &files[i]
	}
	br := bufio.NewReader(r)
	var cur *File
	minus := ""
	fileLines := 0
	oldLeft, newLeft := 0, 0
	for {
		line, err := br.ReadString('\n')
		if line == "" && err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if oldLeft > 0 || newLeft > 0 {
			switch {
			case strings.HasPrefix(line, "-"):
				oldLeft--
			case strings.HasPrefix(line, "+"):
				newLeft--
			case strings.HasPrefix(line, " "):
				// Context between hunks git fused under
				// diff.interHunkContext: in both images, changed in neither.
				oldLeft--
				newLeft--
				continue
			default: // "\ No newline at end of file"
				continue
			}
			if cur != nil {
				h := &cur.Hunks[len(cur.Hunks)-1]
				if len(h.Lines) < MaxHunkLines && fileLines < MaxFileLines {
					h.Lines = append(h.Lines, line)
					fileLines++
				} else {
					h.Omitted++
				}
			}
			continue
		}
		switch {
		case strings.HasPrefix(line, "diff --git "):
			cur, minus, fileLines = nil, "", 0
		case strings.HasPrefix(line, "--- "):
			p, err := headerPath(line[4:])
			if err != nil {
				return err
			}
			minus = p
		case strings.HasPrefix(line, "+++ "):
			p, err := headerPath(line[4:])
			if err != nil {
				return err
			}
			if p == "" {
				p = minus
			}
			cur, fileLines = byPath[p], 0
		case strings.HasPrefix(line, "@@ "):
			h, o, n, err := parseHunkHeader(line)
			if err != nil {
				return err
			}
			oldLeft, newLeft = o, n
			if cur != nil {
				cur.Hunks = append(cur.Hunks, h)
			}
		}
		if err == io.EOF {
			return nil
		}
	}
}

// headerPath is the path of a ---/+++ line without its a/ or b/ prefix, ""
// for /dev/null. Git ends a name holding a space with a tab and C-quotes a
// name holding a quote, a backslash or a control character.
func headerPath(s string) (string, error) {
	s = strings.TrimSuffix(s, "\t")
	if s == "/dev/null" {
		return "", nil
	}
	if strings.HasPrefix(s, `"`) {
		u, err := strconv.Unquote(s)
		if err != nil {
			return "", fmt.Errorf("diff: cannot unquote %s: %w", s, err)
		}
		s = u
	}
	if len(s) > 2 && (s[:2] == "a/" || s[:2] == "b/") {
		s = s[2:]
	}
	return s, nil
}

// parseHunkHeader reads "@@ -a[,b] +c[,d] @@" into the post-image range and
// the number of removed and added lines that follow it.
func parseHunkHeader(line string) (Hunk, int, int, error) {
	parts := strings.Fields(line)
	if len(parts) < 3 || !strings.HasPrefix(parts[1], "-") || !strings.HasPrefix(parts[2], "+") {
		return Hunk{}, 0, 0, fmt.Errorf("diff: bad hunk header %q", line)
	}
	_, oldCount, err := rangeOf(parts[1][1:])
	if err != nil {
		return Hunk{}, 0, 0, fmt.Errorf("diff: bad hunk header %q: %w", line, err)
	}
	start, newCount, err := rangeOf(parts[2][1:])
	if err != nil {
		return Hunk{}, 0, 0, fmt.Errorf("diff: bad hunk header %q: %w", line, err)
	}
	if newCount == 0 {
		at := max(start, 1)
		return Hunk{From: at, To: at}, oldCount, 0, nil
	}
	return Hunk{From: start, To: start + newCount - 1}, oldCount, newCount, nil
}

// rangeOf reads "start[,count]"; a missing count is 1.
func rangeOf(s string) (start, count int, err error) {
	a, b, found := strings.Cut(s, ",")
	if start, err = strconv.Atoi(a); err != nil {
		return 0, 0, err
	}
	if !found {
		return start, 1, nil
	}
	count, err = strconv.Atoi(b)
	return start, count, err
}
