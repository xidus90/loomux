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
	// Every untracked file by name, not the directory git would fold them
	// into: a wiki directory git has never seen comes back as its parent
	// ("?? docs/"), and that parent lies outside the wiki the split below
	// looks for -- a bundle written from scratch read as untouched.
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=all")
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

	// 1. and 2. Code modifications and wiki modifications
	codeChanges, wikiChanges := changesOf(projectRoot, wikiPath)

	// If code changed but the wiki was untouched
	if len(codeChanges) > 0 && len(wikiChanges) == 0 {
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

// changesOf answers what the project changed and what its wiki changed. A
// wiki of the project's own repository is a directory in it, and `git status`
// answers for the whole repository wherever it is asked -- a second call from
// the wiki repeats the project's answer, so the two are told apart by path.
// Asked separately are the two wikis that are no directory of this working
// tree: one beside the project, and one that carries a repository of its own
// (a nested checkout, a linked worktree, a submodule -- each a `.git`).
//
// projectRoot is the top of its working tree; every caller passes a project
// root, and porcelain paths are relative to that top.
func changesOf(projectRoot, wikiPath string) (code, wikiChanges []string) {
	changes := getGitChangedFiles(projectRoot)
	relative, err := filepath.Rel(projectRoot, wikiPath)
	outside := err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator))
	if outside || isOwnRepository(wikiPath) {
		return changes, getGitChangedFiles(wikiPath)
	}
	return splitAtWiki(changes, filepath.ToSlash(relative))
}

// isOwnRepository reads the administrative entry, not its kind: a nested
// checkout keeps a directory there, a linked worktree and a submodule a file.
func isOwnRepository(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

// splitAtWiki sorts repository-relative paths by the wiki's own, which is
// repository-relative too and carries forward slashes like them. The
// separator belongs in the comparison: it keeps "docs/wikipedia.md" out of
// the bundle at "docs/wiki". A wiki declared at the project root holds every
// path: a project whose wiki is the project cannot drift from itself. Only a
// project root that carries no repository of its own arrives here that way --
// a root that does goes the separate way above, which is where the old
// `wikiPath != projectRoot` guard now lives.
func splitAtWiki(changes []string, wiki string) (code, wikiChanges []string) {
	prefix := wiki + "/"
	if wiki == "." {
		prefix = ""
	}
	for _, path := range changes {
		if strings.HasPrefix(path, prefix) {
			wikiChanges = append(wikiChanges, path)
			continue
		}
		code = append(code, path)
	}
	return code, wikiChanges
}
