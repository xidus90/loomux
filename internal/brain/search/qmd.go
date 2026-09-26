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

	"github.com/xidus90/loomux/internal/brain/pytext"
)

const uriPrefix = "qmd://"

var pendingLinePattern = regexp.MustCompile(`(?m)^\s*Pending:\s+(\d+)\s+need embedding`)

// RunnerFunc executes an external command and returns stdout, stderr, exit code, and error.
type RunnerFunc func(argv []string) ([]byte, []byte, int, error)

// QmdPort executes searches via the qmd CLI tool.
type QmdPort struct {
	Executable string
	// Index names the qmd index to address; empty is qmd's default index.
	Index  string
	Runner RunnerFunc
}

// launcherFailure carries an error of Launcher out of DefaultRunner. qmd.py's _invoke turns
// only an OSError into "cannot run ...", so the refusal its launcher raises reaches the caller
// in its own words; invoke tells the two apart through this type to do the same.
type launcherFailure struct{ err error }

func (f *launcherFailure) Error() string { return f.err.Error() }

// DefaultRunner starts argv through Launcher, so an npm shim never hands the arguments to
// cmd.exe, as qmd.py's _default_runner does. The environment is the caller's own: the default
// backbone CUDA appends nothing, where _default_runner pins QMD_LLAMA_GPU=vulkan.
func DefaultRunner(argv []string) ([]byte, []byte, int, error) {
	if len(argv) == 0 {
		return nil, nil, 1, errors.New("empty command arguments")
	}
	launched, err := Launcher(argv[0])
	if err != nil {
		return nil, nil, 1, &launcherFailure{err: err}
	}
	cmd := exec.Command(launched[0], append(launched[1:], argv[1:]...)...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			err = nil
		}
	}
	return stdoutBuf.Bytes(), stderrBuf.Bytes(), exitCode, err
}

// command is qmd's argv for one subcommand. A named index goes right behind
// the program: qmd reads it as a global option, and it separates both the
// collection list (<name>.yml) and the index (<name>.sqlite) while the
// models stay shared.
func (q *QmdPort) command(args ...string) []string {
	exe := q.Executable
	if exe == "" {
		exe = "qmd"
	}
	argv := []string{exe}
	if q.Index != "" {
		argv = append(argv, "--index", q.Index)
	}
	return append(argv, args...)
}

func (q *QmdPort) getRunner() RunnerFunc {
	if q.Runner != nil {
		return q.Runner
	}
	return DefaultRunner
}

// Search executes query against collections using profile and returns up to n hits.
func (q *QmdPort) Search(query string, collections []string, profile Profile, n int) ([]SearchHit, error) {
	subcmd := "query"
	switch profile {
	case ProfileKeyword:
		subcmd = "search"
	case ProfileFast:
		subcmd = "vsearch"
	case ProfileFull:
		subcmd = "query"
	}

	argv := q.command(subcmd, query, "--json", "-n", strconv.Itoa(n))
	for _, col := range collections {
		argv = append(argv, "-c", col)
	}

	stdout, err := q.invoke(argv)
	if err != nil {
		return nil, err
	}
	return parseQmdJSON(stdout, q.Index)
}

type qmdHitRaw struct {
	File    *string  `json:"file"`
	Line    *int     `json:"line"`
	Title   *string  `json:"title"`
	Snippet *string  `json:"snippet"`
	Score   *float64 `json:"score"`
	DocID   *string  `json:"docid"`
}

// parseQmdJSON reads qmd's --json hits. Searching a named index, qmd appends
// ?index=<name> to every file; only that suffix is cut, so a question mark in
// a file name survives.
func parseQmdJSON(stdout []byte, index string) ([]SearchHit, error) {
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
		if index != "" {
			rel = strings.TrimSuffix(rel, "?index="+index)
		}

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
	var launch *launcherFailure
	if errors.As(err, &launch) {
		return nil, launch.err
	}
	if err != nil {
		return nil, fmt.Errorf("cannot run %s: %w", argv[0], err)
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("%s exited with %d: %s", argv[0], exitCode, pytext.Strip(string(stderr)))
	}
	return stdout, nil
}

// Indexed returns every relative path the engine holds for collection. Lines and paths are
// taken as qmd.py takes them: split like str.splitlines, the path kept as written.
func (q *QmdPort) Indexed(collection string) ([]string, error) {
	stdout, err := q.invoke(q.command("ls", collection))
	if err != nil {
		return nil, err
	}
	var found []string
	for _, line := range pytext.SplitLines(string(stdout)) {
		start := strings.Index(line, uriPrefix)
		if start < 0 {
			continue
		}
		rest := line[start+len(uriPrefix):]
		col, rel, _ := strings.Cut(rest, "/")
		if col == collection {
			found = append(found, rel)
		}
	}
	return found, nil
}

// Refresh triggers an update of the search index for collections.
func (q *QmdPort) Refresh(collections []string) error {
	_, err := q.invoke(q.command("update"))
	return err
}

// NotYetSearchable returns the number of documents pending embedding.
func (q *QmdPort) NotYetSearchable() (int, error) {
	stdout, err := q.invoke(q.command("status"))
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
	_, err := q.invoke(q.command("embed"))
	return err
}
