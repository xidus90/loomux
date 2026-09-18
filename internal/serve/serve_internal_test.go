package serve

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// refuseBindAfter makes the binder refuse from the given call on and bind for
// real before that. It hands back the addresses it did bind, so a test can ask
// whether they were given up again.
func refuseBindAfter(t *testing.T, refuseFrom int) *[]string {
	t.Helper()
	bound := &[]string{}
	calls := 0
	saved := listenTCP
	listenTCP = func(network, address string) (net.Listener, error) {
		calls++
		if calls >= refuseFrom {
			return nil, errors.New("no port for you")
		}
		listener, err := saved(network, address)
		if err != nil {
			return nil, err
		}
		*bound = append(*bound, listener.Addr().String())
		return listener, nil
	}
	t.Cleanup(func() { listenTCP = saved })
	return bound
}

// runUntilItReturns runs serve and fails the test rather than hanging the suite.
// Every caller here expects an error before the service is up; a Run that keeps
// running would otherwise block this test for as long as the suite lives.
func runUntilItReturns(t *testing.T, dir string) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- Run(context.Background(), Options{StateDir: dir, RegistryDir: dir, LegacyDir: dir}) }()
	select {
	case err := <-done:
		return err
	case <-time.After(20 * time.Second):
		t.Fatal("Run neither started nor gave up; it is stuck holding the lock")
		return nil
	}
}

// expectTheLockIsFree is the assertion that matters most here: a serve that
// gives up while holding serve.lock locks the machine out of its own service
// until somebody kills the process.
func expectTheLockIsFree(t *testing.T, dir string) {
	t.Helper()
	handle, held, err := lock.TryAcquire(LockPath(dir))
	if err != nil {
		t.Fatalf("TryAcquire: %v", err)
	}
	if !held {
		t.Fatal("serve gave up but kept serve.lock; every later serve would be refused")
	}
	handle.Release()
}

// expectNobodyListens proves the bound port went back to the operating system.
// Shutdown alone would not do it: it closes only the listeners Serve took over.
func expectNobodyListens(t *testing.T, addresses []string) {
	t.Helper()
	for _, address := range addresses {
		conn, err := net.DialTimeout("tcp", address, 2*time.Second)
		if err == nil {
			conn.Close()
			t.Errorf("%s still accepts connections", address)
		}
	}
}

func TestARefusedSecondPortGivesUpTheFirstOneAndTheLock(t *testing.T) {
	dir := t.TempDir()
	// The local listener binds, the cloud one does not. This is the arm that
	// held serve.lock and hung forever: close waited for a Serve that never
	// ran, and Release is deferred earlier, so it runs later -- or never.
	bound := refuseBindAfter(t, 2)

	err := runUntilItReturns(t, dir)
	if err == nil || !strings.Contains(err.Error(), "cloud") {
		t.Fatalf("Run returned %v, want an error naming the cloud listener", err)
	}
	expectTheLockIsFree(t, dir)
	if len(*bound) != 1 {
		t.Fatalf("the binder bound %d ports, want one", len(*bound))
	}
	expectNobodyListens(t, *bound)
}

func TestARefusedFirstPortGivesUpTheLock(t *testing.T) {
	dir := t.TempDir()
	refuseBindAfter(t, 1)

	err := runUntilItReturns(t, dir)
	if err == nil || !strings.Contains(err.Error(), "local") {
		t.Fatalf("Run returned %v, want an error naming the local listener", err)
	}
	expectTheLockIsFree(t, dir)
}

