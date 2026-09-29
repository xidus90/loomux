package switchover

import (
	_ "embed"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// applyTemplate is the text of apply.sh with a token @NAME@ for every value.
// It is text only; Render reads it on first use.
//
//go:embed apply.sh.tmpl
var applyTemplate string

// Params are the values of one project's apply.sh. Every path but VaultOld
// and the entries of OldFiles is absolute; those two are relative to the
// vault and to the project.
type Params struct {
	Name        string   `json:"name"`
	Project     string   `json:"project"`
	Loomux      string   `json:"loomux"`
	ConfigNew   string   `json:"config_new"`
	Registry    string   `json:"registry"`
	RegistryNew string   `json:"registry_new"`
	RegistrySum string   `json:"registry_sum"`
	WikiSrcs    []string `json:"wiki_srcs"`
	WikiDst     string   `json:"wiki_dst"`
	Vault       string   `json:"vault"`
	VaultOld    string   `json:"vault_old"`
	StateArea   string   `json:"state_area"`
	OldFiles    []string `json:"old_files"`
	OldHooks    []string `json:"old_hooks"`
	InitArgs    []string `json:"init_args"`
}

// Render writes the apply.sh of p. It refuses parameters the script could
// act on wrongly: a missing value, a path that is not absolute, an old file or
// vault folder outside its tree, an old file spelt like a short name of
// Windows, a line break that would end an assignment early, and a vault folder
// that is not among the sources of the wiki.
func Render(p Params) (string, error) { return render(applyTemplate, p) }

// render fills template with p. The script must have LF line endings, since
// sh reads a CR as part of a word; every CR goes, from the template and from
// the values.
func render(template string, p Params) (string, error) {
	template = strings.ReplaceAll(template, "\r", "")
	p = withoutCR(p)
	if err := check(p); err != nil {
		return "", err
	}
	// The script strips the folder off the paths git lists, so it has to
	// read as git writes it: without a trailing slash.
	if p.VaultOld != "" {
		p.VaultOld = path.Clean(p.VaultOld)
	}
	values := map[string]string{
		"@VAULT_SRC@":    quote(vaultSource(p)),
		"@NAME@":         quote(p.Name),
		"@PROJECT@":      quote(p.Project),
		"@LOOMUX@":       quote(p.Loomux),
		"@CONFIG_NEW@":   quote(p.ConfigNew),
		"@REGISTRY@":     quote(p.Registry),
		"@REGISTRY_NEW@": quote(p.RegistryNew),
		"@REGISTRY_SUM@": quote(p.RegistrySum),
		"@WIKI_SRCS@":    quote(strings.Join(p.WikiSrcs, "|")),
		"@WIKI_DST@":     quote(p.WikiDst),
		"@VAULT@":        quote(p.Vault),
		"@VAULT_OLD@":    quote(p.VaultOld),
		"@STATE_AREA@":   quote(p.StateArea),
		"@OLD_FILES@":    quote(strings.Join(p.OldFiles, " ")),
		"@OLD_HOOKS@":    quote(strings.Join(p.OldHooks, "|")),
		"@INIT_ARGS@":    quote(strings.Join(p.InitArgs, " ")),
	}
	// The tokens are looked up in the template, not in the result: a value
	// may hold text that looks like a token, and it is replaced in one pass.
	token := regexp.MustCompile(`@[A-Z_]+@`)
	for _, t := range token.FindAllString(template, -1) {
		if _, ok := values[t]; !ok {
			return "", fmt.Errorf("the template has a token %s without a value", t)
		}
	}
	return token.ReplaceAllStringFunc(template, func(t string) string { return values[t] }), nil
}

// quote puts s into single quotes for sh, where nothing is special but the
// quote itself: it ends the quoted text, is escaped, and a new one begins.
func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func withoutCR(p Params) Params {
	drop := func(s string) string { return strings.ReplaceAll(s, "\r", "") }
	each := func(list []string) []string {
		var out []string
		for _, s := range list {
			out = append(out, drop(s))
		}
		return out
	}
	p.Name, p.Project, p.Loomux, p.ConfigNew = drop(p.Name), drop(p.Project), drop(p.Loomux), drop(p.ConfigNew)
	p.Registry, p.RegistryNew, p.RegistrySum = drop(p.Registry), drop(p.RegistryNew), drop(p.RegistrySum)
	p.WikiDst, p.Vault, p.VaultOld, p.StateArea = drop(p.WikiDst), drop(p.Vault), drop(p.VaultOld), drop(p.StateArea)
	p.WikiSrcs, p.OldFiles, p.OldHooks = each(p.WikiSrcs), each(p.OldFiles), each(p.OldHooks)
	return p
}

// field is one value of Params under its JSON name.
type field struct {
	name   string
	values []string
}

func fields(p Params) []field {
	one := func(name, v string) field { return field{name, []string{v}} }
	return []field{
		one("name", p.Name), one("project", p.Project), one("loomux", p.Loomux), one("config_new", p.ConfigNew),
		one("registry", p.Registry), one("registry_new", p.RegistryNew), one("registry_sum", p.RegistrySum),
		{"wiki_srcs", p.WikiSrcs}, one("wiki_dst", p.WikiDst), one("vault", p.Vault), one("vault_old", p.VaultOld),
		one("state_area", p.StateArea), {"old_files", p.OldFiles}, {"old_hooks", p.OldHooks},
	}
}

// check holds p to the rules of Render, in the order of the fields.
func check(p Params) error {
	for _, f := range fields(p) {
		for _, v := range f.values {
			if strings.Contains(v, "\n") {
				return fmt.Errorf("%s: a value holds a line break", f.name)
			}
			if strings.Contains(v, "\x00") {
				return fmt.Errorf("%s: a value holds a NUL", f.name)
			}
		}
	}
	for _, r := range []struct{ name, value string }{
		{"name", p.Name}, {"project", p.Project}, {"loomux", p.Loomux}, {"config_new", p.ConfigNew},
	} {
		if r.value == "" {
			return fmt.Errorf("%s is required", r.name)
		}
	}
	for _, r := range []struct{ name, value, with, by string }{
		{"registry", p.Registry, "registry_new", p.RegistryNew},
		{"registry_sum", p.RegistrySum, "registry_new", p.RegistryNew},
		{"wiki_dst", p.WikiDst, "wiki_srcs", strings.Join(p.WikiSrcs, "|")},
		{"vault", p.Vault, "vault_old", p.VaultOld},
	} {
		if r.by != "" && r.value == "" {
			return fmt.Errorf("%s is required with %s", r.name, r.with)
		}
	}
	for _, f := range fields(p) {
		switch f.name {
		case "project", "config_new", "registry", "registry_new", "wiki_srcs", "wiki_dst", "vault", "state_area":
			for _, v := range f.values {
				if v != "" && !absolute(v) {
					return fmt.Errorf("%s: not an absolute path", f.name)
				}
			}
		}
	}
	// The script puts these words unquoted after `init --yes`, so only a
	// module choice passes: nothing that splits, and nothing that turns init
	// into a dry run or points it at another root.
	module := regexp.MustCompile(`^--(hooks|brain|graph)=(all|each|none)$`)
	for _, a := range p.InitArgs {
		if !module.MatchString(a) {
			return fmt.Errorf("init_args: %q is not --hooks|brain|graph=all|each|none", a)
		}
	}
	if p.RegistrySum != "" && !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(p.RegistrySum) {
		return fmt.Errorf("registry_sum: not a SHA-256 in lower-case hex")
	}
	for _, l := range []struct {
		name string
		list []string
	}{{"wiki_srcs", p.WikiSrcs}, {"old_hooks", p.OldHooks}} {
		for _, v := range l.list {
			if v == "" {
				return fmt.Errorf("%s: an entry is empty", l.name)
			}
			if strings.Contains(v, "|") {
				return fmt.Errorf("%s: an entry holds |", l.name)
			}
		}
	}
	// A source named twice would be staged twice and, for the vault's
	// folder, counted again after it went. Disks of Windows ignore case.
	named := map[string]string{}
	for _, s := range p.WikiSrcs {
		key := strings.ToLower(slashClean(s))
		if first, ok := named[key]; ok {
			return fmt.Errorf("wiki_srcs: %s names %s again", s, first)
		}
		named[key] = s
	}
	// NTFS keeps a second name of the form GIT~1 for a file, and the script's
	// rm finds .git, .loomux or the wiki under it. Which file a short name
	// stands for only the disk knows, so none passes. The vault's folder is
	// left out: git removes it, and git knows a file by the name it tracks.
	short := regexp.MustCompile(`~[0-9]`)
	for _, f := range p.OldFiles {
		if f == "" {
			return fmt.Errorf("old_files: an entry is empty")
		}
		if strings.IndexFunc(f, unicode.IsSpace) >= 0 {
			return fmt.Errorf("old_files: %q holds white space, and the script splits the list there", f)
		}
		if err := inside(f); err != nil {
			return fmt.Errorf("old_files: %w", err)
		}
		if short.MatchString(f) {
			return fmt.Errorf("old_files: %s holds ~ before a digit, the short name a disk of Windows keeps for another file, "+
				"which may be .git or what init writes", f)
		}
		if written(p, path.Clean(f)) {
			return fmt.Errorf("old_files: %s would remove what init writes or the wiki", f)
		}
	}
	if p.VaultOld != "" {
		if err := inside(p.VaultOld); err != nil {
			return fmt.Errorf("vault_old: %w", err)
		}
		if vaultSource(p) == "" {
			return fmt.Errorf("vault_old: %s is not one of wiki_srcs; the vault would lose what was never merged",
				slashClean(p.Vault+"/"+p.VaultOld))
		}
	}
	return nil
}

// vaultSource is the entry of WikiSrcs that names the vault's folder, as it
// is spelt there, or "" without one.
func vaultSource(p Params) string {
	if p.VaultOld == "" {
		return ""
	}
	folder := slashClean(p.Vault + "/" + p.VaultOld)
	for _, s := range p.WikiSrcs {
		if slashClean(s) == folder {
			return s
		}
	}
	return ""
}

// written tells whether removing the old file f (clean, relative to the
// project) would take what the script has just put in place: the
// configuration, the settings with init's hooks, or the wiki. The script
// removes on a disk that ignores case, so the paths are compared without it.
func written(p Params, f string) bool {
	keep := []string{".claude", ".claude/settings.json"}
	whole := []string{".loomux"}
	f = strings.ToLower(f)
	project, wiki := strings.ToLower(slashClean(p.Project)), strings.ToLower(slashClean(p.WikiDst))
	if p.WikiDst != "" && strings.HasPrefix(wiki, project+"/") {
		whole = append(whole, strings.TrimPrefix(wiki, project+"/"))
	}
	// f may be one of them or hold one; inside a whole one, f is a part of it.
	for _, k := range append(keep, whole...) {
		if f == k || strings.HasPrefix(k, f+"/") {
			return true
		}
	}
	for _, k := range whole {
		if strings.HasPrefix(f, k+"/") {
			return true
		}
	}
	return false
}

// absolute accepts a path of this system and a POSIX one: the script runs in
// Git Bash, which reads both.
func absolute(p string) bool {
	return filepath.IsAbs(p) || strings.HasPrefix(p, "/")
}

// inside refuses a relative path that could name anything but a part of the
// tree it is relative to: the root itself, a way out of it, an absolute path
// or a drive, and git's own directory at any depth (a repository below the
// root has one too), in any case a disk of Windows takes for it (.GIT,
// sub/.Git/config).
func inside(rel string) error {
	if strings.ContainsAny(rel, `\:`) || strings.HasPrefix(rel, "/") {
		return fmt.Errorf("%s is not a relative path with forward slashes", rel)
	}
	clean := path.Clean(rel)
	switch {
	case clean == ".":
		return fmt.Errorf("%s names the root itself", rel)
	case clean == ".." || strings.HasPrefix(clean, "../"):
		return fmt.Errorf("%s leaves the root", rel)
	}
	for _, element := range strings.Split(strings.ToLower(clean), "/") {
		if element == ".git" {
			return fmt.Errorf("%s is git's", rel)
		}
	}
	return nil
}

// slashClean is a path with forward slashes, cleaned, for comparing two
// spellings of it.
func slashClean(p string) string {
	return path.Clean(strings.ReplaceAll(p, `\`, "/"))
}
