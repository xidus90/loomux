package hooks

import (
	"slices"
	"strings"
)

// stdinReason refuses an interpreter that would wait on stdin for its
// program, or get it as inline text a shell may have rewritten.
const stdinReason = "loomux refuses an interpreter that reads its program from stdin (`python -`, a bare `python`, `uv run -`, " +
	"a heredoc or here-string into one): with nothing piped in it waits until it is stopped, and a shell may rewrite " +
	"the backslashes of inline program text. Write the script to a file in the scratchpad with the host's file tool " +
	"(Write, write_to_file) and run that file."

// interpreter is how one interpreter reads its arguments, from its manual
// and measured with Python 3.13, Node 24 and Perl 5.42.
type interpreter struct {
	program []string // flags that carry the program on the line
	values  []string // flags whose value is the next word, or the rest of a bundle
	info    []string // whole words after which it reads no program from stdin: it prints and exits, or finds its files itself
}

var python = interpreter{
	program: []string{"-c", "-m"},
	values:  []string{"-W", "-X", "--check-hash-based-pycs"},
	info:    []string{"-V", "-VV", "--version", "-h", "-?", "--help", "--help-env", "--help-xoptions", "--help-all", "-0", "-0p", "--list", "--list-paths"},
}

// interpreters are keyed by verbOf of the program; python3.12 and its kin
// are looked up as python (interpreterOf).
var interpreters = map[string]interpreter{
	"python": python,
	"py":     python,
	"node": {
		program: []string{"-e", "--eval", "-p", "--print"},
		values:  []string{"-r", "--require", "--import", "--loader", "--experimental-loader", "-C", "--conditions", "--input-type"},
		info:    []string{"-v", "--version", "-h", "--help", "--v8-options", "--test"},
	},
	// -c checks the syntax of a program it reads like any other: from a
	// file, or from stdin.
	"perl": {program: []string{"-e", "-E"}, values: []string{"-I", "-M", "-m"}, info: []string{"-v", "-V", "-h", "--version", "--help"}},
	// -E takes an encoding, not a program.
	"ruby": {program: []string{"-e"}, values: []string{"-r", "-I", "-C", "-E"}, info: []string{"-v", "--version", "-h", "--help"}},
}

// uvValues are the flags of uv run and uvx whose value is the next word.
var uvValues = []string{"--with", "--with-editable", "--with-requirements", "--python", "-p", "--project",
	"--directory", "--package", "--extra", "--group", "--env-file", "--index", "--from"}

// readsProgramFromStdin says whether line, or a line it runs from a string,
// starts an interpreter whose program comes from stdin with nothing real on
// it: no file and no pipe from a command that prints.
func readsProgramFromStdin(line string) bool {
	return anyRunLine(withoutHereBodies(line), func(l string, nested bool) bool {
		// The outer line has lost its bodies already; a second pass would
		// take the lines after a heredoc's head for a body.
		if nested {
			l = withoutHereBodies(l)
		}
		return lineReadsStdin(l)
	})
}

// lineReadsStdin is readsProgramFromStdin for one line without the lines it
// runs from strings. It cuts only outside quotes: a quoted ; is data, and
// this rule keeps a task from hanging rather than an agent from a path.
func lineReadsStdin(line string) bool {
	for _, variant := range lineVariants(line) {
		left, at, depth, braces := "", 0, 0, bracesPair(variant)
		for _, segment := range splitSegments(variant, true) {
			// Text inline on the line is no stdin, piped or not, and neither
			// is a pipe whose left side holds it. The second bar of || pipes
			// nothing, inside a group as well.
			inline := strings.Contains(segment, "<<")
			piped := at > 1 && variant[at-1] == '|' && variant[at-2] != '|' &&
				strings.TrimSpace(left) != "" && !strings.Contains(left, "<<")
			at += len(segment) + 1
			left, depth = pipeLeft(variant, at, segment, left, depth, braces)
			for _, words := range readings(segment) {
				if programFromStdin(words) && (inline || !piped && !fromFile(words)) {
					return true
				}
			}
		}
	}
	return false
}

