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

// installed is a state directory whose canonical binary is "old" at v2.7.0
// beta, with the options of a process running it. Unless the test says
// otherwise, the file answers --version with the running version.
func installed(t *testing.T, f *fakeGH) Options {
	t.Helper()
	if f.installed == "" {
		f.installed = "loomux 2.7.0 (beta)\n"
	}
	dir := t.TempDir()
	exe := Canonical(dir)
	if err := os.MkdirAll(filepath.Dir(exe), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	return Options{
		StateDir: dir, Executable: exe, Version: "2.7.0", Channel: "beta",
		GOOS: "windows", GOARCH: "amd64", Run: f.run,
		Now: func() time.Time { return stamp },
	}
}

func canonicalBody(t *testing.T, o Options) string {
	t.Helper()
	data, err := os.ReadFile(Canonical(o.StateDir))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func recorded(t *testing.T, o Options) Status {
	t.Helper()
	st, err := ReadStatus(o.StateDir)
	if err != nil || st == nil {
		t.Fatalf("update.json: %v, %v", st, err)
	}
	return *st
}

func TestRunInstallsANewerRelease(t *testing.T) {
	o := installed(t, release("2.8.0"))
	res := Run(context.Background(), o)
	if res.Outcome != Updated || res.Version != "2.8.0" || res.Err != nil {
		t.Fatalf("Run = %+v", res)
	}
	if got := canonicalBody(t, o); got != "binary 2.8.0" {
		t.Fatalf("canonical binary = %q", got)
	}
	old, _ := os.ReadFile(filepath.Join(filepath.Dir(Canonical(o.StateDir)), "loomux.old.exe"))
	if string(old) != "old" {
		t.Fatalf("the previous binary was not kept aside: %q", old)
	}
	st := recorded(t, o)
	if st.Result != Updated || st.Version != "2.8.0" || st.Running != "2.7.0" || st.Executable != o.Executable || !st.CheckedAt.Equal(stamp) {
		t.Fatalf("update.json = %+v", st)
	}
}

// update.json says who ran the pass: session start warns about serve's
// location only when serve itself wrote it.
func TestRunRecordsItsSource(t *testing.T) {
	o := installed(t, release("2.7.0"))
	o.Source = SourceCLI
	Run(context.Background(), o)
	if st := recorded(t, o); st.Source != SourceCLI {
		t.Fatalf("update.json = %+v", st)
	}
}

// A pass by hand from a checkout says nothing about the machine-wide binary;
// were it recorded, it would hide serve's own record from session start until
// serve's next pass.
func TestRunKeepsServesRecordFromASkippedPassByHand(t *testing.T) {
	f := release("2.8.0")
	o := installed(t, f)
	served := Status{Source: SourceServe, CheckedAt: stamp, Executable: o.Executable, Running: "2.7.0", Result: Failed, Error: "gh not found"}
	if err := WriteStatus(o.StateDir, served); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(StatusPath(o.StateDir))
	if err != nil {
		t.Fatal(err)
	}
	o.Source = SourceCLI
	o.Executable = filepath.Join(o.StateDir, "elsewhere.exe")
	if res := Run(context.Background(), o); res.Outcome != Skipped || res.StatusErr != nil {
		t.Fatalf("Run = %+v", res)
	}
	after, err := os.ReadFile(StatusPath(o.StateDir))
	if err != nil || string(after) != string(before) {
		t.Fatalf("update.json = %q, %v; want serve's record unchanged:\n%s", after, err, before)
	}
}

// A pass the stop of serve cut short came to no outcome of its own; a
// "failed" from it would warn in the next session about nothing.
func TestRunRecordsNoPassItsStopCutShort(t *testing.T) {
	o := installed(t, release("2.8.0"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o.Run = func(context.Context, string, ...string) ([]byte, error) {
		cancel()
		return nil, context.Canceled
	}
	if res := Run(ctx, o); res.Outcome != Failed {
		t.Fatalf("Run = %+v", res)
	}
	if st, err := ReadStatus(o.StateDir); st != nil || err != nil {
		t.Fatalf("a cancelled pass wrote update.json: %+v, %v", st, err)
	}
	if got := canonicalBody(t, o); got != "old" {
		t.Fatalf("canonical binary = %q", got)
	}
}

func TestRunLeavesACurrentBinaryAlone(t *testing.T) {
	f := release("2.7.0")
	o := installed(t, f)
	res := Run(context.Background(), o)
	if res.Outcome != Current || res.Version != "2.7.0" {
		t.Fatalf("Run = %+v", res)
	}
	if strings.Join(f.calls, ",") != "list" {
		t.Fatalf("a current binary downloaded something: %v", f.calls)
	}
	if recorded(t, o).Result != Current {
		t.Fatal("current was not recorded")
	}
}

// serve keeps running the version it started with until a bridge replaces
// it, which may take days. Were the pass to decide by the running version
// alone, it would download the release it already installed every day and
// fill one more slot for old binaries with each swap.
func TestRunLeavesAnInstalledReleaseAlone(t *testing.T) {
	f := release("2.8.0")
	f.installed = "loomux 2.8.0 (beta)\n"
	o := installed(t, f)
	if err := os.WriteFile(Canonical(o.StateDir), []byte("binary 2.8.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := Run(context.Background(), o)
	if res.Outcome != Current || res.Version != "2.8.0" || res.Err != nil {
		t.Fatalf("Run = %+v", res)
	}
	if got := strings.Join(f.calls, ","); got != "list,installed" {
		t.Fatalf("calls = %s; want the list and the installed binary's version only", got)
	}
	if got := canonicalBody(t, o); got != "binary 2.8.0" {
		t.Fatalf("canonical binary = %q", got)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(Canonical(o.StateDir)), "loomux.old.exe")); !os.IsNotExist(err) {
		t.Fatalf("an old-binary slot was taken: %v", err)
	}
	if st := recorded(t, o); st.Result != Current || st.Version != "2.8.0" || st.Running != "2.7.0" {
		t.Fatalf("update.json = %+v", st)
	}
}

// What the installed binary says only ever holds an update back when it names
// the release; any answer the pass cannot use leaves the decision to the
// running version, and one older than that changes nothing either.
func TestRunUpdatesWhenTheInstalledVersionSaysNothingNewer(t *testing.T) {
	for _, c := range []struct {
		name string
		bend func(f *fakeGH)
	}{
		{"a failing call", func(f *fakeGH) { f.fail["installed"] = errors.New("exec format error") }},
		{"an answer that is no version", func(f *fakeGH) { f.installed = "loomux 0.0.0-dev\n" }},
		{"an answer from another program", func(f *fakeGH) { f.installed = "usage: something\n" }},
		{"an older installed binary", func(f *fakeGH) { f.installed = "loomux 2.6.0 (beta)\n" }},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := release("2.8.0")
			c.bend(f)
			o := installed(t, f)
			res := Run(context.Background(), o)
			if res.Outcome != Updated || res.Version != "2.8.0" || res.Err != nil {
				t.Fatalf("Run = %+v", res)
			}
			if got := strings.Join(f.calls, ","); got != "list,installed,download,--version" {
				t.Fatalf("calls = %s", got)
			}
			if got := canonicalBody(t, o); got != "binary 2.8.0" {
				t.Fatalf("canonical binary = %q", got)
			}
		})
	}
}

// A development build at the canonical place was put there by hand; every
// host would run it until someone noticed. The pass replaces it like any
// older release instead of refusing it for good.
func TestRunReplacesADevelopmentBuildAtTheCanonicalPlace(t *testing.T) {
	f := release("2.8.0")
	f.installed = "loomux 0.0.0-dev\n"
	o := installed(t, f)
	o.Version, o.Mode = DevVersion, ModeBeta
	res := Run(context.Background(), o)
	if res.Outcome != Updated || res.Version != "2.8.0" || res.Err != nil {
		t.Fatalf("Run = %+v", res)
	}
	if got := canonicalBody(t, o); got != "binary 2.8.0" {
		t.Fatalf("canonical binary = %q", got)
	}
}

func TestRunSkips(t *testing.T) {
	for _, c := range []struct {
		name string
		bend func(o *Options)
		want string
	}{
		{"off Windows", func(o *Options) { o.GOOS = "linux" }, "updating runs on Windows only"},
		{"a development build elsewhere", func(o *Options) {
			o.Version = DevVersion
			o.Executable = filepath.Join(o.StateDir, "elsewhere.exe")
		}, "development build 0.0.0-dev is never replaced"},
		{"a binary elsewhere", func(o *Options) { o.Executable = filepath.Join(o.StateDir, "elsewhere.exe") }, "running from "},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := release("2.8.0")
			o := installed(t, f)
			c.bend(&o)
			res := Run(context.Background(), o)
			if res.Outcome != Skipped || res.Err == nil || !strings.Contains(res.Err.Error(), c.want) {
				t.Fatalf("Run = %+v", res)
			}
			if len(f.calls) != 0 {
				t.Fatalf("a skipped pass called out: %v", f.calls)
			}
			if st := recorded(t, o); st.Result != Skipped || !strings.Contains(st.Error, c.want) {
				t.Fatalf("update.json = %+v", st)
			}
		})
	}
}

func TestRunFailsAndKeepsTheBinary(t *testing.T) {
	for _, c := range []struct {
		name  string
		spoil func(f *fakeGH)
		want  string
	}{
		{"no gh", func(f *fakeGH) { f.fail["list"] = fmt.Errorf("gh: %w", os.ErrNotExist) }, "gh: "},
		{"a changed asset", func(f *fakeGH) { f.files[AssetName("2.8.0", "windows", "amd64")] = "tampered" }, "checksum mismatch"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := release("2.8.0")
			c.spoil(f)
			o := installed(t, f)
			res := Run(context.Background(), o)
			if res.Outcome != Failed || res.Err == nil || !strings.Contains(res.Err.Error(), c.want) {
				t.Fatalf("Run = %+v", res)
			}
			if got := canonicalBody(t, o); got != "old" {
				t.Fatalf("a failed pass touched the binary: %q", got)
			}
			if st := recorded(t, o); st.Result != Failed || st.Error != res.Err.Error() {
				t.Fatalf("update.json = %+v", st)
			}
		})
	}
}

