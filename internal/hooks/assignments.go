package hooks

import (
	"maps"
	"slices"
	"strings"
)

// assigned are the variables and aliases a line sets, each name in lower
// case with every value it gets, in order.
type assigned struct {
	vars, aliases map[string][]string
}

// maxSubstituted is how many lines substituted makes at most: a name set
// to several values multiplies them.
const maxSubstituted = 16

// assignments are the variables and aliases a line sets, read word by word
// in each segment of the cut that honours quotes: bash's NAME=value at the
// head of a segment or after export, declare, local, readonly, typeset and
// cmd's set; alias NAME=value; PowerShell's $NAME = value, ${NAME}=value and
// $env:NAME = value; Set-Variable, sv, New-Variable and nv; Set-Alias, sal,
// New-Alias and nal. A value is the word as the shell hands it on, its
// quotes gone.
func assignments(line string) assigned {
	a := assigned{}
	add := func(m *map[string][]string, name, value string) {
		if *m == nil {
			*m = map[string][]string{}
		}
		name = strings.ToLower(name)
		(*m)[name] = append((*m)[name], value)
	}
	for _, segment := range splitSegments(line, true) {
		words := tolerantWords(segment)
		if len(words) == 0 {
			continue
		}
		switch head := strings.ToLower(words[0]); {
		case slices.Contains([]string{"export", "declare", "local", "readonly", "typeset", "set", "alias"}, head):
			for _, w := range words[1:] {
				if name, value, ok := shellAssignment(w); ok && head == "alias" {
					add(&a.aliases, name, value)
				} else if ok {
					add(&a.vars, name, value)
				}
			}
		case slices.Contains([]string{"set-variable", "sv", "new-variable", "nv"}, head):
			if name, value, ok := nameAndValue(words[1:]); ok {
				add(&a.vars, name, value)
			}
		case slices.Contains([]string{"set-alias", "sal", "new-alias", "nal"}, head):
			if name, value, ok := nameAndValue(words[1:]); ok {
				add(&a.aliases, name, value)
			}
		case strings.HasPrefix(head, "$"):
			if name, value, ok := powerShellAssignment(words); ok {
				add(&a.vars, name, value)
			}
		default:
			for _, w := range words {
				name, value, ok := shellAssignment(w)
				if !ok {
					break
				}
				add(&a.vars, name, value)
			}
		}
	}
	return a
}

// shellAssignment reads w as NAME=value.
func shellAssignment(w string) (name, value string, ok bool) {
	name, value, ok = strings.Cut(w, "=")
	return name, value, ok && isIdentifier(name)
}

// powerShellAssignment reads words as $NAME = value, with the = and the
// value glued to the name or apart, and the name spelled ${NAME} or
// $env:NAME; == compares and sets nothing.
func powerShellAssignment(words []string) (name, value string, ok bool) {
	name, value, glued := strings.Cut(words[0][1:], "=")
	rest := words[1:]
	if !glued {
		if len(rest) == 0 || !strings.HasPrefix(rest[0], "=") {
			return "", "", false
		}
		value, rest = rest[0][1:], rest[1:]
	}
	if value == "" && len(rest) > 0 {
		value = rest[0]
	}
	name = strings.TrimSuffix(strings.TrimPrefix(name, "{"), "}")
	if strings.HasPrefix(strings.ToLower(name), "env:") {
		name = name[len("env:"):]
	}
	return name, value, isIdentifier(name) && value != "" && !strings.HasPrefix(value, "=")
}

// nameAndValue reads the arguments of Set-Variable or Set-Alias: -Name and
// -Value, or the first two positional words.
func nameAndValue(args []string) (name, value string, ok bool) {
	names, values := psValues(args, "name"), psValues(args, "value")
	if len(names) > 0 && len(values) > 0 {
		return names[0], values[0], true
	}
	words := positional(args)
	if len(words) < 2 {
		return "", "", false
	}
	return words[0], words[1], true
}

// isIdentifier says whether s names a variable: a letter or _, then
// letters, digits and _.
func isIdentifier(s string) bool {
	for i := 0; i < len(s); i++ {
		if !identifierByte(s[i]) || i == 0 && s[i] >= '0' && s[i] <= '9' {
			return false
		}
	}
	return s != ""
}

// identifierByte says whether c may stand in a variable's name.
func identifierByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// substituted are line with the variables and aliases it sets put in where
// it uses them, one line per combination of their values, at most
// maxSubstituted; none when it sets nothing. A substitution only adds
// readings: the line as written stays among the variants.
func substituted(line string) []string {
	a := assignments(line)
	if len(a.vars)+len(a.aliases) == 0 {
		return nil
	}
	out := []string{line}
	for _, name := range slices.Sorted(maps.Keys(a.vars)) {
		out = everyValue(out, a.vars[name], func(s, v string) string { return putVariable(s, name, v) })
	}
	for _, name := range slices.Sorted(maps.Keys(a.aliases)) {
		out = everyValue(out, a.aliases[name], func(s, v string) string { return putAlias(s, name, v) })
	}
	return out
}

// everyValue puts each of values into each of lines, keeping at most
// maxSubstituted lines.
func everyValue(lines, values []string, put func(line, value string) string) []string {
	var out []string
	for _, l := range lines {
		for _, v := range values {
			if r := put(l, v); len(out) < maxSubstituted && !slices.Contains(out, r) {
				out = append(out, r)
			}
		}
	}
	return out
}

// putVariable is s with value where it uses the variable name: $name,
// ${name}, $env:name and %name%, in any case; not where an = sets it.
func putVariable(s, name, value string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if n := variableAt(s[i:], name); n > 0 && !setsAt(s[i+n:]) {
			b.WriteString(value)
			i += n
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// variableAt is the length of a use of the variable name at the start of s,
// or 0.
func variableAt(s, name string) int {
	for _, f := range []struct{ open, close string }{{"${", "}"}, {"$env:", ""}, {"$", ""}, {"%", "%"}} {
		end := len(f.open) + len(name)
		if end+len(f.close) > len(s) || !strings.EqualFold(s[:len(f.open)], f.open) ||
			!strings.EqualFold(s[len(f.open):end], name) || s[end:end+len(f.close)] != f.close {
			continue
		}
		// $DX is another variable than $D.
		if f.close == "" && end < len(s) && identifierByte(s[end]) {
			continue
		}
		return end + len(f.close)
	}
	return 0
}

// setsAt says whether s, the text after a variable, assigns it: = after
// blanks, but not ==.
func setsAt(s string) bool {
	s = strings.TrimLeft(s, " \t")
	return strings.HasPrefix(s, "=") && !strings.HasPrefix(s, "==")
}

// putAlias is s with value where name stands as a command word: after the
// start, a blank or a break, and before the end, a blank or a break.
func putAlias(s, name, value string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		end := i + len(name)
		if end <= len(s) && strings.EqualFold(s[i:end], name) &&
			(i == 0 || strings.IndexByte(" \t;|&(", s[i-1]) >= 0) &&
			(end == len(s) || strings.IndexByte(" \t;|&)", s[end]) >= 0) {
			b.WriteString(value)
			i = end
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
