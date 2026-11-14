package web

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// answering is a service on a loopback port that says it is the interface of
// this PID.
func answering(t *testing.T, pid int) int {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"pid":%d}`, pid)
	}))
	t.Cleanup(srv.Close)
	return portOf(t, srv.URL)
}

func portOf(t *testing.T, raw string) int {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	return port
}

// silentPort is a port nothing listens on any more.
func silentPort(t *testing.T) int {
	t.Helper()
	listener, err := loopbackAny(0)
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

// killed replaces the seam of the irreversible step with one that only
// remembers whom it was asked to end.
func killed(t *testing.T, result error) *[]int {
	t.Helper()
	var pids []int
	saved := killProcess
	killProcess = func(pid int) error {
		pids = append(pids, pid)
		return result
	}
	t.Cleanup(func() { killProcess = saved })
	executable(t, `C:\bin\loomux.exe`, nil)
	return &pids
}

// executable replaces the seam for what a PID runs with a fixed answer.
func executable(t *testing.T, image string, err error) {
	t.Helper()
	saved := imageOf
	imageOf = func(int) (string, error) { return image, err }
	t.Cleanup(func() { imageOf = saved })
}

func put(t *testing.T, dir string, s State) {
	t.Helper()
	if err := writeState(dir, s); err != nil {
		t.Fatal(err)
	}
}

func exists(dir string) bool {
	_, err := os.Stat(StatePath(dir))
	return err == nil
}

func TestPathsHangOffTheStateDirectory(t *testing.T) {
	if got := StatePath("s"); got != filepath.Join("s", "ui.json") {
		t.Errorf("StatePath = %q", got)
	}
	if got := LogPath("s"); got != filepath.Join("s", "logs", "ui.log") {
		t.Errorf("LogPath = %q", got)
	}
}

func TestStopEndsTheInterfaceThatNamesItsPID(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, State{PID: 4242, Port: answering(t, 4242)})
	pids := killed(t, nil)
	stopped, err := Stop(dir)
	if err != nil || !stopped {
		t.Fatalf("Stop = %v, %v, want a stop", stopped, err)
	}
	if len(*pids) != 1 || (*pids)[0] != 4242 {
		t.Errorf("ended %v, want 4242", *pids)
	}
	if exists(dir) {
		t.Error("ui.json outlived the stop")
	}
}

func TestStopWithoutAStateFileStopsNothing(t *testing.T) {
	pids := killed(t, nil)
	stopped, err := Stop(t.TempDir())
	if err != nil || stopped || len(*pids) != 0 {
		t.Fatalf("Stop = %v, %v, ended %v, want nothing", stopped, err, *pids)
	}
}

func TestStopRemovesAStateWhosePortIsSilentWithoutKilling(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, State{PID: 4242, Port: silentPort(t)})
	pids := killed(t, nil)
	stopped, err := Stop(dir)
	if err != nil || stopped || len(*pids) != 0 || exists(dir) {
		t.Fatalf("Stop = %v, %v, ended %v, file there %v: want a removed hint and no kill", stopped, err, *pids, exists(dir))
	}
}

// What listens on the port may be anything -- another program of the user's,
// or something an attacker set up to name a PID of its choice. Only the PID
// the file names counts.
func TestStopRefusesWhenAnotherPIDAnswers(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, State{PID: 4242, Port: answering(t, 4343)})
	pids := killed(t, nil)
	stopped, err := Stop(dir)
	if err == nil || stopped || len(*pids) != 0 || !strings.Contains(err.Error(), "4343") || !strings.Contains(err.Error(), "4242") {
		t.Fatalf("Stop = %v, %v, ended %v: want an error naming both PIDs and no kill", stopped, err, *pids)
	}
	if !exists(dir) {
		t.Error("the state was removed on a mismatch; it is left for a human to look at")
	}
}

// The right PID on the right port is still no license to end a process that
// is not loomux: the file may be old, or planted, and the PID reused.
func TestStopRefusesAProcessThatIsNotLoomux(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, State{PID: 4242, Port: answering(t, 4242)})
	pids := killed(t, nil)
	executable(t, `C:\Windows\System32\notepad.exe`, nil)
	stopped, err := Stop(dir)
	if err == nil || stopped || len(*pids) != 0 || !strings.Contains(err.Error(), "notepad.exe") {
		t.Fatalf("Stop = %v, %v, ended %v: want an error naming the executable and no kill", stopped, err, *pids)
	}
	if !exists(dir) {
		t.Error("the state was removed although nothing was ended")
	}
}

func TestStopRefusesAProcessWhoseExecutableItCannotRead(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, State{PID: 4242, Port: answering(t, 4242)})
	pids := killed(t, nil)
	executable(t, "", errors.New("access is denied"))
	stopped, err := Stop(dir)
	if err == nil || stopped || len(*pids) != 0 || !strings.Contains(err.Error(), "access is denied") {
		t.Fatalf("Stop = %v, %v, ended %v: want the failure and no kill", stopped, err, *pids)
	}
}

func TestOnlyALoomuxFileNameIsALoomuxImage(t *testing.T) {
	for image, want := range map[string]bool{
		`C:\Users\x\AppData\Local\loomux\dev\w\loomux.exe`: true,
		`/usr/local/bin/loomux`:                            true,
		`/tmp/loomux (deleted)`:                            true,
		`C:\bin\LOOMUX.EXE`:                                true,
		`loomux`:                                           true,
		`C:\repo\bin\loomux.old.exe`:                       true,
		`C:\repo\bin\Loomux.Old.12.exe`:                    true,
		`C:\repo\bin\loomux.old..exe`:                      false,
		`C:\repo\bin\loomux.old.x.exe`:                     false,
		`C:\repo\bin\loomux.older.exe`:                     false,
		`/usr/bin/loomux-helper`:                           false,
		`C:\bin\loomux\notepad.exe`:                        false,
		`C:\bin\loomux.exe.bak`:                            false,
		``:                                                 false,
	} {
		if got := isLoomuxImage(image); got != want {
			t.Errorf("isLoomuxImage(%q) = %v, want %v", image, got, want)
		}
	}
}

func TestStopKillsNothingWhenAStrangerAnswersOnThePort(t *testing.T) {
	stranger := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "<html>some other program</html>")
	}))
	defer stranger.Close()
	dir := t.TempDir()
	put(t, dir, State{PID: 4242, Port: portOf(t, stranger.URL)})
	pids := killed(t, nil)
	if stopped, err := Stop(dir); err != nil || stopped || len(*pids) != 0 {
		t.Fatalf("Stop = %v, %v, ended %v, want no kill", stopped, err, *pids)
	}
}

func TestStopNeverEndsAPIDThatIsNoProcess(t *testing.T) {
	for _, pid := range []int{0, -1} {
		dir := t.TempDir()
		put(t, dir, State{PID: pid, Port: answering(t, pid)})
		pids := killed(t, nil)
		if stopped, err := Stop(dir); err != nil || stopped || len(*pids) != 0 {
			t.Errorf("pid %d: Stop = %v, %v, ended %v, want no kill", pid, stopped, err, *pids)
		}
	}
}

func TestStopRemovesAStateFileThatDoesNotParse(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(StatePath(dir), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	pids := killed(t, nil)
	if stopped, err := Stop(dir); err != nil || stopped || len(*pids) != 0 || exists(dir) {
		t.Fatalf("Stop = %v, %v, ended %v, file there %v", stopped, err, *pids, exists(dir))
	}
}

func TestStopReportsAProcessThatWillNotEnd(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, State{PID: 4242, Port: answering(t, 4242)})
	killed(t, errors.New("access is denied"))
	stopped, err := Stop(dir)
	if err == nil || stopped || !strings.Contains(err.Error(), "access is denied") {
		t.Fatalf("Stop = %v, %v, want the failure", stopped, err)
	}
	if !exists(dir) {
		t.Error("the state was removed although the process still runs")
	}
}

func TestStopReportsAStateItCannotRemove(t *testing.T) {
	dir := t.TempDir()
	// A folder with something in it is neither a state nor removable.
	if err := os.MkdirAll(filepath.Join(StatePath(dir), "keep"), 0o700); err != nil {
		t.Fatal(err)
	}
	killed(t, nil)
	if _, err := Stop(dir); err == nil || !strings.Contains(err.Error(), "remove") {
		t.Fatalf("Stop: %v, want the failed removal", err)
	}
}

func TestStopReportsAStateItCannotRemoveAfterTheKill(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, State{PID: 4242, Port: answering(t, 4242)})
	executable(t, `C:\bin\loomux.exe`, nil)
	saved := killProcess
	killProcess = func(int) error {
		// The file is gone and a folder stands in its place.
		os.Remove(StatePath(dir))
		return os.MkdirAll(filepath.Join(StatePath(dir), "keep"), 0o700)
	}
	t.Cleanup(func() { killProcess = saved })
	stopped, err := Stop(dir)
	if err == nil || !stopped {
		t.Fatalf("Stop = %v, %v, want a stop that could not clean up", stopped, err)
	}
}

func TestHealthRefusesWhatIsNotTheInterface(t *testing.T) {
	notOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }))
	defer notOK.Close()
	garbled := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "pid=1") }))
	defer garbled.Close()
	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/elsewhere", http.StatusFound)
	}))
	defer redirecting.Close()
	for name, port := range map[string]int{
		"silent": silentPort(t), "404": portOf(t, notOK.URL), "garbled": portOf(t, garbled.URL), "redirect": portOf(t, redirecting.URL),
	} {
		if pid, err := Health(port); err == nil {
			t.Errorf("%s: Health = %d, want an error", name, pid)
		}
	}
}

func TestReadStateNamesWhatIsWrong(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadState(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a missing file: %v", err)
	}
	if err := os.WriteFile(StatePath(dir), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadState(dir); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Errorf("a garbled file: %v", err)
	}
}

func TestWriteStateFailsWhereItCannotWrite(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeState(file, State{}); err == nil || !strings.Contains(err.Error(), "create") {
		t.Errorf("a state directory that is a file: %v", err)
	}
}

func TestWriteStateLeavesNoTemporaryFileBehind(t *testing.T) {
	dir := t.TempDir()
	// A folder where the file belongs: the replacement fails.
	if err := os.MkdirAll(filepath.Join(StatePath(dir), "keep"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeState(dir, State{}); err == nil {
		t.Fatal("writeState replaced a folder")
	}
	left, _ := filepath.Glob(filepath.Join(dir, "ui.json.*"))
	if len(left) != 0 {
		t.Errorf("the temporary file survived the failed replacement: %v", left)
	}
}

// Two writers of one process meet on the file, not on a temporary one.
func TestWriteStateByTwoWritersAtOnceLeavesOneWholeFile(t *testing.T) {
	dir := t.TempDir()
	var wg sync.WaitGroup
	for i := 1; i <= 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = writeState(dir, State{PID: i, Port: 7000 + i})
		}()
	}
	wg.Wait()
	state, err := ReadState(dir)
	if err != nil || state.PID == 0 || state.Port != 7000+state.PID {
		t.Fatalf("ui.json = %+v, %v; want one writer's whole state", state, err)
	}
	left, _ := filepath.Glob(filepath.Join(dir, "ui.json.*"))
	if len(left) != 0 {
		t.Errorf("temporary files left: %v", left)
	}
}

func TestWaitReadyReturnsOnceTheInterfaceAnswers(t *testing.T) {
	dir := t.TempDir()
	port := answering(t, 4242)
	go func() {
		time.Sleep(120 * time.Millisecond)
		// Written late, and first for a PID the port does not name.
		_ = writeState(dir, State{PID: 1, Port: port})
		time.Sleep(120 * time.Millisecond)
		_ = writeState(dir, State{PID: 4242, Port: port})
	}()
	state, err := WaitReady(dir, 4242, 5*time.Second)
	if err != nil || state.PID != 4242 || state.Port != port {
		t.Fatalf("WaitReady = %+v, %v, want the answering interface", state, err)
	}
}

// An interface that answers but is not the child the starter spawned is not
// the one to hand the login code to.
func TestWaitReadyIgnoresAnInterfaceThatIsNotTheChild(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, State{PID: 4242, Port: answering(t, 4242)})
	if _, err := WaitReady(dir, 7, 80*time.Millisecond); err == nil {
		t.Fatal("WaitReady accepted an interface of another process")
	}
}

func TestWaitReadyGivesUpAndPointsToTheLog(t *testing.T) {
	dir := t.TempDir()
	_, err := WaitReady(dir, 1, 80*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), LogPath(dir)) {
		t.Fatalf("WaitReady: %v, want the log named", err)
	}
}
