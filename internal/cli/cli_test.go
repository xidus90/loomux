package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func run(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(args, strings.NewReader(""), &out, &errb)
	return code, out.String(), errb.String()
}

func TestVersionPrintsTheVersion(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v"} {
		code, out, _ := run(arg)
		if code != 0 || out != "loomux 0.0.0-dev\n" {
			t.Fatalf("%s: code %d, out %q", arg, code, out)
		}
	}
}

func TestVersionNamesAChannelOtherThanStable(t *testing.T) {
	defer func(v, c string) { Version, Channel = v, c }(Version, Channel)
	for channel, want := range map[string]string{
		"":       "loomux 1.2.3\n",
		"stable": "loomux 1.2.3\n",
		"beta":   "loomux 1.2.3 (beta)\n",
	} {
		Version, Channel = "1.2.3", channel
		if _, out, _ := run("version"); out != want {
			t.Fatalf("channel %q: out %q, want %q", channel, out, want)
		}
	}
}

func TestHelpGoesToStdoutAndSucceeds(t *testing.T) {
	code, out, _ := run("help")
	if code != 0 || !strings.Contains(out, "Usage: loomux <command>") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestHelpListsTheCommandsSorted(t *testing.T) {
	noop := func([]string, io.Reader, io.Writer, io.Writer) int { return 0 }
	commands["zeta"] = noop
	commands["alpha"] = noop
	defer delete(commands, "zeta")
	defer delete(commands, "alpha")
	code, out, _ := run("help")
	// Real commands sort between the probes, so only their order is fixed.
	alpha, zeta := strings.Index(out, "\n  alpha\n"), strings.Index(out, "\n  zeta\n")
	if code != 0 || alpha < 0 || zeta < alpha {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestNoArgumentsIsAUsageError(t *testing.T) {
	code, _, errOut := run()
	if code != 2 || !strings.Contains(errOut, "Usage: loomux <command>") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestUnknownCommandIsAUsageError(t *testing.T) {
	code, _, errOut := run("frobnicate")
	if code != 2 || !strings.Contains(errOut, `unknown command "frobnicate"`) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestKnownCommandReceivesTheRest(t *testing.T) {
	var got []string
	commands["probe"] = func(args []string, _ io.Reader, _, _ io.Writer) int {
		got = args
		return 7
	}
	defer delete(commands, "probe")
	code, _, _ := run("probe", "a", "b")
	if code != 7 || strings.Join(got, ",") != "a,b" {
		t.Fatalf("code %d, args %v", code, got)
	}
}
