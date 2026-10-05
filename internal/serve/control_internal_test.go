package serve

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/lock"
)

func TestStopReportsALockTheServiceDidNotLetGoOf(t *testing.T) {
	dir := t.TempDir()
	// A listener that takes the stop request and does nothing else. The real
	// service cannot be made to do that, and this is the arm where a caller
	// would otherwise start the next service straight into a refusal.
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+StopPath, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	// Somebody still holds serve.lock: here the case itself, in the service's
	// place.
	handle, err := lock.Acquire(LockPath(dir))
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	t.Cleanup(func() { _ = handle.Release() })

	saved := stopTimeout
	stopTimeout = 100 * time.Millisecond
	t.Cleanup(func() { stopTimeout = saved })

	if err := WriteState(dir, &State{Local: Endpoint{URL: server.URL + MCPPath}}); err != nil {
		t.Fatalf("WriteState: %v", err)
	}
	err = Stop(dir, false)
	if err == nil {
		t.Fatal("Stop reported success although the lock never came free")
	}
	if !strings.Contains(err.Error(), LockPath(dir)) {
		t.Errorf("Stop: %v, want the lock that is still held named", err)
	}
}

func TestStopWithForceReportsALockTheKilledProcessDidNotLetGoOf(t *testing.T) {
	dir := t.TempDir()
	// Somebody other than the process the PID names holds serve.lock: here
	// the case itself, so the kill cannot free it.
	handle, err := lock.Acquire(LockPath(dir))
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	t.Cleanup(func() { _ = handle.Release() })

	saved := stopTimeout
	stopTimeout = 100 * time.Millisecond
	t.Cleanup(func() { stopTimeout = saved })

	// The test binary again, as a process that only waits to be killed.
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess")
	cmd.Env = append(os.Environ(), "LOOMUX_SERVE_TEST_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start the helper process: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	pid := cmd.Process.Pid

	// A port nothing listens on: taken from the system and given back.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	state := &State{Local: Endpoint{URL: "http://" + address + MCPPath, Token: "unused"}, PID: pid}
	if err := WriteState(dir, state); err != nil {
		t.Fatalf("WriteState: %v", err)
	}

	err = Stop(dir, true)
	if err == nil {
		t.Fatal("Stop reported success although serve.lock stayed held")
	}
	for _, want := range []string{strconv.Itoa(pid), LockPath(dir)} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Stop: %v, want %q named", err, want)
		}
	}

	// Reaped, the PID names nothing, and the kill itself fails. That error
	// comes before any wait on the lock.
	_ = cmd.Wait()
	err = Stop(dir, true)
	if err == nil || !strings.Contains(err.Error(), "kill process "+strconv.Itoa(pid)) {
		t.Errorf("Stop of a PID that is gone: %v, want the failed kill", err)
	}
}
