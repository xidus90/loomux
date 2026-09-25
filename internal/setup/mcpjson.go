package setup

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/xidus90/loomux/internal/setup/hostfile"
)

// mcpPath is the project-scope MCP file of Claude Code.
const mcpPath = ".mcp.json"

// mcpServer is the entry init adds; a struct so command stands before args.
type mcpServer struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// ourServer calls the binary at the path the host entries call, so both
// reach the same file.
func ourServer() mcpServer {
	return mcpServer{Command: "${LOCALAPPDATA}/loomux/bin/loomux.exe", Args: []string{"mcp", "--channel", "local"}}
}

// member is one key of a JSON object with its value as written.
type member struct {
	key string
	raw json.RawMessage
}

// withMCPServer is existing with the server loomux under mcpServers, or
// existing itself when that key is already there. Every other key and
// every other server keeps its place and its text; only the indentation is
// made uniform. A file that is no object, or whose mcpServers is none, is
// refused rather than repaired.
func withMCPServer(existing []byte) ([]byte, error) {
	var top []member
	var err error
	if len(bytes.TrimSpace(existing)) > 0 {
		if top, err = members(existing); err != nil {
			return nil, fmt.Errorf("%s is not a JSON object: %w", mcpPath, err)
		}
	}
	var servers []member
	at := -1
	for i, m := range top {
		if m.key != "mcpServers" {
			continue
		}
		if servers, err = members(m.raw); err != nil {
			return nil, fmt.Errorf("%s: mcpServers is not an object", mcpPath)
		}
		at = i
	}
	for _, s := range servers {
		if s.key == "loomux" {
			return existing, nil
		}
	}
	// A struct of strings always encodes.
	ours := hostfile.EncodeJSON(ourServer(), "", "")
	servers = append(servers, member{"loomux", ours})
	block := object(servers)
	if at < 0 {
		top = append(top, member{"mcpServers", block})
	} else {
		top[at].raw = block
	}
	// object's output is valid JSON, so neither Compact nor Indent can fail
	// on it; Compact first, because Indent keeps the line breaks it finds.
	var compact, out bytes.Buffer
	_ = json.Compact(&compact, object(top))
	_ = json.Indent(&out, compact.Bytes(), "", "  ")
	out.WriteString("\n")
	return out.Bytes(), nil
}

// members reads the keys of the JSON object data in file order.
func members(data []byte) ([]member, error) {
	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid JSON")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, _ := dec.Token(); tok != json.Delim('{') {
		return nil, fmt.Errorf("not an object")
	}
	var out []member
	for dec.More() {
		// The whole document is valid, so a key and its value follow.
		tok, _ := dec.Token()
		var raw json.RawMessage
		_ = dec.Decode(&raw)
		out = append(out, member{tok.(string), raw})
	}
	return out, nil
}

// object writes members as one compact JSON object.
func object(ms []member) json.RawMessage {
	var b bytes.Buffer
	b.WriteString("{")
	for i, m := range ms {
		if i > 0 {
			b.WriteString(",")
		}
		key := hostfile.EncodeJSON(m.key, "", "")
		b.Write(key)
		b.WriteString(":")
		b.Write(m.raw)
	}
	b.WriteString("}")
	return b.Bytes()
}
