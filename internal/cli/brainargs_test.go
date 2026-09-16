package cli

import (
	"math"
	"reflect"
	"strconv"
	"testing"
)

func brainParserNamed(t *testing.T, name string) brainParser {
	t.Helper()
	p, ok := brainParserFor(name)
	if !ok {
		t.Fatalf("no parser for %q", name)
	}
	return p
}

func TestBrainParserForKnowsTheFiveSubcommands(t *testing.T) {
	for _, name := range brainSubcommands {
		if p, ok := brainParserFor(name); !ok || p.name != name {
			t.Fatalf("%s: %v %+v", name, ok, p)
		}
	}
	if _, ok := brainParserFor("reindex"); ok {
		t.Fatal("reindex is not a loomux brain subcommand")
	}
}

func TestBrainUsageLines(t *testing.T) {
	want := map[string]string{
		"search":    "usage: loomux brain search [--scope SCOPE] [--profile {fast,full,keyword}] [-n N] [--channel {local,cloud}] query",
		"catalog":   "usage: loomux brain catalog [--scope SCOPE] [--channel {local,cloud}]",
		"read":      "usage: loomux brain read --scope SCOPE [--section SECTION] [--channel {local,cloud}] relative",
		"neighbors": "usage: loomux brain neighbors --scope SCOPE [--channel {local,cloud}] relative",
		"status":    "usage: loomux brain status [--channel {local,cloud}]",
	}
	for name, line := range want {
		if got := brainParserNamed(t, name).usage(); got != line {
			t.Fatalf("%s:\n got %q\nwant %q", name, got, line)
		}
	}
	if got := brainTopUsage(); got != "usage: loomux brain {search,catalog,read,neighbors,status} ..." {
		t.Fatalf("top usage %q", got)
	}
}

// Every row was measured against the reference: _build_parser().parse_args()
// of ub cli.py under Python 3.14.7.
func TestBrainParseAcceptsWhatArgparseAccepts(t *testing.T) {
	search := func(query, scope, profile string, n int, channel string) brainArgs {
		return brainArgs{positional: query, n: n, values: map[string]string{
			"--scope": scope, "--profile": profile, "-n": strconv.Itoa(n), "--channel": channel}}
	}
	for _, c := range []struct {
		sub  string
		args []string
		want brainArgs
	}{
		{"search", []string{"q"}, search("q", "all", "fast", 5, "local")},
		{"search", []string{"--scope", "a", "q", "--profile=full", "-n3", "--channel=cloud"}, search("q", "a", "full", 3, "cloud")},
		{"search", []string{"q", "-n=3"}, search("q", "all", "fast", 3, "local")},
		{"search", []string{"q", "-n", " 3 "}, search("q", "all", "fast", 3, "local")},
		{"search", []string{"q", "-n", "+3"}, search("q", "all", "fast", 3, "local")},
		{"search", []string{"q", "-n", "3_0"}, search("q", "all", "fast", 30, "local")},
		{"search", []string{"q", "-n", "003"}, search("q", "all", "fast", 3, "local")},
		{"search", []string{"q", "-n", "1_000_000"}, search("q", "all", "fast", 1000000, "local")},
		{"search", []string{"q", "-n", "\u00a07\u2003"}, search("q", "all", "fast", 7, "local")},
		{"search", []string{"--", "-x"}, search("-x", "all", "fast", 5, "local")},
		{"search", []string{"-5"}, search("-5", "all", "fast", 5, "local")},
		{"search", []string{"-5x"}, search("-5x", "all", "fast", 5, "local")},
		{"search", []string{"q", "--scope", "-5"}, search("q", "-5", "fast", 5, "local")},
		{"search", []string{"q", "--scope", "-.5"}, search("q", "-.5", "fast", 5, "local")},
		{"search", []string{"q", "--scope", "-"}, search("q", "-", "fast", 5, "local")},
		{"search", []string{"q", "--scope", "-x y"}, search("q", "-x y", "fast", 5, "local")},
		{"search", []string{"-a b"}, search("-a b", "all", "fast", 5, "local")},
		{"search", []string{"--", "--"}, search("--", "all", "fast", 5, "local")},
		{"search", []string{"q", "--scope", "a", "--scope", "b"}, search("q", "b", "fast", 5, "local")},
		{"search", []string{"q", "--scope="}, search("q", "", "fast", 5, "local")},
		{"search", []string{""}, search("", "all", "fast", 5, "local")},
		{"search", []string{"q", "--"}, search("q", "all", "fast", 5, "local")},
		{"search", []string{"--scope", "a", "--", "q"}, search("q", "a", "fast", 5, "local")},
		{"read", []string{"--scope", "s", "rel", "--section", "T"}, brainArgs{positional: "rel", n: 5,
			values: map[string]string{"--scope": "s", "--section": "T", "--channel": "local"}}},
		{"read", []string{"rel", "--scope", "s", "--section", ""}, brainArgs{positional: "rel", n: 5,
			values: map[string]string{"--scope": "s", "--section": "", "--channel": "local"}}},
		{"neighbors", []string{"--scope=s", "rel"}, brainArgs{positional: "rel", n: 5,
			values: map[string]string{"--scope": "s", "--channel": "local"}}},
		{"catalog", nil, brainArgs{n: 5, values: map[string]string{"--scope": "all", "--channel": "local"}}},
		{"catalog", []string{"--scope="}, brainArgs{n: 5, values: map[string]string{"--scope": "", "--channel": "local"}}},
		{"status", []string{"--channel", "cloud"}, brainArgs{n: 5, values: map[string]string{"--channel": "cloud"}}},
	} {
		got, err := brainParserNamed(t, c.sub).parse(c.args)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%s %q:\n got %+v, %v\nwant %+v", c.sub, c.args, got, err, c.want)
		}
	}
}

