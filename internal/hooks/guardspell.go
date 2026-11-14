package hooks

import (
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
)

// hidesLoomuxCall says whether a line spells a loomux call that its words do
// not show, read as text because the cut into segments loses it: a first
// argument a shell computes (`$x`, `$(…)`, a backtick, `$'…'`, a brace or a
// glob, a PowerShell subexpression, array or splat) of a loomux that stands
// where a program does, and a program that takes a program and its arguments
// as strings (Process.Start, WMI, the task scheduler), whichever they are.
// A word loomux anywhere else (grep loomux *.go, a path, a commit message) is
// no call, and a launcher that only a reading program such as grep names is
// no launch.
func hidesLoomuxCall(line string) bool {
	if !mentionsLoomux(line) {
		return false
	}
	for _, m := range computedArgument().FindAllStringSubmatchIndex(line, -1) {
		if atCommand(line[:m[2]], strings.Trim(line[m[2]:m[3]], `"'`)) {
			return true
		}
	}
	for _, m := range processLauncher().FindAllStringIndex(line, -1) {
		if !namedByReader(line[:m[0]]) {
			return true
		}
	}
	return false
}

// mentionsLoomux says whether line holds the word loomux other than as a
// folder of a path (~/src/loomux/docs): a program, a package (cmd/loomux/…)
// or an argument. A folder is no program, and a launcher on the same line
// that never names the program is no launch of it.
func mentionsLoomux(line string) bool {
	lower := strings.ToLower(line)
	for from := 0; ; {
		i := strings.Index(lower[from:], "loomux")
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len("loomux")
		from = end
		folder := end+1 < len(lower) && strings.ContainsRune(`/\`, rune(lower[end])) && !strings.ContainsRune(" \t\"'", rune(lower[end+1]))
		if !folder || isMainParent(lower[:start]) || startsRelativePath(lower[:start]) {
			return true
		}
	}
}

// startsRelativePath says whether the loomux after before is the first element
// of a relative path (./loomux/., ../loomux/x): from cmd that is the main
// package, so it is read as one rather than as a folder.
func startsRelativePath(before string) bool {
	return strings.HasSuffix(before, "./") || strings.HasSuffix(before, `.\`)
}

// isMainParent says whether before ends in the cmd folder of loomux's main
// package: the loomux that follows it is the package, whatever follows it.
func isMainParent(before string) bool {
	return strings.HasSuffix(before, "cmd/") || strings.HasSuffix(before, `cmd\`)
}

// maskQuotedPipes is before with the | inside a quoted string replaced, so
// that a pattern such as "a|b" does not cut the segment: the quote makes it a
// character of the string. A double-quoted string that runs a substitution
// ($(…) or a backtick) keeps its pipes, since they are real there. After an
// escaped quote nothing is masked at all: bash and PowerShell disagree on
// where the string ends, and the guard does not pick one.
func maskQuotedPipes(before string) string {
	if strings.ContainsAny(before, "\\`") && escapedQuote().MatchString(before) {
		return before
	}
	out := []byte(before)
	var quote byte
	start := 0
	for i := range out {
		switch c := out[i]; {
		case quote == 0 && (c == '"' || c == '\''):
			quote, start = c, i
		case c == quote:
			quote = 0
		case c == '|' && quote != 0 && (quote == '\'' || !strings.ContainsAny(before[start:i], "`") && !strings.Contains(before[start:i], "$(")):
			out[i] = '_'
		}
	}
	return string(out)
}

// escapedQuote finds a quote that a backslash or a backtick comes before.
var escapedQuote = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile("[\\\\`][\"']") })

