package serve

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/testlock"
)

var upkeepNow = time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)

// testUpkeep is an upkeep over dir with a clock that stands still and a pass
// that records its calls and answers what the test hands it.
func testUpkeep(t *testing.T, dir string, answer func() (*maintenance.Report, error)) (*Upkeep, *int) {
	t.Helper()
	calls := 0
	u := NewUpkeep(dir, filepath.Join(dir, "legacy"))
	u.now = func() time.Time { return upkeepNow }
	u.run = func(_ context.Context, now time.Time) (*maintenance.Report, error) {
		calls++
		if !now.Equal(upkeepNow) {
			t.Errorf("the pass ran at %v, want the upkeep's clock", now)
		}
		return answer()
	}
	return u, &calls
}

func writeStamp(t *testing.T, dir string, stamp time.Time) {
	t.Helper()
	path := filepath.Join(dir, "maintenance", "last-run.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(stamp.Format("2006-01-02T15:04:05+00:00")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func oneCase(area string) *maintenance.Report {
	return &maintenance.Report{Cases: []maintenance.Case{{Area: area, Target: "topics/x.md"}}}
}

func TestAPassIsOwedWithoutAStampAndWhenItHasAged(t *testing.T) {
	for name, stamp := range map[string]*time.Time{
		"no stamp":     nil,
		"a day old":    ptr(upkeepNow.Add(-24 * time.Hour)),
		"two days old": ptr(upkeepNow.Add(-48 * time.Hour)),
	} {
		dir := t.TempDir()
		if stamp != nil {
			writeStamp(t, dir, *stamp)
		}
		u, calls := testUpkeep(t, dir, func() (*maintenance.Report, error) { return oneCase("a"), nil })
		u.catchUp(context.Background())
		if *calls != 1 || u.report == nil || !u.Settled() {
			t.Errorf("%s: %d calls, report %v, settled %v", name, *calls, u.report, u.Settled())
		}
	}
}

func ptr(t time.Time) *time.Time { return &t }

func TestAFreshStampOwesNothingAndForgetsTheReport(t *testing.T) {
	// The stamp moved without this process: somebody reconciled by hand, and
	// the cases remembered may be decided by now.
	dir := t.TempDir()
	writeStamp(t, dir, upkeepNow.Add(-time.Hour))
	u, calls := testUpkeep(t, dir, func() (*maintenance.Report, error) { return oneCase("a"), nil })
	u.report, u.failure, u.failed = oneCase("a"), "old", true
	u.catchUp(context.Background())
	if *calls != 0 || u.report != nil || u.failed {
		t.Fatalf("%d calls, report %v, failed %v", *calls, u.report, u.failed)
	}
}

func TestAFailureIsRecordedAndOpensTheGate(t *testing.T) {
	dir := t.TempDir()
	fail := true
	u, _ := testUpkeep(t, dir, func() (*maintenance.Report, error) {
		if fail {
			return nil, errors.New("share gone")
		}
		return oneCase("a"), nil
	})
	u.report = oneCase("b")
	u.catchUp(context.Background())
	if !u.Settled() || !u.failed || u.failure != "share gone" || u.report == nil {
		t.Fatalf("settled %v, failed %v %q, report %v", u.Settled(), u.failed, u.failure, u.report)
	}
	fail = false
	u.catchUp(context.Background())
	if u.failed || u.report.Cases[0].Area != "a" {
		t.Fatalf("the next good pass kept failed %v, report %v", u.failed, u.report)
	}
}

func TestAStampThatCannotBeReadIsAFailure(t *testing.T) {
	dir := t.TempDir()
	writeStamp(t, dir, upkeepNow)
	testlock.Lock(t, filepath.Join(dir, "maintenance", "last-run.txt"))
	u, calls := testUpkeep(t, dir, func() (*maintenance.Report, error) { return nil, nil })
	u.catchUp(context.Background())
	if *calls != 0 || !u.failed {
		t.Fatalf("%d calls, failed %v", *calls, u.failed)
	}
}

func TestCaughtUpWaitsForTheFirstPass(t *testing.T) {
	u, _ := testUpkeep(t, t.TempDir(), func() (*maintenance.Report, error) { return nil, nil })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := u.CaughtUp(ctx); !errors.Is(err, context.Canceled) || u.Settled() {
		t.Fatalf("before the pass: err %v, settled %v", err, u.Settled())
	}
	u.catchUp(context.Background())
	if err := u.CaughtUp(context.Background()); err != nil {
		t.Fatalf("after the pass: %v", err)
	}
}

func TestKeepUpRunsOncePerIntervalUntilTheEnd(t *testing.T) {
	u, calls := testUpkeep(t, t.TempDir(), func() (*maintenance.Report, error) { return nil, nil })
	ticks := make(chan time.Time)
	var asked []time.Duration
	u.after = func(d time.Duration) <-chan time.Time {
		asked = append(asked, d)
		return ticks
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		u.KeepUp(ctx)
		close(done)
	}()
	ticks <- upkeepNow
	ticks <- upkeepNow
	cancel()
	<-done
	if *calls != 3 || len(asked) != 3 || asked[0] != 24*time.Hour {
		t.Fatalf("%d passes, waits %v", *calls, asked)
	}
}

// upkeepRegistry registers a visible area and a local-only one under dir and
// answers their paths.
func upkeepRegistry(t *testing.T, dir string) (open, hidden string) {
	t.Helper()
	open, hidden = t.TempDir(), t.TempDir()
	write := func(path, body string) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(open, ".loomux", "config.toml"), "[area]\nscope = \"project/open\"\n")
	write(filepath.Join(hidden, ".loomux", "config.toml"), "[area]\nscope = \"project/hidden\"\n\n[privacy]\nmode = \"local_only\"\n")
	write(filepath.Join(dir, "registry.toml"),
		"[[area]]\nscope = \"project/open\"\npath = "+strconv.Quote(filepath.ToSlash(open))+"\n"+
			"[[area]]\nscope = \"project/hidden\"\npath = "+strconv.Quote(filepath.ToSlash(hidden))+"\n")
	return open, hidden
}

