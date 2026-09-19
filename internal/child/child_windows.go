//go:build windows

package child

import (
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Seams for the error arms, and afterKill for the test that counts the job.
var (
	createJob = windows.CreateJobObject
	setJob    = windows.SetInformationJobObject
	openProc  = windows.OpenProcess
	assignJob = windows.AssignProcessToJobObject
	resume    = ntResume
	afterKill = func(job windows.Handle) {}
)

type tree struct{ job windows.Handle }

// NtResumeProcess because exec closes the thread handle ResumeThread would
// need; measured on 2026-09-18 with cmd and two ping grandchildren.
func ntResume(h windows.Handle) error {
	status, _, _ := windows.NewLazySystemDLL("ntdll.dll").NewProc("NtResumeProcess").Call(uintptr(h))
	if status != 0 {
		return windows.NTStatus(status)
	}
	return nil
}

// start creates the child suspended, puts it into a job that dies with its
// handle, and only then lets it run: no grandchild is born outside the job.
func start(cmd *exec.Cmd) (*tree, error) {
	job, err := createJob(nil, nil)
	if err != nil {
		return nil, err
	}
	t := &tree{job: job}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := setJob(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		t.release()
		return nil, err
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_SUSPENDED}
	if err := cmd.Start(); err != nil {
		t.release()
		return nil, err
	}
	if err := t.adopt(cmd.Process.Pid); err != nil {
		// A child outside the job dies only by its own handle; the wait
		// reaps it, which Run will not do for a failed start.
		cmd.Process.Kill()
		cmd.Process.Wait()
		t.release()
		return nil, err
	}
	return t, nil
}

func (t *tree) adopt(pid int) error {
	h, err := openProc(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_SUSPEND_RESUME, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	if err := assignJob(t.job, h); err != nil {
		return err
	}
	return resume(h)
}

// killTree is a seam for the test in which the child survives its kill.
var killTree = func(job windows.Handle) { windows.TerminateJobObject(job, 1) }

func (t *tree) kill() {
	killTree(t.job)
	afterKill(t.job)
}

func (t *tree) release() { windows.CloseHandle(t.job) }
