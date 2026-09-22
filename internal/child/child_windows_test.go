//go:build windows

package child

import (
	"errors"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type accounting struct {
	TotalUserTime, TotalKernelTime, ThisPeriodTotalUserTime, ThisPeriodTotalKernelTime int64
	TotalPageFaultCount, TotalProcesses, ActiveProcesses, TotalTerminatedProcesses     uint32
}

// cmd starts ping in the background and a second one in front: without the
// job, the background ping would outlive the kill of cmd. The deadline
// leaves cmd time to start both under load: a kill before the first ping runs
// has no grandchild to prove anything about.
func TestRunKillsTheGrandchildAtTheDeadline(t *testing.T) {
	var active, total uint32 = 99, 0
	old := afterKill
	afterKill = func(job windows.Handle) {
		var a accounting
		windows.QueryInformationJobObject(job, 1, uintptr(unsafe.Pointer(&a)), uint32(unsafe.Sizeof(a)), nil)
		active, total = a.ActiveProcesses, a.TotalProcesses
	}
	t.Cleanup(func() { afterKill = old })
	r := Run(Spec{
		Argv:    []string{"cmd", "/c", "start /b ping -n 60 127.0.0.1 >nul & ping -n 60 127.0.0.1 >nul"},
		Timeout: 5 * time.Second,
	})
	if !r.TimedOut || total < 3 || active != 0 {
		t.Fatalf("%+v total %d active %d", r, total, active)
	}
}

// countKills counts the kills of a job and remembers how many of its
// processes were still alive after the last one.
func countKills(t *testing.T) (kills *int, active *uint32) {
	kills, active = new(int), new(uint32)
	*active = 99
	old := afterKill
	afterKill = func(job windows.Handle) {
		var a accounting
		windows.QueryInformationJobObject(job, 1, uintptr(unsafe.Pointer(&a)), uint32(unsafe.Sizeof(a)), nil)
		*kills, *active = *kills+1, a.ActiveProcesses
	}
	t.Cleanup(func() { afterKill = old })
	return kills, active
}

// The grandchild holding the pipe dies before Run reads what it left, not
// only when the job handle closes.
func TestRunAbandonsOutputAGrandchildHolds(t *testing.T) {
	kills, active := countKills(t)
	start := time.Now()
	r := Run(Spec{Argv: []string{"cmd", "/c", "start /b ping -n 60 127.0.0.1"}})
	if !r.OutputAbandoned || time.Since(start) > DrainGrace+5*time.Second || *kills != 1 || *active != 0 {
		t.Fatalf("%+v after %v, %d kills, %d active", r, time.Since(start), *kills, *active)
	}
}

func TestRunKillsNothingAfterACleanExit(t *testing.T) {
	kills, _ := countKills(t)
	if r := Run(helper("echo")); r.Code != 3 || *kills != 0 {
		t.Fatalf("%+v after %d kills", r, *kills)
	}
}

// A child that survives its kill must not make Run wait twice on one
// deadline: it stays suspended and nothing terminates the job.
func TestRunReturnsWhenTheChildSurvivesTheKill(t *testing.T) {
	oldResume, oldKill := resume, killTree
	var held windows.Handle
	resume = func(windows.Handle) error { return nil }
	killTree = func(job windows.Handle) {
		if held == 0 {
			self := windows.CurrentProcess()
			windows.DuplicateHandle(self, job, self, &held, 0, false, windows.DUPLICATE_SAME_ACCESS)
		}
	}
	t.Cleanup(func() {
		resume, killTree = oldResume, oldKill
		windows.TerminateJobObject(held, 1)
		windows.CloseHandle(held)
	})
	s := helper("sleep")
	s.Timeout = 300 * time.Millisecond
	done := make(chan Result, 1)
	start := time.Now()
	go func() { done <- Run(s) }()
	select {
	case r := <-done:
		if !r.TimedOut || !r.OutputAbandoned || time.Since(start) < DrainGrace {
			t.Fatalf("%+v after %v", r, time.Since(start))
		}
	case <-time.After(3 * DrainGrace):
		t.Fatal("Run hung after a kill that did not kill")
	}
}

// failStart swaps one system call for a failing one and expects Run to report
// that failure, not one a later call ran into after it.
func failStart(t *testing.T, swap func() func()) {
	t.Helper()
	t.Cleanup(swap())
	if r := Run(helper("sleep")); !errors.Is(r.Err, errDenied) || r.Code != -1 {
		t.Fatalf("%+v", r)
	}
}

var errDenied = errors.New("denied")

func TestRunReportsAFailedJob(t *testing.T) {
	failStart(t, func() func() {
		old := createJob
		createJob = func(*windows.SecurityAttributes, *uint16) (windows.Handle, error) { return 0, errDenied }
		return func() { createJob = old }
	})
}

func TestRunReportsAFailedJobLimit(t *testing.T) {
	failStart(t, func() func() {
		old := setJob
		setJob = func(windows.Handle, uint32, uintptr, uint32) (int, error) { return 0, errDenied }
		return func() { setJob = old }
	})
}

func TestRunReportsAFailedOpen(t *testing.T) {
	failStart(t, func() func() {
		old := openProc
		openProc = func(uint32, bool, uint32) (windows.Handle, error) { return 0, errDenied }
		return func() { openProc = old }
	})
}

func TestRunReportsAFailedAssign(t *testing.T) {
	failStart(t, func() func() {
		old := assignJob
		assignJob = func(windows.Handle, windows.Handle) error { return errDenied }
		return func() { assignJob = old }
	})
}

func TestRunReportsAFailedResume(t *testing.T) {
	failStart(t, func() func() {
		old := resume
		resume = func(windows.Handle) error { return errDenied }
		return func() { resume = old }
	})
}

func TestNtResumeReportsABadHandle(t *testing.T) {
	if err := ntResume(0); err == nil {
		t.Fatal("resumed handle 0")
	}
}