func TestTheTrailerOfEachTool(t *testing.T) {
	dir := t.TempDir()
	open, _ := upkeepRegistry(t, dir)
	writeStamp(t, dir, upkeepNow.Add(-48*time.Hour))
	u, _ := testUpkeep(t, dir, nil)
	u.report = &maintenance.Report{
		Cases: []maintenance.Case{{Area: "project/open", Target: "a.md"}, {Area: "project/hidden", Target: "b.md"}},
		Unreadable: []string{
			filepath.Join(open, "review", "project-open", "c1", "case.toml"),
			filepath.Join(open, "review", "project-hidden", "c2", "case.toml"),
		},
	}
	stale := "! the last full reconciliation was 2026-09-21T12:00:00+00:00, more than 24 hours ago: " +
		"a source may have changed without this answer knowing (run `brain reconcile`)"
	for _, c := range []struct {
		cmd  string
		ch   privacy.Channel
		want []string
	}{
		{"status", privacy.ChannelLocal, []string{
			"the daily reconciliation opened a case for project/open/a.md; `loomux cases` lists it",
			"the daily reconciliation opened a case for project/hidden/b.md; `loomux cases` lists it",
			"unreadable case file, skipped by the daily reconciliation: " + filepath.Join(open, "review", "project-open", "c1", "case.toml"),
			"unreadable case file, skipped by the daily reconciliation: " + filepath.Join(open, "review", "project-hidden", "c2", "case.toml"),
		}},
		{"status", privacy.ChannelCloud, []string{
			"the daily reconciliation opened a case for project/open/a.md; `loomux cases` lists it",
			"unreadable case file, skipped by the daily reconciliation: project-open/c1",
		}},
		{"search", privacy.ChannelCloud, []string{"! 1 open case(s) and 1 unreadable case file(s) from the daily reconciliation; `status` lists them"}},
		{"search", privacy.ChannelLocal, []string{"! 2 open case(s) and 2 unreadable case file(s) from the daily reconciliation; `status` lists them"}},
		{"read", privacy.ChannelCloud, []string{"! 1 open case(s) and 1 unreadable case file(s) from the daily reconciliation; `status` lists them", stale}},
	} {
		got, err := u.Trailer(c.cmd, c.ch)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s on %s: %q, %v\nwant %q", c.cmd, c.ch, got, err, c.want)
		}
	}
}