func TestRunFailsWhenEverySlotIsHeld(t *testing.T) {
	o := installed(t, release("2.8.0"))
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
	res := Run(context.Background(), o)
	if res.Outcome != Failed || !strings.Contains(res.Err.Error(), "slots") {
		t.Fatalf("Run = %+v", res)
	}
	if got := canonicalBody(t, o); got != "old" {
		t.Fatalf("canonical binary = %q", got)
	}
}

func TestRunStepsAsideForAPassInProgress(t *testing.T) {
	f := release("2.8.0")
	o := installed(t, f)
	handle, held, err := lock.TryAcquire(filepath.Join(o.StateDir, "update.lock"))
	if err != nil || !held {
		t.Fatalf("take the lock: %v, %v", held, err)
	}
	defer handle.Release()
	res := Run(context.Background(), o)
	if res.Outcome != Busy || res.Err == nil || res.Err.Error() != "update in progress" {
		t.Fatalf("Run = %+v", res)
	}
	if st, _ := ReadStatus(o.StateDir); st != nil {
		t.Fatalf("a busy pass wrote update.json: %+v", st)
	}
	if len(f.calls) != 0 {
		t.Fatalf("a busy pass called out: %v", f.calls)
	}
}

func TestRunFailsWhenTheLockCannotBeOpened(t *testing.T) {
	o := installed(t, release("2.8.0"))
	if err := os.MkdirAll(filepath.Join(o.StateDir, "update.lock", "x"), 0o700); err != nil {
		t.Fatal(err)
	}
	if res := Run(context.Background(), o); res.Outcome != Failed || res.Err == nil {
		t.Fatalf("Run = %+v", res)
	}
}

