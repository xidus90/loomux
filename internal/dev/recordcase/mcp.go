package recordcase

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/cases"
)

// MCPSpec is one recorded call of the reference's MCP front: which reference,
// which world, which tool, which arguments.
//
// There is no Cmd here and no Exe. The front speaks a protocol, not a command
// line, so what a recording pins is the call -- and the program that serves it
// is named once, in Argv, for the daemon and the front alike.
type MCPSpec struct {
	// Argv is the reference's program and its leading arguments; the verbs
	// ("daemon start", "mcp") are appended to it.
	Argv []string
	// World is the directory staged as the reference's state directory.
	World string
	// Tool and Arguments are the call. Arguments is a JSON object and may
	// contain {{WORLD}}; an empty one stands for {}.
	Tool      string
	Arguments string
	// Channel is written into the case. The reference's front serves the cloud
	// channel and nothing else -- its address is its channel -- so this says
	// which address the replay has to dial, it does not pick one here.
	Channel string
	// Out is the case directory to create, Notes the text of notes.md and
	// Compare the grade ("" for text).
	Out     string
	Notes   string
	Compare string
	// Env and PathPrepend are the recorded process's environment, exactly as a
	// command line recording's are.
	Env         []string
	PathPrepend string
}

// The three seams of this file: a recording starts processes, and a test may
// start only the one it runs in.
var (
	startFront  = spawnFront
	startDaemon = spawnDaemon
	runOnce     = execOnce
)

// daemonTimeout bounds the wait for the daemon to answer on both its pipes. It
// is a variable so that a test of the wait does not have to sit out a minute.
var daemonTimeout = 60 * time.Second

// daemonPoll is how often the wait asks. A handshake over a named pipe answers
// in milliseconds, so this only has to be short beside a cold start.
const daemonPoll = 200 * time.Millisecond

// notRunning is what the reference prints for a channel no daemon serves. The
// status command exits 0 either way, so its words are the only answer there
// is.
const notRunning = "no daemon"

// said is a child's stderr, collected rather than passed through.
//
// Passed through, a recording that is meant to fail -- and a test of one --
// prints the reference's complaint into an otherwise green run, where it reads
// as a failure that is not there. Collected, it goes into the error of the
// recording that earned it, which is the only place it explains anything.
//
// Locked, because the recorder reads it while the child may still be writing:
// the error is built before the process is reaped.
type said struct {
	mu   sync.Mutex
	text strings.Builder
}

func (s *said) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.text.Write(p)
}

// String is what the child has said so far, trimmed, and empty when it said
// nothing -- an error must not end in a colon and a blank.
func (s *said) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.TrimSpace(s.text.String())
}

// front is the reference's stdio front while a recording talks to it.
type front struct {
	in   io.WriteCloser
	out  *bufio.Reader
	cmd  *exec.Cmd
	said *said
}

// because puts what the front said on stderr behind the reason it failed.
func (f *front) because(err error) error {
	if complaint := f.said.String(); complaint != "" {
		return fmt.Errorf("%w; the front said: %s", err, complaint)
	}
	return err
}

// close ends the conversation the way a host hanging up does: stdin shut, then
// the process reaped. Neither failure is reported -- the recording is over by
// then, and its answer does not depend on how the front took its leave.
func (f *front) close() {
	_ = f.in.Close()
	_ = f.cmd.Wait()
}

// RecordMCP runs the reference's MCP front in a staged copy of the world and
// writes the case: the call, the CallToolResult and the world it ran in.
//
// The daemon is started here rather than left to the front's own nudge, and it
// is started as `daemon run` -- in the foreground, as a child of this process
// -- rather than as `daemon start`. Both decisions are measured, not tidiness:
//
//   - The nudge runs in the background, and a call that arrives before the pipe
//     exists is answered with "the brain daemon is not answering", which would
//     go into the recording as the reference's answer.
//   - `daemon start` reaches `client._start_outside_job`, which on Windows
//     creates the daemon through WMI's Win32_Process.Create. A process created
//     that way gets the user's default environment, not ours: neither the
//     directory this recording puts in front of PATH nor the fixture variable
//     beside it arrives. The daemon then finds the machine's real qmd, and
//     every answer that asks the engine -- `status` above all -- is the user's
//     index rather than the world's fixture. `daemon run` is an ordinary child
//     and inherits the environment, which is the whole point of staging one.
//
// The daemon is ended again on the way out, whatever happened, so that a
// recording run leaves no process behind.
func RecordMCP(s MCPSpec) error {
	call, err := s.call()
	if err != nil {
		return err
	}
	for _, entry := range s.Env {
		if !strings.Contains(entry, "=") {
			return fmt.Errorf("environment entry %q is not KEY=VALUE", entry)
		}
	}
	tmp, err := mkdirTemp("", "record-mcp-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := cases.StageWorld(s.World, tmp); err != nil {
		return err
	}
	env := recordEnv(runtime.GOOS, os.Environ(), s.Env, s.PathPrepend, tmp)

	running, err := startDaemon(s.Argv, env, tmp)
	if err != nil {
		return fmt.Errorf("daemon run: %w", err)
	}
	// The two ends, in the order they have to happen: the orderly stop over the
	// pipe first, the kill behind it for a daemon that did not take it. Deferred
	// registration runs backwards, so the kill is registered first.
	//
	// Neither failure is the recording's: by then it either has its answer or
	// has already failed with a better one.
	defer running.kill()
	defer func() { _, _ = runOnce(s.Argv, env, tmp, "daemon", "stop") }()
	if err := awaitDaemon(s.Argv, env, tmp); err != nil {
		if complaint := running.said.String(); complaint != "" {
			return fmt.Errorf("%w; the daemon said: %s", err, complaint)
		}
		return err
	}

	result, err := s.converse(env, tmp, call)
	if err != nil {
		return err
	}
	result.Text = string(cases.Normalize([]byte(result.Text), tmp))
	result.RPCError = string(cases.Normalize([]byte(result.RPCError), tmp))
	return s.write(call, result, tmp)
}