// headWords are the words of the segment that ends where before does, read
// like the program's own. A quote the program's path opens belongs to it.
func headWords(before string) []string {
	before = strings.TrimSuffix(strings.TrimSuffix(before, `"`), `'`)
	cut := splitSegments(before, false)
	return tolerantWords(strings.ReplaceAll(cut[len(cut)-1], `\`, "/"))
}

// atCommand says whether program, after the text before it, is what a shell
// runs: nothing but assignments, redirections, reserved words and wrappers
// stands in front of it in its segment, or only go run, for the package.
func atCommand(before, program string) bool {
	words := readPrefixes(append(headWords(before), program)).program
	_, pkg, isRun := goRunCall(words)
	return len(words) == 1 || isRun && pkg == len(words)-1
}

// namedByReader says whether the segment that ends where before does runs a
// program that only reads, so that a launcher after it is an operand: a
// reading verb, or git on a subcommand that shows or records text (git commit
// -m "…"), or gh on a subcommand that writes a title or a body. A flag in
// front of the subcommand (git -c alias.x=!…) is no subcommand. A quoted
// alternation (grep -E "a|b") does not cut the segment (maskQuotedPipes).
func namedByReader(before string) bool {
	program := readPrefixes(headWords(maskQuotedPipes(before))).program
	if len(program) == 0 {
		return false
	}
	if subcommands, ok := subcommandReaders[verbOf(program[0])]; ok {
		return len(program) > 1 && slices.Contains(subcommands, program[1])
	}
	return slices.Contains(readVerbs, verbOf(program[0]))
}

// subcommandReaders are the subcommands, by program, that run nothing from
// their arguments: git's that show or record text, and gh's pull request and
// issue groups, whose title and body are prose.
var subcommandReaders = map[string][]string{
	"git": {"commit", "log", "show", "grep", "diff", "status", "blame", "tag"},
	"gh":  {"pr", "issue"},
}

// computedArgument finds the loomux program, its path in the first group,
// followed by an argument that holds a character a shell or PowerShell
// expands before the program sees it.
var computedArgument = sync.OnceValue(func() *regexp.Regexp {
	const brk = "\\s;|&(){}\"'`"
	// A lone backtick is PowerShell's escape (con`fig); a pair is a bash
	// substitution.
	return regexp.MustCompile(`(?i)(?:^|[` + brk + `])((?:[^` + brk + `]*[/\\])?loomux(?:\.exe)?\.?["']?)[ \t]+` +
		"(?:[^\\s;|&]*[$(@{*?\\[~%]|[^\\s;|&`]*`[^`]*`)")
})

// processLauncher finds the APIs and tools that start a program from strings:
// PowerShell's and .NET's, WMI, the task scheduler, and the process modules of
// Python and Node.
var processLauncher = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)process\]::start|processstartinfo|win32_process|\bwmic\b|\bschtasks\b|scheduledtask|\bsubprocess\b|child_process|\bos\.(?:system|popen|exec\w*|spawn\w*)\b`)
})

// lookedUp finds PowerShell's call operator, or its dot operator (the break
// before it is the first group), on a parenthesis that names loomux
// (`& (gcm loomux)`, `. (gcm loomux)`): the command it yields is loomux.
var lookedUp = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)(?:&|(^|[\s;|&({])\.)\s*\$?\((?:[^()]|\([^()]*\))*\bloomux\b(?:[^()]|\([^()]*\))*\)`)
})

// goWord finds the word go, which is not the end of another (main.go), to put
// a folder behind it.
var goWord = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)(^|[\s"'])(go(?:\.exe)?)(\s|$)`)
})

// computedFolder finds a cd whose folder a shell computes and that ends in
// loomux's main package (`cd "$(git rev-parse --show-toplevel)/cmd/loomux"`):
// whatever stands in front of cmd, the shell lands in the main folder.
var computedFolder = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)\b(?:cd|chdir|sl|set-location|pushd|push-location)\b[^;&|\n]*?\bcmd[/\\]loomux\b[/\\]?["')]*`)
})

// unsafeFolder is what a folder keeps out of the text fromFolder writes back:
// a blank, a quote or a break would cut the line it is read from.
var unsafeFolder = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`[^A-Za-z0-9._/:~+=,-]`)
})

// configurationVariants are the variants of line (lineVariants) and, behind
// them, each variant read where a call operator on a looked-up command is
// loomux, and where go starts in the folder the shell stands in (fromFolder).
func configurationVariants(line, cwd string) []string {
	out := lineVariants(line)
	for _, v := range slices.Clone(out) {
		if r := lookedUp().ReplaceAllString(v, "${1}loomux"); r != v {
			out = append(out, r)
		}
		if f := fromFolder(v, cwd); f != v {
			out = append(out, f)
		}
	}
	return out
}

