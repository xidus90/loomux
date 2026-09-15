package wiki

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/gitenv"
)

type GateViolation struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

// neighbour is the last fallback of Root: a wiki next to the project, of the
// same family (e.g. iam_backend -> iam_wiki) or of the same name.
func neighbour(projectRoot string) string {
	// 3. Neighbour wiki (e.g. iam_backend -> iam_wiki)
	parent := filepath.Dir(projectRoot)
	projectName := filepath.Base(projectRoot)

	for _, suffix := range []string{"_backend", "_frontend", "_workers", "-backend", "-frontend"} {
		if strings.HasSuffix(projectName, suffix) {
			baseName := strings.TrimSuffix(projectName, suffix)
			candidate := filepath.Join(parent, baseName+"_wiki")
			if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
				return candidate
			}
		}
	}

	cand1 := filepath.Join(parent, projectName+"_wiki")
	if fi, err := os.Stat(cand1); err == nil && fi.IsDir() {
		return cand1
	}

	cand2 := filepath.Join(parent, projectName+"-wiki")
	if fi, err := os.Stat(cand2); err == nil && fi.IsDir() {
		return cand2
	}

	return ""
}

func getGitChangedFiles(repoPath string) []string {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	// git's location variables outrank cmd.Dir, and this gate runs as a stop
	// hook -- often inside a git hook that exports them. Inherited, they make
	// the project and its wiki answer about one and the same third repository:
	// both look changed, the drift check never fires, and the gate fails open.
	cmd.Env = gitenv.Environ()
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	// The three fixed characters come off first, and only what is left is
	// trimmed. A porcelain line is two status columns and a blank, and a
	// worktree-only change leaves the first column blank (" M name"), so
	// trimming the whole line first moved the name two characters into the cut
	// and every such path came back without its first two letters.
	var changed []string
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		if len(l) > 3 {
			changed = append(changed, strings.TrimSpace(l[3:]))
		}
	}
	return changed
}

func CheckWikiGate(projectRoot string) []GateViolation {
	var violations []GateViolation

	wikiPath := Root(projectRoot)
	if wikiPath == "" {
		return nil
	}

	// 1. Check code modifications in project
	codeChanges := getGitChangedFiles(projectRoot)
	codeChanged := len(codeChanges) > 0

	// 2. Check wiki modifications
	wikiChanges := getGitChangedFiles(wikiPath)
	wikiChanged := len(wikiChanges) > 0

	// If code changed but wiki was untouched (and wiki is not a subfolder of project or vice versa)
	if codeChanged && !wikiChanged && wikiPath != projectRoot {
		violations = append(violations, GateViolation{
			Name: "wiki-drift",
			Message: fmt.Sprintf("Code in %q was modified (%d changed files), but associated wiki at %q was not updated.",
				filepath.Base(projectRoot), len(codeChanges), wikiPath),
		})
	}

	// 3. Structural Linting
	findings, err := LintBundle(wikiPath)
	if err == nil {
		for _, f := range findings {
			if f.Severity == check.Error {
				violations = append(violations, GateViolation{
					Name:    fmt.Sprintf("wiki-lint:%s", f.Rule),
					Message: fmt.Sprintf("%s: %s", f.Relative, f.Message),
				})
			}
		}
	}

	return violations
}