// call is the recorded call, with the arguments read once so that a broken
// object is refused before anything is started.
func (s MCPSpec) call() (cases.MCPCall, error) {
	if len(s.Argv) == 0 {
		return cases.MCPCall{}, errors.New("a recording names its program in Argv")
	}
	if s.Tool == "" {
		return cases.MCPCall{}, errors.New("a recording names the tool it calls")
	}
	arguments := s.Arguments
	if arguments == "" {
		arguments = "{}"
	}
	var probe map[string]any
	if err := json.Unmarshal([]byte(arguments), &probe); err != nil {
		return cases.MCPCall{}, fmt.Errorf("arguments are no JSON object: %w", err)
	}
	return cases.MCPCall{Tool: s.Tool, Arguments: json.RawMessage(arguments), Channel: s.Channel}, nil
}

// converse speaks one whole session to the front: the handshake, the call, and
// the answer.
func (s MCPSpec) converse(env []string, tmp string, call cases.MCPCall) (cases.MCPResult, error) {
	f, err := startFront(s.Argv, env, tmp)
	if err != nil {
		return cases.MCPResult{}, err
	}
	defer f.close()

	f.ask(1, "initialize", initializeParams)
	if _, err := f.answerTo(1); err != nil {
		return cases.MCPResult{}, f.because(fmt.Errorf("handshake: %w", err))
	}
	f.notify("notifications/initialized")

	arguments := strings.ReplaceAll(string(call.Arguments), cases.WorldToken, filepath.ToSlash(tmp))
	// Built as text rather than marshalled: the arguments are already JSON,
	// and a marshalling step here would only be able to fail on an object that
	// `call` has read once and found sound.
	params := fmt.Sprintf(`{"name":%q,"arguments":%s}`, call.Tool, arguments)
	f.ask(2, "tools/call", params)
	answer, err := f.answerTo(2)
	if err != nil {
		return cases.MCPResult{}, f.because(fmt.Errorf("tools/call: %w", err))
	}
	return answer.result()
}

// initializeParams is the handshake this recorder offers. The revision is the
// one the reference's SDK and ours both know; neither side's negotiated
// envelope is compared, so it only has to be accepted.
const initializeParams = `{"protocolVersion":"2025-06-18","capabilities":{},` +
	`"clientInfo":{"name":"loomux-recorder","version":"1"}}`

// ask sends one request.
//
// A write that fails is not reported here, and that is the honest shape: the
// only way it fails is a front that is already gone, and then the answer never
// comes and answerTo says so -- with the reason, which a broken-pipe error
// would not carry.
func (f *front) ask(id int, method, params string) {
	fmt.Fprintf(f.in, `{"jsonrpc":"2.0","id":%d,"method":%q,"params":%s}`+"\n", id, method, params)
}

// notify sends one notification, which has no id and gets no answer.
func (f *front) notify(method string) {
	fmt.Fprintf(f.in, `{"jsonrpc":"2.0","method":%q}`+"\n", method)
}