// Every message was measured against the reference, as above; top marks the
// ones brain-mcp reports from its top-level parser.
func TestBrainParseRefusesWhatArgparseRefuses(t *testing.T) {
	const channelChoice = "(choose from 'local', 'cloud')"
	for _, c := range []struct {
		sub     string
		args    []string
		top     bool
		message string
	}{
		{"search", nil, false, "the following arguments are required: query"},
		{"search", []string{"--bogus"}, false, "the following arguments are required: query"},
		{"search", []string{"--"}, false, "the following arguments are required: query"},
		{"search", []string{"q", "--channel", "x"}, false, "argument --channel: invalid choice: 'x' " + channelChoice},
		{"search", []string{"q", "--channel=x"}, false, "argument --channel: invalid choice: 'x' " + channelChoice},
		{"search", []string{"q", "--channel=cloud=x"}, false, "argument --channel: invalid choice: 'cloud=x' " + channelChoice},
		{"search", []string{"x'y", "--channel", "x'y"}, false, `argument --channel: invalid choice: "x'y" ` + channelChoice},
		{"search", []string{"q", "--profile", "x"}, false, "argument --profile: invalid choice: 'x' (choose from 'fast', 'full', 'keyword')"},
		{"search", []string{"q", "-n", "0"}, false, "argument -n: must be at least 1, got 0"},
		{"search", []string{"q", "-n", "-3"}, false, "argument -n: must be at least 1, got -3"},
		{"search", []string{"q", "-n0"}, false, "argument -n: must be at least 1, got 0"},
		{"search", []string{"q", "-n=0"}, false, "argument -n: must be at least 1, got 0"},
		{"search", []string{"q", "-n", "-0"}, false, "argument -n: must be at least 1, got 0"},
		{"search", []string{"q", "-n", "-99999999999999999999999"}, false, "argument -n: must be at least 1, got -99999999999999999999999"},
		{"search", []string{"q", "-n", "x"}, false, "argument -n: invalid _at_least_one value: 'x'"},
		{"search", []string{"q", "-n", ""}, false, "argument -n: invalid _at_least_one value: ''"},
		{"search", []string{"q", "-n="}, false, "argument -n: invalid _at_least_one value: ''"},
		{"search", []string{"q", "-nx"}, false, "argument -n: invalid _at_least_one value: 'x'"},
		{"search", []string{"q", "-n", "3=4"}, false, "argument -n: invalid _at_least_one value: '3=4'"},
		{"search", []string{"q", "-n3=4"}, false, "argument -n: invalid _at_least_one value: '3=4'"},
		{"search", []string{"q", "-n", "_3"}, false, "argument -n: invalid _at_least_one value: '_3'"},
		{"search", []string{"q", "-n", "3_"}, false, "argument -n: invalid _at_least_one value: '3_'"},
		{"search", []string{"q", "-n", "3__0"}, false, "argument -n: invalid _at_least_one value: '3__0'"},
		{"search", []string{"q", "-n", "3.0"}, false, "argument -n: invalid _at_least_one value: '3.0'"},
		{"search", []string{"q", "-n", "1 2"}, false, "argument -n: invalid _at_least_one value: '1 2'"},
		{"search", []string{"q", "-n", "-"}, false, "argument -n: invalid _at_least_one value: '-'"},
		{"search", []string{"q", "-n"}, false, "argument -n: expected one argument"},
		{"search", []string{"q", "-n", "--"}, false, "argument -n: expected one argument"},
		{"search", []string{"q", "-n", "-x"}, false, "argument -n: expected one argument"},
		{"search", []string{"q", "--scope"}, false, "argument --scope: expected one argument"},
		{"search", []string{"q", "--scope", "--channel", "local"}, false, "argument --scope: expected one argument"},
		{"search", []string{"--scope", "--", "q"}, false, "argument --scope: expected one argument"},
		{"search", []string{"q", "--scope=a", "--scope"}, false, "argument --scope: expected one argument"},
		{"search", []string{"-n", "0", "--channel", "x"}, false, "argument -n: must be at least 1, got 0"},
		{"search", []string{"--channel", "x", "-n", "0"}, false, "argument --channel: invalid choice: 'x' " + channelChoice},
		{"search", []string{"q", "-n", "3", "-n", "0"}, false, "argument -n: must be at least 1, got 0"},
		{"catalog", []string{"--channel", "x"}, false, "argument --channel: invalid choice: 'x' " + channelChoice},
		{"read", nil, false, "the following arguments are required: relative, --scope"},
		{"read", []string{"rel"}, false, "the following arguments are required: --scope"},
		{"read", []string{"--scope", "s"}, false, "the following arguments are required: relative"},
		{"read", []string{"rel", "--scope", "s", "--section"}, false, "argument --section: expected one argument"},
		{"read", []string{"rel", "extra"}, false, "the following arguments are required: --scope"},
		{"read", []string{"--bogus"}, false, "the following arguments are required: relative, --scope"},
		{"read", []string{"--", "rel", "--scope", "s"}, false, "the following arguments are required: --scope"},
		{"neighbors", nil, false, "the following arguments are required: relative, --scope"},
		{"neighbors", []string{"rel"}, false, "the following arguments are required: --scope"},
		{"status", []string{"--channel", "x"}, false, "argument --channel: invalid choice: 'x' " + channelChoice},
		{"status", []string{"--bogus", "--channel", "x"}, false, "argument --channel: invalid choice: 'x' " + channelChoice},
		{"search", []string{"q", "extra"}, true, "unrecognized arguments: extra"},
		{"search", []string{"q", "extra", "more"}, true, "unrecognized arguments: extra more"},
		{"search", []string{"q", "--bogus"}, true, "unrecognized arguments: --bogus"},
		{"search", []string{"--bogus", "x"}, true, "unrecognized arguments: --bogus"},
		{"search", []string{"--bogus=1", "q"}, true, "unrecognized arguments: --bogus=1"},
		{"search", []string{"q", "--", "extra"}, true, "unrecognized arguments: extra"},
		{"search", []string{"q", "--", "--", "x"}, true, "unrecognized arguments: -- x"},
		{"search", []string{"q", "--scope", "s", "--"}, true, "unrecognized arguments: --"},
		{"search", []string{"--", "q", "-n", "3"}, true, "unrecognized arguments: -n 3"},
		{"search", []string{"q", "-x"}, true, "unrecognized arguments: -x"},
		{"search", []string{"q", "-"}, true, "unrecognized arguments: -"},
		{"search", []string{"q", "-.5"}, true, "unrecognized arguments: -.5"},
		{"search", []string{"q", "-5x"}, true, "unrecognized arguments: -5x"},
		{"search", []string{"q", "-n", "3", "extra"}, true, "unrecognized arguments: extra"},
		{"search", []string{"q", "--scope", "s", "extra", "--channel", "cloud"}, true, "unrecognized arguments: extra"},
		{"catalog", []string{"extra"}, true, "unrecognized arguments: extra"},
		{"read", []string{"a", "b", "--scope", "s"}, true, "unrecognized arguments: b"},
		{"neighbors", []string{"rel", "--scope", "s", "--section", "x"}, true, "unrecognized arguments: --section x"},
		{"status", []string{"extra"}, true, "unrecognized arguments: extra"},
		{"status", []string{"--"}, true, "unrecognized arguments: --"},
		{"status", []string{"--", "x"}, true, "unrecognized arguments: -- x"},
		{"status", []string{"--channel=cloud", "--channel", "local", "z"}, true, "unrecognized arguments: z"},
	} {
		_, err := brainParserNamed(t, c.sub).parse(c.args)
		if err == nil || err.top != c.top || err.message != c.message {
			t.Fatalf("%s %q:\n got %+v\nwant top=%v %q", c.sub, c.args, err, c.top, c.message)
		}
	}
}

