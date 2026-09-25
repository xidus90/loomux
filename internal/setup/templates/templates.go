// Package templates holds the files a project gets from `loomux init` that
// are text rather than configuration: the AGENTS.md skeleton and the skills
// each module brings along. Everything is embedded; nothing is parsed before
// a caller asks for it.
package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"text/template"

	"github.com/xidus90/loomux/internal/hosts"
)

// embedded is the raw tree under files/. It is a file system, not parsed
// data: the AGENTS.md template is parsed inside AgentsMD, on first use.
//
//go:embed files
var embedded embed.FS

// File is one file a caller writes into a project.
type File struct {
	Path string // relative to the project root, slash-separated
	Text string
}

// Vars is what the AGENTS.md template reads.
type Vars struct {
	Project        string
	CommitLanguage string
	Hosts          []hosts.Host
}

// AgentsMD renders the AGENTS.md a project starts with.
func AgentsMD(v Vars) (string, error) {
	return agentsMD(embedded, v)
}

// agentsMD takes the file system as an argument so a test can hand it a
// broken template; the embedded one always parses.
func agentsMD(fsys fs.FS, v Vars) (string, error) {
	raw, err := fs.ReadFile(fsys, "files/AGENTS.md.tmpl")
	if err != nil {
		return "", fmt.Errorf("read AGENTS.md template: %w", err)
	}
	has := func(host hosts.Host) bool { return slices.Contains(v.Hosts, host) }
	parsed, err := template.New("AGENTS.md").
		Funcs(template.FuncMap{"has": has}).
		Option("missingkey=error").
		Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("parse AGENTS.md template: %w", err)
	}
	var out strings.Builder
	if err := parsed.Execute(&out, v); err != nil {
		return "", fmt.Errorf("render AGENTS.md: %w", err)
	}
	return out.String(), nil
}

// Skills returns the named skills at the path the host reads project skills
// from. A host whose skill location is not known yet gets none.
func Skills(names []string, host hosts.Host) ([]File, error) {
	var root string
	switch host {
	case hosts.HostClaude:
		root = ".claude/skills"
	case hosts.HostAntigravity:
		// Where Antigravity looks for a project's skills has not been
		// measured yet; until it is, writing a guess would leave files no
		// host reads. The installer's measurement decides the path.
		return nil, nil
	default:
		return nil, fmt.Errorf("no skill location known for host %q", host)
	}
	files := make([]File, 0, len(names))
	for _, name := range names {
		raw, err := fs.ReadFile(embedded, "files/skills/"+name+"/SKILL.md")
		if err != nil {
			return nil, fmt.Errorf("skill %q: %w", name, err)
		}
		files = append(files, File{Path: root + "/" + name + "/SKILL.md", Text: string(raw)})
	}
	return files, nil
}

// SkillNames names the skills a module brings along; a module without skills
// gets none.
func SkillNames(module string) []string {
	switch module {
	case "hooks":
		return []string{"verify-until-green"}
	case "brain":
		return []string{"brain-ingest", "brain-land", "brain-research", "brain-review", "brain-wiki-plan"}
	default:
		return nil
	}
}
