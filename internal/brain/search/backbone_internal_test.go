package search

import (
	"errors"
	"testing"
)

func unusedConnect(map[string]string) (Session, error) { return nil, errors.New("unused") }

// The port says what it runs on, and its maintenance command line runs on the
// same.
func TestTheMcpPortsCommandLineRunsOnItsBackbone(t *testing.T) {
	port := NewQmdMcpPort(WithBackbone(BackboneCPU), WithConnect(unusedConnect))
	if port.Backbone() != BackboneCPU {
		t.Fatalf("%q", port.Backbone())
	}
	if cli, ok := port.cli.(*QmdPort); !ok || cli.Backbone != BackboneCPU || cli.Executable != "qmd" {
		t.Fatalf("%+v", port.cli)
	}
	if NewQmdMcpPort(WithConnect(unusedConnect)).Backbone() != DefaultBackbone {
		t.Fatal("the default")
	}
}

// TestThePortKnowsWhetherItStartedTheDaemon: the default connect's notice is
// the port's only word that it started the daemon, and the port keeps it
// whether or not its caller asked to hear the notice.
func TestThePortKnowsWhetherItStartedTheDaemon(t *testing.T) {
	var notice func(string)
	connectDefault = func(_ string, _ int, n func(string)) ConnectFunc {
		notice = n
		return unusedConnect
	}
	defer func() { connectDefault = DefaultConnect }()

	var heard []string
	port := NewQmdMcpPort(WithNotice(func(m string) { heard = append(heard, m) }))
	if port.StartedDaemon() {
		t.Fatal("started before anything was asked")
	}
	notice(WarmingNotice)
	if !port.StartedDaemon() || len(heard) != 1 {
		t.Fatalf("started %v, heard %q", port.StartedDaemon(), heard)
	}
	quiet := NewQmdMcpPort()
	notice(WarmingNotice)
	if !quiet.StartedDaemon() {
		t.Fatal("a port without a notice lost the start")
	}
	// A connect of the caller's own says nothing about starts.
	if NewQmdMcpPort(WithConnect(unusedConnect)).StartedDaemon() {
		t.Fatal("started without a default connect")
	}
}
