package answer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
)

// emptyVault is a state directory with a registry that holds no area: the
// smallest world an answer can be asked in. Without the file every command
// fails in privacy.VisibleAreas before it reaches its engine, so a bare
// t.TempDir() would test the missing file and nothing else.
func emptyVault(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "registry.toml"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// stubbedStatusPort answers the backlog question from memory. The default port
// runs the qmd executable, and a status test that asks the machine it runs on
// reports the machine, not the answer.
func stubbedStatusPort() answer.Ports {
	ports := answer.DefaultPorts()
	port := search.NewFakePort()
	ports.Status = func() search.SearchPort { return port }
	return ports
}

func TestRunStatusAnswersWithoutANotice(t *testing.T) {
	dir := emptyVault(t)
	var heard []string
	text, notes, err := answer.RunWith(stubbedStatusPort(), answer.Request{
		Command: "status",
		Channel: privacy.ChannelLocal,
	}, dir, func(m string) { heard = append(heard, m) })
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if text == "" {
		t.Error("status answered with empty text")
	}
	if len(notes) != 0 {
		t.Errorf("status produced notes: %v", notes)
	}
	if len(heard) != 0 {
		t.Errorf("status produced a notice: %v", heard)
	}
}

func TestRunRefusesAnUnknownCommand(t *testing.T) {
	dir := t.TempDir()
	_, _, err := answer.Run(answer.Request{Command: "nonesuch"}, dir, func(string) {})
	if err == nil {
		t.Fatal("expected an error for an unknown command")
	}
	if !strings.Contains(err.Error(), "nonesuch") {
		t.Errorf("error does not name the command: %v", err)
	}
}

func TestRunToleratesANilNotice(t *testing.T) {
	// serve and the corpus both pass a notice; a caller that does not must not
	// crash the answer.
	dir := emptyVault(t)
	if _, _, err := answer.RunWith(stubbedStatusPort(), answer.Request{
		Command: "status",
		Channel: privacy.ChannelLocal,
	}, dir, nil); err != nil {
		t.Fatalf("Run with nil notice: %v", err)
	}
}

// TestDefaultPortsWithHandsTheOptionsToTheQmdPort: what a caller says about
// the qmd port has to arrive there, and the lock file is where it shows. The
// search fails -- there is no qmd on this PATH -- but the lock is taken before
// the start, and it is taken where the caller said, not in the global state
// directory.
func TestDefaultPortsWithHandsTheOptionsToTheQmdPort(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	global := t.TempDir()
	t.Setenv(config.StateDirEnv, global)
	mine := filepath.Join(t.TempDir(), "qmd.lock")

	ports := answer.DefaultPortsWith(global, search.WithQmdLock(mine), search.WithPort(64994))
	if _, err := ports.Search(nil).Search("q", []string{"c"}, search.ProfileFast, 1); err == nil {
		t.Fatal("a search without qmd on PATH answered")
	}
	if _, err := os.Stat(mine); err != nil {
		t.Errorf("the lock the caller named was not taken: %v", err)
	}
	if _, err := os.Stat(filepath.Join(global, "qmd.lock")); err == nil {
		t.Error("the global state directory was locked although the caller named its own")
	}
}

// TestRunForAnswersLikeRun: the function serve keeps has the shape of Run and
// answers the same way. An unknown command never reaches an engine, which is
// what makes it safe to ask here.
func TestRunForAnswersLikeRun(t *testing.T) {
	dir := t.TempDir()
	run := answer.RunFor(dir, search.WithQmdLock(filepath.Join(dir, "qmd.lock")))
	_, _, err := run(answer.Request{Command: "nonesuch"}, dir, nil)
	if err == nil || !strings.Contains(err.Error(), "nonesuch") {
		t.Fatalf("got %v", err)
	}
}

// globalState is a state directory whose machine-wide config.toml says text.
func globalState(t *testing.T, text string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestThePortsRunOnTheMachinesBackbone: the search and the status engine both
// take [search] backbone from the state directory they are built for, and a
// caller's own option still has the last word.
func TestThePortsRunOnTheMachinesBackbone(t *testing.T) {
	state := globalState(t, "[search]\nbackbone = \"vulkan\"\n")
	ports := answer.DefaultPortsWith(state)
	if port, ok := ports.Search(nil).(*search.QmdMcpPort); !ok || port.Backbone() != search.BackboneVulkan {
		t.Fatalf("search %+v", ports.Search(nil))
	}
	if port, ok := ports.Status().(*search.QmdPort); !ok || port.Backbone != search.BackboneVulkan || port.Runner != nil {
		t.Fatalf("status %+v", ports.Status())
	}
	mine := answer.DefaultPortsWith(state, search.WithBackbone(search.BackboneCPU))
	if port := mine.Search(nil).(*search.QmdMcpPort); port.Backbone() != search.BackboneCPU {
		t.Fatalf("the caller's option was overridden: %q", port.Backbone())
	}
	// The command line's ports read the state directory of the environment.
	t.Setenv(config.StateDirEnv, globalState(t, "[search]\nbackbone = \"cpu\"\n"))
	if port := answer.DefaultPorts().Search(nil).(*search.QmdMcpPort); port.Backbone() != search.BackboneCPU {
		t.Fatalf("default ports %q", port.Backbone())
	}
}

// TestABrokenBackboneStopsOnlyTheEngine: a [search] block that does not read
// is the answer of every engine question, naming the file; the ports are
// still built, so a command that asks no engine answers.
func TestABrokenBackboneStopsOnlyTheEngine(t *testing.T) {
	state := globalState(t, "[search]\nbackbone = \"metal\"\n")
	path := filepath.Join(state, "config.toml")
	ports := answer.DefaultPortsWith(state)
	if _, err := ports.Search(nil).Search("q", nil, search.ProfileFast, 1); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("search: %v", err)
	}
	if _, err := ports.Status().NotYetSearchable(); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("status: %v", err)
	}
}
