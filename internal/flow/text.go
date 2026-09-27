package flow

import (
	"bytes"
	"fmt"
	"io/fs"

	"github.com/xidus90/loomux/internal/flow/tmpl"
)

// ReadText returns a text's bytes: the file among the flow's texts, or the
// text itself when it is inline. A carriage return before a line feed is
// dropped on the way in, so the bytes -- and the definition hash taken over
// them -- do not depend on the editor or on core.autocrlf. That holds for an
// inline text too: the TOML decoder keeps the line ends of a multi-line
// string as the flow.toml has them.
func ReadText(texts fs.FS, t Text) ([]byte, error) {
	raw := []byte(t.Inline)
	if t.Path != "" {
		var err error
		if raw, err = fs.ReadFile(texts, t.Path); err != nil {
			return nil, fmt.Errorf("reading %s %s: %w", t.Key, t.Path, err)
		}
	}
	return bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n")), nil
}

// Render fills a text's placeholders from the state and the run's parameters.
//
// One map for both: a name is declared under [params] or under [state] and
// never under both, which the loader enforces. Blocks render through here and
// nowhere else, so a gate's question and an agent's instruction cannot drift
// apart in what a placeholder means.
func Render(raw []byte, state State, params Params) (string, error) {
	template, err := tmpl.Parse(string(raw))
	if err != nil {
		return "", err
	}
	values := make(map[string]any, len(params)+len(state.Fields))
	for name, value := range params {
		values[name] = value
	}
	for name, value := range state.Fields {
		values[name] = value
	}
	return template.Render(values)
}
