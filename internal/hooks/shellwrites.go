package hooks

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/shellwords"
)

// shellTarget is one path a call writes or removes, as the call spells it.
type shellTarget struct {
	path    string
	removes bool // a removal, or the source of a move: what lies below it counts too
	shell   bool // spelled by a shell: braces and globs are still to unfold
	// filters, when there are any, narrow a removal to what lies at and
	// below path and one of them matches; start is path as the line wrote
	// it, which find prints in front of each match.
	filters []nameFilter
	start   string
	// refusal, when set, is the reason the guard cannot tell what the call
	// writes; the target names no path.
	refusal string
}

// The verb table, by base name in lower case without .exe (verbOf).
var (
	// everyFileWrites write every file they name.
	everyFileWrites = []string{"tee", "tee-object", "set-content", "add-content", "ac", "out-file",
		"clear-content", "clc", "truncate", "touch"}
	// everyFileRemoves remove every file or folder they name, a folder with all it holds.
	everyFileRemoves = []string{"rm", "del", "erase", "remove-item", "ri", "rd", "rmdir", "unlink", "shred"}
	// moveVerbs remove their sources and write their destination.
	moveVerbs = []string{"mv", "move", "move-item", "mi"}
	// renameVerbs remove the item and write its new name in the same folder.
	renameVerbs = []string{"rename-item", "ren", "rni", "rename"}
	// copyVerbs write only their destination.
	copyVerbs = []string{"cp", "copy", "copy-item", "cpi", "install", "rsync", "ln"}
	// readVerbs read what they name, or only name it.
	readVerbs = []string{"cat", "type", "get-content", "gc", "less", "more", "head", "tail", "wc",
		"stat", "file", "jq", "ls", "dir", "get-childitem", "gci", "grep", "rg", "select-string",
		"sls", "diff", "echo", "printf", "write-output", "cd", "set-location", "sl", "pushd",
		"popd", "test-path"}
	// otherWriteVerbs are the verbs verbWrites reads one by one, for
	// knownProgram; git stands in knownTools.
	otherWriteVerbs = []string{"dd", "tar", "unzip", "expand-archive", "robocopy", "xcopy", "new-item",
		"ni", "curl", "wget", "invoke-webrequest", "iwr", "find", "sed", "perl", "patch"}
	// changeDirectory move the place relative paths after them start from.
	changeDirectory = []string{"cd", "set-location", "sl", "pushd"}
)

// unknownCall is a program the verb table does not know, with its
// arguments, and the place an earlier cd moved its relative paths to.
type unknownCall struct {
	base string
	args []string
}

// shellWrites reads a shell line for the paths it writes or removes, by a
// table of verbs rather than an expression per path. unknown holds, for every
// reading of a trusted segment whose program is in neither the table nor
// readVerbs, that program and its arguments, those of a line a shell runs
// from a string among them; strict mode judges those.
//
// Every variant of the line (lineVariants, and the line with each
// substitution folded to one word, foldSubstitutions) and both cuts of each
// (splitSegments) are read, and every reading adds targets, never removes
// one: a misread quote costs a false refusal, not a pass. The quote-blind cut
// also breaks inside quoted text, so the guesses for a segment no strict split
// reads and the unknown programs come from the quote-aware cut alone: a
// commit message that names .loomux/state/runs behind a parenthesis would
// otherwise be refused. Within a cut a cd, Set-Location or pushd moves the
// place later relative targets start from; popd goes back to the root. A > that
// every shell reads inside quotes redirects nothing (maskQuotedRedirects), and
// a removing verb fed by a pipe removes what the segment before it names
// (pipedRemovals). The string a shell runs (sh -c, pwsh -Command, cmd /c) is
// read as a line of its own (innerLine).
func shellWrites(root, line string) (targets []shellTarget, unknown []unknownCall) {
	return shellWritesAt(root, line, 0)
}

// maxInnerDepth is how many shells deep a line inside a string is still
// read; each level reads every variant, cut and reading of the one above.
const maxInnerDepth = 3

// shellWritesAt is shellWrites for a line depth shells inside the one the
// call runs.
func shellWritesAt(root, line string, depth int) (targets []shellTarget, unknown []unknownCall) {
	targets = heredocPatches(line)
	variants := lineVariants(line)
	if folded := foldSubstitutions(joinPaths(line)); !slices.Contains(variants, folded) {
		variants = append(variants, folded)
	}
	for _, variant := range variants {
		targets = append(targets, dotNetTargets(variant)...)
		cuts := [][]string{splitSegments(variant, false)}
		if strings.ContainsAny(variant, `"'`) {
			cuts = append(cuts, splitSegments(variant, true))
		}
		for n, cut := range cuts {
			// A line without quotes cuts the same both ways.
			trusted := n == 1 || len(cuts) == 1
			base, prev, at := "", "", 0
			for _, segment := range cut {
				// The segments are contiguous, each after a one-byte break.
				piped := at > 0 && variant[at-1] == '|'
				at += len(segment) + 1
				if _, err := shellwords.Split(segment); err != nil && trusted {
					targets = append(targets, under(base, guessedTargets(segment))...)
				}
				all := readings(maskQuotedRedirects(segment))
				var place []string
				for i, words := range all {
					// The field reading honours no quote, so the words after a
					// shell's string are only its $0, $1, … in the others.
					read := segmentWrites(joinPlace(root, base), words, depth, i < len(all)-1)
					targets = append(targets, under(base, read.targets)...)
					// A here-string or the segment before a pipe hands a shell
					// that reads stdin its line.
					for _, fed := range fedLines(prev, words, piped) {
						if depth < maxInnerDepth {
							f, calls := shellWritesAt(joinPlace(root, base), fed, depth+1)
							targets = append(targets, under(base, f)...)
							read.inner = append(read.inner, calls...)
						}
					}
					if piped && trusted {
						targets = append(targets, under(base, pipedRemovals(prev, read.args, read.viaXargs))...)
					}
					if !read.known && trusted {
						unknown = append(unknown, unknownCall{followed(base, read.dir), read.args})
					}
					for _, call := range read.named {
						if trusted {
							unknown = append(unknown, unknownCall{base, call})
						}
					}
					for _, call := range read.inner {
						if trusted {
							unknown = append(unknown, unknownCall{followed(base, call.base), call.args})
						}
					}
					// The tolerant reading keeps a PowerShell path whole. A
					// cd a wrapper runs moves no shell.
					if i == len(all)-2 && !read.spawns {
						place = read.args
					}
				}
				base, prev = changedDirectory(base, place), segment
			}
		}
	}
	return targets, unknown
}