func TestRunReportsAStatusItCouldNotWrite(t *testing.T) {
	o := installed(t, release("2.7.0"))
	if err := os.MkdirAll(filepath.Join(StatusPath(o.StateDir), "x"), 0o700); err != nil {
		t.Fatal(err)
	}
	res := Run(context.Background(), o)
	// The pass itself came to its outcome; only its record is missing.
	if res.Outcome != Current || res.Err != nil || res.StatusErr == nil {
		t.Fatalf("Run = %+v", res)
	}
}

// releasesOf is release(ver) with a list of its own: ver is the one asset
// the download serves.
func releasesOf(ver, list string) *fakeGH {
	f := release(ver)
	f.list = list
	return f
}

func TestRunTheBridgeTakesTheRestart(t *testing.T) {
	f := releasesOf("1.0.0", `[{"tagName":"v7.2.0","isPrerelease":true},{"tagName":"v1.0.0","isPrerelease":false}]`)
	f.version = "loomux 1.0.0\n"
	o := installed(t, f)
	o.Version, f.installed = "7.2.0", "loomux 7.2.0 (beta)\n"
	if res := Run(context.Background(), o); res.Outcome != Updated || res.Version != "1.0.0" {
		t.Fatalf("res = %+v", res)
	}
}

func TestRunTheBridgeStaysOffANewBetaUnlessMarked(t *testing.T) {
	for _, marked := range []bool{false, true} {
		want := "1.0.0"
		if marked {
			want = "1.1.0-beta.1"
		}
		f := releasesOf(want, `[{"tagName":"v7.2.0","isPrerelease":true},{"tagName":"v1.0.0","isPrerelease":false},{"tagName":"v1.1.0-beta.1","isPrerelease":true}]`)
		f.version = "loomux " + want + "\n"
		o := installed(t, f)
		o.Version, f.installed = "7.2.0", "loomux 7.2.0 (beta)\n"
		if marked {
			if err := WriteChannel(o.StateDir, true); err != nil {
				t.Fatal(err)
			}
		}
		if res := Run(context.Background(), o); res.Outcome != Updated || res.Version != want {
			t.Fatalf("marked=%v: res = %+v", marked, res)
		}
	}
}

