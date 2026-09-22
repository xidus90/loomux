package cli

import (
	"flag"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// parseInterspersed parses flags wherever they stand among the positional
// arguments and answers the positionals in order, as argparse reads a command
// line: `case --package <id>` and `case <id> --package` are one command.
//
// Go's `flag` stops at the first argument that is not a flag, so the parse is
// resumed after every positional. A flag's value is consumed by the parse
// itself and never mistaken for a positional.
func parseInterspersed(flags *flag.FlagSet, args []string) ([]string, error) {
	if err := refuseOptionValue(flags, args); err != nil {
		return nil, err
	}
	var free []string
	for {
		if err := flags.Parse(args); err != nil {
			return nil, err
		}
		rest := flags.Args()
		if len(rest) == 0 {
			return free, nil
		}
		// `flag` swallows a `--` and stops there, so the argument it stopped
		// at may be the first after the terminator. Resuming past it would
		// read a later `--package` as a flag again and make the terminator
		// mean its opposite; past it everything is positional, which is also
		// the one way to name a case that looks like a flag.
		if at := len(args) - len(rest) - 1; at >= 0 && args[at] == "--" {
			return append(free, rest...), nil
		}
		free = append(free, rest[0])
		args = rest[1:]
	}
}

// refuseOptionValue turns away `--amend --` and `--amend --reject` before
// `flag` reads them.
//
// argparse never takes what looks like an option as a flag's value, it
// answers "expected one argument"; `flag` would take it and hand it on as a
// path. So the command line is walked the way the parse will walk it: a flag
// that wants a value consumes the next argument, and the first `--` nothing
// consumed is the terminator, past which nothing is a flag. `--amend=--`
// names its value outright and argparse takes it too.
func refuseOptionValue(flags *flag.FlagSet, args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return nil
		}
		name := strings.TrimLeft(arg, "-")
		if name == arg || strings.Contains(name, "=") || !wantsValue(flags, name) {
			continue
		}
		if i+1 < len(args) && looksLikeOption(args[i+1]) {
			err := fmt.Errorf("argument --%s: expected one argument", name)
			fmt.Fprintf(flags.Output(), "%s: %v\n", flags.Name(), err)
			return err
		}
		i++
	}
	return nil
}

// wantsValue says whether the flag of that name takes the next argument as
// its value: a known flag that is not a switch.
func wantsValue(flags *flag.FlagSet, name string) bool {
	known := flags.Lookup(name)
	if known == nil {
		return false
	}
	switched, ok := known.Value.(interface{ IsBoolFlag() bool })
	return !ok || !switched.IsBoolFlag()
}

// looksLikeOption is argparse's `_parse_optional` answering "optional" for an
// argument it met where a value was due: it starts with `-`, and it is not a
// lone `-`, a number-like word or a word with a space in it -- those three
// argparse takes as values.
func looksLikeOption(arg string) bool {
	if len(arg) < 2 || arg[0] != '-' || strings.Contains(arg, " ") {
		return false
	}
	return !numberLike(arg[1:])
}

// numberLike is what follows the `-` of Python 3.14's
// `_negative_number_matcher`, `^-\.?\d` matched as a prefix: a digit, or a
// point and a digit, whatever comes after. So `-1e5`, `-5x.md` and
// `-2026-notes.md` are values, `-inf` and `-.x` are options. Python's `\d` is
// every Unicode decimal digit, which unicode.IsDigit is.
func numberLike(rest string) bool {
	rest = strings.TrimPrefix(rest, ".")
	first, _ := utf8.DecodeRuneInString(rest)
	return unicode.IsDigit(first)
}