func TestAFailureLeadsTheHeadlineAndHidesItsCauseFromTheCloud(t *testing.T) {
	dir := t.TempDir()
	upkeepRegistry(t, dir)
	u, _ := testUpkeep(t, dir, nil)
	u.failure, u.failed = `C:\share\x: gone`, true
	u.report = oneCase("project/open")
	local, _ := u.Trailer("status", privacy.ChannelLocal)
	cloud, _ := u.Trailer("catalog", privacy.ChannelCloud)
	wantLocal := []string{
		"the daily reconciliation failed: C:\\share\\x: gone; sources changed since the last pass are not yet cases -- fix the cause, then run `loomux reconcile`",
		"the daily reconciliation opened a case for project/open/topics/x.md; `loomux cases` lists it",
	}
	wantCloud := []string{"! the daily reconciliation failed (the cause is named on the local channel); " +
		"sources changed since the last pass are not yet cases -- fix the cause, then run `loomux reconcile`"}
	if !reflect.DeepEqual(local, wantLocal) || !reflect.DeepEqual(cloud, wantCloud) {
		t.Fatalf("local %q\ncloud %q", local, cloud)
	}
}

func TestAQuietUpkeepAddsNothing(t *testing.T) {
	dir := t.TempDir()
	upkeepRegistry(t, dir)
	u, _ := testUpkeep(t, dir, nil)
	if got, err := u.Trailer("read", privacy.ChannelLocal); err != nil || len(got) != 0 {
		t.Fatalf("trailer %q, %v", got, err)
	}
}

func TestTheTrailerReportsWhatItCannotRead(t *testing.T) {
	broken := t.TempDir()
	if err := os.WriteFile(filepath.Join(broken, "registry.toml"), []byte("{{{"), 0o600); err != nil {
		t.Fatal(err)
	}
	u, _ := testUpkeep(t, broken, nil)
	u.report = oneCase("project/open")
	if _, err := u.Trailer("status", privacy.ChannelLocal); err == nil {
		t.Error("an unreadable registry gave notes")
	}
	if _, err := u.Trailer("read", privacy.ChannelLocal); err == nil {
		t.Error("an unreadable registry gave a headline")
	}
	dir := t.TempDir()
	upkeepRegistry(t, dir)
	writeStamp(t, dir, upkeepNow)
	testlock.Lock(t, filepath.Join(dir, "maintenance", "last-run.txt"))
	u, _ = testUpkeep(t, dir, nil)
	if _, err := u.Trailer("read", privacy.ChannelLocal); err == nil {
		t.Error("an unreadable stamp gave a trailer")
	}
}

func TestAnUnreadableCaseOutsideEverySeenAreaIsNotNamed(t *testing.T) {
	seen := map[string]config.Area{"project/open": {Scope: "project/open", Path: `C:\vault`}}
	if _, ok := unreadableLine(`C:\elsewhere\review\project-open\c1\case.toml`, privacy.ChannelLocal, seen); ok {
		t.Error("a case file under no seen area was named")
	}
	if !withinFolded(`C:\Vault\Review\x`, `c:\vault`) || withinFolded(`C:\v`, `C:\vault`) || withinFolded(`C:\x`, `C:\vault\deep`) {
		t.Error("withinFolded does not compare as Windows does")
	}
}

