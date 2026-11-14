package hooks

import (
	"cmp"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/pathkey"
)

// stateFilesReason refuses an agent a write to a file the whole machine
// shares, as opposed to the project's own manifest.
const stateFilesReason = "loomux's machine-wide files are a human's to write; an agent proposes a change with `loomux config set|unset … --global --propose`"

// stateFiles are the files of the state directory an agent never writes.
// Proposals, logs and the run files below it stay open.
var stateFiles = []string{"config.toml", "ui.toml", "ui.json", "registry.toml"}

// stateFileReasons judges the absolute path of one target against the
// machine-wide files of stateDir: the file itself, and for a removal also
// any folder above it. The state directory itself counts for any write: a
// copy, a move or an extraction into it names no file the guard can see; a
// link made onto it or onto a folder above it counts too (linked), since
// the write through it names a path that does not exist yet.
// Both sides are compared as written and as the file system resolves them,
// so a link or a short name does not hide the file, and by pathkey, so a
// case the platform ignores does not either.
func stateFileReasons(absolute, stateDir string, removes, linked bool) []string {
	if stateDir == "" {
		return nil
	}
	targets, bases := keysOf(absolute), keysOf(stateDir)
	for _, base := range bases {
		// A path that does not resolve has the key of the empty path, ".",
		// which no file is.
		if base != "." && slices.Contains(targets, base) {
			return []string{stateFilesReason}
		}
		if linked && slices.ContainsFunc(targets, func(target string) bool { return above(target, base) }) {
			return []string{stateFilesReason}
		}
		for _, name := range stateFiles {
			file := pathkey.Key(filepath.Join(base, name))
			for _, target := range targets {
				if target == file || (removes && above(target, file)) {
					return []string{stateFilesReason}
				}
			}
		}
	}
	return nil
}

// above says whether the key target is a folder that holds the key file.
func above(target, file string) bool {
	return strings.HasPrefix(file, strings.TrimSuffix(target, string(filepath.Separator))+string(filepath.Separator))
}

// keysOf is the key of p as written and as the file system resolves it, and
// the same for p as Win32 names it; a p that does not resolve adds the key
// of nothing, which names no file.
func keysOf(p string) []string {
	var keys []string
	for _, q := range slices.Compact([]string{p, win32Name(runtime.GOOS, p)}) {
		resolved, _ := guard.ResolvePath(q)
		keys = append(keys, pathkey.Key(q), pathkey.Key(resolved))
	}
	return keys
}

// trailingDotsAndSpaces is a run of dots and blanks that ends a path element.
var trailingDotsAndSpaces = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`([^./\\ ])[. ]+([/\\]|$)`) })

// win32Name is p as Win32 names it: it drops the dots and blanks that end an
// element, so a file that does not exist yet is written as config.toml. by
// that name. Other systems keep them.
func win32Name(goos, p string) string {
	if goos != "windows" {
		return p
	}
	return trailingDotsAndSpaces().ReplaceAllString(p, "$1$2")
}

// stateReasons judges the named paths of one target against the state
// files, in every spelling a shell or the host's working folder gives them.
func (j judge) stateReasons(named []string, removes, linked bool) []string {
	for _, rel := range named {
		for _, place := range stateSpellings(j.root, j.cwd, rel) {
			if reasons := stateFileReasons(place, j.stateDir, removes, linked); len(reasons) > 0 {
				return reasons
			}
		}
	}
	return nil
}