// pipeLeft carries the left side of a pipe past segment, whose break sits
// just before at: a ;, &, line break or the second bar of || ends it while
// no parenthesis or brace is open. The left side of a bar is so the whole
// pipeline element in front of it, a subshell, a group or a $( ) included;
// depth counts what is open, and a stray closer counts as none. Braces
// count only when the line's braces pair up (bracesPair).
func pipeLeft(variant string, at int, segment, left string, depth int, braces bool) (string, int) {
	for _, w := range strings.Fields(segment) {
		switch {
		case braces && w == "{":
			depth++
		case braces && w == "}":
			depth--
		}
	}
	if at > len(variant) {
		return left, depth
	}
	switch variant[at-1] {
	case '(':
		depth++
	case ')':
		depth--
	}
	ends := strings.IndexByte(";&\n", variant[at-1]) >= 0 || variant[at-1] == '|' && at > 1 && variant[at-2] == '|'
	if ends && depth <= 0 {
		return "", 0
	}
	return left + segment + " ", depth
}

// bracesPair says whether the lone { and } words of variant pair up, each
// closer after its opener. A glued closer ({ $_.Name}) or a quoted brace
// (' { ') leaves them unpaired, and counting them then would hold a group
// open to the end of the line.
func bracesPair(variant string) bool {
	open := 0
	for _, w := range strings.Fields(variant) {
		switch w {
		case "{":
			open++
		case "}":
			open--
		}
		if open < 0 {
			return false
		}
	}
	return open == 0
}

// fromFile says whether words redirect stdin with < or 0<. A segment with a
// heredoc or here-string never gets here as fed (lineReadsStdin).
func fromFile(words []string) bool {
	return slices.ContainsFunc(words, func(w string) bool {
		return strings.HasPrefix(strings.TrimPrefix(w, "0"), "<")
	})
}

// programFromStdin says whether words, a reading of one segment, run an
// interpreter that takes its program from stdin: directly, behind uv run or
// uvx, or as uv run - itself.
func programFromStdin(words []string) bool {
	if describesCommand(words) {
		return false
	}
	program := dropPrefixes(words)
	if len(program) == 0 {
		return false
	}
	if args, ok := uvRunArgs(program); ok {
		if len(args) == 0 {
			// uv run without a command fails at once.
			return false
		}
		if args[0] == "-" {
			return true
		}
		program = args
	}
	in, ok := interpreterOf(program[0])
	return ok && in.fromStdin(program[1:])
}

// describesCommand says whether words run command -v or -V, which name a
// program and run none; dropPrefixes would strip them and leave the program
// bare. Assignments and the reserved words of a condition (if, then, !) may
// stand in front.
func describesCommand(words []string) bool {
	for len(words) > 0 && (strings.Contains(words[0], "=") && !strings.HasPrefix(words[0], "-") ||
		slices.Contains(shellWords, words[0]) && words[0] != "command") {
		words = words[1:]
	}
	return len(words) > 1 && words[0] == "command" && strings.HasPrefix(words[1], "-") && strings.ContainsAny(words[1], "vV")
}

// uvRunArgs are the words after uv run or uvx and their flags: the command
// they run, or - for a script on stdin.
func uvRunArgs(words []string) ([]string, bool) {
	var rest []string
	switch {
	case verbOf(words[0]) == "uvx":
		rest = words[1:]
	case verbOf(words[0]) == "uv" && len(words) > 1 && words[1] == "run":
		rest = words[2:]
	default:
		return nil, false
	}
	for len(rest) > 0 && len(rest[0]) > 1 && rest[0][0] == '-' {
		n := 1
		if slices.Contains(uvValues, rest[0]) {
			n = 2
		}
		rest = rest[min(n, len(rest)):]
	}
	return rest, true
}

// interpreterOf is the interpreter word names by its base name without
// .exe; python followed by a version, also a free-threaded one (python3,
// python3.12, python3.14t), is python.
func interpreterOf(word string) (interpreter, bool) {
	name := verbOf(word)
	if version, ok := strings.CutPrefix(name, "python"); ok && strings.Trim(version, "0123456789.t") == "" {
		name = "python"
	}
	in, ok := interpreters[name]
	return in, ok
}