func TestTheRealPassRunsOverTheRegistry(t *testing.T) {
	// No registry and an empty one are serve's first minute, not a failure;
	// a registry without a review centre is one.
	empty := t.TempDir()
	u := NewUpkeep(empty, filepath.Join(empty, "legacy"))
	u.catchUp(context.Background())
	if u.failed || u.report != nil {
		t.Fatalf("no registry: failed %v %q, report %v", u.failed, u.failure, u.report)
	}
	if err := os.WriteFile(filepath.Join(empty, "registry.toml"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if report, err := reconcileRegistered(context.Background(), u.lookup, upkeepNow); report != nil || err != nil {
		t.Fatalf("empty registry: %v, %v", report, err)
	}
	dir := t.TempDir()
	upkeepRegistry(t, dir)
	lookup := NewUpkeep(dir, filepath.Join(dir, "legacy")).lookup
	if _, err := reconcileRegistered(context.Background(), lookup, upkeepNow); !errors.Is(err, maintenance.ErrNoReviewCentre) {
		t.Fatalf("no review centre: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "registry.toml"), []byte("{{{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := reconcileRegistered(context.Background(), lookup, upkeepNow); err == nil {
		t.Fatal("a broken registry passed")
	}
}

func TestTheRealPassReportsWhatItFound(t *testing.T) {
	dir := t.TempDir()
	open, _ := upkeepRegistry(t, dir)
	if err := os.WriteFile(filepath.Join(open, ".loomux", "config.toml"),
		[]byte("[area]\nscope = \"project/open\"\n\n[layout]\nreview = \"review\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := reconcileRegistered(context.Background(), NewUpkeep(dir, filepath.Join(dir, "legacy")).lookup, upkeepNow)
	if err != nil || report == nil || len(report.Cases) != 0 {
		t.Fatalf("report %v, err %v", report, err)
	}
}

// The edges the mutation round of 2026-09-23 found untested.

func TestAFailureWithoutAReportIsTheOnlyLine(t *testing.T) {
	dir := t.TempDir()
	upkeepRegistry(t, dir)
	u, _ := testUpkeep(t, dir, nil)
	u.failure, u.failed = "share gone", true
	got, err := u.Notes(privacy.ChannelLocal)
	if err != nil || len(got) != 1 {
		t.Fatalf("notes %q, %v", got, err)
	}
}

func TestAnEmptyReportAsksNoRegistry(t *testing.T) {
	// A pass that opened nothing has nothing to gate; a registry that went
	// bad since must not turn every answer into an error.
	broken := t.TempDir()
	if err := os.WriteFile(filepath.Join(broken, "registry.toml"), []byte("{{{"), 0o600); err != nil {
		t.Fatal(err)
	}
	u, _ := testUpkeep(t, broken, nil)
	u.report = &maintenance.Report{}
	if got, err := u.Notes(privacy.ChannelLocal); err != nil || got != nil {
		t.Fatalf("notes %q, %v", got, err)
	}
}

func TestWithinFoldedAtEqualAndShorterPaths(t *testing.T) {
	if !withinFolded(`C:\Vault`, `c:\vault`) {
		t.Error("a directory does not lie within itself")
	}
	if withinFolded(`C:\vault`, `C:\vault\deep`) {
		t.Error("a parent lies within its child")
	}
}

// The pass runs under KeepUp's context, so ending serve ends a pass that is
// still stating and hashing instead of holding the shutdown until it is done.
func TestKeepUpHandsItsContextToThePass(t *testing.T) {
	u, _ := testUpkeep(t, t.TempDir(), func() (*maintenance.Report, error) { return nil, nil })
	started := make(chan struct{})
	u.run = func(ctx context.Context, _ time.Time) (*maintenance.Report, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		u.KeepUp(ctx)
		close(done)
	}()
	<-started
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("KeepUp still waits for a pass whose context has ended")
	}
}

// The headline counts what it names: a skipped case file is no open case, so
// a report of only cases says only cases, and one of only skipped files says
// only those.
func TestTheHeadlineCountsCasesAndUnreadableFilesApart(t *testing.T) {
	dir := t.TempDir()
	open, _ := upkeepRegistry(t, dir)
	writeStamp(t, dir, upkeepNow)
	u, _ := testUpkeep(t, dir, nil)
	u.report = oneCase("project/open")
	if got, _ := u.Trailer("search", privacy.ChannelLocal); !reflect.DeepEqual(got,
		[]string{"! 1 open case(s) from the daily reconciliation; `status` lists them"}) {
		t.Errorf("cases only: %q", got)
	}
	u.report = &maintenance.Report{Unreadable: []string{filepath.Join(open, "review", "project-open", "c1", "case.toml")}}
	if got, _ := u.Trailer("search", privacy.ChannelLocal); !reflect.DeepEqual(got,
		[]string{"! 1 unreadable case file(s) from the daily reconciliation; `status` lists them"}) {
		t.Errorf("unreadable only: %q", got)
	}
}
