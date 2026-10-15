package linecov

import (
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

//go:embed tools.toml
var toolsText string

// loadProfiles holds a function, not parsed data: no package variable may
// parse embedded data, and OnceValues parses on the first call only.
var loadProfiles = sync.OnceValues(func() ([]Profile, error) { return parseProfiles(toolsText) })

// LoadProfiles returns the built-in tool profiles, sorted by name. The slice
// is a copy; the slices nested in each profile are shared and read-only.
func LoadProfiles() ([]Profile, error) {
	ps, err := loadProfiles()
	return slices.Clone(ps), err
}

// Profile describes one coverage tool as data: what it measures, where it
// writes, which settings files it reads and what they mean. A tool whose
// settings use a known syntax needs nothing but an entry in tools.toml.
type Profile struct {
	Name       string         `toml:"-"`
	Extensions []string       `toml:"extensions"`
	Reports    []string       `toml:"reports"`
	Settings   []SettingsFile `toml:"settings"`
	Include    KeySet         `toml:"include"`
	Exclude    KeySet         `toml:"exclude"`
	Scope      string         `toml:"scope"`
	Switches   []Switch       `toml:"switches"`
	Patterns   string         `toml:"patterns"`
	Markers    []MarkerSpec   `toml:"markers"`
	MarkerKeys []MarkerSpec   `toml:"marker_keys"`
}

// SettingsFile is one candidate in the tool's own lookup order.
type SettingsFile struct {
	File    string `toml:"file"`
	Syntax  string `toml:"syntax"`
	Section string `toml:"section"`
	Table   string `toml:"table"`
	Path    string `toml:"path"`
}

// KeySet names settings keys: Dirs hold directories or files, Globs hold
// patterns in the profile's dialect, Defaults apply when no glob key is set,
// Always apply whatever the settings say.
type KeySet struct {
	Dirs     []string `toml:"dirs"`
	Globs    []string `toml:"globs"`
	Defaults []string `toml:"defaults"`
	Always   []string `toml:"always"`
}

// Switch is a boolean key that adds exclusions while it is on.
type Switch struct {
	Key     string   `toml:"key"`
	Default bool     `toml:"default"`
	Exclude []string `toml:"exclude"`
}

// MarkerSpec is a native exclusion marker: a fixed Pattern, or the values of
// a settings Key, optionally put into a Template at {}. Block is "", "start",
// "stop" or "file".
type MarkerSpec struct {
	Label    string `toml:"label"`
	Pattern  string `toml:"pattern"`
	Key      string `toml:"key"`
	Template string `toml:"template"`
	Block    string `toml:"block"`
	NoReason bool   `toml:"no_reason"`
}

// Marker is a compiled MarkerSpec. A group named reason in Re holds the
// reason; without it the text after the match does. A marker counts only in
// files with one of Extensions, those of the profile that brings it.
type Marker struct {
	Label      string
	Re         *regexp.Regexp
	NoReason   bool
	Block      string
	Extensions []string
}

func (s MarkerSpec) compile(value string) (Marker, error) {
	pattern, label := s.Pattern, s.Label
	if s.Key != "" {
		pattern, label = value, value
		if s.Template != "" {
			pattern = strings.ReplaceAll(s.Template, "{}", regexp.QuoteMeta(value))
			label = strings.ReplaceAll(strings.ReplaceAll(s.Template, `\b`, ""), "{}", value)
		}
	}
	re, err := regexp.Compile(pattern)
	return Marker{Label: label, Re: re, NoReason: s.NoReason, Block: s.Block}, err
}

func parseProfiles(data string) ([]Profile, error) {
	var doc struct {
		Tool map[string]Profile `toml:"tool"`
	}
	md, err := toml.Decode(data, &doc)
	if err != nil {
		return nil, fmt.Errorf("tools.toml: %w", err)
	}
	if un := md.Undecoded(); len(un) > 0 {
		return nil, fmt.Errorf("tools.toml: unknown key %s", un[0])
	}
	var out []Profile
	for _, name := range slices.Sorted(maps.Keys(doc.Tool)) {
		p := doc.Tool[name]
		p.Name = name
		if err := p.validate(); err != nil {
			return nil, fmt.Errorf("tools.toml: tool %s: %w", name, err)
		}
		out = append(out, p)
	}
	return out, nil
}

func (p Profile) validate() error {
	if len(p.Extensions) == 0 {
		return errors.New("no extensions")
	}
	if p.Scope != "" && p.Scope != "all" {
		return fmt.Errorf("unknown scope %q", p.Scope)
	}
	if _, ok := dialects()[p.Patterns]; p.Patterns != "" && !ok {
		return fmt.Errorf("unknown patterns dialect %q", p.Patterns)
	}
	globs := len(p.Include.Globs) + len(p.Include.Defaults) + len(p.Include.Always) +
		len(p.Exclude.Globs) + len(p.Exclude.Defaults) + len(p.Exclude.Always) + len(p.Switches)
	if globs > 0 && p.Patterns == "" {
		return errors.New("globs need a patterns dialect")
	}
	for _, s := range p.Settings {
		if _, ok := syntaxes()[s.Syntax]; !ok {
			return fmt.Errorf("settings %s: unknown syntax %q", s.File, s.Syntax)
		}
		if s.Syntax == "xml" && s.Path == "" {
			return fmt.Errorf("settings %s: xml needs a path", s.File)
		}
	}
	for _, s := range p.Markers {
		if s.Label == "" || s.Pattern == "" {
			return errors.New("a marker needs a label and a pattern")
		}
		if s.Key != "" || s.Template != "" {
			return fmt.Errorf("marker %s: a fixed marker takes no key or template", s.Label)
		}
		if _, err := s.compile(""); err != nil {
			return fmt.Errorf("marker %s: %w", s.Label, err)
		}
	}
	for _, s := range append(slices.Clone(p.MarkerKeys), p.Markers...) {
		if !slices.Contains([]string{"", "start", "stop", "file"}, s.Block) {
			return fmt.Errorf("unknown block %q", s.Block)
		}
	}
	for _, s := range p.MarkerKeys {
		if s.Key == "" {
			return errors.New("a marker key needs a key")
		}
		if s.Pattern != "" {
			return fmt.Errorf("marker key %s: a configured marker takes no pattern", s.Key)
		}
	}
	return nil
}
