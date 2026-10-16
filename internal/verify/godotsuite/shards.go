package godotsuite

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Plan spreads suites over at most n shards, longest suite first onto the
// lightest shard (LPT). A suite without a recorded time weighs the mean of
// the recorded ones, or 1 when there are none, so a first run goes by path.
func Plan(suites []string, times map[string]float64, n int) [][]string {
	if n < 1 {
		n = 1
	}
	var sum float64
	known := 0
	for _, s := range suites {
		if t, ok := times[s]; ok {
			sum += t
			known++
		}
	}
	fallback := 1.0
	if known > 0 {
		fallback = sum / float64(known)
	}
	weight := func(s string) float64 {
		if t, ok := times[s]; ok {
			return t
		}
		return fallback
	}

	order := append([]string(nil), suites...)
	sort.Slice(order, func(i, j int) bool {
		if wi, wj := weight(order[i]), weight(order[j]); wi != wj {
			return wi > wj
		}
		return order[i] < order[j]
	})

	shards := make([][]string, n)
	load := make([]float64, n)
	for _, s := range order {
		lightest := 0
		for i := range load {
			if load[i] < load[lightest] {
				lightest = i
			}
		}
		shards[lightest] = append(shards[lightest], s)
		load[lightest] += weight(s)
	}
	var out [][]string
	for _, sh := range shards {
		if len(sh) > 0 {
			out = append(out, sh)
		}
	}
	return out
}

// ShardCount is min(8, cpus, suites), at least 1.
func ShardCount(suites, cpus int) int {
	return max(1, min(8, cpus, suites))
}

// LoadTimes reads the timings file. A missing file is a first run and gets
// no note; anything else that cannot be read is planned without.
func LoadTimes(path string) (map[string]float64, string) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]float64{}, ""
	}
	m := map[string]float64{}
	if err == nil {
		err = json.Unmarshal(data, &m)
	}
	if err != nil || m == nil {
		return map[string]float64{}, ".loomux/state/godot-suite/" + filepath.Base(path) +
			" is unreadable; planning without timings"
	}
	return m, ""
}

// SaveTimes merges update into the file. The write goes to a temp file next
// to it and is renamed over it, so a reader never sees half a file and a
// crash leaves at most a stray *.tmp that nothing reads.
func SaveTimes(path string, update map[string]float64) error {
	merged, _ := LoadTimes(path)
	for k, v := range update {
		merged[k] = v
	}
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := ""
	f, err := os.CreateTemp(filepath.Dir(path), "*.tmp")
	if err == nil {
		tmp = f.Name()
		_, err = f.Write(data)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil && tmp != "" {
		os.Remove(tmp)
	}
	return err
}

// areaName is an area as one path element.
func areaName(area string) string {
	if area == "." {
		return "root"
	}
	return strings.NewReplacer("/", "_", "\\", "_").Replace(area)
}

// TimesPath is the timings file of one area of a checkout.
func TimesPath(checkoutRoot, area string) string {
	return filepath.Join(checkoutRoot, ".loomux", "state", "godot-suite", areaName(area)+".json")
}