// stateSpellings are the absolute paths a word may name for a shell whose
// working folder is the root or the host's cwd: the word with the home
// folder and the environment variables a shell expands before it runs
// (the guard has no variables of the line, only its own environment), and
// the Windows drive spellings Git Bash and the administrative shares add.
// A writing tool's path keeps a stream name, ::$DATA among them, which a
// shell target has lost; it is judged without one as well.
func stateSpellings(root, cwd, word string) []string {
	spelled := append([]string{word, withoutStream(word)}, homeForms(word)...)
	for _, p := range slices.Clone(spelled) {
		spelled = append(spelled, expandEnv(p))
	}
	for _, p := range slices.Clone(spelled) {
		spelled = append(spelled, driveForms(runtime.GOOS, p)...)
	}
	var out []string
	for _, p := range spelled {
		place := filepath.FromSlash(p)
		if filepath.IsAbs(place) {
			out = append(out, place)
			continue
		}
		out = append(out, filepath.Join(root, place))
		if cwd != "" {
			out = append(out, filepath.Join(cwd, place))
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// envReference is $NAME, ${NAME}, PowerShell's $env:NAME, ${env:NAME} and
// $($env:NAME), cmd's %NAME%, and bash's ${NAME} with an operator after the
// name: ${NAME:-default} and ${NAME:=default}, ${NAME:+other}. The other
// operators (${NAME%/}, ${NAME:?}) read as the plain value: what they make of
// it (trimming, replacing) belongs to the computed-path gap, which words
// cannot close. Its groups are the
// name in $(…), the name in ${…}, the operator, the operand, the plain name
// and the name in %…%.
var envReference = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`\$\(\s*\$(?i:env:)?(\w+)\s*\)|` +
		`\$\{(?i:env:)?(\w+)(?:(:?[-=+])([^}]*)|[:#%/^,@!?][^}]*)?\}|` +
		`\$(?i:env:)?(\w+)|%(\w+)%`)
})

// expandEnv is p with each variable the environment holds replaced by its
// value, a ${NAME:-default} that the environment does not serve by its
// default; a plain variable it does not hold stays as written.
func expandEnv(p string) string {
	return envReference().ReplaceAllStringFunc(p, func(ref string) string {
		sub := envReference().FindStringSubmatch(ref)
		value, ok := os.LookupEnv(cmp.Or(sub[1], sub[2], sub[5], sub[6]))
		if op := sub[3]; op != "" {
			// A colon asks for a value that is not empty, too.
			usable := ok && (!strings.HasPrefix(op, ":") || value != "")
			switch {
			case strings.HasSuffix(op, "+") && usable:
				return sub[4]
			case strings.HasSuffix(op, "+"):
				return ""
			case usable:
				return value
			}
			return sub[4]
		}
		if ok {
			return value
		}
		return ref
	})
}

// homeForms are p with a leading ~ replaced by each home folder the guard
// knows: the platform's and $HOME, which Git Bash spells as a drive path.
func homeForms(p string) []string {
	rest, ok := strings.CutPrefix(p, "~")
	if !ok || (rest != "" && !isSlash(rest[0])) {
		return nil
	}
	var out []string
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, home+rest)
	}
	if home := os.Getenv("HOME"); home != "" {
		out = append(out, home+rest)
	}
	return out
}

var (
	// gitBashDrive is Git Bash's and Cygwin's /c/… for C:\….
	gitBashDrive = regexp.MustCompile(`^/(?:cygdrive/)?([a-zA-Z])(/.*)?$`)
	// adminShare is a local drive reached as //host/c$/… (host is any name).
	adminShare = regexp.MustCompile(`^//[^/]+/([a-zA-Z])\$(/.*)?$`)
	// devicePath is \\?\C:\… and \\.\C:\…, and \\?\UNC\host\c$\… for a share.
	devicePath = regexp.MustCompile(`^//[?.]/(.+)$`)
)

// driveForms are p as a Windows drive path where a Windows spelling of one
// is not the path the file system names: /c/… and //host/c$/… and
// \\?\C:\… and \\?\UNC\host\c$\…. Other systems have none.
func driveForms(goos, p string) []string {
	if goos != "windows" {
		return nil
	}
	// Not filepath.ToSlash: a backslash is a separator in these spellings on
	// every system the guard runs on.
	slash := strings.ReplaceAll(p, `\`, "/")
	if m := devicePath.FindStringSubmatch(slash); m != nil {
		// UNC\host\… is a share.
		if len(m[1]) <= len("UNC/") || !strings.EqualFold(m[1][:len("UNC/")], "UNC/") {
			return []string{m[1]}
		}
		slash = "//" + m[1][len("UNC/"):]
	}
	for _, re := range []*regexp.Regexp{gitBashDrive, adminShare} {
		if m := re.FindStringSubmatch(slash); m != nil {
			return []string{strings.ToUpper(m[1]) + ":" + cmp.Or(m[2], "/")}
		}
	}
	return nil
}
