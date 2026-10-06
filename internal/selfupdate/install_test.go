package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/lock"
)

// fresh is a state directory with nothing installed, and the options of an
// init that runs from anywhere. The releases of these tests are of the old
// count, so the pass is a beta one: a development build is on no channel.
func fresh(t *testing.T, f *fakeGH) Options {
	t.Helper()
	return Options{
		StateDir: filepath.Join(t.TempDir(), "state"), Executable: `C:\src\loomux\bin\loomux.exe`,
		Version: DevVersion, Channel: "beta", Mode: ModeBeta, GOOS: "windows", GOARCH: "amd64",
		Run: f.run, Now: func() time.Time { return stamp },
	}
}

func TestInstallPutsTheNewestReleaseWhereNothingWas(t *testing.T) {
	o := fresh(t, release("2.12.1"))
	res := Install(context.Background(), o)
	if res.Outcome != Updated || res.Version != "2.12.1" || res.Err != nil {
		t.Fatalf("Install = %+v", res)
	}
	if got := canonicalBody(t, o); got != "binary 2.12.1" {
		t.Fatalf("canonical binary = %q", got)
	}
	if _, err := os.Stat(StatusPath(o.StateDir)); !os.IsNotExist(err) {
		t.Fatalf("Install must not write update.json: %v", err)
	}
}