func TestRunANewBinaryLeavesTheOldCountAlone(t *testing.T) {
	for _, marked := range []bool{false, true} {
		f := releasesOf("7.1.0", `[{"tagName":"v7.1.0","isPrerelease":true},{"tagName":"v1.0.0","isPrerelease":false}]`)
		o := installed(t, f)
		o.Version, o.Channel, f.installed = "1.0.0", "", "loomux 1.0.0\n"
		if marked {
			if err := WriteChannel(o.StateDir, true); err != nil {
				t.Fatal(err)
			}
		}
		if res := Run(context.Background(), o); res.Outcome != Current || res.Version != "1.0.0" {
			t.Fatalf("marked=%v: res = %+v", marked, res)
		}
	}
}

func TestRunAMarkedBetaStaysOnBetasAfterAStableRelease(t *testing.T) {
	f := releasesOf("1.2.0-beta.1", `[{"tagName":"v1.1.0","isPrerelease":false},{"tagName":"v1.2.0-beta.1","isPrerelease":true}]`)
	f.version = "loomux 1.2.0-beta.1\n"
	o := installed(t, f)
	o.Version, o.Channel, f.installed = "1.1.0-beta.2", "", "loomux 1.1.0-beta.2\n"
	if err := WriteChannel(o.StateDir, true); err != nil {
		t.Fatal(err)
	}
	if res := Run(context.Background(), o); res.Outcome != Updated || res.Version != "1.2.0-beta.1" {
		t.Fatalf("res = %+v", res)
	}
	if beta, _ := ReadChannel(o.StateDir); !beta {
		t.Fatal("the marker went")
	}
}

func TestRunAMarkedBetaTakesTheStableReleaseThatOvertakesIt(t *testing.T) {
	f := releasesOf("1.1.0", `[{"tagName":"v1.1.0","isPrerelease":false},{"tagName":"v1.1.0-beta.2","isPrerelease":true}]`)
	f.version = "loomux 1.1.0\n"
	o := installed(t, f)
	o.Version, o.Channel, f.installed = "1.1.0-beta.2", "", "loomux 1.1.0-beta.2\n"
	if err := WriteChannel(o.StateDir, true); err != nil {
		t.Fatal(err)
	}
	if res := Run(context.Background(), o); res.Outcome != Updated || res.Version != "1.1.0" {
		t.Fatalf("res = %+v", res)
	}
	if beta, _ := ReadChannel(o.StateDir); !beta {
		t.Fatal("the marker went")
	}
}

func TestRunUnmarkedNewBinaryTakesNoBeta(t *testing.T) {
	f := releasesOf("1.2.0-beta.1", `[{"tagName":"v1.1.0","isPrerelease":false},{"tagName":"v1.2.0-beta.1","isPrerelease":true}]`)
	o := installed(t, f)
	o.Version, o.Channel, f.installed = "1.1.0", "", "loomux 1.1.0\n"
	if res := Run(context.Background(), o); res.Outcome != Current {
		t.Fatalf("res = %+v", res)
	}
}

