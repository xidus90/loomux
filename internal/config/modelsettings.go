package config

// The local model's settings and their cascade, after
// `src/brain/model/settings.py` of the reference.
//
// Two levels decide: the state (`<state>/config.toml`) and the area's
// declaration. Off beats on, and only in that direction: an area can switch
// off what is on globally, never switch on what is off -- otherwise the
// global switch-off would be none. Unknown keys in [model] pass unjudged, as
// in the reference; an unknown role is refused, because a typo would leave a
// role off that the person believes on.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	DefaultModelEndpoint = "http://127.0.0.1:11434"
	DefaultModelName     = "hf.co/unsloth/gemma-4-E4B-it-qat-GGUF:UD-Q4_K_XL"
)

// readModelFile is os.ReadFile, as a variable so that a test can fail the
// read of a regular file that is there -- a failure Windows gives no way to
// provoke from the file system.
var readModelFile = os.ReadFile

// ModelRoleNames are the three things the model may be asked to do.
func ModelRoleNames() []string { return []string{"describe", "place", "propose"} }

// GlobalModelKeys are the keys ParseModelSettings reads; the schema of
// `loomux config --global` is held against them.
func GlobalModelKeys() []string {
	return []string{"enabled", "endpoint", "name", "roles", "temperature"}
}

// ModelSettings is one level of the cascade, or the result of both.
type ModelSettings struct {
	Enabled     bool
	Endpoint    string
	Name        string
	Temperature float64
	Roles       map[string]bool // the roles switched on, and only those
}

// ReadModelSettings reads the global file. Anything but a regular file
// declares nothing and yields the defaults.
func ReadModelSettings(stateDir string) (ModelSettings, error) {
	path := filepath.Join(stateDir, "config.toml")
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return ParseModelSettings(path, "")
	}
	data, err := readModelFile(path)
	if err != nil {
		return ModelSettings{}, fmt.Errorf("%s: cannot be read: %w", path, err)
	}
	return ParseModelSettings(path, string(data))
}

// ParseModelSettings reads the [model] block of text; path only names the
// file in a refusal.
func ParseModelSettings(path, text string) (ModelSettings, error) {
	s, err := parseModelSettings(text)
	if err != nil {
		return ModelSettings{}, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

func parseModelSettings(text string) (ModelSettings, error) {
	document := map[string]any{}
	if err := toml.Unmarshal([]byte(text), &document); err != nil {
		return ModelSettings{}, fmt.Errorf("not valid TOML: %w", err)
	}
	block := map[string]any{}
	if value, present := document["model"]; present {
		table, ok := value.(map[string]any)
		if !ok {
			return ModelSettings{}, errors.New("[model] must be a table")
		}
		block = table
	}
	enabled, err := optionalBool(block, "enabled", "[model]", " ")
	if err != nil {
		return ModelSettings{}, err
	}
	endpoint, err := modelString(block, "endpoint", DefaultModelEndpoint)
	if err != nil {
		return ModelSettings{}, err
	}
	name, err := modelString(block, "name", DefaultModelName)
	if err != nil {
		return ModelSettings{}, err
	}
	temperature, err := modelTemperature(block)
	if err != nil {
		return ModelSettings{}, err
	}
	roles, err := declaredRoles(block)
	if err != nil {
		return ModelSettings{}, err
	}
	if roles == nil {
		roles = map[string]bool{}
		for _, role := range ModelRoleNames() {
			roles[role] = true
		}
	}
	return ModelSettings{Enabled: enabled, Endpoint: endpoint, Name: name, Temperature: temperature, Roles: roles}, nil
}

func modelString(block map[string]any, key, fallback string) (string, error) {
	if _, present := block[key]; !present {
		return fallback, nil
	}
	return optionalString(block, key, "[model]", " ")
}

// modelTemperature takes an integer as well, as Python's `int | float` does;
// a boolean is no number although TOML's reader would not confuse them.
func modelTemperature(block map[string]any) (float64, error) {
	value, present := block["temperature"]
	if !present {
		return 0, nil
	}
	var number float64
	switch v := value.(type) {
	case int64:
		number = float64(v)
	case float64:
		number = v
	default:
		return 0, fmt.Errorf("[model] temperature must be a number, found %s", tomlType(value))
	}
	// Written as the range it must lie in, so that NaN, which compares false
	// to everything, falls outside it.
	if !(number >= 0 && number <= 2) {
		return 0, fmt.Errorf("[model] temperature must lie between 0 and 2, found %v", number)
	}
	return number, nil
}

// declaredRoles is nil where the block names no roles, and otherwise the
// roles switched on.
func declaredRoles(block map[string]any) (map[string]bool, error) {
	if err := checkRoles(block); err != nil {
		return nil, err
	}
	raw, present := block["roles"].(map[string]any)
	if !present {
		return nil, nil
	}
	roles := map[string]bool{}
	for name, value := range raw {
		if value == true {
			roles[name] = true
		}
	}
	return roles, nil
}

// optionalFlag is optionalBool that tells an unsaid key from false.
func optionalFlag(table map[string]any, key, owner string) (*bool, error) {
	if _, present := table[key]; !present {
		return nil, nil
	}
	flag, err := optionalBool(table, key, owner, " ")
	if err != nil {
		return nil, err
	}
	return &flag, nil
}

// Narrowed is s with the area's word on it: enabled only where the area does
// not say false, and the roles cut to those the area names.
func (s ModelSettings) Narrowed(m *Manifest) ModelSettings {
	out := s
	out.Enabled = s.Enabled && (m.ModelEnabled == nil || *m.ModelEnabled)
	if m.ModelRoles != nil {
		roles := map[string]bool{}
		for role := range s.Roles {
			if m.ModelRoles[role] {
				roles[role] = true
			}
		}
		out.Roles = roles
	}
	return out
}

// RoleOn says whether the model may be asked for role at all.
func (s ModelSettings) RoleOn(role string) bool { return s.Enabled && s.Roles[role] }
