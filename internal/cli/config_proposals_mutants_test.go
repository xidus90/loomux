package cli

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// isolateProposalMutantEnv points every directory the config commands may read
// from the environment at a temporary one, so no test meets the machine's.
func isolateProposalMutantEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"LOOMUX_STATE_DIR", "XDG_CONFIG_HOME", "LOCALAPPDATA"} {
		t.Setenv(name, t.TempDir())
	}
}

// proposalSelfLoop makes path a link to itself: it shows up in its directory's
// listing, but opening it fails with an error that is not "does not exist".
func proposalSelfLoop(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if filepath.Separator == '\\' {
		// A junction is the link Windows lets an unelevated user make.
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", path, path).CombinedOutput(); err != nil {
			t.Skipf("no junction on this machine: %v (%s)", err, out)
		}
		return
	}
	if err := os.Symlink(path, path); err != nil {
		t.Skipf("no symlink on this machine: %v", err)
	}
}

// stubProposalLink replaces the hard link for one test.
func stubProposalLink(t *testing.T, link func(string, string) error) {
	t.Helper()
	restore := linkFile
	linkFile = link
	t.Cleanup(func() { linkFile = restore })
}

// The name is claimed by linking the complete temporary file: what is
// linked is a file that exists and already holds the proposal.
func TestStoreProposalLinksTheWrittenTemporaryFile(t *testing.T) {
	var linked []string
	stubProposalLink(t, func(oldname, newname string) error {
		data, err := os.ReadFile(oldname)
		if err != nil || !strings.Contains(string(data), `"key": "a"`) {
			t.Errorf("linked %q: %q %v", oldname, data, err)
		}
		linked = append(linked, oldname)
		return os.Link(oldname, newname)
	})
	dir := t.TempDir()
	id, err := storeProposal(dir, time.Date(2026, 9, 24, 10, 15, 30, 0, time.UTC), proposal{Op: "set", Key: "a"})
	if err != nil || id != "20260924T101530Z-001" || len(linked) != 1 {
		t.Fatalf("%s %v %v", id, err, linked)
	}
	if names := proposalFiles(t, dir); len(names) != 1 {
		t.Fatalf("files %v", names)
	}
}

// A temporary file that cannot be written ends the store; no name is
// claimed for a proposal that was never written.
func TestStoreProposalStopsWhenTheTemporaryFileFails(t *testing.T) {
	stubProposalLink(t, func(oldname, newname string) error {
		t.Errorf("link %q -> %q after the temporary file failed", oldname, newname)
		return errors.New("not supported")
	})
	_, err := storeProposal(filepath.Join(t.TempDir(), "missing", "\x00"), time.Now(), proposal{})
	if err == nil {
		t.Fatal("no error")
	}
}

// A link that finds its name taken moves on to the next name; the exclusive
// create is the stand-in for a file system without links, not a second try
// at a name another writer holds.
func TestStoreProposalMovesOnWhenTheLinkFindsTheNameTaken(t *testing.T) {
	calls := 0
	stubProposalLink(t, func(oldname, newname string) error {
		calls++
		if calls == 1 {
			return &fs.PathError{Op: "link", Path: newname, Err: fs.ErrExist}
		}
		return os.Link(oldname, newname)
	})
	dir := t.TempDir()
	id, err := storeProposal(dir, time.Date(2026, 9, 24, 10, 15, 30, 0, time.UTC), proposal{Op: "set", Key: "a"})
	if err != nil || id != "20260924T101530Z-002" {
		t.Fatalf("%s %v", id, err)
	}
	if names := proposalFiles(t, dir); strings.Join(names, " ") != "20260924T101530Z-002.json" {
		t.Fatalf("files %v", names)
	}
}

// A proposal directory that cannot be listed fails the command with the
// listing's own error, even when the config file reads.
func TestConfigProposalCommandsReportADirectoryThatCannotBeListed(t *testing.T) {
	isolateProposalMutantEnv(t)
	root := configRoot(t, "[commit]\nthreshold = 4\n")
	proposalSelfLoop(t, proposalDir(root))
	if _, err := os.ReadDir(proposalDir(root)); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Skipf("the loop lists or reads as absent here: %v", err)
	}
	// The listing's error names the directory; "no open proposal" does not.
	listed := filepath.Join("state", "config", "proposals")
	if code, out, errOut := runConfig(t, "", "proposals", "--root", root); code != 1 || out != "" || !strings.Contains(errOut, listed) {
		t.Errorf("proposals: %d %q %q", code, out, errOut)
	}
	for _, args := range [][]string{{"apply", "a-1", "--yes"}, {"reject", "a-1"}} {
		code, _, errOut := runConfig(t, "", append(args, "--root", root)...)
		if code != 1 || strings.Contains(errOut, "no open proposal") || !strings.Contains(errOut, listed) {
			t.Errorf("%v: %d %q", args, code, errOut)
		}
	}
}

// A proposal file that is listed but cannot be opened is reported with the
// error of opening it, not as a file that failed to parse.
func TestConfigProposalsReportsAProposalFileThatCannotBeOpened(t *testing.T) {
	isolateProposalMutantEnv(t)
	root := configRoot(t, "")
	loop := filepath.Join(proposalDir(root), "x.json")
	proposalSelfLoop(t, loop)
	if _, err := os.ReadFile(loop); err == nil {
		t.Skip("the loop reads here")
	}
	code, _, errOut := runConfig(t, "", "proposals", "--root", root)
	if code != 1 || !strings.Contains(errOut, "proposal x:") || !strings.Contains(errOut, "x.json") ||
		strings.Contains(errOut, "JSON") {
		t.Fatalf("%d %q", code, errOut)
	}
}

// A proposal file that does not parse is reported by its parse error; apply
// never goes on to compute an empty proposal against the file.
func TestConfigApplyReportsAProposalThatDoesNotParse(t *testing.T) {
	isolateProposalMutantEnv(t)
	root := configRoot(t, "")
	forgeRaw(t, proposalDir(root), "a-1", "{")
	code, _, errOut := runConfig(t, "", "apply", "a-1", "--yes", "--root", root)
	if code != 1 || !strings.Contains(errOut, "proposal a-1: unexpected end of JSON input; the proposal stays") ||
		strings.Contains(errOut, "unknown operation") {
		t.Fatalf("%d %q", code, errOut)
	}
}

// A config file that does not read stops apply before anything is computed
// or written.
func TestConfigApplyWritesNothingWhenTheFileDoesNotRead(t *testing.T) {
	isolateProposalMutantEnv(t)
	restore := applyWrite
	applyWrite = func(configTarget, string, string) error {
		t.Error("wrote although the file did not read")
		return nil
	}
	t.Cleanup(func() { applyWrite = restore })
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	forgeProposal(t, proposalDir(root), proposal{ID: "a-1", Op: "set", Key: "commit.threshold", Input: "4"})
	code, _, errOut := runConfig(t, "", "apply", "a-1", "--yes", "--root", root)
	if code != 1 || !strings.Contains(errOut, "config.toml") || !strings.Contains(errOut, "the proposal stays") ||
		len(proposalFiles(t, proposalDir(root))) != 1 {
		t.Fatalf("%d %q", code, errOut)
	}
}
