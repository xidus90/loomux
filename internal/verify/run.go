package verify

import (
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/xidus90/loomux/internal/child"
)

// Outcome is how one job ended. BlockedBy names the predecessor whose red
// kept it from starting. Probation says its red does not fail the run: for a
// blocked lane it is the answer of the lane that blocks it, whatever its own
// arming; for every other lane, that the lane is not armed.
type Outcome struct {
	Job       Job
	State     State
	Output    string
	Duration  time.Duration
	BlockedBy string
	Probation bool
}

// RunOptions is what a run needs from outside. Timeout caps each process, 0
// meaning none; Budget, when positive, caps the whole run from its start.
// Armed says whether a lane's red fails the run; nil arms every lane.
type RunOptions struct {
	Scope           Scope
	MaxParallel     int
	Timeout, Budget time.Duration
	Start           func(child.Spec) child.Result
	Look            func(string) (string, error)
	Now             func() time.Time
	Armed           func(Job) bool
}

const abandoned = "output abandoned: a process the tool started kept its pipe open"

// Red says whether a state fails a run. A tool that is missing or a project
// that is not ready is a finding for a check, but an edit cannot fix it.
func Red(s State, scope Scope) bool {
	switch s {
	case StateFailed, StateTimedOut, StateBlocked:
		return true
	case StateMissingTool, StateUnready:
		return scope == ScopeCheck
	}
	return false
}

// runner is one call of Run: its options, its process cap and its deadline,
// zero when there is no budget.
type runner struct {
	opt      RunOptions
	sem      chan struct{}
	deadline time.Time
}

// step is one command's verdict and what it printed.
type step struct {
	state  State
	output string
}

// Run drives every job in its own goroutine; a job waits for the one its
// After names, wherever that sits in the slice. The cap is taken around each
// process and never while waiting, so a chain cannot hold the slot it needs.
func Run(jobs []Job, opt RunOptions) []Outcome {
	r := &runner{opt: opt, sem: make(chan struct{}, max(opt.MaxParallel, 1))}
	if opt.Budget > 0 {
		r.deadline = opt.Now().Add(opt.Budget)
	}
	out := make([]Outcome, len(jobs))
	done := make([]chan struct{}, len(jobs))
	for i := range done {
		done[i] = make(chan struct{})
	}
	for i, job := range jobs {
		go func() {
			defer close(done[i])
			if job.After >= 0 {
				<-done[job.After]
				if o, settled := r.inherit(job, out[job.After]); settled {
					out[i] = o
					return
				}
			}
			start := opt.Now()
			o := r.lane(job)
			o.Duration = opt.Now().Sub(start)
			o.Probation = r.probation(job)
			out[i] = o
		}()
	}
	for _, d := range done {
		<-d
	}
	return out
}

// inherit decides a job by its predecessor: a red one blocks it, and one that
// could not judge leaves it unable to judge as well. The predecessor blocks
// whether or not it is armed -- what it should have written is missing either
// way -- but a block counts as red only where the lane that blocks does.
func (r *runner) inherit(job Job, pred Outcome) (Outcome, bool) {
	if Red(pred.State, r.opt.Scope) {
		return Outcome{Job: job, State: StateBlocked, BlockedBy: pred.Job.Name, Probation: pred.Probation}, true
	}
	switch pred.State {
	case StateUnavailable, StateNotApplicable:
		return Outcome{Job: job, State: pred.State, Probation: r.probation(job)}, true
	case StateBudget, StateMissingTool, StateUnready:
		// In a check only a spent budget gets here, the other two are red. A
		// lane that reads its predecessor's files has nothing to read then;
		// one that only waits for it runs as it would have.
		if r.opt.Scope == ScopeEdit || job.Consumes {
			return Outcome{Job: job, State: pred.State, Probation: r.probation(job)}, true
		}
	}
	return Outcome{}, false
}

// probation says whether job's lane is not armed.
func (r *runner) probation(job Job) bool {
	return r.opt.Armed != nil && !r.opt.Armed(job)
}

// lane runs one job whose predecessor let it: a planned state, an in-process
// function, or its commands after the tools and coverage files are there.
func (r *runner) lane(job Job) Outcome {
	o := Outcome{Job: job}
	switch {
	case job.Pre != "":
		o.State, o.Output = job.Pre, job.Note
	case job.Fn != nil:
		o.State, o.Output = fnResult(job.Fn)
	default:
		o.State, o.Output = r.commands(job)
	}
	return o
}

