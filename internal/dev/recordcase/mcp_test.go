package recordcase

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/cases"
)

// scriptEnv carries which conversation the helper is to play. It is a second
// variable beside helperEnv because helperEnv already says which shape of old
// tool this binary stands in for, and the MCP front is one shape with many
// scripts.
const scriptEnv = "LOOMUX_RECORDMCP_SCRIPT"

// helperMCP is the reference played by this binary: the daemon verbs, which
// only have to exit, and the front, which speaks a whole session. It runs
// inside TestHelperProcess, so the recorder meets a real process over real
// pipes.
func helperMCP(args []string) {
	script := os.Getenv(scriptEnv)
	verb := strings.Join(args, " ")
	switch {
	case script == "fail-daemon" && verb == "daemon run":
		fmt.Fprintln(os.Stderr, "the daemon refused")
		os.Exit(3)
	case verb == "daemon run":
		// The daemon serves until it is killed. A sleep rather than a bare
		// select: Go's deadlock detector ends a process that has nothing left
		// to run, and this one has to wait to be killed.
		time.Sleep(time.Hour)
	case verb == "daemon status" && (script == "never-ready" || script == "fail-daemon"):
		fmt.Println("no daemon on the pipe (cloud); run `brain daemon start`")
		os.Exit(0)
	case verb == "daemon status":
		fmt.Println("daemon 0.1.0 on the pipe (local)")
		fmt.Println("daemon 0.1.0 on the pipe (cloud)")
		os.Exit(0)
	case verb != "mcp":
		os.Exit(0)
	}
	helperFront(script)
}

