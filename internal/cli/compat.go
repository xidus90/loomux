package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/compat"
	"github.com/xidus90/loomux/internal/verify/godotsuite"
)

type listFlag []string

func (l *listFlag) String() string     { return strings.Join(*l, ",") }
func (l *listFlag) Set(v string) error { *l = append(*l, v); return nil }

// compatRun is compat.Run; a test stands in for the real runs.
var compatRun = compat.Run

func devCompat(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	switch {
	case len(args) > 0 && args[0] == "matrix":
		return devCompatMatrix(args[1:], stdout, stderr)
	case len(args) > 0 && args[0] == "run":
		return devCompatRun(args[1:], stdout, stderr)
	case len(args) > 0 && args[0] == "scout":
		return devCompatScout(args[1:], stdout, stderr)
	}
	fmt.Fprintln(stderr, "loomux dev compat: subcommand required: matrix, run, scout")
	return 2
}

func knownEntry(key string) bool {
	entries, _ := compat.Load() // the embedded list is validated by compat's tests
	return slices.ContainsFunc(entries, func(e compat.Entry) bool { return e.Key() == key })
}

// devCompatRun runs one entry for real: the key comes first and the flags
// after it, as with check.
func devCompatRun(args []string, stdout, stderr io.Writer) int {
	key := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		key, args = args[0], args[1:]
	}
	fs := flag.NewFlagSet("dev compat run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	work := fs.String("work", "", "directory the work copies are laid out in")
	addons := fs.String("addons", "", "directory holding one folder per addon of the entry, named like the addon")
	jsonPath := fs.String("json", "", "also write the report as JSON to this file")
	describe := fs.Bool("describe", false, "print the runtime and addon sources of the entry as JSON instead of running it")
	entryJSON := fs.String("entry-json", "", "run the entry in this JSON file, which need not be in the list, instead of a key")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	var probe *compat.Entry
	switch {
	case (key == "") == (*entryJSON == ""):
		fmt.Fprintln(stderr, "loomux dev compat run: name the entry key before the flags, or give --entry-json")
		return 2
	case fs.NArg() > 0:
		fmt.Fprintf(stderr, "loomux dev compat run: unexpected argument %q\n", fs.Arg(0))
		return 2
	case key != "" && !knownEntry(key):
		fmt.Fprintf(stderr, "loomux dev compat run: unknown entry %q\n", key)
		return 2
	}
	if *entryJSON != "" {
		e, err := compat.LoadEntry(*entryJSON)
		if err != nil {
			fmt.Fprintf(stderr, "loomux dev compat run: %v\n", err)
			return 2
		}
		probe, key = &e, e.Key()
	}
	switch {
	case *describe:
		var d compat.Description
		var err error
		if probe != nil {
			// A framework of the stack may still be no addon CI can install.
			d, err = compat.DescribeEntry(*probe)
		} else {
			d, _ = compat.Describe(key) // the key is known, and the list is validated by the tests
		}
		if err != nil {
			fmt.Fprintf(stderr, "loomux dev compat run: %v\n", err)
			return 2
		}
		data, _ := json.Marshal(d) // plain strings always encode
		fmt.Fprintf(stdout, "%s\n", data)
		return 0
	case *work == "":
		fmt.Fprintln(stderr, "loomux dev compat run: --work is required")
		return 2
	}
	rep, err := compatRun(compat.RunOptions{
		Key: key, Entry: probe, Work: *work, Addons: *addons, Rescan: compatRescan,
		Check: func(root string, kinds []string) ([]compat.LaneResult, error) { return runCheck(root, kinds, stderr) },
	})
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev compat run: %v\n", err)
		return 1
	}
	fmt.Fprint(stdout, rep.Markdown())
	if *jsonPath != "" {
		data, _ := json.MarshalIndent(rep, "", "  ") // plain strings always encode
		if err := os.WriteFile(*jsonPath, append(data, '\n'), 0o644); err != nil {
			fmt.Fprintf(stderr, "loomux dev compat run: %v\n", err)
			return 1
		}
	}
	if !rep.Green {
		return 1
	}
	return 0
}

