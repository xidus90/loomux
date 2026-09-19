package commit

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/child"
)

// DefaultThresholds are the threshold levels measured by calibrate.
var DefaultThresholds = []int{1, 2, 3, 4}

// GitTimeout is the deadline for reading git commit logs.
const GitTimeout = 60 * time.Second

// GitRunner is the process seam used by ReadMessages.
var GitRunner = defaultGitRunner

func defaultGitRunner(dir string, timeout time.Duration, argv ...string) (child.Result, error) {
	res := child.Run(child.Spec{
		Dir:     dir,
		Argv:    argv,
		Timeout: timeout,
	})
	// A start failure carries its cause only in Err; without it a missing git
	// would be reported as "git exited 1".
	return res, res.Err
}

// Calibrate measures which messages would be refused for each threshold.
// It returns a map of threshold to slice of 0-based indices into messages.
func Calibrate(messages []string, lang string, thresholds []int, allow []*regexp.Regexp) map[int][]int {
	result := make(map[int][]int, len(thresholds))
	for _, threshold := range thresholds {
		var refused []int
		for idx, msg := range messages {
			if len(Scan(msg, lang, threshold, allow)) > 0 {
				refused = append(refused, idx)
			}
		}
		result[threshold] = refused
	}
	return result
}

// ReadMessages reads the last count commit messages from git log, newest first.
func ReadMessages(root string, count int) ([]string, error) {
	res, err := GitRunner(root, GitTimeout, "git", "log", "-z", "--format=%B", "-n", strconv.Itoa(count))
	if err != nil {
		return nil, fmt.Errorf("cannot read the history in %s: %w", root, err)
	}
	if res.TimedOut {
		return nil, fmt.Errorf("reading the history in %s took longer than %.0fs", root, GitTimeout.Seconds())
	}
	if res.Code != 0 {
		detail := strings.TrimSpace(res.Stderr)
		if detail == "" {
			detail = fmt.Sprintf("git exited %d", res.Code)
		}
		return nil, fmt.Errorf("cannot read the history in %s: %s", root, detail)
	}

	chunks := strings.Split(res.Stdout, "\x00")
	var messages []string
	for _, chunk := range chunks {
		if strings.TrimSpace(chunk) != "" {
			messages = append(messages, chunk)
		}
	}
	return messages, nil
}

// Render prints the calibration table to w.
func Render(messages []string, lang string, thresholds []int, w io.Writer, allow []*regexp.Regexp) {
	result := Calibrate(messages, lang, thresholds, allow)
	fmt.Fprintf(w, "%d messages, checked as %s\n", len(messages), lang)
	for _, threshold := range thresholds {
		refused := result[threshold]
		fmt.Fprintf(w, "  threshold %d: %d refused\n", threshold, len(refused))
		for _, idx := range refused {
			fmt.Fprintf(w, "    #%d  %s\n", idx+1, Subject(messages[idx]))
		}
	}
}

// Subject returns the first line of a message carrying text, trimmed.
func Subject(msg string) string {
	// splitLines and not a plain split: the table names the line the scan
	// counted, so both have to see the same line boundaries.
	for _, line := range splitLines(msg) {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
