package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/BurntSushi/toml"
)

// PathRule refuses writes to paths matching any of its globs.
type PathRule struct {
	Match  []string
	Reason string
}

// CommandRule refuses shell commands its expression matches.
type CommandRule struct {
	Regex  *regexp.Regexp
	Source string
	Reason string
}

// Policy is the [policy] table of .loomux/config.toml, and from [guard] how
// hard the guard reads a shell line.
type Policy struct {
	Paths    []PathRule
	Commands []CommandRule
	// Strict is [guard] mode = "strict": the guard also holds against an
	// agent that means to get round it. The zero value is the default mode.
	Strict bool
}

// ManifestPath is where a project's one configuration file lives.
func ManifestPath(root string) string {
	return filepath.Join(root, manifestNames[0])
}

// PolicyKeys are the keys of the two rule lists ReadPolicy reads. The reader
// decodes through the struct tags of policyFile; a test holds the two equal.
func PolicyKeys() map[string][]string {
	return map[string][]string{
		"policy.paths.rules":    {"match", "reason"},
		"policy.commands.rules": {"regex", "reason"},
	}
}

type policyFile struct {
	Policy struct {
		Paths struct {
			Rules []struct {
				Match  any    `toml:"match"`
				Reason string `toml:"reason"`
			} `toml:"rules"`
		} `toml:"paths"`
		Commands struct {
			Rules []struct {
				Regex  string `toml:"regex"`
				Reason string `toml:"reason"`
			} `toml:"rules"`
		} `toml:"commands"`
	} `toml:"policy"`
	Guard any `toml:"guard"`
}

// ReadPolicy reads the policy of the project at root. A missing file is an
// empty policy; every other failure is an error naming the file, because the
// guard refuses on it and the human who fixes the file needs to find it.
//
// Every expression is compiled here. A compile error dropped would leave a
// rule with a lookahead that never matches, without a word (spec, 2026-09-14).
func ReadPolicy(root string) (Policy, error) {
	path := ManifestPath(root)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Policy{}, nil
	}
	if err != nil {
		return Policy{}, fmt.Errorf("%s: %w", path, err)
	}
	var file policyFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return Policy{}, fmt.Errorf("%s: %w", path, err)
	}
	var policy Policy
	for i, rule := range file.Policy.Paths.Rules {
		globs, err := globList(rule.Match)
		if err != nil {
			return Policy{}, fmt.Errorf("%s: [[policy.paths.rules]] #%d match: %w", path, i+1, err)
		}
		if rule.Reason == "" {
			return Policy{}, fmt.Errorf("%s: [[policy.paths.rules]] #%d needs a reason", path, i+1)
		}
		policy.Paths = append(policy.Paths, PathRule{Match: globs, Reason: rule.Reason})
	}
	for i, rule := range file.Policy.Commands.Rules {
		if rule.Regex == "" {
			return Policy{}, fmt.Errorf("%s: [[policy.commands.rules]] #%d needs a regex", path, i+1)
		}
		compiled, err := regexp.Compile(rule.Regex)
		if err != nil {
			return Policy{}, fmt.Errorf("%s: [[policy.commands.rules]] #%d regex %s does not compile: %w", path, i+1, rule.Regex, err)
		}
		if rule.Reason == "" {
			return Policy{}, fmt.Errorf("%s: [[policy.commands.rules]] #%d needs a reason", path, i+1)
		}
		policy.Commands = append(policy.Commands, CommandRule{Regex: compiled, Source: rule.Regex, Reason: rule.Reason})
	}
	strict, err := parseGuard(path, file.Guard)
	if err != nil {
		return Policy{}, err
	}
	policy.Strict = strict
	return policy, nil
}

// globList also refuses what would load and never match: no glob at all, an
// empty one, or one whose syntax `path.Match` -- the guard's matcher -- rejects.
//
// A malformed glob is the path half of the defect the commands side already
// refuses two rules up: `match = "secrets/[a-z.env"` loaded happily and never
// matched, so the policy named a path it did not protect. `path.Match`
// answers `ErrBadPattern` for a pattern it cannot read, and an empty name is
// enough to make it read one -- with one gap, which is why the guard side
// still refuses loudly on a match error rather than trusting this: a bad class
// in a chunk after the first (`foo/*[x`) is not reached here, because `Match`
// stops at the first chunk that does not match an empty name.
func globList(value any) ([]string, error) {
	var globs []string
	switch v := value.(type) {
	case string:
		globs = []string{v}
	case []any:
		globs = make([]string, 0, len(v))
		for _, element := range v {
			text, ok := element.(string)
			if !ok {
				return nil, fmt.Errorf("list element %v is not a string", element)
			}
			globs = append(globs, text)
		}
	default:
		return nil, fmt.Errorf("must be a string or a list of strings, found %T", value)
	}
	if len(globs) == 0 {
		return nil, errors.New("needs at least one glob")
	}
	for i, glob := range globs {
		if glob == "" {
			return nil, fmt.Errorf("glob #%d is empty", i+1)
		}
		if _, err := path.Match(glob, ""); err != nil {
			return nil, fmt.Errorf("glob #%d %q is malformed: %w", i+1, glob, err)
		}
	}
	return globs, nil
}