// fnResult turns an in-process lane's answer into a verdict; its error is
// the last line of the output.
func fnResult(fn func() (string, error)) (State, string) {
	out, err := fn()
	if err == nil {
		return StateOK, out
	}
	return StateFailed, withLine(out, err.Error())
}

// commands runs a lane's processes: every tool looked up before any starts,
// the measure step, the files it should have left, then every command, even
// after a red one.
func (r *runner) commands(job Job) (State, string) {
	all := job.Argvs
	if len(job.Measure) > 0 {
		all = append([][]string{job.Measure}, all...)
	}
	for _, argv := range all {
		if _, err := r.opt.Look(argv[0]); err != nil {
			return StateMissingTool, `"` + argv[0] + `" is not on PATH: ` + strings.Join(argv, " ")
		}
	}
	if len(job.Measure) > 0 {
		// A measure that did not finish leaves nothing to report on; only
		// the budget keeps its own name, so an edit can still pass it by.
		if s := r.run(job, job.Measure); s.state != StateOK {
			if s.state != StateBudget {
				s.state = StateFailed
			}
			return s.state, s.output
		}
	}
	for _, p := range job.Reads {
		if _, err := os.Stat(p); err != nil {
			return StateFailed, p + " is missing: the measuring run did not write it"
		}
	}
	steps := make([]step, len(job.Argvs))
	if job.Threaded {
		var wg sync.WaitGroup
		for i, argv := range job.Argvs {
			wg.Go(func() { steps[i] = r.run(job, argv) })
		}
		wg.Wait()
	} else {
		for i, argv := range job.Argvs {
			steps[i] = r.run(job, argv)
		}
	}
	return mergeSteps(job.Argvs, steps, r.opt.Scope)
}

// run starts one command under the cap, with the smaller of its own timeout
// and what is left of the budget. The rest is read once the slot is held, so
// the wait for it is spent from the budget too.
func (r *runner) run(job Job, argv []string) step {
	r.sem <- struct{}{}
	defer func() { <-r.sem }()
	timeout, capped := r.opt.Timeout, false
	if !r.deadline.IsZero() {
		rest := r.deadline.Sub(r.opt.Now())
		if rest <= 0 {
			return step{StateBudget, "not started: the budget is spent"}
		}
		if timeout <= 0 || rest < timeout {
			timeout, capped = rest, true
		}
	}
	res := r.opt.Start(child.Spec{Argv: argv, Dir: job.Dir, Env: job.Env, Timeout: timeout})
	out := res.Stdout + res.Stderr
	switch {
	case res.Err != nil:
		return step{StateFailed, res.Err.Error()}
	case res.TimedOut && capped:
		return step{StateBudget, out}
	case res.TimedOut:
		return step{StateTimedOut, out}
	case res.OutputAbandoned:
		return step{StateFailed, withLine(out, abandoned)}
	case res.Code != 0:
		return step{StateFailed, out}
	}
	return step{StateOK, out}
}

// withLine puts line under out, on a line of its own.
func withLine(out, line string) string {
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out + line + "\n"
}

// mergeSteps makes one verdict and one report of a lane's commands. A finding
// outranks the budget: a lane that failed somewhere is red even if another
// command ran out of time. One command reports as it came; several each get
// a heading, in the configured order, marked when it is red.
func mergeSteps(argvs [][]string, steps []step, scope Scope) (State, string) {
	state := StateOK
	for _, s := range steps {
		if s.state == StateBudget && state == StateOK {
			state = StateBudget
		} else if s.state != StateOK && s.state != StateBudget && (state == StateOK || state == StateBudget) {
			state = s.state
		}
	}
	if len(steps) == 1 {
		return state, steps[0].output
	}
	blocks := make([]string, len(steps))
	for i, s := range steps {
		head := "$ " + strings.Join(argvs[i], " ")
		if Red(s.state, scope) {
			head += " (failed)"
		}
		blocks[i] = head + "\n" + strings.TrimRightFunc(s.output, unicode.IsSpace)
	}
	return state, strings.Join(blocks, "\n\n") + "\n"
}