// rpcAnswer is one response, as much of it as a recording reads.
type rpcAnswer struct {
	ID     *int            `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// answerTo reads until the answer to id arrives. Everything else on the way is
// skipped: the front sends progress notifications while a call is in flight,
// and a reader that took the first line for its answer would record one of
// them instead.
func (f *front) answerTo(id int) (rpcAnswer, error) {
	for {
		line, err := f.out.ReadString('\n')
		if err != nil {
			return rpcAnswer{}, fmt.Errorf("the front ended without answering %d: %w", id, err)
		}
		var answer rpcAnswer
		if json.Unmarshal([]byte(line), &answer) != nil || answer.ID == nil || *answer.ID != id {
			continue
		}
		return answer, nil
	}
}

// result is the answer as a case records it: the text of the CallToolResult
// and isError, or -- for a call the reference never turned into a result --
// the message of the protocol error.
func (a rpcAnswer) result() (cases.MCPResult, error) {
	if a.Error != nil {
		return cases.MCPResult{RPCError: a.Error.Message}, nil
	}
	var payload struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(a.Result, &payload); err != nil {
		return cases.MCPResult{}, fmt.Errorf("the result is no CallToolResult: %w", err)
	}
	var texts []string
	for _, block := range payload.Content {
		texts = append(texts, block.Text)
	}
	return cases.MCPResult{IsError: payload.IsError, Text: strings.Join(texts, "\n")}, nil
}

// write puts the case on disk: the call as it was written, the result as it
// came back, and the world as it was before the run.
//
// There is no world_after. What the run leaves in the state directory is the
// daemon's -- its lock, its pid file and the reconciliation it carries -- and
// none of it belongs to the tool whose answer this case pins.
func (s MCPSpec) write(call cases.MCPCall, result cases.MCPResult, tmp string) error {
	files := map[string]any{"call": call, "result": result}
	if err := os.MkdirAll(s.Out, 0o755); err != nil {
		return err
	}
	for name, value := range files {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(s.Out, name), append(data, '\n'), 0o644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(s.Out, "notes.md"), []byte(s.Notes+"\n"), 0o644); err != nil {
		return err
	}
	if s.Compare != "" {
		if err := os.WriteFile(filepath.Join(s.Out, "compare"), []byte(s.Compare+"\n"), 0o644); err != nil {
			return err
		}
	}
	// The pristine world, not the staged one: a daemon ran in the staged copy.
	return copyTree(s.World, filepath.Join(s.Out, "world"), nil)
}

// spawnFront starts the reference's front with its stdio in our hands.
func spawnFront(argv, env []string, dir string) (*front, error) {
	cmd := exec.Command(argv[0], append(append([]string{}, argv[1:]...), "mcp")...)
	complaint := &said{}
	cmd.Dir, cmd.Env, cmd.Stderr = dir, env, complaint
	in, err := cmd.StdinPipe()
	if err == nil {
		var out io.ReadCloser
		if out, err = cmd.StdoutPipe(); err == nil {
			if err = cmd.Start(); err == nil {
				return &front{in: in, out: bufio.NewReader(out), cmd: cmd, said: complaint}, nil
			}
		}
	}
	return nil, err
}

// execOnce runs the reference once, waits for it, and hands back what it said
// on stdout.
func execOnce(argv, env []string, dir string, args ...string) (string, error) {
	cmd := exec.Command(argv[0], append(append([]string{}, argv[1:]...), args...)...)
	cmd.Dir, cmd.Env = dir, env
	// Stderr deliberately left unset: Output then collects it into the
	// ExitError, where the caller can put it in its own message, instead of
	// printing it into a run that may be perfectly green.
	out, err := cmd.Output()
	return string(out), err
}

// daemon is the reference's daemon while a recording runs beside it.
type daemon struct {
	cmd  *exec.Cmd
	said *said
}

// kill is the backstop behind the orderly stop: a daemon that did not take the
// stop must not outlive the recording.
func (d *daemon) kill() {
	_ = d.cmd.Process.Kill()
	_ = d.cmd.Wait()
}

// spawnDaemon starts `daemon run` beside this process.
//
// Its stdout is discarded and its stderr collected: the daemon says nothing on
// stdout, and what it says while failing is what a broken recording is read by
// -- in the error of that recording, not in the output of a green run.
func spawnDaemon(argv, env []string, dir string) (*daemon, error) {
	cmd := exec.Command(argv[0], append(append([]string{}, argv[1:]...), "daemon", "run")...)
	complaint := &said{}
	cmd.Dir, cmd.Env, cmd.Stderr = dir, env, complaint
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &daemon{cmd: cmd, said: complaint}, nil
}

// awaitDaemon holds the recording until both pipes answer.
//
// It reads the words of `daemon status` rather than its exit code, because the
// reference exits 0 whether or not anything is running: the command reports,
// it does not judge. A channel nothing serves is named with notRunning, so an
// answer without that phrase is a daemon on every channel it listed.
func awaitDaemon(argv, env []string, dir string) error {
	deadline := time.Now().Add(daemonTimeout)
	for {
		out, err := runOnce(argv, env, dir, "daemon", "status")
		if err == nil && out != "" && !strings.Contains(out, notRunning) {
			return nil
		}
		if time.Now().After(deadline) {
			// %v and not %w: the ordinary way to get here is a daemon that
			// never came up, where the status command exits 0 and err is nil.
			// A %w on a nil prints %!w(<nil>), and nothing unwraps this.
			return fmt.Errorf("the daemon did not answer within %s: %s%v", daemonTimeout, out, err)
		}
		time.Sleep(daemonPoll)
	}
}