// pipedRemovals are the removals of a removing verb (rm, Remove-Item, also
// behind xargs) that names no path of its own and reads its paths from a
// pipe: what the segment before the pipe lists (listingRemovals).
func pipedRemovals(prev string, args []string, viaXargs bool) []shellTarget {
	if len(args) == 0 || !slices.Contains(everyFileRemoves, verbOf(args[0])) || (!viaXargs && len(positional(args[1:])) > 0) {
		return nil
	}
	var out []shellTarget
	var read [][]string
	for _, words := range readings(prev) {
		// The readings mostly agree.
		words = dropPrefixes(words)
		if slices.ContainsFunc(read, func(r []string) bool { return slices.Equal(r, words) }) {
			continue
		}
		read = append(read, words)
		out = append(out, listingRemovals(prev, words)...)
	}
	return out
}

// listingRemovals are the paths a listing prints into a pipe, each taken for
// removed: what find or Get-ChildItem keeps under a name filter (filtered),
// else every word of the segment that looks like a path.
func listingRemovals(prev string, words []string) []shellTarget {
	if len(words) > 0 {
		switch verbOf(words[0]) {
		case "find":
			if filters := findFilters(words[1:]); filters != nil {
				return filtered(findStarts(words[1:]), filters)
			}
		case "get-childitem", "gci", "ls", "dir":
			if starts, filters := childItemFilters(words[1:]); filters != nil {
				return filtered(starts, filters)
			}
		}
	}
	out := guessedTargets(prev)
	for i := range out {
		out[i].removes = true
	}
	return out
}

// foldSubstitutions folds each balanced $(…) and each pair of backticks in
// line into the one word $_, so that a path which goes on after a
// substitution ($(pwd)/.loomux/config.toml) keeps its fixed tail; the cuts
// break a substitution apart and would lose it. The line as written still
// reads the command inside.
func foldSubstitutions(line string) string {
	var b strings.Builder
	for i := 0; i < len(line); i++ {
		end := -1
		switch {
		case strings.HasPrefix(line[i:], "$("):
			end = closingParen(line, i+1)
		case line[i] == '`':
			if next := strings.IndexByte(line[i+1:], '`'); next >= 0 {
				end = i + 1 + next
			}
		}
		if end < 0 {
			b.WriteByte(line[i])
			continue
		}
		b.WriteString("$_")
		i = end
	}
	return b.String()
}

// closingParen is the index of the ) that closes the ( at open, or -1.
func closingParen(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				return i
			}
		}
	}
	return -1
}

// joinPathCall finds a PowerShell (Join-Path …) and the arguments inside it.
var joinPathCall = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)\(\s*join-path\s+([^()]*)\)`)
})

// joinPaths writes each (Join-Path A B …) of line as the path A/B/… it
// builds: parameter names dropped, quotes stripped, $PWD as the working
// folder. The cuts break the call apart at its parentheses.
func joinPaths(line string) string {
	return joinPathCall().ReplaceAllStringFunc(line, func(call string) string {
		var parts []string
		for _, w := range tolerantWords(joinPathCall().FindStringSubmatch(call)[1]) {
			switch {
			case strings.HasPrefix(w, "-"):
			case strings.EqualFold(w, "$pwd"):
				parts = append(parts, ".")
			default:
				parts = append(parts, w)
			}
		}
		return strings.Join(parts, "/")
	})
}

// quotedRedirect stands in a masked segment for a > inside quotes.
const quotedRedirect = "\x00"

// maskQuotedRedirects replaces each > that lies inside quotes under every
// reading of the escapes -- none, bash's backslash, PowerShell's backtick --
// with quotedRedirect, so that grep '>' f reads f and writes nothing. A > that
// any of the three reads outside quotes stays a redirection: a misread quote
// costs a false refusal, not a pass. segmentWrites puts the > back.
func maskQuotedRedirects(segment string) string {
	plain, bash, pwsh := quotedAt(segment, 0), quotedAt(segment, '\\'), quotedAt(segment, '`')
	b := []byte(segment)
	for i, c := range b {
		if c == '>' && plain[i] && bash[i] && pwsh[i] {
			b[i] = quotedRedirect[0]
		}
	}
	return string(b)
}

// quotedAt says for each byte of s whether it lies inside ' or ", with esc
// escaping the next byte outside single quotes (0 for no escape); a quote
// left open runs to the end.
func quotedAt(s string, esc byte) []bool {
	in := make([]bool, len(s))
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case esc != 0 && c == esc && quote != '\'':
			i++
		case quote != 0:
			in[i] = c != quote
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		}
	}
	return in
}

