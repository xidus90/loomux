package hooks

import (
	"errors"
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/verify"
)

func goLanesWorld(t *testing.T) string {
	t.Helper()
	root := goProject(t)
	writeWorldFile(t, root, "a_test.go", "package m\n")
	return root
}

// A lane with nothing to run -- types/go has no command -- is no lane of the
// gate, one with nothing to check -- a module without tests -- is none
// either, and the graph lane is one only where a graph was built.
func TestGateLanesAreTheLanesWithSomethingToRun(t *testing.T) {
	root := goLanesWorld(t)
	got, err := GateLanes(root)
	if err != nil || !slices.Equal(got, []string{"coverage/go@.", "lint/go@.", "test/go@."}) {
		t.Fatalf("%v %v", got, err)
	}
	writeWorldFile(t, root, ".loomux/state/graph/wiring.json", "{}")
	got, err = GateLanes(root)
	if err != nil || !slices.Equal(got, []string{"coverage/go@.", "graph/go@.", "lint/go@.", "test/go@."}) {
		t.Fatalf("with a graph: %v %v", got, err)
	}
	// No test file: the test lane finds no tests and coverage has nothing to
	// measure. Neither would ever leave the list.
	got, err = GateLanes(goProject(t))
	if err != nil || !slices.Equal(got, []string{"lint/go@."}) {
		t.Fatalf("without tests: %v %v", got, err)
	}
}

// The wiki's lane and a project lane are built from kind, stack and area as
// every other: lint/wiki@. and lint/project@., whatever their names print.
func TestGateLanesNameTheWikiAndTheProjectLanes(t *testing.T) {
	root, _ := wikiProject(t, "[verify.project]\nlint = \"echo hi\"\n", cleanPage)
	got, err := GateLanes(root)
	if err != nil || !slices.Equal(got, []string{"lint/project@.", "lint/wiki@."}) {
		t.Fatalf("%v %v", got, err)
	}
}

func TestGateLanesReportAConfigAndAPlanThatFail(t *testing.T) {
	root := goLanesWorld(t)
	writeWorldFile(t, root, ".loomux/config.toml", "[verify\n")
	if _, err := GateLanes(root); err == nil {
		t.Fatal("a config that does not parse went through")
	}
	root = goLanesWorld(t)
	old := stopPlan
	stopPlan = func(verify.Effective, verify.Request, verify.PlanEnv) ([]verify.Job, error) {
		return nil, errors.New("no plan")
	}
	t.Cleanup(func() { stopPlan = old })
	if _, err := GateLanes(root); err == nil || err.Error() != "no plan" {
		t.Fatalf("%v", err)
	}
}

func TestLaneStatesSortEveryLaneAndEveryEntry(t *testing.T) {
	root := goLanesWorld(t)
	if states, exists, err := ReadLaneStates(root); exists || err != nil || len(states.Probation) != 0 {
		t.Fatalf("no file: %+v %v %v", states, exists, err)
	}
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = [\"lint/go@.\", \"lint/python@.\"]\n")
	states, exists, err := ReadLaneStates(root)
	if !exists || err != nil ||
		!slices.Equal(states.Armed, []string{"lint/go@."}) ||
		!slices.Equal(states.Probation, []string{"coverage/go@.", "test/go@."}) ||
		!slices.Equal(states.Orphans, []string{"lint/python@."}) {
		t.Fatalf("%+v %v %v", states, exists, err)
	}
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = 1\n")
	if _, _, err := ReadLaneStates(root); err == nil {
		t.Fatal("an unreadable file went through")
	}
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = []\n")
	writeWorldFile(t, root, ".loomux/config.toml", "[verify\n")
	if _, _, err := ReadLaneStates(root); err == nil {
		t.Fatal("lanes that cannot be planned went through")
	}
}

// A project that ignores .loomux, or the file, never gets it into a commit:
// what it says holds on this machine only, and the states say so. Ignoring
// the state directory alone, as init sets a project up, is no such case.
func TestLaneStatesSayWhenGitIgnoresTheFile(t *testing.T) {
	for ignore, want := range map[string]bool{
		".loomux/\n":           true,
		".loomux/armed.toml\n": true,
		"*.toml\n":             true,
		"/.loomux/state/\n":    false,
		"":                     false,
	} {
		root := goLanesWorld(t)
		gitInit(t, root)
		writeWorldFile(t, root, ".gitignore", ignore)
		writeWorldFile(t, root, ".loomux/armed.toml", "armed = []\n")
		states, exists, err := ReadLaneStates(root)
		if err != nil || !exists || states.Ignored != want {
			t.Errorf("%q: ignored %v, want %v (%v)", ignore, states.Ignored, want, err)
		}
	}
	// No repository: nothing ignores anything.
	root := goLanesWorld(t)
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = []\n")
	if states, _, err := ReadLaneStates(root); err != nil || states.Ignored {
		t.Fatalf("outside a repository: %+v %v", states, err)
	}
}
