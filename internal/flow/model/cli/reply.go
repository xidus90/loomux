package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/xidus90/loomux/internal/flow/model"
)

var jsonTypes = map[model.ReplyType]string{
	model.ReplyString: "string",
	model.ReplyInt:    "integer",
	model.ReplyBool:   "boolean",
}

// Schema renders the JSON Schema a reply must satisfy; map keys marshal sorted.
func Schema(reply map[string]model.ReplyType) string {
	props := map[string]map[string]string{}
	required := []string{}
	for name, t := range reply {
		props[name] = map[string]string{"type": jsonTypes[t]}
		required = append(required, name)
	}
	sort.Strings(required)
	b, _ := json.Marshal(map[string]any{
		"type": "object", "properties": props, "required": required, "additionalProperties": false,
	})
	return string(b)
}

// Fields checks that raw is a JSON object with exactly the fields and types
// of reply, and returns the values (int fields as int).
func Fields(raw []byte, reply map[string]model.ReplyType) (map[string]any, error) {
	var in map[string]json.RawMessage
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("reply is not a JSON object: %w", err)
	}
	for name := range in {
		if _, ok := reply[name]; !ok {
			return nil, fmt.Errorf("reply carries %q, which the node did not ask for", name)
		}
	}
	out := map[string]any{}
	for name, t := range reply {
		v, ok := in[name]
		if !ok {
			return nil, fmt.Errorf("reply lacks %q", name)
		}
		val, ok := decode(v, t)
		if !ok {
			return nil, fmt.Errorf("reply field %q is %s, not %s", name, v, t)
		}
		out[name] = val
	}
	return out, nil
}

func decode(v json.RawMessage, t model.ReplyType) (any, bool) {
	if string(bytes.TrimSpace(v)) == "null" {
		return nil, false // Unmarshal would accept null as a no-op
	}
	switch t {
	case model.ReplyString:
		var s string
		return s, json.Unmarshal(v, &s) == nil
	case model.ReplyBool:
		var b bool
		return b, json.Unmarshal(v, &b) == nil
	}
	// A JSON number without fraction or exponent is an int; Unmarshal into
	// int rejects "3", 1.5 and overflow alike.
	var n int
	return n, json.Unmarshal(v, &n) == nil
}