// compatRescan imports a work copy with the engine {godot} finds for it. The
// copy is no git checkout, so the engine's other changes are not asked for.
func compatRescan(root string) error {
	bin, finding, err := godotBinary("", root, root)
	if err == nil && finding != "" {
		err = errors.New(finding)
	}
	if err == nil {
		_, err = godotRescan(godotsuite.RescanOptions{Root: root, Godot: bin, Start: checkStart,
			Status: func(string) (string, error) { return "", nil }})
	}
	return err
}

func devCompatMatrix(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev compat matrix", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var sel compat.Selection
	var keys listFlag
	fs.BoolVar(&sel.Full, "full", false, "select every entry")
	fs.Var(&keys, "entry", "select this entry key (repeatable)")
	since := fs.String("changed-since", "", "select what changed since this git ref")
	probes := fs.String("probes", "", "select these probe entries (a JSON list, as scout writes them) and nothing else")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *probes != "" {
		es, err := compat.LoadProbes(*probes)
		if err != nil {
			fmt.Fprintf(stderr, "loomux dev compat matrix: %v\n", err)
			return 1
		}
		out, _ := json.Marshal(struct {
			Include []compat.Job `json:"include"`
		}{compat.ProbeMatrix(es)}) // plain strings always encode
		fmt.Fprintf(stdout, "%s\n", out)
		return 0
	}
	sel.Keys = keys
	for _, k := range keys {
		if !knownEntry(k) {
			fmt.Fprintf(stderr, "loomux dev compat matrix: unknown entry %q\n", k)
			return 2
		}
	}
	if *since != "" {
		var err error
		if sel.ChangedFixtures, sel.ChangedEntries, err = compat.ChangedSince(".", *since); err != nil {
			fmt.Fprintf(stderr, "loomux dev compat matrix: %v\n", err)
			return 1
		}
	}
	entries, _ := compat.Load()
	out, _ := json.Marshal(struct {
		Include []compat.Job `json:"include"`
	}{compat.Matrix(entries, sel)}) // plain strings always encode
	fmt.Fprintf(stdout, "%s\n", out)
	return 0
}

// compatUpstream is the real release sources; a test stands in for them.
var compatUpstream = func() compat.Upstream { return compat.HTTPUpstream{Token: os.Getenv("GITHUB_TOKEN")} }

// devCompatScout asks the upstream sources what the list lacks. With
// --resolved it answers instead whether the list now holds the version an
// issue title announces.
func devCompatScout(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev compat scout", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonPath := fs.String("json", "", "also write the findings as JSON to this file")
	resolved := fs.String("resolved", "", "print resolved or open for this issue title instead of scouting")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "loomux dev compat scout: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	entries, _ := compat.Load()
	if *resolved != "" {
		if compat.Resolved(entries, *resolved) {
			fmt.Fprintln(stdout, "resolved")
		} else {
			fmt.Fprintln(stdout, "open")
		}
		return 0
	}
	findings, err := compat.Scout(entries, compatUpstream())
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev compat scout: %v\n", err)
		return 1
	}
	if findings == nil {
		findings = []compat.Finding{} // encodes as an empty array
	}
	for _, f := range findings {
		fmt.Fprintln(stdout, compat.IssueTitle(f))
	}
	if len(findings) == 0 {
		fmt.Fprintln(stdout, "nothing new")
	}
	if *jsonPath != "" {
		data, _ := json.MarshalIndent(findings, "", "  ") // plain strings always encode
		if err := os.WriteFile(*jsonPath, append(data, '\n'), 0o644); err != nil {
			fmt.Fprintf(stderr, "loomux dev compat scout: %v\n", err)
			return 1
		}
	}
	return 0
}
