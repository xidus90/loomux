package runs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Marker is a run's marker file: which flow the run belongs to and how it was
// started. The journal records what nodes did, not which graph they came from,
// so resume and replay would have nothing to load without it.
type Marker struct {
	Flow     string
	Origin   string   // where the flow came from: "project", "bundled" or "bundled+overlay"
	Overlays []string // the files of the bundled flow an overlay replaced
	Options  map[string]string
	Baseline *Baseline
	Version  string // the loomux version that started the run; empty when unknown
}

// The option names the marker uses for itself.
const (
	keyBaseline       = "baseline"
	keyBaselineCommit = "baseline_commit"
	keyOrigin         = "origin"
	keyOverlays       = "overlays"
	keyVersion        = "loomux_version"
)

var reserved = []string{keyBaseline, keyBaselineCommit, keyOrigin, keyOverlays, keyVersion}

// Reserved are the option names the marker uses for itself, which no option
// of a run may take: the loader refuses a parameter of such a name.
func Reserved() []string { return slices.Clone(reserved) }

// WriteMarker writes the marker format: the flow name on the first line, then
// one name=<JSON string> line per option, sorted by name. JSON keeps a value
// with a newline -- a list of paths -- on its own line.
//
// A write that fails after the create leaves half a marker that NextID
// counts; it costs one number and needs a full disk. ReadMarker refuses it
// when the cut falls inside an option line. A cut at the end of a line reads
// as a marker with fewer options, and a cut inside the first line as one
// whose flow name is cut short.
func WriteMarker(path string, marker Marker) error {
	if marker.Flow == "" {
		return fmt.Errorf("%s: a marker needs the name of its flow", path)
	}
	options := make(map[string]string, len(marker.Options)+len(reserved))
	for name, value := range marker.Options {
		if slices.Contains(reserved, name) {
			return fmt.Errorf("%s: option %q is reserved for the marker itself", path, name)
		}
		options[name] = value
	}
	if marker.Baseline != nil {
		options[keyBaseline] = encodeList(marker.Baseline.Dirty)
		options[keyBaselineCommit] = marker.Baseline.Commit
	}
	if marker.Origin != "" {
		options[keyOrigin] = marker.Origin
	}
	if len(marker.Overlays) > 0 {
		options[keyOverlays] = encodeList(marker.Overlays)
	}
	if marker.Version != "" {
		options[keyVersion] = marker.Version
	}
	names := make([]string, 0, len(options))
	for name := range options {
		names = append(names, name)
	}
	slices.Sort(names)
	lines := []string{marker.Flow}
	for _, name := range names {
		// A string always marshals.
		value, _ := json.Marshal(options[name])
		lines = append(lines, name+"="+string(value))
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	// O_EXCL: a marker is claimed, never overwritten. Two runs that computed
	// the same number would otherwise share one journal without a word.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return writing(path, err)
	}
	_, err = file.WriteString(strings.Join(lines, "\n") + "\n")
	return writing(path, errors.Join(err, file.Close()))
}

// encodeList sorts a list of paths and joins it one path per line, so that
// the same set always spells the same marker.
func encodeList(paths []string) string {
	sorted := slices.Clone(paths)
	slices.Sort(sorted)
	return strings.Join(sorted, "\n")
}

// writing names the marker a write failed on, and is nil when nothing failed.
// One exit for the claim and the write alike, because a write into a file that
// was just created cannot be made to fail from a test.
func writing(path string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("writing %s: %w", path, err)
}

// ReadMarker reads a marker WriteMarker wrote. An absent marker is (nil, nil).
func ReadMarker(path string) (*Marker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if strings.TrimSpace(lines[0]) == "" {
		return nil, fmt.Errorf("%s: says nothing -- not even which flow it belongs to", path)
	}
	marker := &Marker{Flow: strings.TrimSpace(lines[0]), Options: map[string]string{}}
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		name, raw, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("%s: option line without '=': %q", path, line)
		}
		// WriteMarker spells every value as a JSON string, so anything else
		// was cut short or written by hand. A null leaves the pointer nil.
		var value *string
		if err := json.Unmarshal([]byte(raw), &value); err != nil || value == nil {
			return nil, fmt.Errorf("%s: option %q is not a JSON string: %s", path, name, raw)
		}
		marker.Options[name] = *value
	}
	dirty := pop(marker.Options, keyBaseline)
	if commit, present := marker.Options[keyBaselineCommit]; present {
		marker.Baseline = &Baseline{Commit: commit, Dirty: decodeList(dirty)}
	}
	pop(marker.Options, keyBaselineCommit)
	marker.Origin = pop(marker.Options, keyOrigin)
	marker.Overlays = decodeList(pop(marker.Options, keyOverlays))
	marker.Version = pop(marker.Options, keyVersion)
	return marker, nil
}

func decodeList(recorded string) []string {
	var list []string
	for _, line := range strings.Split(recorded, "\n") {
		if line != "" {
			list = append(list, line)
		}
	}
	return list
}

func pop(options map[string]string, name string) string {
	value := options[name]
	delete(options, name)
	return value
}