func TestRunRecordsAForeignMarkerAndTakesStable(t *testing.T) {
	f := releasesOf("1.2.0-beta.1", `[{"tagName":"v1.1.0","isPrerelease":false},{"tagName":"v1.2.0-beta.1","isPrerelease":true}]`)
	o := installed(t, f)
	o.Version, o.Channel, f.installed = "1.1.0", "", "loomux 1.1.0\n"
	if err := os.WriteFile(ChannelPath(o.StateDir), []byte("nightly"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := Run(context.Background(), o)
	if res.Outcome != Current || !strings.Contains(recorded(t, o).Error, `channel file holds "nightly"`) {
		t.Fatalf("res = %+v, status = %+v", res, recorded(t, o))
	}
}

func TestRunModes(t *testing.T) {
	list := `[{"tagName":"v1.1.0","isPrerelease":false},{"tagName":"v1.2.0-beta.1","isPrerelease":true},{"tagName":"v1.0.0","isPrerelease":false}]`
	for _, c := range []struct {
		name, mode, pin, running, asset string
		marked                          bool
		outcome                         Outcome
		version                         string
		markerAfter                     bool
	}{
		{"beta sets the marker and takes the newest", ModeBeta, "", "1.1.0", "1.2.0-beta.1", false, Updated, "1.2.0-beta.1", true},
		{"beta when current still sets the marker", ModeBeta, "", "1.2.0-beta.1", "1.2.0-beta.1", false, Current, "1.2.0-beta.1", true},
		{"stable goes back from a beta", ModeStable, "", "1.2.0-beta.1", "1.1.0", true, Updated, "1.1.0", false},
		{"pin a beta sets the marker", "", "1.2.0-beta.1", "1.1.0", "1.2.0-beta.1", false, Updated, "1.2.0-beta.1", true},
		{"pin a stable downgrades and clears", "", "1.0.0", "1.2.0-beta.1", "1.0.0", true, Updated, "1.0.0", false},
		{"pin what runs is current", "", "1.1.0", "1.1.0", "1.1.0", true, Current, "1.1.0", false},
		{"pin a stable of another number", "", "1.0.0", "1.1.0", "1.0.0", false, Updated, "1.0.0", false},
		{"stable over its own beta", ModeStable, "", "1.1.0-beta.2", "1.1.0", true, Updated, "1.1.0", false},
		{"pin the next beta", "", "1.2.0-beta.2", "1.2.0-beta.1", "1.2.0-beta.2", true, Updated, "1.2.0-beta.2", true},
		{"pin when what runs says nothing", "", "1.1.0", "garbage", "1.1.0", false, Updated, "1.1.0", false},
	} {
		f := releasesOf(c.asset, list)
		f.view = fmt.Sprintf(`{"tagName":"v%s","isPrerelease":%v}`, c.pin, strings.Contains(c.pin, "-"))
		f.version = "loomux " + c.asset + "\n"
		o := installed(t, f)
		o.Version, o.Channel, f.installed = c.running, "", "loomux "+c.running+"\n"
		o.Mode, o.Pin = c.mode, c.pin
		if c.marked {
			if err := WriteChannel(o.StateDir, true); err != nil {
				t.Fatal(err)
			}
		}
		res := Run(context.Background(), o)
		if res.Outcome != c.outcome || res.Version != c.version {
			t.Errorf("%s: res = %+v", c.name, res)
		}
		if beta, _ := ReadChannel(o.StateDir); beta != c.markerAfter {
			t.Errorf("%s: marker = %v, want %v", c.name, beta, c.markerAfter)
		}
	}
}

func TestRunPinOfAMissingVersionFailsAndKeepsTheMarker(t *testing.T) {
	f := release("1.1.0")
	f.fail["view"] = errors.New("release not found")
	o := installed(t, f)
	o.Pin = "9.9.9"
	if err := WriteChannel(o.StateDir, true); err != nil {
		t.Fatal(err)
	}
	res := Run(context.Background(), o)
	if res.Outcome != Failed || res.Err.Error() != "no release 9.9.9: release not found" {
		t.Fatalf("res = %+v", res)
	}
	if beta, _ := ReadChannel(o.StateDir); !beta {
		t.Fatal("a failed pass changed the marker")
	}
}

func TestRunReportsAMarkerItCouldNotWrite(t *testing.T) {
	f := release("2.8.0")
	o := installed(t, f)
	o.Mode = ModeBeta
	if err := os.MkdirAll(filepath.Join(ChannelPath(o.StateDir), "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := Run(context.Background(), o)
	if res.Outcome != Updated || res.Err == nil || !strings.Contains(res.Err.Error(), "channel") {
		t.Fatalf("res = %+v", res)
	}
}