// Where loomux answers differently from the reference on purpose; each row is
// a line of the parity list.
func TestBrainParseDeviatesFromArgparseWhereTheParityListSays(t *testing.T) {
	// argparse widens --prof to --profile and refuses --s as ambiguous; loomux
	// knows only whole option names.
	for _, c := range []struct {
		args    []string
		message string
	}{
		{[]string{"q", "--prof", "x"}, "unrecognized arguments: --prof x"},
		{[]string{"q", "--prof=full"}, "unrecognized arguments: --prof=full"},
		{[]string{"q", "--s", "x"}, "unrecognized arguments: --s x"},
		{[]string{"q", "-h"}, "unrecognized arguments: -h"},
		{[]string{"q", "--state-dir", "x"}, "unrecognized arguments: --state-dir x"},
	} {
		_, err := brainParserNamed(t, "search").parse(c.args)
		if err == nil || !err.top || err.message != c.message {
			t.Fatalf("%q: got %+v, want %q", c.args, err, c.message)
		}
	}
	// int() accepts every Unicode decimal digit; loomux only ASCII ones.
	for _, digit := range []string{"\u0663", "\uff13"} {
		_, err := brainParserNamed(t, "search").parse([]string{"q", "-n", digit})
		if err == nil || err.message != "argument -n: invalid _at_least_one value: '"+digit+"'" {
			t.Fatalf("%q: got %+v", digit, err)
		}
	}
	// Python keeps a count of any size; loomux holds it at the largest int.
	got, err := brainParserNamed(t, "search").parse([]string{"q", "-n", "99999999999999999999999"})
	if err != nil || got.n != math.MaxInt {
		t.Fatalf("got %+v, %v", got, err)
	}
}