// placed puts body at the canonical location of o, as an earlier install would.
func placed(t *testing.T, o Options, body string) {
	t.Helper()
	exe := Canonical(o.StateDir)
	if err := os.MkdirAll(filepath.Dir(exe), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestInstallLeavesANewerOrEqualBinaryAlone(t *testing.T) {
	for _, have := range []string{"2.12.1", "2.13.0"} {
		t.Run(have, func(t *testing.T) {
			f := release("2.12.1")
			f.installed = "loomux " + have + " (beta)\n"
			o := fresh(t, f)
			placed(t, o, "kept")
			res := Install(context.Background(), o)
			if res.Outcome != Current || res.Version != have {
				t.Fatalf("Install = %+v", res)
			}
			if got := canonicalBody(t, o); got != "kept" {
				t.Fatalf("binary replaced: %q", got)
			}
		})
	}
}

func TestInstallReplacesAnOlderBinary(t *testing.T) {
	f := release("2.12.1")
	f.installed = "loomux 2.11.0 (beta)\n"
	o := fresh(t, f)
	placed(t, o, "old")
	if res := Install(context.Background(), o); res.Outcome != Updated || res.Version != "2.12.1" {
		t.Fatalf("Install = %+v", res)
	}
	if got := canonicalBody(t, o); got != "binary 2.12.1" {
		t.Fatalf("canonical binary = %q", got)
	}
}

func TestInstallSkipsOffWindows(t *testing.T) {
	o := fresh(t, release("2.12.1"))
	o.GOOS = "linux"
	if res := Install(context.Background(), o); res.Outcome != Skipped || res.Err == nil {
		t.Fatalf("Install = %+v", res)
	}
}

func TestInstallFailsWithoutGh(t *testing.T) {
	f := release("2.12.1")
	f.fail["list"] = errors.New("gh not found; install GitHub CLI and run gh auth login")
	o := fresh(t, f)
	res := Install(context.Background(), o)
	if res.Outcome != Failed || res.Err == nil {
		t.Fatalf("Install = %+v", res)
	}
	if _, err := os.Stat(Canonical(o.StateDir)); !os.IsNotExist(err) {
		t.Fatalf("a failed install left a binary: %v", err)
	}
}

func TestInstallFailsWhenTheDownloadFails(t *testing.T) {
	f := release("2.12.1")
	f.fail["download"] = errors.New("HTTP 404")
	o := fresh(t, f)
	if res := Install(context.Background(), o); res.Outcome != Failed || res.Err == nil {
		t.Fatalf("Install = %+v", res)
	}
	if _, err := os.Stat(Canonical(o.StateDir)); !os.IsNotExist(err) {
		t.Fatalf("a failed download left a binary: %v", err)
	}
}

func TestInstallFailsWhenTheStateDirectoryCannotBeMade(t *testing.T) {
	o := fresh(t, release("2.12.1"))
	if err := os.WriteFile(o.StateDir, nil, 0o600); err != nil { // a file where the directory goes
		t.Fatal(err)
	}
	if res := Install(context.Background(), o); res.Outcome != Failed {
		t.Fatalf("Install = %+v", res)
	}
}

func TestInstallStepsAsideForAPassInProgress(t *testing.T) {
	f := release("2.12.1")
	o := fresh(t, f)
	_ = os.MkdirAll(o.StateDir, 0o700)
	held, ok, err := lock.TryAcquire(filepath.Join(o.StateDir, "update.lock"))
	if err != nil || !ok {
		t.Fatal(err)
	}
	defer held.Release()
	if res := Install(context.Background(), o); res.Outcome != Busy {
		t.Fatalf("Install = %+v", res)
	}
	if len(f.calls) != 0 {
		t.Fatalf("a busy install called out: %v", f.calls)
	}
}

func TestInstallFailsWhenTheLockCannotBeOpened(t *testing.T) {
	o := fresh(t, release("2.12.1"))
	if err := os.MkdirAll(filepath.Join(o.StateDir, "update.lock", "x"), 0o700); err != nil {
		t.Fatal(err)
	}
	if res := Install(context.Background(), o); res.Outcome != Failed || res.Err == nil {
		t.Fatalf("Install = %+v", res)
	}
}

func TestInstallFailsWhenEverySlotIsHeld(t *testing.T) {
	f := release("2.12.1")
	f.installed = "loomux 2.11.0 (beta)\n"
	o := fresh(t, f)
	placed(t, o, "old")
	bin := filepath.Dir(Canonical(o.StateDir))
	for i := range 16 {
		name := "loomux.old.exe"
		if i > 0 {
			name = fmt.Sprintf("loomux.old.%d.exe", i)
		}
		if err := os.MkdirAll(filepath.Join(bin, name, "held"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	res := Install(context.Background(), o)
	if res.Outcome != Failed || res.Version != "2.12.1" || !strings.Contains(res.Err.Error(), "slots") {
		t.Fatalf("Install = %+v", res)
	}
	if got := canonicalBody(t, o); got != "old" {
		t.Fatalf("canonical binary = %q", got)
	}
}

// The path init takes: a binary of the old count, no mode, no marker, nothing
// installed yet. Its own count puts it on the beta channel.
func TestInstallOfTheOldCountTakesTheHighestPreRelease(t *testing.T) {
	f := releasesOf("7.3.0", `[{"tagName":"v7.2.0","isPrerelease":true},{"tagName":"v7.3.0","isPrerelease":true}]`)
	o := fresh(t, f)
	o.Version, o.Channel, o.Mode = "7.2.0", "beta", ModeChannel
	res := Install(context.Background(), o)
	if res.Outcome != Updated || res.Version != "7.3.0" || res.Err != nil {
		t.Fatalf("Install = %+v", res)
	}
	if got := canonicalBody(t, o); got != "binary 7.3.0" {
		t.Fatalf("canonical binary = %q", got)
	}
}

// An init that runs from a beta brings the beta channel along: the entries it
// writes must not call an older binary than the one that wrote them.
func TestInstallFromABetaFollowsTheBetaChannel(t *testing.T) {
	f := releasesOf("1.1.0-beta.2", `[{"tagName":"v1.0.0","isPrerelease":false},{"tagName":"v1.1.0-beta.2","isPrerelease":true}]`)
	o := fresh(t, f)
	o.Version, o.Channel, o.Mode = "1.1.0-beta.1", "", ModeChannel
	res := Install(context.Background(), o)
	if res.Outcome != Updated || res.Version != "1.1.0-beta.2" || res.Err != nil {
		t.Fatalf("Install = %+v", res)
	}
	if beta, err := ReadChannel(o.StateDir); !beta || err != nil {
		t.Fatalf("ReadChannel = %v, %v; want true", beta, err)
	}
}

// A pin is its own request and keeps its kind; the beta of the running binary
// does not turn it into a beta pass.
func TestInstallFromABetaKeepsAPin(t *testing.T) {
	f := releasesOf("1.0.0", `[{"tagName":"v1.0.0","isPrerelease":false}]`)
	f.version, f.view = "loomux 1.0.0\n", `{"tagName":"v1.0.0","isPrerelease":false}`
	o := fresh(t, f)
	o.Version, o.Channel, o.Mode, o.Pin = "1.1.0-beta.1", "", ModeChannel, "1.0.0"
	res := Install(context.Background(), o)
	if res.Outcome != Updated || res.Version != "1.0.0" || res.Err != nil {
		t.Fatalf("Install = %+v", res)
	}
	if beta, _ := ReadChannel(o.StateDir); beta {
		t.Fatal("a pin of a stable release must not set the beta marker")
	}
}

// An explicit --stable outranks the beta of the running binary.
func TestInstallFromABetaKeepsAnExplicitStableMode(t *testing.T) {
	f := releasesOf("1.0.0", `[{"tagName":"v1.0.0","isPrerelease":false},{"tagName":"v1.1.0-beta.2","isPrerelease":true}]`)
	f.version = "loomux 1.0.0\n"
	o := fresh(t, f)
	o.Version, o.Channel, o.Mode = "1.1.0-beta.1", "", ModeStable
	res := Install(context.Background(), o)
	if res.Outcome != Updated || res.Version != "1.0.0" || res.Err != nil {
		t.Fatalf("Install = %+v", res)
	}
	if beta, _ := ReadChannel(o.StateDir); beta {
		t.Fatal("--stable must clear the beta marker")
	}
}

func TestInstallFromAStableReleaseSetsNoMarker(t *testing.T) {
	f := releasesOf("1.0.0", `[{"tagName":"v1.0.0","isPrerelease":false},{"tagName":"v1.1.0-beta.2","isPrerelease":true}]`)
	f.version = "loomux 1.0.0\n"
	o := fresh(t, f)
	o.Version, o.Channel, o.Mode = "1.0.0", "", ModeChannel
	res := Install(context.Background(), o)
	if res.Outcome != Updated || res.Version != "1.0.0" || res.Err != nil {
		t.Fatalf("Install = %+v", res)
	}
	if beta, _ := ReadChannel(o.StateDir); beta {
		t.Fatal("an init from a stable release must not set the beta marker")
	}
}
