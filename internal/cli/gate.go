package cli

import (
	"cmp"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/hooks"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/verify"
)

const gateUsage = "usage: loomux gate status [--root <dir>]\n" +
	"       loomux gate arm <lane>... [--root <dir>]\n" +
	"       loomux gate disarm <lane>...|--all [--root <dir>]"

// gateWrite is the seam for a file that cannot be written after it was read,
// which no world provokes: what stands in the file's way stops the read first.
var gateWrite = verify.WriteArmed

// gateCommand is `loomux gate`: what a human says about which lanes fail the
// gate. A group of its own and not under check, where every first word is a
// profile or a kind. status reads; arm and disarm write .loomux/armed.toml,
// which the guard keeps from agents.
func gateCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	usage := func() int {
		fmt.Fprintln(stderr, gateUsage)
		return 2
	}
	if len(args) == 0 || !slices.Contains([]string{"status", "arm", "disarm"}, args[0]) {
		return usage()
	}
	sub := args[0]
	flags := flag.NewFlagSet("loomux gate "+sub, flag.ContinueOnError)
	flags.SetOutput(stderr)
	rootFlag := flags.String("root", "", "path to the project root; found upwards when empty")
	all := new(bool)
	if sub == "disarm" {
		all = flags.Bool("all", false, "put every lane into probation: write the file without an entry")
	}
	keys, err := parseInterspersed(flags, args[1:])
	if err != nil {
		return 2
	}
	// A key spelled with Windows separators names the lane LaneKey writes
	// with forward ones; no part of a key holds a backslash of its own.
	for i, key := range keys {
		keys[i] = strings.ReplaceAll(key, `\`, "/")
	}
	root := *rootFlag
	if root == "" {
		root = gateRoot(".")
	}
	if abs, err := filepath.Abs(cmp.Or(root, ".")); err == nil {
		root = abs
	}
	fail := func(err error) int {
		fmt.Fprintf(stderr, "loomux gate %s: %v\n", sub, err)
		return 1
	}
	switch {
	case sub == "status" && len(keys) == 0:
		return gateStatus(root, stdout, stderr, fail)
	case sub == "arm" && len(keys) > 0:
		return gateArm(root, keys, stdout, fail)
	case sub == "disarm" && *all && len(keys) == 0:
		if err := gateWrite(root, verify.ArmedSet{Exists: true}); err != nil {
			return fail(err)
		}
		fmt.Fprintln(stdout, "probation: every lane")
		return 0
	case sub == "disarm" && !*all && len(keys) > 0:
		return gateDisarm(root, keys, stdout, fail)
	}
	return usage()
}

// gateRoot is the project gate works on when no --root names it: the
// directory of the configuration as check finds it, else the nearest one
// holding the armed lanes, else the repository's top level, else "" for the
// directory gate runs in. A project in probation may have no configuration --
// a Go project runs on the presets -- and from one of its subdirectories gate
// must still write the file the gate reads, not a second one beside it.
func gateRoot(start string) string {
	if root, err := hosts.FindRoot(start); err == nil {
		return root
	}
	// Outside a repository nothing says how far up the project reaches: a
	// file above may be anybody's.
	top, err := gitwork.TopLevel(start)
	if err != nil {
		return ""
	}
	// The search walks git's own spelling, the top and the directories below
	// it down to start, and reads the top as well: a file above it belongs to
	// the project of an outer repository, and a working directory reached
	// through a junction has parents outside the repository. A prefix git
	// cannot name leaves the top alone to look at.
	prefix, _ := gitwork.Prefix(start)
	parts := strings.FieldsFunc(prefix, func(r rune) bool { return r == '/' })
	for n := len(parts); n >= 0; n-- {
		dir := filepath.Join(append([]string{top}, parts[:n]...)...)
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(verify.ArmedFile))); err == nil {
			return dir
		}
	}
	return top
}

const noArmedFile = "no " + verify.ArmedFile + ": every lane is armed"

// gateStatus prints every lane and every entry without a lane, by key.
func gateStatus(root string, stdout, stderr io.Writer, fail func(error) int) int {
	states, exists, err := hooks.ReadLaneStates(root)
	if err != nil {
		return fail(err)
	}
	if !exists {
		fmt.Fprintln(stdout, noArmedFile)
		return 0
	}
	said := map[string]string{}
	for state, keys := range map[string][]string{"armed": states.Armed, "probation": states.Probation, "orphan": states.Orphans} {
		for _, key := range keys {
			said[key] = state
		}
	}
	for _, key := range slices.Sorted(maps.Keys(said)) {
		fmt.Fprintf(stdout, "%s: %s\n", key, said[key])
	}
	if states.Ignored {
		// A warning, not a refusal: the list above holds, on this machine.
		fmt.Fprintf(stderr, "loomux gate status: %s is ignored by git: it reaches no commit and holds on this machine only\n", verify.ArmedFile)
	}
	return 0
}

// unknownLane is the refusal for the first key no lane answers to.
func unknownLane(keys, lanes []string) error {
	for _, key := range keys {
		if !slices.Contains(lanes, key) {
			return fmt.Errorf("no lane %q; lanes: %s", key, strings.Join(lanes, ", "))
		}
	}
	return nil
}

// gateArm enters lanes whatever their state: from now on they count. A key
// no lane answers to refuses the whole call.
func gateArm(root string, keys []string, stdout io.Writer, fail func(error) int) int {
	armed, err := verify.ReadArmed(root)
	var lanes []string
	if err == nil {
		lanes, err = hooks.GateLanes(root)
	}
	// A key no lane answers to is a mistake before anything else, with the
	// file and without it: a typo must not read as "every lane is armed".
	if err == nil {
		err = unknownLane(keys, lanes)
	}
	if err != nil {
		return fail(err)
	}
	if !armed.Exists {
		// Every lane is armed already; a file naming only these would put
		// every other lane into probation.
		fmt.Fprintln(stdout, noArmedFile)
		return 0
	}
	if err := gateWrite(root, armed.With(keys...)); err != nil {
		return fail(err)
	}
	fmt.Fprintf(stdout, "armed: %s\n", strings.Join(verify.ArmedSet{}.With(keys...).Keys, ", "))
	return 0
}

// gateDisarm takes entries out, an orphaned one among them. Without the file
// every lane is armed, so taking one out leaves a file that names every
// other lane; only there must the key be a lane.
func gateDisarm(root string, keys []string, stdout io.Writer, fail func(error) int) int {
	armed, err := verify.ReadArmed(root)
	if err != nil {
		return fail(err)
	}
	if !armed.Exists {
		lanes, err := hooks.GateLanes(root)
		if err == nil {
			err = unknownLane(keys, lanes)
		}
		if err != nil {
			return fail(err)
		}
		armed = verify.ArmedSet{}.With(lanes...)
	}
	var gone, absent []string
	for _, key := range (verify.ArmedSet{}).With(keys...).Keys {
		if slices.Contains(armed.Keys, key) {
			gone = append(gone, key)
		} else {
			absent = append(absent, key)
		}
	}
	if err := gateWrite(root, armed.Without(keys...)); err != nil {
		return fail(err)
	}
	if len(gone) > 0 {
		fmt.Fprintf(stdout, "probation: %s\n", strings.Join(gone, ", "))
	}
	for _, key := range absent {
		fmt.Fprintf(stdout, "%s: was not armed\n", key)
	}
	return 0
}
