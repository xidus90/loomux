package graph

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// ErrNotIndexed is what a missing graph.json means, and it is named because a
// caller draws that state rather than reporting a failure: an area that is
// registered but was never indexed is a state of its own, not an error. The
// message is the whole suffix of the error ReadGraph builds, so naming the
// state costs nothing at the reader's end -- the sentence on screen is the
// one that stood here before the sentinel existed.
var ErrNotIndexed = errors.New("never indexed; run `brain reindex`")

// ReadGraph loads and validates graph.json for the specified area. A
// read-only area keeps it in stateDir, with fallbackDir -- ultra-brain's --
// as the fallback until `loomux migrate` has moved the area.
//
// Like core._graph (src/brain/core.py:584-588) it asks Path.exists() first,
// which answers False for any failure to stat, and reads the file as strict
// UTF-8 with universal newlines. A read error already names the file.
func ReadGraph(area config.Area, stateDir, fallbackDir string) (*Graph, error) {
	manifestDir := config.ResolvedAreaDir(area, stateDir, fallbackDir)
	graphPath := filepath.Join(manifestDir, "graph.json")

	if _, err := os.Stat(graphPath); err != nil {
		return nil, fmt.Errorf("%s: %w", area.Scope, ErrNotIndexed)
	}
	text, err := pytext.ReadText(graphPath)
	if err != nil {
		return nil, err
	}

	return ParseGraph([]byte(text), graphPath)
}

// ParseGraph unmarshals and validates the contents of a graph.json file.
func ParseGraph(data []byte, path string) (*Graph, error) {
	loaded, err := decodeKeepingNumbers(data)
	if err != nil {
		return nil, fmt.Errorf("%s: graph is not valid JSON (%v); delete it and run `brain reindex`", path, err)
	}

	// A root that is not an object has neither key, as a nil map has none.
	raw, _ := loaded.(map[string]any)
	var missing []string
	if _, ok := raw["edges"]; !ok {
		missing = append(missing, "edges")
	}
	if _, ok := raw["links"]; !ok {
		missing = append(missing, "links")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%s: graph is missing %s; delete it and run `brain reindex`", path, strings.Join(missing, ", "))
	}

	if err := checkShape(raw, path); err != nil {
		return nil, err
	}

	var g Graph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &g, nil
}

// decodeKeepingNumbers decodes one JSON value as json.Unmarshal would, but
// keeps numbers as json.Number, so that 3 and 3.0 stay apart the way
// Python's json.loads keeps int and float apart. Whatever the decoder refuses,
// or leaves behind after the value, json.Unmarshal is asked to name.
func decodeKeepingNumbers(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var loaded any
	decodeErr := decoder.Decode(&loaded)
	_, trailingErr := decoder.Token()
	if decodeErr != nil || trailingErr != io.EOF {
		return nil, json.Unmarshal(data, new(any))
	}
	return loaded, nil
}

func checkShape(raw map[string]any, path string) error {
	edgesRaw, ok := raw["edges"].([]any)
	if !ok {
		return fmt.Errorf("%s: edges is not a list of from/to entries; delete it and run `brain reindex`", path)
	}
	for _, edgeItem := range edgesRaw {
		edgeMap, ok := edgeItem.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: edges is not a list of from/to entries; delete it and run `brain reindex`", path)
		}
		if _, hasFrom := edgeMap["from"]; !hasFrom {
			return fmt.Errorf("%s: edges is not a list of from/to entries; delete it and run `brain reindex`", path)
		}
		if _, hasTo := edgeMap["to"]; !hasTo {
			return fmt.Errorf("%s: edges is not a list of from/to entries; delete it and run `brain reindex`", path)
		}
	}

	linksRaw, ok := raw["links"].(map[string]any)
	if !ok {
		return fmt.Errorf("%s: links is not a total/resolved/dropped record; delete it and run `brain reindex`", path)
	}

	totalVal, hasTotal := linksRaw["total"]
	resolvedVal, hasResolved := linksRaw["resolved"]
	droppedVal, hasDropped := linksRaw["dropped"]

	if !hasTotal || !hasResolved || !hasDropped {
		return fmt.Errorf("%s: links is not a total/resolved/dropped record; delete it and run `brain reindex`", path)
	}

	droppedMap, ok := droppedVal.(map[string]any)
	if !ok {
		return fmt.Errorf("%s: links.dropped is not a table of reasons; delete it and run `brain reindex`", path)
	}

	if !isNumber(totalVal) || !isNumber(resolvedVal) {
		return fmt.Errorf("%s: the link counts are not numbers; delete it and run `brain reindex`", path)
	}

	for _, countVal := range droppedMap {
		if !isNumber(countVal) {
			return fmt.Errorf("%s: the link counts are not numbers; delete it and run `brain reindex`", path)
		}
	}

	return nil
}

// isNumber is Python's isinstance(value, int) on what json.loads returns: an
// integer literal is an int, 3.0 and 1e2 are floats. JSON's grammar leaves a
// literal integral exactly when it has no fraction and no exponent. A boolean,
// which Python lets through, is refused (parity list).
func isNumber(v any) bool {
	number, ok := v.(json.Number)
	return ok && !strings.ContainsAny(string(number), ".eE")
}
