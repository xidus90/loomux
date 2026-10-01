package hooks

import (
	"encoding/base64"
	"encoding/binary"
	"slices"
	"strings"
	"unicode/utf16"
)

// stringShells are the programs innerLine reads a string of; cmd /c and
// env -S are prefixes dropPrefixes strips.
var stringShells = []string{"sh", "bash", "zsh", "dash", "pwsh", "powershell", "eval", "iex", "invoke-expression"}

// innerLine is the command line a shell among args runs from a string: the
// word after -c of sh, bash, zsh or dash (also in a bundle, -lc, and behind
// -o or -O with its value), every word after -c or -Command of pwsh or
// powershell, abbreviated or not, the string of -EncodedCommand decoded, the
// words after /c or /k of cmd (also glued, /c"…"), and the value of env -S
// or --split-string with the words after it; and the words after eval, iex
// or Invoke-Expression, joined, when one of them is the program. after are
// the words that follow the string of sh -c, its $0, $1, ….
func innerLine(args []string) (line string, after []string, ok bool) {
	if program := dropPrefixes(args); len(program) > 0 {
		switch verbOf(program[0]) {
		case "eval":
			return strings.Join(program[1:], " "), nil, true
		case "iex", "invoke-expression":
			rest := program[1:]
			if len(rest) > 0 && isParameter(rest[0], "command") {
				rest = rest[1:]
			}
			return strings.Join(rest, " "), nil, len(rest) > 0
		}
	}
	for i, a := range args {
		var found bool
		switch verbOf(a) {
		case "sh", "bash", "zsh", "dash":
			line, after, found = posixString(args[i+1:])
		case "pwsh", "powershell":
			line, found = powerShellString(args[i+1:])
		case "cmd":
			line, found = cmdString(args[i+1:])
		case "env":
			line, found = splitString(args[i+1:])
		}
		if found {
			return line, after, true
		}
	}
	return "", nil, false
}

// posixString is the string of sh -c among a POSIX shell's arguments, and
// the words after it. Options come first: a bundle with c takes the string
// as the next word, one that ends in o or O takes an option name, and a long
// option stands alone.
func posixString(args []string) (line string, after []string, ok bool) {
	for j := 0; j+1 < len(args); j++ {
		a := args[j]
		switch {
		case a == "" || a[0] != '-' && a[0] != '+':
			return "", nil, false
		case !letters(a[1:]):
		case a[0] == '-' && strings.ContainsRune(a, 'c'):
			return args[j+1], args[j+2:], true
		case strings.HasSuffix(a, "o") || strings.HasSuffix(a, "O"):
			j++
		}
	}
	return "", nil, false
}

// powerShellString is the line pwsh or powershell runs from its arguments:
// every word after -Command, or the decoded string of -EncodedCommand.
func powerShellString(args []string) (string, bool) {
	for j, a := range args {
		switch {
		case isParameter(a, "command"):
			return strings.Join(args[j+1:], " "), true
		case isParameter(a, "encodedcommand") && j+1 < len(args):
			if line, ok := decodeCommand(args[j+1]); ok {
				return line, true
			}
		}
	}
	return "", false
}

// isParameter says whether a spells the PowerShell parameter name, in any
// case and abbreviated to at least its first letter.
func isParameter(a, name string) bool {
	f := strings.ToLower(a)
	return len(f) > 1 && strings.HasPrefix("-"+name, f)
}

// decodeCommand is the string of -EncodedCommand: base64 of UTF-16LE.
func decodeCommand(s string) (string, bool) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(b)%2 != 0 {
		return "", false
	}
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(b[2*i:])
	}
	return string(utf16.Decode(units)), true
}

// cmdString is the line cmd runs: the words after /c, /k or /r among its
// leading switches, which may be glued to the switch (/c"…").
func cmdString(args []string) (string, bool) {
	for j, a := range args {
		f := strings.ToLower(a)
		if len(f) < 2 || f[0] != '/' {
			return "", false
		}
		if !strings.HasPrefix(f, "/c") && !strings.HasPrefix(f, "/k") && !strings.HasPrefix(f, "/r") {
			continue
		}
		rest := args[j+1:]
		if glued := a[2:]; glued != "" {
			rest = append([]string{glued}, rest...)
		}
		return strings.Join(rest, " "), true
	}
	return "", false
}