// helperFront speaks the stdio side of one conversation.
func helperFront(script string) {
	if script == "silent" {
		// On stderr first: a front that says nothing to its caller may still
		// say why, and the recorder puts that into its error.
		fmt.Fprintln(os.Stderr, "the front found no state directory")
		os.Exit(0)
	}
	in := bufio.NewReader(os.Stdin)
	answers := map[string]string{
		"result":  `{"content":[{"type":"text","text":"the answer"}],"isError":false}`,
		"error":   `{"content":[{"type":"text","text":"no such scope"}],"isError":true}`,
		"world":   `{"content":[{"type":"text","text":"WORLDHERE"}],"isError":false}`,
		"junk":    `"not an object"`,
		"halfway": "",
	}
	for id := 1; ; {
		line, err := in.ReadString('\n')
		if err != nil {
			os.Exit(0)
		}
		var req struct {
			ID     *int            `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal([]byte(line), &req) != nil || req.ID == nil {
			continue
		}
		id = *req.ID
		switch {
		case req.Method == "initialize":
			fmt.Printf(`{"jsonrpc":"2.0","id":%d,"result":{"protocolVersion":"2025-06-18"}}`+"\n", id)
		case script == "rpc-error":
			fmt.Printf(`{"jsonrpc":"2.0","id":%d,"error":{"code":-32602,"message":"bad tool"}}`+"\n", id)
		case script == "halfway":
			os.Exit(0)
		default:
			// A progress notification first: a recorder that read the first
			// line as its answer would record nothing.
			fmt.Println(`{"jsonrpc":"2.0","method":"notifications/progress","params":{"message":"warming"}}`)
			answer := answers[script]
			answer = strings.ReplaceAll(answer, "WORLDHERE", filepath.ToSlash(os.Getenv("BRAIN_STATE_DIR")))
			fmt.Printf(`{"jsonrpc":"2.0","id":%d,"result":%s}`+"\n", id, answer)
		}
	}
}

// helperArgv is the command line that runs this binary as the reference.
func helperArgv(t *testing.T) []string {
	t.Helper()
	t.Setenv(helperEnv, "mcp")
	return []string{os.Args[0], "-test.run=TestHelperProcess", "--"}
}

func mcpSpec(t *testing.T, script string) MCPSpec {
	t.Helper()
	world := filepath.Join(t.TempDir(), "world")
	if err := os.MkdirAll(world, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(world, "registry.toml"), []byte("# empty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return MCPSpec{
		Argv:      helperArgv(t),
		World:     world,
		Tool:      "catalog",
		Arguments: `{"scope":"notes"}`,
		Channel:   "cloud",
		Out:       filepath.Join(t.TempDir(), "out"),
		Notes:     "what it shows",
		Env:       []string{scriptEnv + "=" + script},
	}
}

func readCase(t *testing.T, dir string) (cases.MCPCall, cases.MCPResult) {
	t.Helper()
	var call cases.MCPCall
	var result cases.MCPResult
	for name, into := range map[string]any{"call": &call, "result": &result} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, into); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// The case file is indented for a reader, and MarshalIndent indents the
	// arguments along with it; what a call means is the compact form.
	var compact bytes.Buffer
	if err := json.Compact(&compact, call.Arguments); err != nil {
		t.Fatal(err)
	}
	call.Arguments = json.RawMessage(compact.String())
	return call, result
}

func TestRecordMCPWritesTheCall(t *testing.T) {
	s := mcpSpec(t, "result")
	if err := RecordMCP(s); err != nil {
		t.Fatal(err)
	}
	call, result := readCase(t, s.Out)
	if call.Tool != "catalog" || call.Channel != "cloud" || string(call.Arguments) != `{"scope":"notes"}` {
		t.Fatalf("call: %+v", call)
	}
	if result.Text != "the answer" || result.IsError || result.RPCError != "" {
		t.Fatalf("result: %+v", result)
	}
	notes, err := os.ReadFile(filepath.Join(s.Out, "notes.md"))
	if err != nil || string(notes) != "what it shows\n" {
		t.Fatalf("notes: %q %v", notes, err)
	}
	if _, err := os.Stat(filepath.Join(s.Out, "world", "registry.toml")); err != nil {
		t.Fatalf("the world was not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Out, "compare")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a text case writes no compare file: %v", err)
	}
}

func TestRecordMCPWritesAnErrorResultAndTheGrade(t *testing.T) {
	s := mcpSpec(t, "error")
	s.Compare = cases.CompareOutcome
	if err := RecordMCP(s); err != nil {
		t.Fatal(err)
	}
	_, result := readCase(t, s.Out)
	if !result.IsError || result.Text != "no such scope" {
		t.Fatalf("result: %+v", result)
	}
	grade, err := os.ReadFile(filepath.Join(s.Out, "compare"))
	if err != nil || string(grade) != "outcome\n" {
		t.Fatalf("compare: %q %v", grade, err)
	}
}

func TestRecordMCPWritesAProtocolError(t *testing.T) {
	s := mcpSpec(t, "rpc-error")
	if err := RecordMCP(s); err != nil {
		t.Fatal(err)
	}
	_, result := readCase(t, s.Out)
	if result.RPCError != "bad tool" {
		t.Fatalf("result: %+v", result)
	}
}

// The staged world is a different directory on every machine, so what the
// reference says about it has to come back as the token.
func TestRecordMCPNormalisesTheStagedWorld(t *testing.T) {
	s := mcpSpec(t, "world")
	if err := RecordMCP(s); err != nil {
		t.Fatal(err)
	}
	_, result := readCase(t, s.Out)
	if result.Text != cases.WorldToken {
		t.Fatalf("result: %q", result.Text)
	}
}

func TestRecordMCPRefusesAnIncompleteSpec(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*MCPSpec)
		want string
	}{
		{"no program", func(s *MCPSpec) { s.Argv = nil }, "names its program"},
		{"no tool", func(s *MCPSpec) { s.Tool = "" }, "names the tool"},
		{"broken arguments", func(s *MCPSpec) { s.Arguments = "{" }, "arguments"},
		{"broken environment", func(s *MCPSpec) { s.Env = []string{"NOEQUALS"} }, "KEY=VALUE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := mcpSpec(t, "result")
			tc.edit(&s)
			err := RecordMCP(s)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

// Arguments left out are an empty object, which is what `status` is called
// with and what a recorder must not turn into a null.
func TestRecordMCPDefaultsTheArgumentsToAnEmptyObject(t *testing.T) {
	s := mcpSpec(t, "result")
	s.Arguments = ""
	if err := RecordMCP(s); err != nil {
		t.Fatal(err)
	}
	call, _ := readCase(t, s.Out)
	if string(call.Arguments) != "{}" {
		t.Fatalf("arguments: %s", call.Arguments)
	}
}

func TestRecordMCPReportsAWorldItCannotStage(t *testing.T) {
	s := mcpSpec(t, "result")
	s.World = filepath.Join(t.TempDir(), "nowhere")
	if err := RecordMCP(s); err == nil {
		t.Fatal("expected the staging error")
	}
}

func TestRecordMCPReportsATempDirectoryItCannotMake(t *testing.T) {
	mkdirTemp = func(string, string) (string, error) { return "", errors.New("no temp") }
	defer func() { mkdirTemp = os.MkdirTemp }()
	if err := RecordMCP(mcpSpec(t, "result")); err == nil {
		t.Fatal("expected the temp directory error")
	}
}

// A `daemon run` that dies at once: the wait is what reports it, because the
// start itself succeeded.
func TestRecordMCPReportsADaemonThatWillNotStart(t *testing.T) {
	daemonTimeout = 300 * time.Millisecond
	t.Cleanup(func() { daemonTimeout = 60 * time.Second })
	err := RecordMCP(mcpSpec(t, "fail-daemon"))
	if err == nil || !strings.Contains(err.Error(), "did not answer within") {
		t.Fatalf("expected the wait to be named, got %v", err)
	}
}

func TestRecordMCPReportsAFrontThatSaysNothing(t *testing.T) {
	err := RecordMCP(mcpSpec(t, "silent"))
	if err == nil || !strings.Contains(err.Error(), "handshake") {
		t.Fatalf("expected the handshake to be named, got %v", err)
	}
	// And what it said on stderr while saying nothing on stdout: passed
	// through it would print into a green run, so it travels in the error.
	if !strings.Contains(err.Error(), "the front said: the front found no state directory") {
		t.Errorf("the front's own words are missing from %v", err)
	}
}

func TestRecordMCPReportsAFrontThatHangsUpMidCall(t *testing.T) {
	err := RecordMCP(mcpSpec(t, "halfway"))
	if err == nil || !strings.Contains(err.Error(), "tools/call") {
		t.Fatalf("expected the call to be named, got %v", err)
	}
}

func TestRecordMCPReportsAnOutputDirectoryItCannotMake(t *testing.T) {
	s := mcpSpec(t, "result")
	// A file where the case directory should go: MkdirAll cannot make one.
	if err := os.WriteFile(s.Out, []byte("in the way\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RecordMCP(s); err == nil {
		t.Fatal("expected the output directory error")
	}
}

func TestRecordMCPReportsAProgramThatIsNotThere(t *testing.T) {
	s := mcpSpec(t, "result")
	s.Argv = []string{filepath.Join(t.TempDir(), "no-such-program")}
	if err := RecordMCP(s); err == nil {
		t.Fatal("expected the missing program to be reported")
	}
}

// The front is started with the spec's directory in front of PATH, because the
// reference's engine is a fake put there for the recording.
func TestRecordMCPPutsTheSpecsDirectoryOnPath(t *testing.T) {
	s := mcpSpec(t, "result")
	s.PathPrepend = filepath.Join(t.TempDir(), "bin")
	if err := RecordMCP(s); err != nil {
		t.Fatal(err)
	}
}

// startFront, startDaemon and runOnce are what the seams above stand in for;
// this is the one test that runs them without a recording around them.
func TestTheProcessHelpersRunARealProgram(t *testing.T) {
	// The argv first: it sets the marker this binary reads to know it is the
	// helper, and the environment below is a copy taken after that.
	argv := helperArgv(t)
	env := append(os.Environ(), scriptEnv+"=result")
	out, err := runOnce(argv, env, t.TempDir(), "daemon", "status")
	if err != nil || !strings.Contains(out, "daemon 0.1.0") {
		t.Fatalf("runOnce: %q %v", out, err)
	}
	f, err := startFront(argv, env, t.TempDir())
	if err != nil {
		t.Fatalf("startFront: %v", err)
	}
	f.close()
	d, err := startDaemon(argv, env, t.TempDir())
	if err != nil {
		t.Fatalf("startDaemon: %v", err)
	}
	d.kill()

	gone := []string{filepath.Join(t.TempDir(), "gone")}
	if _, err := runOnce(gone, env, t.TempDir(), "probe"); err == nil {
		t.Fatal("expected a missing program to fail")
	}
	if _, err := startFront(gone, env, t.TempDir()); err == nil {
		t.Fatal("expected a missing program to fail")
	}
	if _, err := startDaemon(gone, env, t.TempDir()); err == nil {
		t.Fatal("expected a missing program to fail")
	}
	var exit *exec.ExitError
	if _, err := runOnce(argv, append(env, scriptEnv+"=fail-daemon"), t.TempDir(), "daemon", "run"); !errors.As(err, &exit) {
		t.Fatalf("expected the exit status, got %v", err)
	}
}

// A daemon that never comes up ends the recording rather than hanging it.
func TestRecordMCPReportsADaemonThatNeverAnswers(t *testing.T) {
	daemonTimeout = 300 * time.Millisecond
	t.Cleanup(func() { daemonTimeout = 60 * time.Second })
	err := RecordMCP(mcpSpec(t, "never-ready"))
	if err == nil || !strings.Contains(err.Error(), "did not answer within") {
		t.Fatalf("expected the wait to be named, got %v", err)
	}
}

func TestRecordMCPReportsAResultThatIsNoCallToolResult(t *testing.T) {
	err := RecordMCP(mcpSpec(t, "junk"))
	if err == nil || !strings.Contains(err.Error(), "CallToolResult") {
		t.Fatalf("expected the shape to be named, got %v", err)
	}
}

// The files of a case, each blocked by a directory of its own name: MkdirAll
// makes the case directory, and WriteFile cannot write over a directory.
func TestRecordMCPReportsAFileItCannotWrite(t *testing.T) {
	for _, name := range []string{"call", "result", "notes.md", "compare"} {
		t.Run(name, func(t *testing.T) {
			s := mcpSpec(t, "result")
			s.Compare = cases.CompareOutcome
			if err := os.MkdirAll(filepath.Join(s.Out, name), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := RecordMCP(s); err == nil {
				t.Fatalf("expected %s to stop the recording", name)
			}
		})
	}
}

// Arguments that are no JSON never reach write through RecordMCP -- the spec
// is read first -- so the refusal is measured where it lives.
func TestWriteRefusesArgumentsThatAreNoJSON(t *testing.T) {
	s := mcpSpec(t, "result")
	err := s.write(cases.MCPCall{Tool: "catalog", Arguments: json.RawMessage("{")}, cases.MCPResult{}, t.TempDir())
	if err == nil {
		t.Fatal("expected the broken arguments to be refused")
	}
}

// A front that cannot be started at all. RecordMCP never gets here with the
// same Argv -- the daemon fails first -- so the arm is measured where it lives.
func TestConverseReportsAFrontThatWillNotStart(t *testing.T) {
	s := mcpSpec(t, "result")
	s.Argv = []string{filepath.Join(t.TempDir(), "no-such-program")}
	if _, err := s.converse(os.Environ(), t.TempDir(), cases.MCPCall{Tool: "catalog", Arguments: json.RawMessage("{}")}); err == nil {
		t.Fatal("expected the missing program to be reported")
	}
}