// fromStdin says whether the interpreter, given args, reads its program from
// stdin: a lone - as its first positional, or no positional at all. A
// program flag or an info word means it does not; a redirection is no
// argument.
func (in interpreter) fromStdin(args []string) bool {
	flags := true
	for i := 0; i < len(args); i++ {
		a := args[i]
		if isRedirect, bare := redirection(a); isRedirect {
			if bare {
				i++
			}
			continue
		}
		switch {
		case flags && a == "--":
			flags = false
		case flags && slices.Contains(in.info, a):
			return false
		case flags && strings.HasPrefix(a, "--"):
			name, _, glued := strings.Cut(a, "=")
			if slices.Contains(in.program, name) {
				return false
			}
			if !glued && slices.Contains(in.values, name) {
				i++
			}
		case flags && len(a) > 1 && a[0] == '-':
			program, next := in.bundle(a)
			if program {
				return false
			}
			if next {
				i++
			}
		default:
			return a == "-"
		}
	}
	return true
}

// bundle reads a short flag word letter by letter (-uc, -Wignore, -lne):
// program says a letter carries the program, next that the last letter
// takes the next word as its value. A value letter before the last takes
// the rest of the word.
func (in interpreter) bundle(word string) (program, next bool) {
	for j := 1; j < len(word); j++ {
		letter := "-" + word[j:j+1]
		if slices.Contains(in.program, letter) {
			return true, false
		}
		if slices.Contains(in.values, letter) {
			return false, j == len(word)-1
		}
	}
	return false, false
}

// hereDoc is the end word of one heredoc a line opens, and whether it was
// opened with <<-, which strips leading tabs from the lines.
type hereDoc struct {
	word string
	dash bool
}

// withoutHereBodies is line without the bodies of its heredocs and with
// each PowerShell here-string replaced by the word <<@: what they hold is
// data, not commands. A heredoc's head (<<EOF) stays on its line, so that a
// segment still shows it takes inline text.
func withoutHereBodies(line string) string {
	lines := strings.Split(line, "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if head, closing, ok := hereStringHead(l); ok {
			j := i + 1
			for j < len(lines) && !strings.HasPrefix(lines[j], closing) {
				j++
			}
			if j < len(lines) {
				head += lines[j][len(closing):]
			}
			out = append(out, head)
			i = j
			continue
		}
		out = append(out, l)
		for _, doc := range hereDocs(l) {
			for i+1 < len(lines) {
				i++
				body := strings.TrimRight(lines[i], "\r")
				if doc.dash {
					body = strings.TrimLeft(body, "\t")
				}
				if body == doc.word {
					break
				}
			}
		}
	}
	return strings.Join(out, "\n")
}

// hereStringHead reads a line that opens a PowerShell here-string: it ends
// in @' or @" after a blank, = or ( or alone. head is the line with the
// opener replaced by <<@, closing the text a line must start with to end it.
func hereStringHead(l string) (head, closing string, ok bool) {
	t := strings.TrimRight(l, " \t\r")
	if !strings.HasSuffix(t, "@'") && !strings.HasSuffix(t, `@"`) {
		return "", "", false
	}
	if len(t) > 2 && !strings.ContainsRune(" \t=(", rune(t[len(t)-3])) {
		return "", "", false
	}
	return t[:len(t)-2] + "<<@", t[len(t)-1:] + "@", true
}

// hereDocs are the heredocs l opens, in order: << or <<- outside quotes,
// with the end word unquoted. The third < of a here-string's <<< ends the
// word before it starts, so <<< opens none.
func hereDocs(l string) []hereDoc {
	var out []hereDoc
	var quote byte
	for i := 0; i < len(l); i++ {
		c := l[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case strings.HasPrefix(l[i:], "<<"):
			rest, dash := strings.CutPrefix(l[i+2:], "-")
			rest = strings.TrimLeft(rest, " \t")
			end := strings.IndexAny(rest, " \t\r;|&<>()")
			if end < 0 {
				end = len(rest)
			}
			if word := strings.NewReplacer(`'`, "", `"`, "", `\`, "").Replace(rest[:end]); word != "" {
				out = append(out, hereDoc{word: word, dash: dash})
			}
			i++
		}
	}
	return out
}