// TestTheServicesQmdPortLocksItsOwnStateDirectory is the reason qmdOptions
// exists. The port is built as the service builds it, against a daemon that is
// up but refuses every query, and two things have to show: the lock file lies
// in this service's state directory and nowhere else, and the query was tried
// twice -- one try and exactly one retry, where the command line would try
// three times.
func TestTheServicesQmdPortLocksItsOwnStateDirectory(t *testing.T) {
	global := t.TempDir()
	t.Setenv(config.StateDirEnv, global)
	dir := t.TempDir()

	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err == nil && strings.Contains(string(body), "tools/call") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "id": 1, "error": map[string]any{"message": "no"},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]any{}})
	}))
	defer daemon.Close()
	address, err := url.Parse(daemon.URL)
	if err != nil {
		t.Fatalf("parse the daemon's address: %v", err)
	}
	port, err := strconv.Atoi(address.Port())
	if err != nil {
		t.Fatalf("read the daemon's port: %v", err)
	}

	engine := search.NewQmdMcpPort(append(qmdOptions(dir), search.WithPort(port))...)
	_, err = engine.Search("q", []string{"c"}, search.ProfileFast, 1)
	if err == nil || !strings.Contains(err.Error(), "did not answer in 2 attempts") {
		t.Fatalf("got %v, want two attempts", err)
	}
	if _, err := os.Stat(QmdLockPath(dir)); err != nil {
		t.Errorf("the service did not lock its own state directory: %v", err)
	}
	if _, err := os.Stat(QmdLockPath(global)); err == nil {
		t.Error("the service locked the global state directory instead of its own")
	}
}

// TestTheDefaultAnswerIsTheServicesOwn: nothing but a caller's own answer
// displaces the one built with qmdOptions.
func TestTheDefaultAnswerIsTheServicesOwn(t *testing.T) {
	if (Options{StateDir: t.TempDir()}).answerFunc() == nil {
		t.Fatal("a service without an answer of its own got none")
	}
}

// writeRegistryWithOneArea is the smallest registry a search can be run
// against: one writable area whose manifest declares nothing but its scope.
func writeRegistryWithOneArea(t *testing.T, registryDir string) {
	t.Helper()
	area := t.TempDir()
	manifest := filepath.Join(area, ".loomux", "config.toml")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o700); err != nil {
		t.Fatalf("make the manifest directory: %v", err)
	}
	if err := os.WriteFile(manifest, []byte("[area]\nscope = \"project/a\"\n"), 0o600); err != nil {
		t.Fatalf("write the manifest: %v", err)
	}
	registry := "[[area]]\nscope = \"project/a\"\npath = " + strconv.Quote(filepath.ToSlash(area)) + "\n"
	if err := os.WriteFile(filepath.Join(registryDir, "registry.toml"), []byte(registry), 0o600); err != nil {
		t.Fatalf("write the registry: %v", err)
	}
}

// TestTheDefaultAnswerTakesThisServicesQmdLock covers the call site of
// qmdOptions rather than its body. The re-review of task 10 found the mutation
// that survives everything above: answerFunc reverted to answer.Run answers
// just as well, and only a run of the whole answer says which lock it took.
//
// Neither state directory exists, and that is what makes the test safe as well
// as sharp: the lock cannot be made in either, so the answer fails at the first
// step and names the path it tried -- no port is probed, no daemon is started,
// and nothing is left behind.
func TestTheDefaultAnswerTakesThisServicesQmdLock(t *testing.T) {
	root := t.TempDir()
	global, service := filepath.Join(root, "global"), filepath.Join(root, "service")
	t.Setenv(config.StateDirEnv, global)
	registryDir, legacyDir := t.TempDir(), t.TempDir()
	writeRegistryWithOneArea(t, registryDir)

	_, _, err := (Options{StateDir: service}).answerFunc()(answer.Request{
		Command: "search",
		Query:   "q",
		Scope:   "all",
		Profile: string(search.ProfileFast),
		Count:   1,
		Channel: privacy.ChannelLocal,
	}, registryDir, legacyDir, nil)

	if err == nil {
		t.Fatal("the answer succeeded although no state directory exists")
	}
	if !strings.Contains(err.Error(), QmdLockPath(service)) {
		t.Errorf("the answer did not take %s: %v", QmdLockPath(service), err)
	}
	if strings.Contains(err.Error(), QmdLockPath(global)) {
		t.Errorf("the answer took the global lock %s instead: %v", QmdLockPath(global), err)
	}
}
