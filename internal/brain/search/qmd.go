package search

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const uriPrefix = "qmd://"

var pendingLinePattern = regexp.MustCompile(`(?m)^\s*Pending:\s+(\d+)\s+need embedding`)

// RunnerFunc executes an external command and returns stdout, stderr, exit code, and error.
type RunnerFunc func(argv []string) ([]byte, []byte, int, error)

// QmdPort executes searches via the qmd CLI tool.
type QmdPort struct {
	Executable string
	Runner     RunnerFunc
}

// DefaultRunner executes argv using os/exec.
func DefaultRunner(argv []string) ([]byte, []byte, int, error) {
	if len(argv) == 0 {
		return nil, nil, 1, errors.New("empty command arguments")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			err = nil
		}
	}
	return stdoutBuf.Bytes(), stderrBuf.Bytes(), exitCode, err
}

func (q *QmdPort) getRunner() RunnerFunc {
	if q.Runner != nil {
		return q.Runner
	}
	return DefaultRunner
}

// Search executes query against collections using profile and returns up to n hits.
func (q *QmdPort) Search(query string, collections []string, profile Profile, n int) ([]SearchHit, error) {
	exe := q.Executable
	if exe == "" {
		exe = "qmd"
	}

	subcmd := "query"
	switch profile {
	case ProfileKeyword:
		subcmd = "search"
	case ProfileFast:
		subcmd = "vsearch"
	case ProfileFull:
		subcmd = "query"
	}

	argv := []string{exe, subcmd, query, "--json", "-n", strconv.Itoa(n)}
	for _, col := range collections {
		argv = append(argv, "-c", col)
	}

	runner := q.getRunner()
	stdout, stderr, exitCode, err := runner(argv)
	if err != nil {
		return nil, fmt.Errorf("cannot run %s: %w", argv[0], err)
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("%s exited with %d: %s", argv[0], exitCode, strings.TrimSpace(string(stderr)))
	}

	return parseQmdJSON(stdout)
}

type qmdHitRaw struct {
	File    *string  `json:"file"`
	Line    *int     `json:"line"`
	Title   *string  `json:"title"`
	Snippet *string  `json:"snippet"`
	Score   *float64 `json:"score"`
	DocID   *string  `json:"docid"`
}

func parseQmdJSON(stdout []byte) ([]SearchHit, error) {
	var rawHits []qmdHitRaw
	if err := json.Unmarshal(stdout, &rawHits); err != nil {
		var obj any
		if jsonErr := json.Unmarshal(stdout, &obj); jsonErr == nil {
			if _, isArray := obj.([]any); !isArray {
				return nil, errors.New("expected a list of hits")
			}
		}
		return nil, fmt.Errorf("could not read the search output: %w", err)
	}

	hits := make([]SearchHit, len(rawHits))
	for i, raw := range rawHits {
		if raw.File == nil {
			return nil, errors.New("the hit is missing 'file'")
		}
		if !strings.HasPrefix(*raw.File, uriPrefix) {
			return nil, fmt.Errorf("expected a %s location, found %q", uriPrefix, *raw.File)
		}
		uriRest := (*raw.File)[len(uriPrefix):]
		col, rel, _ := strings.Cut(uriRest, "/")

		if raw.DocID == nil {
			return nil, errors.New("the hit is missing 'docid'")
		}

		line := 0
		if raw.Line != nil {
			line = *raw.Line
		}
		score := 0.0
		if raw.Score != nil {
			score = *raw.Score
		}
		title := ""
		if raw.Title != nil {
			title = *raw.Title
		}
		snippet := ""
		if raw.Snippet != nil {
			snippet = *raw.Snippet
		}

		hits[i] = SearchHit{
			Collection: col,
			Relative:   rel,
			Line:       line,
			Title:      title,
			Snippet:    snippet,
			Score:      score,
			ContentKey: *raw.DocID,
		}
	}
	return hits, nil
}

func (q *QmdPort) invoke(argv []string) ([]byte, error) {
	runner := q.getRunner()
	stdout, stderr, exitCode, err := runner(argv)
	if err != nil {
		return nil, fmt.Errorf("cannot run %s: %w", argv[0], err)
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("%s exited with %d: %s", argv[0], exitCode, strings.TrimSpace(string(stderr)))
	}
	return stdout, nil
}

// Indexed returns every relative path the engine holds for collection.
func (q *QmdPort) Indexed(collection string) ([]string, error) {
	exe := q.Executable
	if exe == "" {
		exe = "qmd"
	}
	stdout, err := q.invoke([]string{exe, "ls", collection})
	if err != nil {
		return nil, err
	}
	var found []string
	lines := strings.Split(string(stdout), "\n")
	for _, line := range lines {
		start := strings.Index(line, uriPrefix)
		if start < 0 {
			continue
		}
		rest := line[start+len(uriPrefix):]
		col, rel, _ := strings.Cut(rest, "/")
		if col == collection {
			found = append(found, strings.TrimSpace(rel))
		}
	}
	return found, nil
}

// Refresh triggers an update of the search index for collections.
func (q *QmdPort) Refresh(collections []string) error {
	exe := q.Executable
	if exe == "" {
		exe = "qmd"
	}
	_, err := q.invoke([]string{exe, "update"})
	return err
}

// NotYetSearchable returns the number of documents pending embedding.
func (q *QmdPort) NotYetSearchable() (int, error) {
	exe := q.Executable
	if exe == "" {
		exe = "qmd"
	}
	stdout, err := q.invoke([]string{exe, "status"})
	if err != nil {
		return 0, err
	}
	match := pendingLinePattern.FindSubmatch(stdout)
	if match == nil {
		return 0, nil
	}
	count, _ := strconv.Atoi(string(match[1]))
	return count, nil
}

// Embed triggers embedding generation for pending documents.
func (q *QmdPort) Embed(collections []string) error {
	exe := q.Executable
	if exe == "" {
		exe = "qmd"
	}
	_, err := q.invoke([]string{exe, "embed"})
	return err
}
