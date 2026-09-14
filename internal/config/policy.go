package config

import (
	"errors"
	"fmt"
	"os"
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

// Policy is the [policy] table of .loomux/config.toml.
type Policy struct {
	Paths    []PathRule
	Commands []CommandRule
}

// ManifestPath is where a project's one configuration file lives.
func ManifestPath(root string) string {
	return filepath.Join(root, manifestNames[0])
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
}

// ReadPolicy reads the policy of the project at root. A missing file is an
// empty policy; every other failure is an error naming the file, because the
// guard refuses on it and the human who fixes the file needs to find it.
//
// Every expression is compiled here. ulguard dropped the compile error and a
// rule with a lookahead never matched, without a word (spec, 2026-09-14).
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
	return policy, nil
}

// globList also refuses what would load and never match: no glob at all, or an
// empty one.
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
	}
	return globs, nil
}