// fromFolder is line with a -C behind every go: the folder the shell stands in
// at that point, where cwd is, moved by the cds before it on the line, so that
// go run . and go run ./loomux read as the package they run. The folder keeps
// its elements, which is all isMainFolder reads.
func fromFolder(line, cwd string) string {
	base := folderText(cwd)
	line = computedFolder().ReplaceAllString(line, "cd cmd/loomux")
	var out strings.Builder
	pos := 0
	for _, segment := range splitSegments(line, false) {
		base = changedDirectory(base, tolerantWords(strings.ReplaceAll(segment, `\`, "/")))
		if base != "" {
			segment = goWord().ReplaceAllString(segment, "${1}${2} -C "+folderText(base)+"${3}")
		}
		out.WriteString(segment)
		// A break follows each segment but the last.
		pos += len(segment)
		if pos < len(line) {
			out.WriteByte(line[pos])
			pos++
		}
	}
	return out.String()
}

// folderText is p in forward slashes, with each byte unsafeFolder names
// replaced by an underscore.
func folderText(p string) string {
	return unsafeFolder().ReplaceAllString(strings.ReplaceAll(p, `\`, "/"), "_")
}

// isMainFolder says whether p, a folder in either slash and any case, is
// loomux's main package: cmd/loomux, alone or at the end of a longer path.
func isMainFolder(p string) bool {
	return endsIn(p, "cmd", "loomux")
}

// endsIn says whether the last elements of p, a path in either slash and any
// case, are the names, each one as a glob the path's element may spell it.
func endsIn(p string, names ...string) bool {
	elements := strings.Split(path.Clean(strings.ToLower(strings.ReplaceAll(p, `\`, "/"))), "/")
	if len(elements) < len(names) {
		return false
	}
	for i, name := range names {
		if ok, _ := path.Match(elements[len(elements)-len(names)+i], name); !ok {
			return false
		}
	}
	return true
}

// goRunCall reads words as go [-C dir] run [flags] package: the folder go
// changes to by its -C flags, and the index of the first package argument,
// -1 when there is none.
func goRunCall(words []string) (dir string, pkg int, ok bool) {
	if verbOf(words[0]) != "go" {
		return "", 0, false
	}
	i := goFlags(words, 1, &dir)
	if i >= len(words) || words[i] != "run" {
		return "", 0, false
	}
	i = goFlags(words, i+1, &dir)
	if i >= len(words) {
		return dir, -1, true
	}
	return dir, i, true
}

// goFlags skips the flags of words from i on, with the value of those that
// take one, and adds each -C to dir. It answers the index of the first word
// that is no flag.
func goFlags(words []string, i int, dir *string) int {
	for i < len(words) && strings.HasPrefix(words[i], "-") {
		name, value, glued := strings.Cut(strings.TrimLeft(words[i], "-"), "=")
		i++
		if !glued && goFlagTakesValue(name) && i < len(words) {
			value, i = words[i], i+1
		}
		if name == "C" {
			*dir = path.Join(*dir, strings.ReplaceAll(value, `\`, "/"))
		}
	}
	return i
}

// isLoomuxPackage says whether a go run argument is loomux's main package:
// cmd/loomux, its main.go or a glob that finds it (m*.go, cmd/l*), or the packages
// below a folder that holds it (cmd/loomux/..., ./...), relative or under a
// module path, at any version, in any spelling path.Clean folds. It compares
// whole path elements, so mycmd/loomux is another package.
func isLoomuxPackage(p string) bool {
	p = strings.ToLower(strings.ReplaceAll(p, `\`, "/"))
	if at := strings.LastIndexByte(p, '@'); at >= 0 {
		p = p[:at]
	}
	p = path.Clean(p)
	last := path.Base(p)
	if matched, _ := path.Match(last, "main.go"); matched && strings.HasSuffix(last, ".go") {
		return isMainFolder(path.Dir(p))
	}
	if last == "..." {
		p = path.Dir(p)
		return p == "." || endsIn(p, "cmd") || isMainFolder(p)
	}
	return isMainFolder(p)
}
