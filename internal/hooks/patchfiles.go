package hooks

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// patchPaths are the file names a patch's header lines carry, as written,
// each once, in the order they first appear: both names of diff --git,
// the name after --- and +++ up to a tab, and the names after rename from,
// rename to and copy to. /dev/null names no file.
func patchPaths(text string) []string {
	var out []string
	add := func(p string) {
		p = strings.Trim(p, `"`)
		if p != "" && p != "/dev/null" && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	// Fields and TrimSpace drop the line end, \r\n as well as \n.
	for line := range strings.Lines(text) {
		if rest, ok := strings.CutPrefix(line, "diff --git "); ok {
			for _, word := range strings.Fields(rest) {
				add(word)
			}
			continue
		}
		for _, head := range []string{"--- ", "+++ ", "rename from ", "rename to ", "copy to "} {
			if rest, ok := strings.CutPrefix(line, head); ok {
				name, _, _ := strings.Cut(rest, "\t")
				add(strings.TrimSpace(name))
			}
		}
	}
	return out
}

// strippedPaths is raw with strip leading elements dropped, as patch -pN
// and git apply -pN read it; nothing when raw has no more elements. A
// negative strip, patch without -p, gives every level down to the base name.
func strippedPaths(raw string, strip int) []string {
	parts := strings.Split(raw, "/")
	if strip >= 0 {
		if strip >= len(parts) {
			return nil
		}
		return []string{strings.Join(parts[strip:], "/")}
	}
	var out []string
	for i := range parts {
		out = append(out, strings.Join(parts[i:], "/"))
	}
	return out
}

// patchTargets are the files the patches in files change, read from disk
// under dir, with strip elements dropped from each name and prefix put in
// front. A patch that cannot be read is a refusal: what it changes is
// unknown.
func patchTargets(dir string, files []string, strip int, prefix string) []shellTarget {
	var out []shellTarget
	for _, file := range files {
		place := file
		if !filepath.IsAbs(place) {
			place = filepath.Join(dir, place)
		}
		text, err := os.ReadFile(place)
		if err != nil {
			out = append(out, shellTarget{refusal: fmt.Sprintf(
				"loomux cannot read the patch %s, so it cannot tell which files it changes and refuses: %v", file, err)})
			continue
		}
		out = append(out, namedInPatch(string(text), strip, prefix)...)
	}
	return out
}

// namedInPatch are the files a patch text changes, with strip elements
// dropped and prefix in front.
func namedInPatch(text string, strip int, prefix string) []shellTarget {
	var out []shellTarget
	for _, raw := range patchPaths(text) {
		for _, p := range strippedPaths(raw, strip) {
			out = append(out, shellTarget{path: path.Join(prefix, p), shell: true})
		}
	}
	return out
}

// patchWrites is what patch writes: the file it names (the first word that
// is no option's value), the -o and -r files, and, without a named file,
// the files its patch changes -- read from -i, or from input, the file a <
// redirection hands it -- under -d.
func patchWrites(dir string, args []string, input string) []shellTarget {
	strip, words, values := -1, []string(nil), map[string][]string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, value, glued := optionValue(a, map[string]string{
			"-p": "strip", "--strip": "strip", "-i": "input", "--input": "input", "-o": "output",
			"--output": "output", "-d": "directory", "--directory": "directory", "-r": "reject", "--reject-file": "reject",
		})
		switch {
		case name != "" && !glued && i+1 < len(args):
			i++
			values[name] = append(values[name], args[i])
		case name != "":
			values[name] = append(values[name], value)
		case !strings.HasPrefix(a, "-"):
			words = append(words, a)
		}
	}
	if n, err := strconv.Atoi(last(values["strip"])); err == nil {
		strip = n
	}
	prefix := last(values["directory"])
	out := targetsOf(slices.Concat(values["output"], values["reject"]), false)
	if len(words) > 0 {
		return append(out, targetsOf(words[:1], false)...)
	}
	// -d changes the folder before patch reads -i; a redirection is opened
	// by the shell, where the line stands.
	if len(values["input"]) > 0 {
		return append(out, patchTargets(filepath.Join(dir, prefix), values["input"], strip, prefix)...)
	}
	if input != "" {
		return append(out, patchTargets(dir, []string{input}, strip, prefix)...)
	}
	return out
}

// gitPatchWrites is what git apply and git am write: the files their
// patches change, read from each word that is no option's value, or from
// input, the file a < redirection hands them; -p is 1 unless given, and
// --directory goes in front.
func gitPatchWrites(dir string, args []string, input string) []shellTarget {
	strip, prefix := 1, ""
	var patches []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, value, glued := optionValue(a, map[string]string{
			"-p": "strip", "--directory": "directory", "--exclude": "skip", "--include": "skip", "-C": "skip",
		})
		if name != "" && !glued && i+1 < len(args) {
			i++
			value = args[i]
		}
		switch {
		case name == "strip":
			if n, err := strconv.Atoi(value); err == nil {
				strip = n
			}
		case name == "directory":
			prefix = value
		case !strings.HasPrefix(a, "-"):
			patches = append(patches, a)
		}
	}
	if len(patches) == 0 && input != "" {
		patches = []string{input}
	}
	return patchTargets(dir, patches, strip, prefix)
}

// optionValue reads a as one of options (spelling → name): the name, and a
// value glued to it (-p1, --strip=1); glued is false when the value is the
// next word.
func optionValue(a string, options map[string]string) (name, value string, glued bool) {
	for spelling, n := range options {
		switch {
		case a == spelling:
			return n, "", false
		case strings.HasPrefix(spelling, "--") && strings.HasPrefix(a, spelling+"="):
			return n, a[len(spelling)+1:], true
		case !strings.HasPrefix(spelling, "--") && strings.HasPrefix(a, spelling) && len(a) > len(spelling):
			return n, a[len(spelling):], true
		}
	}
	return "", "", false
}

// last is the last of values, or "".
func last(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1]
}

// heredocPatches are the files a patch in a heredoc of line changes, at
// every level, when line hands a heredoc to patch, git apply or git am:
// the body is part of the line, and its options are not read here.
func heredocPatches(line string) []shellTarget {
	if !strings.Contains(line, "<<") || !slices.ContainsFunc(strings.Fields(line), func(w string) bool {
		return w == "patch" || w == "apply" || w == "am"
	}) {
		return nil
	}
	return namedInPatch(line, -1, "")
}