// segmentRead is what segmentWrites reads from one reading of one segment.
type segmentRead struct {
	// targets are what it writes, relative to the shell's folder.
	targets []shellTarget
	// args are the program and its arguments without prefixes and
	// redirections; known says whether the program is in the verb table or
	// readVerbs.
	args  []string
	known bool
	// inner are the unknown programs of a line a shell among args runs.
	inner []unknownCall
	// dir, spawns and named are those of the prefixes (prefixes).
	dir      string
	spawns   bool
	viaXargs bool
	named    [][]string
}

// segmentWrites is what one reading of one segment writes: its redirections
// wherever they stand, then what its program does to its arguments, in the
// folder env -C or sudo -D runs it in. dir is where relative paths start, for
// git checkout's question whether a path exists. words come from a masked
// segment; args and targets carry each > again. The line a shell among args
// runs from a string adds its targets and its unknown programs (inner), up to
// maxInnerDepth shells deep; a shell that is the program then counts as
// known, for the line it runs is judged, and the words after its string, when
// the reading grouped its quotes, are an unknown call of their own.
func segmentWrites(dir string, words []string, depth int, grouped bool) segmentRead {
	var targets []shellTarget
	var args []string
	var inner []unknownCall
	unmasked := func(w string) string { return strings.ReplaceAll(w, quotedRedirect, ">") }
	// input is the file a < hands the program, for the verbs that read what
	// they write from it.
	var input string
	for i := 0; i < len(words); i++ {
		w := words[i]
		redirect, writes, bare, target := redirectTarget(w)
		if op := strings.TrimLeft(w, "0123456789"); redirect && strings.HasPrefix(op, "<") && !strings.HasPrefix(op, "<<") {
			input = op[1:]
			if bare && i+1 < len(words) {
				input = words[i+1]
			}
		}
		if !redirect {
			op := strings.IndexByte(w, '>')
			if op < 0 {
				args = append(args, unmasked(w))
				continue
			}
			// echo x>f: the redirection is glued to the word before it.
			args = append(args, unmasked(w[:op]))
			_, writes, bare, target = redirectTarget(w[op:])
		}
		if bare && i+1 < len(words) {
			i++
			if writes {
				target = words[i]
			}
		}
		if target = strings.TrimSpace(unmasked(target)); target != "" {
			targets = append(targets, shellTarget{path: target, shell: true})
		}
	}
	line, after, ok := innerLine(args)
	ran := ok && depth < maxInnerDepth
	if ran {
		found, calls := shellWritesAt(dir, line, depth+1)
		targets, inner = append(targets, found...), calls
	}
	pre := readPrefixes(args)
	read := segmentRead{args: pre.program, known: true, inner: inner, dir: pre.dir, spawns: pre.spawns, viaXargs: pre.viaXargs, named: pre.named}
	if len(read.args) == 0 {
		read.targets = targets
		return read
	}
	found, known := verbWrites(joinPlace(dir, pre.dir), read.args, input)
	if ran && slices.Contains(stringShells, verbOf(read.args[0])) {
		known = true
		// The words after the string reach the script as $0, $1, …, where
		// any command in it may take them for a path.
		if len(after) > 0 && grouped {
			read.inner = append(read.inner, unknownCall{"", append([]string{read.args[0]}, after...)})
		}
	}
	read.known = known
	read.targets = append(targets, under(filepath.ToSlash(pre.dir), found)...)
	return read
}

// redirectTarget reads w as a redirection: whether it is one, whether it
// writes, whether its target is the next word, and the target glued to it.
// > >> >| 2> &> and PowerShell's *> write; < and << read; a target that
// begins with & is a descriptor (2>&1, >&2).
func redirectTarget(w string) (redirect, writes, bare bool, target string) {
	i := 0
	for i < len(w) && (w[i] >= '0' && w[i] <= '9' || w[i] == '&' || w[i] == '*') {
		i++
	}
	if i == len(w) || w[i] != '<' && w[i] != '>' {
		return false, false, false, ""
	}
	writes = w[i] == '>'
	for i < len(w) && strings.IndexByte("<>|!", w[i]) >= 0 {
		i++
	}
	rest := w[i:]
	if strings.HasPrefix(rest, "&") {
		return true, false, false, ""
	}
	if !writes {
		return true, false, rest == "", ""
	}
	return true, true, rest == "", rest
}

// verbWrites is what a program does to its arguments, by the verb table;
// input is the file a < redirection hands it, or "".
func verbWrites(dir string, args []string, input string) ([]shellTarget, bool) {
	verb := verbOf(args[0])
	rest := args[1:]
	switch {
	case slices.Contains(everyFileWrites, verb):
		return targetsOf(positional(rest), false), true
	case slices.Contains(everyFileRemoves, verb):
		return targetsOf(positional(rest), true), true
	case slices.Contains(moveVerbs, verb):
		return moved(dir, rest), true
	case slices.Contains(renameVerbs, verb):
		return renamed(rest), true
	case slices.Contains(copyVerbs, verb):
		return destination(dir, verb, rest), true
	}
	switch verb {
	case "dd":
		var out []string
		for _, a := range rest {
			if value, ok := strings.CutPrefix(a, "of="); ok {
				out = append(out, value)
			}
		}
		return targetsOf(out, false), true
	case "tar":
		return tarWrites(rest), true
	case "unzip":
		return targetsOf(posixValues(rest, "-d", ""), false), true
	case "expand-archive":
		return targetsOf(psValues(rest, "destinationpath"), false), true
	case "robocopy", "xcopy":
		return mirrored(dir, verb, rest), true
	case "new-item", "ni":
		return newItem(rest), true
	case "curl", "wget", "invoke-webrequest", "iwr":
		return downloads(verb, rest), true
	case "find":
		return findRemoves(rest), true
	case "sed":
		return inPlace(rest, "ef"), true
	case "perl":
		return inPlace(rest, "eE"), true
	case "patch":
		return patchWrites(dir, rest, input), true
	case "git":
		return gitWrites(dir, rest, input)
	}
	return nil, reads(args)
}

