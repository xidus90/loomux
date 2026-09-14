package cases

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Case represents an isolated test case in the language-independent corpus.
type Case struct {
	Verb          string
	Name          string
	Path          string
	Cmd           string
	Stdin         []byte
	Stdout        []byte
	ExitCode      int
	Notes         string
	HasWorldAfter bool
	// Compare is "data" (exit and stdout) or "message" (exit only).
	Compare string
}

// LoadCase loads a single case from its directory.
func LoadCase(dir string) (*Case, error) {
	cleanDir := filepath.Clean(dir)

	worldInfo, err := os.Stat(filepath.Join(cleanDir, "world"))
	if err != nil || !worldInfo.IsDir() {
		return nil, fmt.Errorf("missing world directory in %s", cleanDir)
	}

	cmdBytes, err := os.ReadFile(filepath.Join(cleanDir, "cmd"))
	if err != nil {
		return nil, fmt.Errorf("missing cmd in %s: %w", cleanDir, err)
	}
	cmd := strings.TrimSpace(string(cmdBytes))
	if cmd == "" {
		return nil, fmt.Errorf("empty cmd in %s", cleanDir)
	}

	exitBytes, err := os.ReadFile(filepath.Join(cleanDir, "exit"))
	if err != nil {
		return nil, fmt.Errorf("missing exit in %s: %w", cleanDir, err)
	}
	exitCode, err := strconv.Atoi(strings.TrimSpace(string(exitBytes)))
	if err != nil {
		return nil, fmt.Errorf("invalid exit code in %s: %w", cleanDir, err)
	}

	stdoutBytes, err := os.ReadFile(filepath.Join(cleanDir, "stdout"))
	if err != nil {
		return nil, fmt.Errorf("missing stdout in %s: %w", cleanDir, err)
	}

	var stdin []byte
	if data, err := os.ReadFile(filepath.Join(cleanDir, "stdin")); err == nil {
		stdin = data
	}

	var notes string
	if data, err := os.ReadFile(filepath.Join(cleanDir, "notes.md")); err == nil {
		notes = string(data)
	}

	compare := "data"
	if data, err := os.ReadFile(filepath.Join(cleanDir, "compare")); err == nil {
		compare = strings.TrimSpace(string(data))
		if compare != "data" && compare != "message" {
			return nil, fmt.Errorf("unknown compare %q in %s", compare, cleanDir)
		}
	}

	hasWorldAfter := false
	if info, err := os.Stat(filepath.Join(cleanDir, "world_after")); err == nil && info.IsDir() {
		hasWorldAfter = true
	}

	name := filepath.Base(cleanDir)
	verb := filepath.Base(filepath.Dir(cleanDir))

	return &Case{
		Verb:          verb,
		Name:          name,
		Path:          cleanDir,
		Cmd:           cmd,
		Stdin:         stdin,
		Stdout:        stdoutBytes,
		ExitCode:      exitCode,
		Notes:         notes,
		HasWorldAfter: hasWorldAfter,
		Compare:       compare,
	}, nil
}

// DiscoverCases scans root recursively for cases, optionally filtering by verb.
func DiscoverCases(root string, verbFilter string) ([]*Case, error) {
	var results []*Case
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		cmdPath := filepath.Join(path, "cmd")
		if info, err := os.Stat(cmdPath); err == nil && !info.IsDir() {
			c, err := LoadCase(path)
			if err != nil {
				return err
			}
			if verbFilter == "" || c.Verb == verbFilter {
				results = append(results, c)
			}
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Verb != results[j].Verb {
			return results[i].Verb < results[j].Verb
		}
		return results[i].Name < results[j].Name
	})
	return results, nil
}
