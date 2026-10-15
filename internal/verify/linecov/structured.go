package linecov

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/BurntSushi/toml"
)

func readTOML(content []byte, w Where) (Values, bool, error) {
	var doc map[string]any
	if _, err := toml.Decode(text(content), &doc); err != nil {
		return nil, false, err
	}
	return section(doc, splitDots(w.Table))
}

func readJSON(content []byte, w Where) (Values, bool, error) {
	var doc any
	if err := json.Unmarshal([]byte(text(content)), &doc); err != nil {
		return nil, false, err
	}
	return section(doc, splitDots(w.Path))
}

func splitDots(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ".")
}

// section walks keys down nested maps and flattens the scalars and scalar
// lists it finds there; nested tables are not values.
func section(doc any, keys []string) (Values, bool, error) {
	cur, ok := doc.(map[string]any)
	for _, k := range keys {
		if !ok {
			break
		}
		cur, ok = cur[k].(map[string]any)
	}
	if !ok {
		return Values{}, false, nil
	}
	vals := Values{}
	for k, v := range cur {
		vals.add(k, scalars(v)...)
	}
	return vals, true, nil
}

func scalars(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case bool, int64, float64:
		return []string{fmt.Sprint(x)}
	case []any:
		var out []string
		for _, e := range x {
			out = append(out, scalars(e)...)
		}
		return out
	}
	return nil
}

type xmlNode struct {
	name  string
	attrs []xml.Attr
	text  strings.Builder
	kids  []*xmlNode
}

func parseXML(content []byte) (*xmlNode, error) {
	d := xml.NewDecoder(bytes.NewReader([]byte(text(content))))
	top := &xmlNode{}
	stack := []*xmlNode{top}
	for {
		tok, err := d.Token()
		if err != nil {
			// Clean end of input; a truncated document is a syntax error instead.
			if errors.Is(err, io.EOF) {
				if len(top.kids) != 1 {
					return nil, errors.New("unreadable XML: want exactly one root element")
				}
				return top.kids[0], nil
			}
			return nil, fmt.Errorf("unreadable XML: %w", err)
		}
		switch x := tok.(type) {
		case xml.StartElement:
			n := &xmlNode{name: x.Name.Local, attrs: x.Attr}
			parent := stack[len(stack)-1]
			parent.kids = append(parent.kids, n)
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			stack[len(stack)-1].text.Write(x)
		}
	}
}

// readXML selects every element on w.Path (slash names from the root
// element) and reads what lies below: a leaf's text under its relative path,
// an attribute as <path>@<name>.
func readXML(content []byte, w Where) (Values, bool, error) {
	root, err := parseXML(content)
	if err != nil {
		return nil, false, err
	}
	steps := strings.Split(w.Path, "/")
	if steps[0] != root.name {
		return Values{}, false, nil
	}
	nodes := []*xmlNode{root}
	for _, step := range steps[1:] {
		var next []*xmlNode
		for _, n := range nodes {
			for _, k := range n.kids {
				if k.name == step {
					next = append(next, k)
				}
			}
		}
		nodes = next
	}
	if len(nodes) == 0 {
		return Values{}, false, nil
	}
	vals := Values{}
	for _, n := range nodes {
		collect(n, "", vals)
	}
	return vals, true, nil
}

func collect(n *xmlNode, prefix string, vals Values) {
	for _, k := range n.kids {
		key := strings.TrimPrefix(prefix+"/"+k.name, "/")
		if len(k.kids) == 0 {
			vals.add(key, splitList(k.text.String())...)
		}
		for _, a := range k.attrs {
			vals.add(key+"@"+a.Name.Local, a.Value)
		}
		collect(k, key, vals)
	}
}
