package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xidus90/loomux/internal/release"
)

var releaseGo release.GoBuild = release.ExecGoBuild

var releaseCommands = map[string]command{
	"build":            devReleaseBuild,
	"changelog-insert": devReleaseChangelog,
	"next-beta":        devReleaseNextBeta,
	"next-version":     devReleaseNextVersion,
	"parse-body":       devReleaseParseBody,
}

func devRelease(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "loomux dev release: subcommand required")
		return 2
	}
	sub, ok := releaseCommands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "loomux dev release: unknown subcommand %q\n", args[0])
		return 2
	}
	return sub(args[1:], stdin, stdout, stderr)
}

// readInput reads a named file, or stdin for "-".
func readInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(path)
}

func devReleaseNextVersion(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev release next-version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	bump := fs.String("bump", "", "major, minor or patch")
	tags := fs.String("tags", "-", "file with one tag per line, - for stdin")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	data, err := readInput(*tags, stdin)
	if err == nil {
		var v string
		if v, err = release.NextVersion(strings.Fields(string(data)), *bump); err == nil {
			fmt.Fprintln(stdout, v)
			return 0
		}
	}
	fmt.Fprintf(stderr, "loomux dev release next-version: %v\n", err)
	return 2
}

func devReleaseNextBeta(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev release next-beta", flag.ContinueOnError)
	fs.SetOutput(stderr)
	bump := fs.String("bump", "", "major, minor or patch")
	tags := fs.String("tags", "-", "file with one tag per line, - for stdin")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	data, err := readInput(*tags, stdin)
	if err == nil {
		var v string
		if v, err = release.NextBeta(strings.Fields(string(data)), *bump); err == nil {
			fmt.Fprintln(stdout, v)
			return 0
		}
	}
	fmt.Fprintf(stderr, "loomux dev release next-beta: %v\n", err)
	return 2
}

func devReleaseParseBody(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev release parse-body", flag.ContinueOnError)
	fs.SetOutput(stderr)
	labels := fs.String("labels", "", "comma-separated labels of the pull request")
	body := fs.String("body", "-", "file with the pull request body, - for stdin")
	commits := fs.String("commits", "", "file with a JSON array of the pull request's commit messages")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	data, err := readInput(*body, stdin)
	var messages []string
	if err == nil && *commits != "" {
		var raw []byte
		if raw, err = os.ReadFile(*commits); err == nil {
			err = json.Unmarshal(raw, &messages)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev release parse-body: %v\n", err)
		return 2
	}
	parsed, problems := release.ParseBody(strings.FieldsFunc(*labels, func(r rune) bool { return r == ',' }), string(data))
	if len(problems) == 0 {
		problems = release.CheckCommits(parsed.Bump, messages)
	}
	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintf(stderr, "loomux dev release parse-body: %s\n", p)
		}
		return 1
	}
	out, _ := json.Marshal(parsed)
	fmt.Fprintf(stdout, "%s\n", out)
	return 0
}

func devReleaseChangelog(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev release changelog-insert", flag.ContinueOnError)
	fs.SetOutput(stderr)
	version := fs.String("version", "", "release version without v")
	date := fs.String("date", "", "release date, YYYY-MM-DD")
	link := fs.String("link", "", "URL of the pull request")
	file := fs.String("file", "CHANGELOG.md", "changelog to update")
	notes := fs.String("notes", "-", "file with the changelog block, - for stdin")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *version == "" || *date == "" || *link == "" {
		fmt.Fprintln(stderr, "loomux dev release changelog-insert: --version, --date and --link are required")
		return 2
	}
	block, err := readInput(*notes, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev release changelog-insert: %v\n", err)
		return 2
	}
	existing, err := os.ReadFile(*file)
	if errors.Is(err, os.ErrNotExist) {
		existing, err = nil, nil
	}
	var updated []byte
	if err == nil {
		updated, err = release.InsertChangelog(existing, *version, *date, *link, string(block))
	}
	if err == nil {
		err = os.WriteFile(*file, updated, 0o644)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev release changelog-insert: %v\n", err)
		if errors.Is(err, release.ErrDuplicate) {
			return 1
		}
		return 2
	}
	return 0
}

func devReleaseBuild(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev release build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	version := fs.String("version", "", "release version without v")
	channel := fs.String("channel", "", "release channel, e.g. beta")
	out := fs.String("out", "dist", "output directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *version == "" {
		fmt.Fprintln(stderr, "loomux dev release build: --version is required")
		return 2
	}
	names, err := release.Build(*version, *channel, *out, releaseGo)
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev release build: %v\n", err)
		return 1
	}
	for _, n := range names {
		fmt.Fprintln(stdout, n)
	}
	return 0
}