// verbOf is a program's name as the verb table spells it: its base name in
// lower case, without .exe.
func verbOf(program string) string {
	return strings.TrimSuffix(baseName(program), ".exe")
}

// targetsOf makes shell targets of paths, skipping empty words.
func targetsOf(paths []string, removes bool) []shellTarget {
	var out []shellTarget
	for _, p := range paths {
		if p != "" {
			out = append(out, shellTarget{path: p, removes: removes, shell: true})
		}
	}
	return out
}

// positional are the arguments that are neither a flag (-x, --x) nor a cmd
// switch (/s, /E:ON), with the value of a PowerShell parameter glued after a
// colon (-FilePath:x) among them; after -- every word counts, one that begins
// with - too. A flag's separate value is positional as well: the guard does
// not know which flags take one, and a wrong path only ever refuses.
func positional(args []string) []string {
	var out []string
	for i, a := range args {
		switch {
		case a == "--":
			return append(out, args[i+1:]...)
		case strings.HasPrefix(a, "-"):
			if _, value, glued := strings.Cut(a, ":"); glued && value != "" {
				out = append(out, value)
			}
		case cmdSwitch(a):
		default:
			out = append(out, a)
		}
	}
	return out
}

// cmdSwitch says whether a is a switch of a cmd program (/s, /Q, /E:ON): a
// slash and one element, which no path the rules keep ever is.
func cmdSwitch(a string) bool {
	return len(a) > 1 && a[0] == '/' && !strings.ContainsAny(a[1:], `/\`)
}

// psValues are the values of the PowerShell parameter name, in any case and
// abbreviated to no fewer than three letters, glued after a colon or as the
// next word.
func psValues(args []string, name string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) < 4 || a[0] != '-' || a[1] == '-' {
			continue
		}
		spelled, value, glued := strings.Cut(a[1:], ":")
		if len(spelled) < 3 || !strings.HasPrefix(name, strings.ToLower(spelled)) {
			continue
		}
		if glued {
			out = append(out, value)
		} else if i+1 < len(args) {
			i++
			out = append(out, args[i])
		}
	}
	return out
}

// posixValues are the values of a POSIX flag: -o f, -of, a bundle ending in
// the letter (-sSLo f), --output f and --output=f. Either spelling may be
// empty; -- ends the flags.
func posixValues(args []string, short, long string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		separate := false
		switch {
		case a == "--":
			return out
		case short != "" && a == short || long != "" && a == long:
			separate = true
		case long != "" && strings.HasPrefix(a, long+"="):
			out = append(out, a[len(long)+1:])
		case short == "" || strings.HasPrefix(a, "--") || len(a) < 3 || a[0] != '-':
		case a[1] == short[1]:
			out = append(out, a[2:])
		case letters(a[1:]) && a[len(a)-1] == short[1]:
			separate = true
		}
		if separate && i+1 < len(args) {
			i++
			out = append(out, args[i])
		}
	}
	return out
}

// moved is a move: the destination -- -Destination, -t or
// --target-directory, or else the last positional argument -- is written,
// and every other positional argument, a source, is removed. A source moved
// into a folder is written there under its name (landed).
func moved(dir string, args []string) []shellTarget {
	words := positional(args)
	folders := posixValues(args, "-t", "--target-directory")
	dest := append(psValues(args, "destination"), folders...)
	if len(dest) == 0 && len(words) > 1 {
		dest = words[len(words)-1:]
	}
	sources := without(words, dest)
	out := append(targetsOf(sources, true), targetsOf(dest, false)...)
	return append(out, landed(dir, sources, dest, len(folders) > 0, false)...)
}

// without is words without the ones in drop.
func without(words, drop []string) []string {
	var out []string
	for _, w := range words {
		if !slices.Contains(drop, w) {
			out = append(out, w)
		}
	}
	return out
}

// landed are the places sources land in when a copy or move puts them into
// a folder: each under its own name in each destination. A destination is a
// folder when it ends in a slash, when intoFolder says so (it came from -t),
// when there are several sources, or when it is a folder on disk under dir.
// A copy of a tree (tree) may overwrite anything below where it lands, so
// that place counts as removed; a source ending in /. lands in the folder
// itself, which then counts so.
func landed(dir string, sources, dests []string, intoFolder, tree bool) []shellTarget {
	var out []shellTarget
	for _, d := range dests {
		// A trailing backslash is a slash in the reading that keeps
		// PowerShell paths, which shellWrites always reads as well.
		folder := intoFolder || len(sources) > 1 || strings.HasSuffix(d, "/") || isFolder(dir, d)
		if !folder {
			continue
		}
		for _, s := range sources {
			name := path.Base(strings.ReplaceAll(s, `\`, "/"))
			out = append(out, shellTarget{path: path.Join(strings.ReplaceAll(d, `\`, "/"), name), removes: tree, shell: true})
		}
	}
	return out
}

// isFolder says whether p, relative to dir unless absolute, is a folder on
// disk.
func isFolder(dir, p string) bool {
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p)
	}
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// renamed is Rename-Item and ren: the item is removed, and its new name,
// which lies in the item's folder, written.
func renamed(args []string) []shellTarget {
	items := psValues(args, "path")
	names := psValues(args, "newname")
	for _, w := range positional(args) {
		switch {
		case slices.Contains(items, w) || slices.Contains(names, w):
		case len(items) == 0:
			items = append(items, w)
		case len(names) == 0:
			names = append(names, w)
		}
	}
	out := targetsOf(items, true)
	for _, item := range items {
		for _, name := range names {
			out = append(out, shellTarget{path: path.Join(path.Dir(filepath.ToSlash(item)), name), shell: true})
		}
	}
	return out
}

// destination is where a copy lands: -Destination (PowerShell), -t or
// --target-directory where the verb takes one (rsync's -t keeps times),
// else the last positional argument; and each source under its name when
// that is a folder (landed). rsync copies the content of a source that ends
// in a slash into the destination itself, which then counts as removed.
func destination(dir, verb string, args []string) []shellTarget {
	dest := psValues(args, "destination")
	var folders []string
	if verb != "rsync" {
		folders = posixValues(args, "-t", "--target-directory")
	}
	dest = append(dest, folders...)
	words := positional(args)
	if len(dest) == 0 && len(words) > 0 {
		dest = words[len(words)-1:]
	}
	var sources, contents []string
	for _, s := range without(words, dest) {
		if verb == "rsync" && strings.HasSuffix(s, "/") {
			contents = append(contents, s)
		} else {
			sources = append(sources, s)
		}
	}
	out := append(targetsOf(dest, false), landed(dir, sources, dest, len(folders) > 0, copiesTrees(verb, args))...)
	if len(contents) > 0 {
		out = append(out, targetsOf(dest, true)...)
	}
	return out
}

// copiesTrees says whether a copy takes folders with all they hold: cp
// with -r, -R, -a (alone or in a bundle), --recursive or --archive,
// Copy-Item and its aliases with -Recurse, and rsync always, whose -r a
// script may leave out while -a brings it in.
func copiesTrees(verb string, args []string) bool {
	switch verb {
	case "rsync":
		return true
	case "cp":
		return slices.ContainsFunc(args, func(a string) bool {
			bundle, dashed := strings.CutPrefix(a, "-")
			return a == "--recursive" || a == "--archive" ||
				dashed && letters(bundle) && strings.ContainsAny(bundle, "rRa")
		})
	case "copy-item", "cpi", "copy":
		return slices.ContainsFunc(args, func(a string) bool { return isParameter(a, "recurse") })
	}
	return false
}

// mirrored is where robocopy and xcopy write: the destination, the files
// robocopy names after it there, and for xcopy the source under its name
// when the destination is a folder. With /E, /S or /MIR the content of the
// source folder lands in the destination, which then counts as removed.
func mirrored(dir, verb string, args []string) []shellTarget {
	words := positional(args)
	if len(words) < 2 {
		return nil
	}
	dest := words[1:2]
	out := targetsOf(dest, false)
	if slices.ContainsFunc(args, func(a string) bool {
		switch strings.ToLower(a) {
		case "/e", "/s", "/mir":
			return true
		}
		return false
	}) {
		return append(out, targetsOf(dest, true)...)
	}
	if verb == "robocopy" {
		for _, file := range words[2:] {
			out = append(out, shellTarget{path: path.Join(strings.ReplaceAll(dest[0], `\`, "/"), file), shell: true})
		}
		return out
	}
	return append(out, landed(dir, words[:1], dest, false, false)...)
}

// tarWrites are where tar writes: the folder it extracts into (-C,
// --directory), and when it creates -- a c in a flag word or in its first,
// dashless word, or --create -- the archive (-f, --file, an f ending a
// bundle). What an archive holds is unknown, so an extraction into the
// working folder without -C writes nothing the guard can name.
func tarWrites(args []string) []shellTarget {
	out := posixValues(args, "-C", "--directory")
	creates := slices.Contains(args, "--create")
	for i, a := range args {
		bundle, dashed := strings.CutPrefix(a, "-")
		if !strings.HasPrefix(a, "--") && letters(bundle) && (dashed || i == 0) && strings.ContainsRune(bundle, 'c') {
			creates = true
		}
	}
	if creates {
		out = append(out, posixValues(args, "-f", "--file")...)
		if len(args) > 1 && !strings.HasPrefix(args[0], "-") && letters(args[0]) && strings.HasSuffix(args[0], "f") {
			out = append(out, args[1])
		}
	}
	return targetsOf(out, false)
}

// newItem is where New-Item creates: -Path, or every positional argument,
// each alone and joined with -Name when one is given.
func newItem(args []string) []shellTarget {
	paths := psValues(args, "path")
	names := psValues(args, "name")
	if len(paths) == 0 {
		for _, w := range positional(args) {
			if !slices.Contains(names, w) {
				paths = append(paths, w)
			}
		}
	}
	out := slices.Clone(paths)
	for _, p := range paths {
		for _, n := range names {
			out = append(out, p+"/"+n)
		}
	}
	if len(paths) == 0 {
		out = names
	}
	return targetsOf(out, false)
}

// downloads are the files curl, wget and Invoke-WebRequest write: -OutFile,
// which PowerShell's curl and wget aliases take too; curl's -o and --output,
// and with -O or --remote-name the URL's name under --output-dir; wget's -O
// and --output-document, or else the URL's name under -P or
// --directory-prefix.
func downloads(verb string, args []string) []shellTarget {
	out := psValues(args, "outfile")
	switch verb {
	case "curl":
		out = append(out, posixValues(args, "-o", "--output")...)
		if remoteName(args) {
			out = append(out, fetchedNames(args, posixValues(args, "", "--output-dir"))...)
		}
	case "wget":
		named := posixValues(args, "-O", "--output-document")
		out = append(out, named...)
		if len(named) == 0 {
			out = append(out, fetchedNames(args, posixValues(args, "-P", "--directory-prefix"))...)
		}
	}
	return targetsOf(out, false)
}

// remoteName says whether curl names its file after the URL: -O alone or in
// a bundle, --remote-name or --remote-name-all, before any --.
func remoteName(args []string) bool {
	for _, a := range args {
		switch {
		case a == "--":
			return false
		case a == "--remote-name" || a == "--remote-name-all":
			return true
		case len(a) > 1 && a[0] == '-' && letters(a[1:]) && strings.ContainsRune(a, 'O'):
			return true
		}
	}
	return false
}

// fetchedNames are the files a download named after its URLs writes: each
// URL's last element without query or fragment, under each of dirs, or in
// the working folder without one. A URL without a path names no file.
func fetchedNames(args, dirs []string) []string {
	var out []string
	for _, w := range positional(args) {
		_, rest, isURL := strings.Cut(w, "://")
		rest, _, _ = strings.Cut(rest, "?")
		rest, _, _ = strings.Cut(rest, "#")
		_, name, hasPath := strings.Cut(rest, "/")
		if name = path.Base("/" + name); !isURL || !hasPath || name == "/" {
			continue
		}
		if len(dirs) == 0 {
			out = append(out, name)
		}
		for _, d := range dirs {
			out = append(out, d+"/"+name)
		}
	}
	return out
}

// findRemoves are what find deletes when it deletes -- with -delete, or
// -exec, -execdir, -ok or -okdir running a removing verb: what its name
// filters keep below its start paths (filtered), or the start paths whole.
func findRemoves(args []string) []shellTarget {
	for i, a := range args {
		switch a {
		case "-delete":
			return filtered(findStarts(args), findFilters(args))
		case "-exec", "-execdir", "-ok", "-okdir":
			if i+1 < len(args) && slices.Contains(everyFileRemoves, verbOf(args[i+1])) {
				return filtered(findStarts(args), findFilters(args))
			}
		}
	}
	return nil
}

// findStarts are find's start paths, the working folder without one.
func findStarts(args []string) []string {
	var starts []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") || a == "(" || a == "!" {
			break
		}
		starts = append(starts, a)
	}
	if len(starts) == 0 {
		starts = []string{"."}
	}
	return starts
}

// nameFilter is one test of a listing that keeps what glob matches: the base
// name, or with path the whole path as find prints it; fold matches in any
// case.
type nameFilter struct {
	glob       string
	fold, path bool
}

// findFilters are find's name tests -- -name, -iname, -path, -ipath -- when
// they narrow what it keeps, nil when any other test may keep more: no name
// test, one that matches every name, -regex, a negation, an -o, or a name
// test after the first action, which find runs on every entry before the
// test. Only an and of name tests is left, so what matches one of them is a
// superset. A pattern is read with find's escapes (unescapeFind).
func findFilters(args []string) []nameFilter {
	var out []nameFilter
	acted := false
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "-name", "-iname", "-path", "-ipath":
			if acted || i+1 == len(args) {
				return nil
			}
			i++
			glob, ok := unescapeFind(args[i])
			if !ok || strings.Trim(glob, "*") == "" {
				return nil
			}
			out = append(out, nameFilter{glob: glob, fold: a[1] == 'i', path: strings.HasSuffix(a, "path")})
		case "-regex", "-iregex", "!", "-not", "-o", "-or", ",":
			return nil
		case "-delete", "-exec", "-execdir", "-ok", "-okdir":
			acted = true
		}
	}
	return out
}

// unescapeFind reads a find pattern's escapes: \x stands for x. The escaped
// byte then counts as a pattern character again, which only matches more; a
// lone \ at the end is no pattern find reads (false).
func unescapeFind(glob string) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(glob); i++ {
		if glob[i] == '\\' {
			if i++; i == len(glob) {
				return "", false
			}
		}
		b.WriteByte(glob[i])
	}
	return b.String(), true
}

// childItemFilters are the start paths of Get-ChildItem -- -Path,
// -LiteralPath, or each positional argument that is no filter, the working
// folder without one -- and its -Include and -Filter patterns, split at
// commas and matched in any case; nil filters when it names none, one that
// matches every name, or a -Filter the file system may match against an 8.3
// short name: one with a three-letter extension (*.tom keeps config.toml,
// CONFIG~1.TOM) or a ~.
func childItemFilters(args []string) ([]string, []nameFilter) {
	shortNamed := psValues(args, "filter")
	for _, glob := range shortNamed {
		if dot := strings.LastIndexByte(glob, '.'); dot >= 0 && len(glob)-dot-1 == 3 || strings.Contains(glob, "~") {
			return nil, nil
		}
	}
	values := append(psValues(args, "include"), shortNamed...)
	var filters []nameFilter
	for _, v := range values {
		for _, glob := range strings.Split(v, ",") {
			if strings.Trim(glob, "*") == "" {
				return nil, nil
			}
			filters = append(filters, nameFilter{glob: glob, fold: true})
		}
	}
	starts := append(psValues(args, "path"), psValues(args, "literalpath")...)
	if len(starts) == 0 {
		for _, w := range positional(args) {
			if !slices.Contains(values, w) {
				starts = append(starts, w)
			}
		}
	}
	if len(starts) == 0 {
		starts = []string{"."}
	}
	return starts, filters
}

// filtered are the removals of a listing whose name filters keep what they
// match below starts: each start path with the filters, which the judge
// matches against the disk, or the start paths whole without a filter.
func filtered(starts []string, filters []nameFilter) []shellTarget {
	out := targetsOf(starts, true)
	for i := range out {
		out[i].filters, out[i].start = filters, out[i].path
	}
	return out
}

// matches says whether the filter keeps a path find prints as printed.
func (f nameFilter) matches(printed string) bool {
	subject, glob := printed, f.glob
	if !f.path {
		subject = path.Base(printed)
	}
	if f.fold {
		subject, glob = strings.ToLower(subject), strings.ToLower(glob)
	}
	return fnmatch(glob, subject)
}

// fnmatch matches s against a find pattern: * any run of bytes, a slash
// among them, ? one byte, [...] one byte of a class (! or ^ negates it, a-z
// a range), any other byte itself. A [ without its ] is a byte of its own.
func fnmatch(glob, s string) bool {
	for glob != "" {
		switch c := glob[0]; {
		case c == '*':
			for i := len(s); i >= 0; i-- {
				if fnmatch(glob[1:], s[i:]) {
					return true
				}
			}
			return false
		case s == "":
			return false
		case c == '?':
		case c == '[' && strings.IndexByte(glob[1:], ']') >= 0:
			end := strings.IndexByte(glob[1:], ']') + 1
			class := glob[1:end]
			negate := strings.HasPrefix(class, "!") || strings.HasPrefix(class, "^")
			if negate {
				class = class[1:]
			}
			if inClass(class, s[0]) == negate {
				return false
			}
			glob = glob[end:]
		case c != s[0]:
			return false
		}
		glob, s = glob[1:], s[1:]
	}
	return s == ""
}

// inClass says whether c is one of a class's bytes or ranges.
func inClass(class string, c byte) bool {
	for i := 0; i < len(class); i++ {
		if i+2 < len(class) && class[i+1] == '-' {
			if class[i] <= c && c <= class[i+2] {
				return true
			}
			i += 2
		} else if class[i] == c {
			return true
		}
	}
	return false
}

// inPlace are the files sed -i or perl -i edits in place: every positional
// argument but the script, which is the first one unless a flag brings it --
// one of scriptFlags alone or ending a bundle, or --expression, --file. -i is
// case-sensitive, because perl -I is an include path; it counts alone, with
// a suffix (-i.bak), in a bundle (-Ei, -pi) and as --in-place.
func inPlace(args []string, scriptFlags string) []shellTarget {
	edits, scripted := false, false
	var words []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			words = append(words, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(a, "--"):
			name, _, glued := strings.Cut(a[2:], "=")
			switch {
			case strings.HasPrefix(name, "in-place"):
				edits = true
			case name == "expression" || name == "file":
				scripted = true
				if !glued {
					i++
				}
			}
		case len(a) > 1 && a[0] == '-':
			flags, _, _ := strings.Cut(a[1:], ".")
			edits = edits || strings.ContainsRune(flags, 'i')
			if flags != "" && strings.ContainsRune(scriptFlags, rune(flags[len(flags)-1])) {
				scripted = true
				i++
			}
		default:
			words = append(words, a)
		}
	}
	if !edits {
		return nil
	}
	if !scripted && len(words) > 0 {
		words = words[1:]
	}
	return targetsOf(words, false)
}

// gitWrites is what a git subcommand writes: mv and rm their paths, checkout
// and restore the paths checkedOut names, clean what cleaned names, apply
// and am the files their patches change (input is the file a < hands
// them). diff, log, show, status, blame, add and commit read; every other
// subcommand is unknown.
func gitWrites(dir string, args []string, input string) ([]shellTarget, bool) {
	sub, rest := gitSubcommand(args)
	switch sub {
	case "apply", "am":
		return gitPatchWrites(dir, rest, input), true
	case "mv":
		return moved(dir, rest), true
	case "rm":
		return targetsOf(positional(rest), true), true
	case "checkout", "restore":
		return checkedOut(dir, sub, rest), true
	case "clean":
		return cleaned(rest), true
	case "diff", "log", "show", "status", "blame", "add", "commit":
		return nil, true
	}
	return nil, false
}

// gitSubcommand skips git's global options -- with the next word as the
// value of -C, -c, --git-dir, --work-tree and --namespace -- and answers the
// subcommand and its arguments. Paths after -C count from the working
// folder all the same: a named limit.
func gitSubcommand(args []string) (string, []string) {
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "-C" || a == "-c" || a == "--git-dir" || a == "--work-tree" || a == "--namespace":
			i++
		case strings.HasPrefix(a, "-"):
		default:
			return a, args[i+1:]
		}
	}
	return "", nil
}

// checkedOut are the paths git checkout or restore overwrites: every word
// after --, or without --, each argument that exists as a path. Without one
// it switches a branch, and restore --staged without --worktree touches only
// the index: neither writes a file.
func checkedOut(dir, sub string, args []string) []shellTarget {
	if sub == "restore" && (slices.Contains(args, "--staged") || slices.Contains(args, "-S")) &&
		!slices.Contains(args, "--worktree") && !slices.Contains(args, "-W") {
		return nil
	}
	if at := slices.Index(args, "--"); at >= 0 {
		return targetsOf(args[at+1:], false)
	}
	var paths []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-b" || a == "-B" || a == "--orphan" || a == "-s" || a == "--source":
			i++
		case strings.HasPrefix(a, "-"):
		default:
			place := a
			if !filepath.IsAbs(place) {
				place = filepath.Join(dir, place)
			}
			if _, err := os.Lstat(place); err == nil {
				paths = append(paths, a)
			}
		}
	}
	return targetsOf(paths, false)
}

// cleaned is what git clean removes: the paths it names, or the root without
// one. -n in a bundle and --dry-run only list; -e, a bundle ending in it
// (-fdxe) and --exclude bring a pattern, not a path.
func cleaned(args []string) []shellTarget {
	var paths []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			paths = append(paths, args[i+1:]...)
			i = len(args)
		case a == "--dry-run" || len(a) > 1 && a[0] == '-' && a[1] != '-' && a[1] != 'e' && letters(a[1:]) && strings.ContainsRune(a, 'n'):
			return nil
		case a == "--exclude" || len(a) > 1 && a[0] == '-' && letters(a[1:]) && strings.HasSuffix(a, "e"):
			i++
		case strings.HasPrefix(a, "-"):
		default:
			paths = append(paths, a)
		}
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	return targetsOf(paths, true)
}

// reads says whether a program only reads: one of readVerbs, or a loomux
// command that reads -- flow list and show, config get, list and proposals,
// and every check.
func reads(args []string) bool {
	if slices.Contains(readVerbs, verbOf(args[0])) {
		return true
	}
	sub, ok := loomuxArgs(args)
	if !ok || len(sub) == 0 {
		return false
	}
	switch sub[0] {
	case "check":
		return true
	case "flow":
		return len(sub) > 1 && (sub[1] == "list" || sub[1] == "show")
	case "config":
		return len(sub) > 1 && (sub[1] == "get" || sub[1] == "list" || sub[1] == "proposals")
	}
	return false
}

// dotNetCall finds a .NET file call in PowerShell: the class, the method and
// where the text after its opening parenthesis starts.
var dotNetCall = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)\[(?:system\.)?io\.(file|directory)\]::(\w+)\s*\(`)
})

