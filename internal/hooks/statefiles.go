package hooks

import (
	"cmp"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

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
// copy, a move or an extraction into it names no file the guard can see.
// Both sides are compared as written and as the file system resolves them,
// so a link or a short name does not hide the file, and by pathkey, so a
// case the platform ignores does not either.
func stateFileReasons(absolute, stateDir string, removes bool) []string {
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
		for _, name := range stateFiles {
			file := pathkey.Key(filepath.Join(base, name))
			for _, target := range targets {
				if target == file || (removes && strings.HasPrefix(file, target+string(filepath.Separator))) {
					return []string{stateFilesReason}
				}
			}
		}
	}
	return nil
}

// keysOf is the key of p as written and as the file system resolves it; a
// p that does not resolve adds the key of nothing, which names no file.
func keysOf(p string) []string {
	resolved, _ := guard.ResolvePath(p)
	return []string{pathkey.Key(p), pathkey.Key(resolved)}
}

// stateReasons judges the named paths of one target against the state
// files, in every spelling a shell or the host's working folder gives them.
func (j judge) stateReasons(named []string, removes bool) []string {
	for _, rel := range named {
		for _, place := range stateSpellings(j.root, j.cwd, rel) {
			if reasons := stateFileReasons(place, j.stateDir, removes); len(reasons) > 0 {
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
func stateSpellings(root, cwd, word string) []string {
	spelled := append([]string{word}, homeForms(word)...)
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

// envReference is $NAME, ${NAME}, PowerShell's $env:NAME and ${env:NAME}, and
// cmd's %NAME%.
var envReference = regexp.MustCompile(`\$\{(?i:env:)?(\w+)\}|\$(?i:env:)?(\w+)|%(\w+)%`)

// expandEnv is p with each variable the environment holds replaced by its
// value; one it does not hold stays as written.
func expandEnv(p string) string {
	return envReference.ReplaceAllStringFunc(p, func(ref string) string {
		sub := envReference.FindStringSubmatch(ref)
		if value, ok := os.LookupEnv(sub[1] + sub[2] + sub[3]); ok {
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
	// devicePath is \\?\C:\… and \\.\C:\….
	devicePath = regexp.MustCompile(`^//[?.]/(.+)$`)
)

// driveForms are p as a Windows drive path where a Windows spelling of one
// is not the path the file system names: /c/… and //host/c$/… and
// \\?\C:\…. Other systems have none.
func driveForms(goos, p string) []string {
	if goos != "windows" {
		return nil
	}
	// Not filepath.ToSlash: a backslash is a separator in these spellings on
	// every system the guard runs on.
	slash := strings.ReplaceAll(p, `\`, "/")
	if m := devicePath.FindStringSubmatch(slash); m != nil {
		return []string{m[1]}
	}
	for _, re := range []*regexp.Regexp{gitBashDrive, adminShare} {
		if m := re.FindStringSubmatch(slash); m != nil {
			return []string{strings.ToUpper(m[1]) + ":" + cmp.Or(m[2], "/")}
		}
	}
	return nil
}
