package commit

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/xidus90/loomux/internal/config"
)

// Policy holds the project-level rules for commit message language and header checks.
type Policy struct {
	Language     string
	Threshold    int
	Conventional bool
	Allow        []*regexp.Regexp
}

// DefaultPolicy returns the built-in policy when [commit] is not configured.
func DefaultPolicy() Policy {
	return Policy{
		Language:     "en",
		Threshold:    2,
		Conventional: true,
	}
}

var knownCommitKeys = []string{"allow", "conventional", "language", "threshold"}
var knownAllowKeys = []string{"reason", "regex"}

// ReadPolicy reads the [commit] section of .loomux/config.toml.
// If the configuration file is missing or has no [commit] section, DefaultPolicy is returned.
func ReadPolicy(root string) (Policy, error) {
	path := config.ManifestPath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultPolicy(), nil
		}
		return Policy{}, fmt.Errorf("%s: %w", path, err)
	}

	var raw map[string]any
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return Policy{}, fmt.Errorf("%s: %w", path, err)
	}

	commitVal, exists := raw["commit"]
	if !exists {
		return DefaultPolicy(), nil
	}

	commitTable, ok := commitVal.(map[string]any)
	if !ok {
		return Policy{}, fmt.Errorf("%s: [commit] must be a table", path)
	}

	if err := refuseUnknownKeys(commitTable, knownCommitKeys, "[commit]", path); err != nil {
		return Policy{}, err
	}

	policy := DefaultPolicy()

	if langVal, ok := commitTable["language"]; ok {
		langStr, ok := langVal.(string)
		if !ok || (langStr != "en" && langStr != "de") {
			return Policy{}, fmt.Errorf("%s: [commit].language must be one of (\"en\", \"de\"), not %s", path, formatVal(langVal))
		}
		policy.Language = langStr
	}

	if threshVal, ok := commitTable["threshold"]; ok {
		threshInt, ok := threshVal.(int64)
		if !ok {
			return Policy{}, fmt.Errorf("%s: [commit].threshold must be an integer", path)
		}
		if threshInt <= 0 {
			return Policy{}, fmt.Errorf("%s: [commit].threshold must be greater than zero", path)
		}
		policy.Threshold = int(threshInt)
	}

	if convVal, ok := commitTable["conventional"]; ok {
		convBool, ok := convVal.(bool)
		if !ok {
			return Policy{}, fmt.Errorf("%s: [commit].conventional must be a boolean", path)
		}
		policy.Conventional = convBool
	}

	if allowVal, ok := commitTable["allow"]; ok {
		switch list := allowVal.(type) {
		case []map[string]any:
			for i, item := range list {
				where := fmt.Sprintf("[[commit.allow]] #%d", i+1)
				re, err := parseAllowItem(item, where, path)
				if err != nil {
					return Policy{}, err
				}
				policy.Allow = append(policy.Allow, re)
			}
		case []any:
			for i, item := range list {
				itemTable, ok := item.(map[string]any)
				if !ok {
					return Policy{}, fmt.Errorf("%s: [[commit.allow]] must be a list of tables", path)
				}
				where := fmt.Sprintf("[[commit.allow]] #%d", i+1)
				re, err := parseAllowItem(itemTable, where, path)
				if err != nil {
					return Policy{}, err
				}
				policy.Allow = append(policy.Allow, re)
			}
		default:
			return Policy{}, fmt.Errorf("%s: [[commit.allow]] must be a list of tables", path)
		}
	}

	return policy, nil
}

func parseAllowItem(item map[string]any, where, path string) (*regexp.Regexp, error) {
	if _, ok := item["match"]; ok {
		return nil, fmt.Errorf("%s: %s has no `match` -- remove it and write a `regex`; unlike the policy's path rules a glob has no clear meaning against a line of text", path, where)
	}
	if err := refuseUnknownKeys(item, knownAllowKeys, where, path); err != nil {
		return nil, err
	}
	regexVal, ok := item["regex"]
	if !ok {
		return nil, fmt.Errorf("%s: %s needs a `regex`; unlike the policy's path rules there is no `match`, because a glob has no clear meaning against a line of text", path, where)
	}
	reasonVal, ok := item["reason"]
	if !ok {
		return nil, fmt.Errorf("%s: %s needs a `reason`", path, where)
	}
	reasonStr, ok := reasonVal.(string)
	if !ok || reasonStr == "" {
		return nil, fmt.Errorf("%s: %s needs a `reason`", path, where)
	}
	pattern, ok := regexVal.(string)
	if !ok {
		return nil, fmt.Errorf("%s: %s must be a string", path, where)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: invalid regex: %v", path, where, err)
	}
	return re, nil
}

func refuseUnknownKeys(table map[string]any, known []string, where, path string) error {
	var unknown []string
	knownSet := make(map[string]bool)
	for _, k := range known {
		knownSet[k] = true
	}
	for k := range table {
		if !knownSet[k] {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	var unknownQuoted []string
	for _, u := range unknown {
		unknownQuoted = append(unknownQuoted, fmt.Sprintf("'%s'", u))
	}
	var knownQuoted []string
	for _, k := range known {
		knownQuoted = append(knownQuoted, fmt.Sprintf("'%s'", k))
	}
	return fmt.Errorf("%s: %s does not know %s; it takes %s",
		path, where, strings.Join(unknownQuoted, ", "), strings.Join(knownQuoted, ", "))
}

func formatVal(v any) string {
	if s, ok := v.(string); ok {
		return fmt.Sprintf("'%s'", s)
	}
	return fmt.Sprintf("%v", v)
}
