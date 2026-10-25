package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestRunTimesOut(t *testing.T) {
	o := Options{Timeout: 10 * time.Millisecond, Exec: func(ctx context.Context, c Command) ([]byte, []byte, error) {
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}}
	_, err := run(context.Background(), o, Command{Name: "claude"})
	if err == nil || !strings.Contains(err.Error(), "timed out after 10ms") || !strings.Contains(err.Error(), "[agent] timeout") {
		t.Fatalf("got %v", err)
	}
}

func TestRunWithoutTimeout(t *testing.T) {
	o := Options{Exec: func(ctx context.Context, c Command) ([]byte, []byte, error) {
		if _, ok := ctx.Deadline(); ok {
			t.Error("deadline set")
		}
		return []byte("out"), nil, nil
	}}
	out, err := run(context.Background(), o, Command{Name: "claude"})
	if err != nil || string(out) != "out" {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestRunParentCancelIsNotATimeout(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	o := Options{Exec: func(ctx context.Context, c Command) ([]byte, []byte, error) { return nil, nil, ctx.Err() }}
	_, err := run(ctx, o, Command{Name: "claude"})
	if err == nil || strings.Contains(err.Error(), "timed out") || err.Error() != "claude failed: context deadline exceeded" {
		t.Fatalf("got %v", err)
	}
}

func TestRunNamesMissingCLI(t *testing.T) {
	o := Options{Exec: func(context.Context, Command) ([]byte, []byte, error) { return nil, nil, exec.ErrNotFound }}
	_, err := run(context.Background(), o, Command{Name: "claude"})
	if err == nil || !strings.Contains(err.Error(), "claude CLI not found on PATH") {
		t.Fatalf("got %v", err)
	}
}

func TestRunExitCodeShowsStderr(t *testing.T) {
	var lines []string
	for i := 1; i <= 12; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	o := Options{Exec: func(context.Context, Command) ([]byte, []byte, error) {
		return nil, []byte(strings.Join(lines, "\n")), errors.New("exit status 1")
	}}
	_, err := run(context.Background(), o, Command{Name: "claude"})
	if err == nil || !strings.Contains(err.Error(), "line5") || strings.Contains(err.Error(), "line6") {
		t.Fatalf("got %v", err)
	}
}

func TestRunSetsDir(t *testing.T) {
	o := Options{Root: "/some/root", Exec: func(_ context.Context, c Command) ([]byte, []byte, error) {
		if c.Dir != "/some/root" {
			t.Errorf("dir %q", c.Dir)
		}
		return nil, nil, nil
	}}
	if _, err := run(context.Background(), o, Command{Name: "claude"}); err != nil {
		t.Fatal(err)
	}
}

func TestSystemRunsAProcess(t *testing.T) {
	c := Command{Name: os.Args[0], Args: []string{"-test.run=TestHelperProcess"}, Env: []string{"GO_HELPER=1"}, Stdin: []byte("hello")}
	out, _, err := System(context.Background(), c)
	if err != nil || string(out) != "echo:hello:1" {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_HELPER") != "1" {
		return
	}
	var b [64]byte
	n, _ := os.Stdin.Read(b[:])
	fmt.Fprintf(os.Stdout, "echo:%s:%s", b[:n], os.Getenv("GO_HELPER"))
	os.Exit(0)
}
