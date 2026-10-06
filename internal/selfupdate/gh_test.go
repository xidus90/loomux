package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestLatestTakesTheHighestReleaseOfTheChannel(t *testing.T) {
	f := &fakeGH{list: `[{"tagName":"v2.7.0","isPrerelease":true},{"tagName":"v2.10.0","isPrerelease":true}]`}
	var args []string
	run := func(ctx context.Context, name string, a ...string) ([]byte, error) {
		args = a
		return f.run(ctx, name, a...)
	}
	rel, err := latest(context.Background(), run, takesAll)
	if err != nil || rel.Tag != "v2.10.0" {
		t.Fatalf("latest = %v, %v; want v2.10.0", rel, err)
	}
	for _, want := range []string{"list", "--repo", Repo, "--exclude-drafts", "tagName,isPrerelease"} {
		if !slices.Contains(args, want) {
			t.Errorf("gh release list misses %q: %v", want, args)
		}
	}
}

// Only the stable channel asks GitHub to leave pre-releases out: many betas
// would otherwise push the newest stable release out of the 30 listed.
func TestLatestExcludesPreReleasesOnlyForTheStableChannel(t *testing.T) {
	for _, tc := range []struct {
		name string
		t    takes
		want bool
	}{{"stable", takesStable, true}, {"old", takesOld, false}, {"all", takesAll, false}} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeGH{list: `[{"tagName":"v1.0.0","isPrerelease":false}]`}
			var args []string
			run := func(ctx context.Context, name string, a ...string) ([]byte, error) {
				args = a
				return f.run(ctx, name, a...)
			}
			if _, err := latest(context.Background(), run, tc.t); err != nil {
				t.Fatal(err)
			}
			if got := slices.Contains(args, "--exclude-pre-releases"); got != tc.want {
				t.Fatalf("--exclude-pre-releases = %v, want %v: %v", got, tc.want, args)
			}
		})
	}
}

// A repository that has releases and gives none is a fault, not a state:
// before the channel was read, exactly this made every pass say "current".
func TestLatestFailsWhenTheChannelHasNoRelease(t *testing.T) {
	f := &fakeGH{list: `[{"tagName":"v2.7.0","isPrerelease":true}]`}
	_, err := latest(context.Background(), f.run, takesStable)
	if err == nil || err.Error() != "no release in channel stable" {
		t.Fatalf("err = %v", err)
	}
}

func TestLatestNamesTheChannelItSearched(t *testing.T) {
	f := &fakeGH{list: `[]`}
	_, err := latest(context.Background(), f.run, takesAll)
	if err == nil || err.Error() != "no release in channel beta" {
		t.Fatalf("err = %v", err)
	}
}

// Tags that are no version are passed over; when none is left, that is the
// same fault as a channel without releases.
func TestLatestFailsWhenNoTagIsAVersion(t *testing.T) {
	f := &fakeGH{list: `[{"tagName":"nightly","isPrerelease":true},{"tagName":"v2.8","isPrerelease":false}]`}
	_, err := latest(context.Background(), f.run, takesAll)
	if err == nil || err.Error() != "no release in channel beta" {
		t.Fatalf("err = %v", err)
	}
}

func TestLatestRefusesAnswersItCannotRead(t *testing.T) {
	f := &fakeGH{list: `not json`}
	if _, err := latest(context.Background(), f.run, takesAll); err == nil || !strings.Contains(err.Error(), "parse gh release list") {
		t.Fatalf("err = %v", err)
	}
}

func TestLatestNamesAMissingGh(t *testing.T) {
	f := &fakeGH{fail: map[string]error{"list": fmt.Errorf("gh: %w", exec.ErrNotFound)}}
	_, err := latest(context.Background(), f.run, takesAll)
	if err == nil || err.Error() != "gh not found; install GitHub CLI and run gh auth login" {
		t.Fatalf("err = %v", err)
	}
}

func TestLatestPassesOtherFailuresOn(t *testing.T) {
	f := &fakeGH{fail: map[string]error{"list": errors.New("gh: HTTP 404: Not Found")}}
	if _, err := latest(context.Background(), f.run, takesAll); err == nil || err.Error() != "gh: HTTP 404: Not Found" {
		t.Fatalf("err = %v", err)
	}
}

func TestACallThatOutlivesItsDeadlineSaysSo(t *testing.T) {
	callTimeout = 10 * time.Millisecond
	t.Cleanup(func() { callTimeout = 2 * time.Minute })
	hang := func(ctx context.Context, _ string, _ ...string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if _, err := latest(context.Background(), hang, takesAll); err == nil || !strings.Contains(err.Error(), "gh timed out") {
		t.Fatalf("err = %v", err)
	}
}

func TestFirstLine(t *testing.T) {
	for in, want := range map[string]string{
		"":                            "",
		"one":                         "one",
		"  first \r\nsecond\n":        "first",
		"\nHTTP 401: Bad credentials": "HTTP 401: Bad credentials",
	} {
		if got := firstLine(in); got != want {
			t.Errorf("firstLine(%q) = %q, want %q", in, got, want)
		}
	}
}

// ExecRunner against real processes, no gh and no network: the test binary
// stands in both for a program that succeeds and for one that fails and says
// why on standard error.
func TestExecRunner(t *testing.T) {
	t.Run("a program that does not exist", func(t *testing.T) {
		_, err := ExecRunner(context.Background(), "loomux-no-such-program")
		// call() tells a missing gh apart by this sentinel, so the wrap must keep it.
		if !errors.Is(err, exec.ErrNotFound) || !strings.HasPrefix(err.Error(), "loomux-no-such-program: ") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("a program that succeeds", func(t *testing.T) {
		if _, err := ExecRunner(context.Background(), os.Args[0], "-test.run=^$"); err != nil {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("a program that fails and says why", func(t *testing.T) {
		_, err := ExecRunner(context.Background(), os.Args[0], "-test.no-such-flag")
		var exit *exec.ExitError
		if !errors.As(err, &exit) || !strings.Contains(err.Error(), "flag provided but not defined") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestViewAsksForOneTag(t *testing.T) {
	f := &fakeGH{view: `{"tagName":"v1.1.0-beta.2","isPrerelease":true}`}
	var args []string
	run := func(ctx context.Context, name string, a ...string) ([]byte, error) {
		args = a
		return f.run(ctx, name, a...)
	}
	rel, err := view(context.Background(), run, "1.1.0-beta.2")
	if err != nil || rel != (Release{Tag: "v1.1.0-beta.2", Prerelease: true}) {
		t.Fatalf("view = %+v, %v", rel, err)
	}
	if !slices.Equal(args[:3], []string{"release", "view", "v1.1.0-beta.2"}) || !slices.Contains(args, Repo) {
		t.Errorf("gh args = %v", args)
	}
}

func TestViewNamesAVersionThatIsNotThere(t *testing.T) {
	f := &fakeGH{fail: map[string]error{"view": errors.New("release not found")}}
	_, err := view(context.Background(), f.run, "9.9.9")
	if err == nil || err.Error() != "no release 9.9.9: release not found" {
		t.Fatalf("err = %v", err)
	}
}

func TestViewRefusesWhatIsNoVersion(t *testing.T) {
	f := &fakeGH{view: `{"tagName":"nightly"}`}
	if _, err := view(context.Background(), f.run, "nightly"); err == nil {
		t.Fatal("took nightly")
	}
	f = &fakeGH{view: `not json`}
	if _, err := view(context.Background(), f.run, "1.0.0"); err == nil || !strings.Contains(err.Error(), "parse gh release view") {
		t.Fatal("took a broken answer")
	}
}
