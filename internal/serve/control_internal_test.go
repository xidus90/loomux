package serve

import (
	"net/http"
	"net/http/httptest"
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