// dotNetTargets are the paths the .NET calls of a line write or remove:
// File's Write*, Append*, Create*, Copy* and Replace*, Delete and Move, and
// Directory's CreateDirectory, Delete and Move. splitSegments cuts at the
// parenthesis, so the arguments are read from the line up to the one that
// closes it, with slashes for backslashes.
func dotNetTargets(line string) []shellTarget {
	var out []shellTarget
	for _, m := range dotNetCall().FindAllStringSubmatchIndex(line, -1) {
		class, method := strings.ToLower(line[m[2]:m[3]]), strings.ToLower(line[m[4]:m[5]])
		removes, writes := dotNetMethod(class, method)
		if !writes {
			continue
		}
		for _, arg := range callArguments(line[m[1]:]) {
			out = append(out, shellTarget{path: strings.ReplaceAll(arg, `\`, "/"), removes: removes, shell: true})
		}
	}
	return out
}

// dotNetMethod says whether a .NET file method removes and whether it
// writes at all.
func dotNetMethod(class, method string) (removes, writes bool) {
	switch {
	case method == "delete" || method == "move":
		return true, true
	case class == "directory":
		return false, method == "createdirectory"
	}
	for _, prefix := range []string{"write", "append", "create", "copy", "replace"} {
		if strings.HasPrefix(method, prefix) {
			return false, true
		}
	}
	return false, false
}

// callArguments splits the text after an opening parenthesis at its commas,
// up to the parenthesis that closes it, and answers each quoted argument
// without its quotes and each unquoted one that looks like a path.
func callArguments(text string) []string {
	var out []string
	take := func(arg string) {
		arg = strings.TrimSpace(arg)
		switch {
		case len(arg) >= 2 && (arg[0] == '\'' || arg[0] == '"') && arg[len(arg)-1] == arg[0]:
			out = append(out, arg[1:len(arg)-1])
		case pathLike(arg):
			out = append(out, arg)
		}
	}
	depth, start := 0, 0
	var quote byte
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '(':
			depth++
		case c == ')' && depth == 0:
			take(text[start:i])
			return out
		case c == ')':
			depth--
		case c == ',' && depth == 0:
			take(text[start:i])
			start = i + 1
		}
	}
	take(text[start:])
	return out
}

// guessedTargets are the words of a segment no strict split reads that look
// like a path, each taken for a write: what the segment does to them is
// unknown, and a removal would make a lone . the whole project.
func guessedTargets(segment string) []shellTarget {
	var out []shellTarget
	for _, w := range strings.Fields(segment) {
		if w = strings.Trim(w, `"'`); pathLike(w) {
			out = append(out, shellTarget{path: w, shell: true})
		}
	}
	return out
}

// pathLike says whether a word looks like a path: it holds a slash or a
// backslash, or begins with a dot.
func pathLike(w string) bool {
	return strings.ContainsAny(w, `/\`) || strings.HasPrefix(w, ".")
}

// rooted says whether p names a place without the working folder: an
// absolute path on either platform, or one under a drive letter.
func rooted(p string) bool {
	return filepath.IsAbs(p) || strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`) ||
		len(p) >= 2 && p[1] == ':' && isLetter(p[0])
}

// under puts relative targets below base, the place an earlier cd moved to.
func under(base string, targets []shellTarget) []shellTarget {
	if base == "" {
		return targets
	}
	for i, t := range targets {
		if !rooted(t.path) {
			targets[i].path = base + "/" + t.path
		}
	}
	return targets
}

// joinPlace is where a relative path under base lies on the disk.
func joinPlace(root, base string) string {
	if rooted(base) {
		return base
	}
	return filepath.Join(root, filepath.FromSlash(base))
}

// changedDirectory is base after one segment: a cd, Set-Location or pushd
// moves it to its last positional argument (home without one), popd back to
// the root, anything else leaves it.
func changedDirectory(base string, args []string) string {
	if len(args) == 0 {
		return base
	}
	verb := verbOf(args[0])
	if verb == "popd" {
		return ""
	}
	if !slices.Contains(changeDirectory, verb) {
		return base
	}
	words := positional(args[1:])
	if len(words) == 0 {
		return "~"
	}
	return followed(base, words[len(words)-1])
}

// followed is base after a move to next, relative to it unless rooted; an
// empty next stays at base.
func followed(base, next string) string {
	next = filepath.ToSlash(next)
	switch {
	case next == "":
		return base
	case rooted(next) || base == "":
		return next
	}
	return base + "/" + next
}