// splitString is the line env -S runs: its value, as a word of its own,
// glued to -S or after --split-string=, and the words after it. env's
// other flags and its VAR=value assignments come before it.
func splitString(args []string) (string, bool) {
	for j, a := range args {
		var value string
		switch {
		case a == "-S" || a == "--split-string":
			if j+1 == len(args) {
				return "", false
			}
			value, j = args[j+1], j+1
		case strings.HasPrefix(a, "--split-string="):
			value = strings.TrimPrefix(a, "--split-string=")
		case strings.HasPrefix(a, "-S"):
			value = a[2:]
		case strings.HasPrefix(a, "-") || strings.Contains(a, "="):
			continue
		default:
			return "", false
		}
		return strings.Join(append([]string{value}, args[j+1:]...), " "), true
	}
	return "", false
}

// pipedLine is the line a shell reads from the segment before its pipe,
// when the shell reads its script from stdin (readsStdin): the words after
// echo or Write-Output, the words after printf's format, or the words of a
// segment that is only a string ('…' | iex).
func pipedLine(prev string, args []string) (string, bool) {
	if !readsStdin(args) {
		return "", false
	}
	prev = strings.TrimSpace(prev)
	words := tolerantWords(prev)
	if len(words) == 0 {
		return "", false
	}
	switch verbOf(words[0]) {
	case "echo", "write-output":
		rest := words[1:]
		for len(rest) > 0 && strings.HasPrefix(rest[0], "-") {
			rest = rest[1:]
		}
		return strings.Join(rest, " "), len(rest) > 0
	case "printf":
		if len(words) > 2 {
			return strings.Join(words[2:], " "), true
		}
		return strings.Join(words[1:], " "), len(words) > 1
	}
	if prev[0] == '\'' || prev[0] == '"' {
		return strings.Join(words, " "), true
	}
	return "", false
}

// readsStdin says whether a program reads the commands it runs from stdin:
// a POSIX shell with options only (-s, and the value of -o or -O, allowed)
// or with -s before its positional words; pwsh or powershell with -Command -
// or -File -, or without either; cmd without /c or /k; iex or
// Invoke-Expression without an argument.
func readsStdin(args []string) bool {
	program := args
	// dropPrefixes takes cmd and its switches for a wrapper of what follows
	// /c; without /c there is nothing to follow.
	if len(args) == 0 || verbOf(args[0]) != "cmd" {
		program = dropPrefixes(args)
	}
	if len(program) == 0 {
		return false
	}
	rest := program[1:]
	switch verbOf(program[0]) {
	case "sh", "bash", "zsh", "dash":
		for j := 0; j < len(rest); j++ {
			a := rest[j]
			switch {
			case a == "" || a[0] != '-' && a[0] != '+':
				return false
			case a == "-s":
				return true
			case !letters(a[1:]):
			// -c takes the next word for its string, which ends the options.
			case strings.HasSuffix(a, "o") || strings.HasSuffix(a, "O"):
				j++
			}
		}
		return true
	case "pwsh", "powershell":
		for j, a := range rest {
			switch {
			case isParameter(a, "command"):
				return j+1 < len(rest) && rest[j+1] == "-"
			case !strings.HasPrefix(a, "-"):
				// A script after -File or as the first word; -File - is
				// stdin, and - is read as a flag.
				return false
			}
		}
		return true
	case "cmd":
		_, runs := cmdString(rest)
		return !runs
	case "iex", "invoke-expression":
		return len(rest) == 0
	}
	return false
}

// fedLines are the lines one reading of a segment hands a shell on its
// stdin: a here-string, and when the segment follows a pipe, what the
// segment before it prints. A > a reading masked inside quotes is a > again.
func fedLines(prev string, words []string, piped bool) []string {
	var out []string
	if line, ok := hereString(words); ok {
		out = append(out, strings.ReplaceAll(line, quotedRedirect, ">"))
	}
	if line, ok := pipedLine(prev, words); ok && piped {
		out = append(out, line)
	}
	return out
}

// hereString is the line a shell reads from a here-string (bash <<< '…'):
// the word after <<<, or the text glued to it, when the program without it
// reads its commands from stdin.
func hereString(args []string) (string, bool) {
	for i, a := range args {
		value, found := strings.CutPrefix(a, "<<<")
		if !found {
			continue
		}
		rest := slices.Concat(args[:i], args[i+1:])
		if value == "" {
			if i+1 == len(args) {
				return "", false
			}
			value, rest = args[i+1], slices.Concat(args[:i], args[i+2:])
		}
		return value, readsStdin(rest)
	}
	return "", false
}
