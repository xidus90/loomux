package journal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Canonical is the one JSON spelling the writer and both hashes use: object
// keys sorted at every depth, no HTML escaping, no trailing newline.
//
// encoding/json sorts map keys but writes struct fields in declaration order,
// so the value is marshalled once, read back as generic JSON with numbers kept
// as written, and marshalled again. Every object is a map by then.
func Canonical(value any) ([]byte, error) {
	first, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("cannot serialize: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(first))
	// Without UseNumber a large integer would pass through a float64 and come
	// out as a different number.
	decoder.UseNumber()
	var generic any
	// json.Marshal's own output always decodes; there is no error to handle.
	_ = decoder.Decode(&generic)

	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	// Only maps, slices, strings, bools, nil and json.Number are left, and all
	// of them encode.
	_ = encoder.Encode(generic)
	return bytes.TrimSuffix(out.Bytes(), []byte("\n")), nil
}

// InputHash takes the role of journal.py's input_hash: a stable fingerprint of
// what a node saw, by which a resume finds a node's earlier result.
//
// The bytes, and so the hashes, differ from journal.py's on purpose: they are
// Canonical's, and json.dumps spells the same value differently -- a space
// after every separator, for one.
func InputHash(node string, data any) (string, error) {
	return digest(map[string]any{"node": node, "data": data})
}

// DefinitionHash fingerprints what a node was told: its entry in the flow file,
// the raw bytes of its instruction or question, the resolved model, effort and
// tool list. A replay compares it to say which nodes ran on an older definition.
func DefinitionHash(node any, text []byte, model, effort string, tools []string) (string, error) {
	return digest(map[string]any{"node": node, "text": text, "model": model, "effort": effort, "tools": tools})
}

func digest(value any) (string, error) {
	blob, err := Canonical(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:]), nil
}
