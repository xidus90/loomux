// Package child runs one checking tool: a deadline, both streams, and the
// whole process tree dead afterwards -- not only the process it started.
package child

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/gitenv"
)

// DrainGrace is shared by the kill, the reap and the readers after a
// deadline or a clean exit; it is one budget, not one per step.
const DrainGrace = 5 * time.Second

// pipe is a seam: only exhausted handles make os.Pipe fail.
var pipe = os.Pipe

// Spec is one tool run. Env holds extra KEY=VALUE entries; a Timeout of 0
// means no deadline.
type Spec struct {
	Argv    []string
	Dir     string
	Env     []string
	Timeout time.Duration
}

// Result is what the run left behind. Code is -1 when there is none; Err is
// set only when the tool could not be started.
type Result struct {
	Code            int
	Stdout          string
	Stderr          string
	TimedOut        bool
	OutputAbandoned bool
	Err             error
}

// environ puts the forced encoding last: in exec, a later KEY=VALUE wins, so
// no caller can override it.
func environ(extra []string) []string {
	env := append(gitenv.Clean(os.Environ()), extra...)
	return append(env, "PYTHONIOENCODING=utf-8")
}

func text(b []byte) string {
	s := strings.ToValidUTF8(string(b), "�")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// lockedBuffer: Run may read a stream a grandchild still writes to, after it
// gave up on the reader.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf.Bytes()...)
}

type reader struct {
	buf  lockedBuffer
	done chan struct{}
}

func drain(r io.Reader) *reader {
	rd := &reader{done: make(chan struct{})}
	go func() {
		io.Copy(&rd.buf, r)
		close(rd.done)
	}()
	return rd
}

// abandonedBy waits for every reader until the given time and says whether
// one was still open then. A reader that has ended is asked first: once the
// time has passed, a select between its closed channel and an expired timer
// would pick either at random.
func abandonedBy(rds []*reader, until time.Time) bool {
	abandoned := false
	for _, rd := range rds {
		select {
		case <-rd.done:
			continue
		default:
		}
		select {
		case <-rd.done:
		case <-time.After(time.Until(until)):
			abandoned = true
		}
	}
	return abandoned
}

func errStart(argv0 string, err error) error {
	return errors.Join(errors.New(argv0), err)
}

// Run uses os.Pipe and Process.Wait, not StdoutPipe and cmd.Wait: cmd.Wait
// also waits for the pipes to close, so a grandchild holding one would hide
// that the child itself had ended, and OutputAbandoned could not be seen.
func Run(s Spec) Result {
	cmd := exec.Command(s.Argv[0], s.Argv[1:]...)
	cmd.Dir = s.Dir
	cmd.Env = environ(s.Env)
	outR, outW, err := pipe()
	if err != nil {
		return Result{Code: -1, Err: err}
	}
	errR, errW, err := pipe()
	if err != nil {
		outR.Close()
		outW.Close()
		return Result{Code: -1, Err: err}
	}
	cmd.Stdout, cmd.Stderr = outW, errW
	t, err := start(cmd)
	// The parent's write ends close either way: only the child's copies may
	// keep a reader waiting.
	outW.Close()
	errW.Close()
	if err != nil {
		outR.Close()
		errR.Close()
		return Result{Code: -1, Err: errStart(s.Argv[0], err)}
	}
	out, errOut := drain(outR), drain(errR)
	defer outR.Close()
	defer errR.Close()

	var state *os.ProcessState
	exited := make(chan struct{})
	go func() { state, _ = cmd.Process.Wait(); close(exited) }()

	var deadline <-chan time.Time
	if s.Timeout > 0 {
		timer := time.NewTimer(s.Timeout)
		defer timer.Stop()
		deadline = timer.C
	}
	res := Result{Code: -1}
	// One point in time, not one channel: a time.After channel fires once,
	// and a second wait on it after it fired would block for ever.
	var until time.Time
	select {
	case <-exited:
		until = time.Now().Add(DrainGrace)
	case <-deadline:
		res.TimedOut = true
		t.kill()
		until = time.Now().Add(DrainGrace)
		select {
		case <-exited:
		case <-time.After(time.Until(until)):
		}
	}
	res.OutputAbandoned = abandonedBy([]*reader{out, errOut}, until)
	if res.OutputAbandoned {
		t.kill()
	}
	t.release()
	select {
	case <-exited:
		res.Code = state.ExitCode()
	default:
	}
	res.Stdout, res.Stderr = text(out.buf.bytes()), text(errOut.buf.bytes())
	return res
}
