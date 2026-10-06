package verify

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/lock"
)

// lockTick is how often a lane asks for its lock again, as lock.WaitFree.
const lockTick = 250 * time.Millisecond

// LockPath is the lock file of a stack's area in the checkout at root, named
// as CoverPaths names its files. A linked worktree has its own state
// directory and with it its own lock.
func LockPath(root, stack, area string) string {
	if area == "." {
		area = "root"
	}
	return filepath.Join(root, ".loomux", "state", "locks", stack+"-"+strings.ReplaceAll(area, "/", "_")+".lock")
}

// WaitingTo is RunOptions.Waiting for a run that reports on w.
func WaitingTo(w io.Writer) func(Job, string) {
	return func(j Job, who string) { fmt.Fprintf(w, "%s: waiting for the lock (%s)\n", j.Name, who) }
}

// takeLock takes job's lock and answers how to let it go, or the state the
// lane ends in. It waits outside the process slots, as a lane waits for its
// predecessor, and no longer than the budget or, without one, the timeout.
// The lock file stays empty -- a Windows lock denies reads of its range --,
// so who holds it is written beside it; the lock is the truth, the holder
// file a courtesy for whoever waits.
func (r *runner) takeLock(job Job) (release func(), state State, msg string) {
	if err := os.MkdirAll(filepath.Dir(job.Lock), 0o755); err != nil {
		return nil, StateFailed, err.Error()
	}
	limit, over := r.deadline, StateBudget
	if limit.IsZero() && r.opt.Timeout > 0 {
		limit, over = r.opt.Now().Add(r.opt.Timeout), StateTimedOut
	}
	told := false
	for {
		h, held, err := lock.TryAcquire(job.Lock)
		if err != nil {
			return nil, StateFailed, err.Error()
		}
		if held {
			holder := fmt.Sprintf("pid=%d\ncaller=%s\nsince=%s\n", os.Getpid(), r.opt.Caller, r.opt.Now().UTC().Format(time.RFC3339))
			_ = lock.ReplaceText(job.Lock+".who", holder) // a courtesy; the lock holds without it
			return func() {
				os.Remove(job.Lock + ".who")
				h.Release()
			}, "", ""
		}
		who := holderOf(job.Lock, r.opt.Now())
		if !told && r.opt.Waiting != nil {
			r.opt.Waiting(job, who)
		}
		told = true
		if !limit.IsZero() && !r.opt.Now().Before(limit) {
			return nil, over, "not started: waited for the lock, " + who
		}
		r.opt.Sleep(lockTick)
	}
}

// holderOf reads the holder file beside a lock someone holds.
func holderOf(path string, now time.Time) string {
	data, err := os.ReadFile(path + ".who")
	if err != nil {
		return "held by another run"
	}
	fields := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), "="); ok {
			fields[k] = v
		}
	}
	// Lanes of one stack and area in one run share the lock, so the holder can
	// be this process.
	who := "held by loomux pid " + fields["pid"]
	if fields["pid"] == strconv.Itoa(os.Getpid()) {
		who = "held by another lane of this run"
	}
	if fields["caller"] != "" {
		who += ", " + fields["caller"]
	}
	if since, err := time.Parse(time.RFC3339, fields["since"]); err == nil {
		who += fmt.Sprintf(", since %ds", int(now.Sub(since).Seconds()))
	}
	return who
}
