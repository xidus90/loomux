package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// ErrNoArea says that a configuration carries no [area] table. A project that
// uses only loomux's policy declares no brain area, and every caller reads
// this as "no declaration here" (stage 1a, R7a).
var ErrNoArea = errors.New("the configuration declares no [area]")

// ReadDeclaration reads one area declaration and refuses it whole where a
// value loomux reads, or one the Python reference refused, has the wrong type
// or an unusable value. Unknown keys are not judged: real manifests carry
// keys of tools that are not loomux. The rules and their order are M1 to M17
// of `docs/.superpowers/specs/2026-09-16-loomux-registry-manifest-pruefungen-design.md`
// in the working papers of the archive release `archive/parity-recordings`.
func ReadDeclaration(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: cannot be read: %w", path, err)
	}
	document := map[string]any{}
	if err := toml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("%s: not valid TOML: %w", path, err)
	}
	manifest, err := declaration(document)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	manifest.Path = path
	return manifest, nil
}

// DeclarationKeys are the keys ReadDeclaration reads, per section. The
// reader takes its lists from here, so the schema of `loomux config` and the
// reader cannot drift apart.
func DeclarationKeys() map[string][]string {
	return map[string][]string{
		"area":        {"scope"},
		"privacy":     {"mode", "never"},
		"wiki":        {"types", "untouched_days"},
		"maintenance": {"on_merge", "branch"},
		"model":       {"enabled", "roles"},
		"layout":      {"wiki", "hub", "review", "inbox"},
		"index":       {"include", "exclude", "unsearched"},
	}
}

// declarationSections are the tables a declaration reads, in the order their
// shape is checked.
func declarationSections() []string {
	return []string{"area", "privacy", "wiki", "maintenance", "model", "layout", "index"}
}

func declaration(document map[string]any) (*Manifest, error) {
	if _, present := document["area"]; !present {
		return nil, ErrNoArea
	}
	sections := map[string]map[string]any{}
	for _, name := range declarationSections() {
		value, present := document[name]
		if !present {
			sections[name] = map[string]any{}
			continue
		}
		section, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("[%s] must be a table, found %s", name, tomlType(value))
		}
		sections[name] = section
	}
	scope, err := requiredString(sections["area"], "scope", "[area]", " ")
	if err != nil {
		return nil, err
	}
	mode, err := privacyMode(sections["privacy"])
	if err != nil {
		return nil, err
	}
	types, err := stringList(sections["wiki"], "types", "[wiki]")
	if err != nil {
		return nil, err
	}
	onMerge, err := optionalBool(sections["maintenance"], "on_merge", "[maintenance]", " ")
	if err != nil {
		return nil, err
	}
	branch, err := optionalString(sections["maintenance"], "branch", "[maintenance]", " ")
	if err != nil {
		return nil, err
	}
	modelEnabled, err := optionalFlag(sections["model"], "enabled", "[model]")
	if err != nil {
		return nil, err
	}
	modelRoles, err := declaredRoles(sections["model"])
	if err != nil {
		return nil, err
	}
	keys := DeclarationKeys()
	layout := map[string]string{}
	for _, key := range keys["layout"] {
		value, present := sections["layout"][key]
		if !present {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("[layout] %s must be a string, found %s", key, tomlType(value))
		}
		layout[key] = text
	}
	globs := map[string][]string{}
	for _, key := range keys["index"] {
		if globs[key], err = stringList(sections["index"], key, "[index]"); err != nil {
			return nil, err
		}
	}
	never, err := stringList(sections["privacy"], "never", "[privacy]")
	if err != nil {
		return nil, err
	}
	days, err := untouchedDays(sections["wiki"])
	if err != nil {
		return nil, err
	}
	return &Manifest{
		Scope:           scope,
		DeclaredTypes:   types,
		UntouchedDays:   days,
		LayoutWiki:      layout["wiki"],
		LayoutHub:       layout["hub"],
		LayoutReview:    layout["review"],
		LayoutInbox:     layout["inbox"],
		PrivacyMode:     mode,
		NeverGlobs:      never,
		IndexInclude:    globs["include"],
		IndexExclude:    globs["exclude"],
		IndexUnsearched: globs["unsearched"],
		OnMerge:         onMerge,
		MergeBranch:     mergeBranch(branch),
		ModelEnabled:    modelEnabled,
		ModelRoles:      modelRoles,
	}, nil
}

// privacyMode is the declared mode, manual_cloud where none is declared. A
// misspelt mode must fail loudly: falling back would turn the strictest
// setting into the most permissive one.
func privacyMode(privacy map[string]any) (string, error) {
	value, present := privacy["mode"]
	if !present {
		return "manual_cloud", nil
	}
	switch mode, _ := value.(string); mode {
	case "automatic_cloud", "local_only", "manual_cloud":
		return mode, nil
	}
	return "", fmt.Errorf("[privacy] mode must be one of automatic_cloud, local_only, manual_cloud, found %s", found(value))
}

// stringList is an optional array of strings. Written as a bare string, a
// glob list would otherwise match nothing and a run would succeed on zero
// files.
func stringList(section map[string]any, key, owner string) ([]string, error) {
	value, present := section[key]
	if !present {
		return nil, nil
	}
	list, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%s %s must be an array of strings, found %s", owner, key, tomlType(value))
	}
	texts := make([]string, 0, len(list))
	for i, element := range list {
		text, ok := element.(string)
		if !ok {
			return nil, fmt.Errorf("%s %s #%d must be a string, found %s", owner, key, i+1, tomlType(element))
		}
		texts = append(texts, text)
	}
	return texts, nil
}

// checkRoles refuses an unknown role rather than skipping it: a typo would
// leave a role switched off that the person believes they switched on.
func checkRoles(model map[string]any) error {
	value, present := model["roles"]
	if !present {
		return nil
	}
	roles, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("[model] roles must be a table, found %s", tomlType(value))
	}
	known := ModelRoleNames()
	var unknown []string
	for name := range roles {
		if !slices.Contains(known, name) {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		quoted := make([]string, len(unknown))
		for i, name := range unknown {
			quoted[i] = strconv.Quote(name)
		}
		return fmt.Errorf("[model] roles has unknown %s; known are %s", strings.Join(quoted, ", "), strings.Join(known, ", "))
	}
	for _, name := range known {
		if _, err := optionalBool(roles, name, "[model]", " roles."); err != nil {
			return err
		}
	}
	return nil
}

// untouchedDays is the lint threshold; a boolean is no number of days.
func untouchedDays(wiki map[string]any) (int, error) {
	value, present := wiki["untouched_days"]
	if !present {
		return DefaultUntouchedDays, nil
	}
	days, ok := value.(int64)
	if !ok {
		return 0, fmt.Errorf("[wiki] untouched_days must be an integer >= 1, found %s", tomlType(value))
	}
	if days < 1 {
		return 0, fmt.Errorf("[wiki] untouched_days must be an integer >= 1, found %d", days)
	}
	return int(days), nil
}

// InboxLayout is where the area's inbox sits, checked, or "" when the
// declaration names none. An absolute value would leave the area tree when
// joined onto it. Absolute is filepath.IsAbs on the running platform, so on
// Windows a rooted `/in` without a drive passes, as it did in the barrier.
func (m *Manifest) InboxLayout() (string, error) {
	if filepath.IsAbs(m.LayoutInbox) {
		return "", fmt.Errorf("%s: [layout] inbox must be relative to the area, found %q", m.Path, m.LayoutInbox)
	}
	return m.LayoutInbox, nil
}
