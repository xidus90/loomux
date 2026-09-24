package cli

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/selfupdate"
)

// fakeSelfUpdate replaces the pass with one that answers res and counts its
// calls; the real one reaches GitHub.
func fakeSelfUpdate(t *testing.T, res selfupdate.Result) *int {
	t.Helper()
	calls := 0
	selfUpdateRun = func(context.Context, selfupdate.Options) selfupdate.Result {
		calls++
		return res
	}
	t.Cleanup(func() { selfUpdateRun = selfupdate.Run })
	return &calls
}

func TestSelfUpdateCommand(t *testing.T) {
	for _, c := range []struct {
		name      string
		res       selfupdate.Result
		code      int
		out, errs string
	}{
		{"current", selfupdate.Result{Outcome: selfupdate.Current, Version: "2.7.0"}, 0, "already current (v2.7.0)\n", ""},
		{"updated", selfupdate.Result{Outcome: selfupdate.Updated, Version: "2.8.0"}, 0, "updated to v2.8.0; serve switches on the next bridge\n", ""},
		{"skipped", selfupdate.Result{Outcome: selfupdate.Skipped, Err: errors.New("development build 0.0.0-dev is never replaced")}, 2, "", "loomux self-update: skipped: development build 0.0.0-dev is never replaced\n"},
		{"failed", selfupdate.Result{Outcome: selfupdate.Failed, Err: errors.New("checksum mismatch for x")}, 1, "", "loomux self-update: checksum mismatch for x\n"},
		{"busy", selfupdate.Result{Outcome: selfupdate.Busy, Err: errors.New("update in progress")}, 1, "", "loomux self-update: update in progress\n"},
		{"status unwritten", selfupdate.Result{Outcome: selfupdate.Current, Version: "2.7.0", StatusErr: errors.New("disk full")}, 0, "already current (v2.7.0)\n", "loomux self-update: record update.json: disk full\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			fakeSelfUpdate(t, c.res)
			var out, errs bytes.Buffer
			code := selfUpdateCommand(nil, nil, &out, &errs)
			if code != c.code || out.String() != c.out || errs.String() != c.errs {
				t.Fatalf("code %d, out %q, err %q", code, out.String(), errs.String())
			}
		})
	}
}

// A pass by hand may run from a checkout; it says so, so that session start
// does not take its location for serve's.
func TestSelfUpdateCommandRunsAsCLI(t *testing.T) {
	var source string
	selfUpdateRun = func(_ context.Context, o selfupdate.Options) selfupdate.Result {
		source = o.Source
		return selfupdate.Result{Outcome: selfupdate.Current}
	}
	t.Cleanup(func() { selfUpdateRun = selfupdate.Run })
	selfUpdateCommand(nil, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if source != selfupdate.SourceCLI {
		t.Fatalf("source = %q", source)
	}
}

func TestSelfUpdateTakesNoArguments(t *testing.T) {
	calls := fakeSelfUpdate(t, selfupdate.Result{})
	var errs bytes.Buffer
	if code := selfUpdateCommand([]string{"--to", "x"}, nil, &bytes.Buffer{}, &errs); code != 2 {
		t.Fatalf("code = %d", code)
	}
	if *calls != 0 || !strings.Contains(errs.String(), "usage: loomux self-update") {
		t.Fatalf("calls %d, err %q", *calls, errs.String())
	}
}

func TestSelfUpdateOptionsDescribeThisProcess(t *testing.T) {
	t.Setenv(config.StateDirEnv, t.TempDir())
	o := selfUpdateOptions(selfupdate.SourceServe)
	if o.Source != selfupdate.SourceServe || o.StateDir != config.StateDir() || o.Version != Version || o.Channel != Channel ||
		o.GOOS != runtime.GOOS || o.GOARCH != runtime.GOARCH || o.Executable == "" || o.Run == nil || o.Now == nil {
		t.Fatalf("options = %+v", o)
	}
}
