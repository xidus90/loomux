package cases

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The two grades of the MCP comparison. They are not the byte comparison's
// grades with another name: what a recorded MCP case holds is one tool call
// and one CallToolResult, never a command line and a stream of stdout.
//
// The envelope is deliberately outside both. The reference speaks through the
// Python MCP SDK and loomux through the Go one, so initialize alone differs in
// capabilities, in serverInfo and in the revision the two negotiate. Comparing
// that would compare the two libraries; the specification is what the envelope
// is held against, and it is held against it elsewhere.
const (
	// CompareText pins the text of the result and isError.
	CompareText = "text"
	// CompareOutcome pins isError alone. The wording of a refusal is loomux's
	// own, exactly as a `message` case's stdout is.
	CompareOutcome = "outcome"
)

// MCPCall is one tool call as a case records it. Channel is the address the
// call goes to, never an argument of the call itself.
type MCPCall struct {
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
	Channel   string          `json:"channel"`
}

// MCPResult is what the reference answered: the text of the CallToolResult,
// whether it was an error result, and -- instead of both -- the message of a
// JSON-RPC error, for a call the reference never turned into a result at all.
type MCPResult struct {
	IsError bool   `json:"isError"`
	Text    string `json:"text"`
	// RPCError is empty unless the reference answered with a protocol error.
	RPCError string `json:"rpcError,omitempty"`
}

// MCPOutcome is the same three things, as one run produced them.
type MCPOutcome struct {
	IsError  bool
	Text     string
	RPCError string
}

// MCPCase is one recorded call of the MCP front.
type MCPCase struct {
	Verb    string
	Name    string
	Path    string
	Call    MCPCall
	Result  MCPResult
	Notes   string
	Compare string
}

// LoadMCPCase loads a single MCP case from its directory.
func LoadMCPCase(dir string) (*MCPCase, error) {
	cleanDir := filepath.Clean(dir)

	worldInfo, err := os.Stat(filepath.Join(cleanDir, "world"))
	if err != nil || !worldInfo.IsDir() {
		return nil, fmt.Errorf("missing world directory in %s", cleanDir)
	}

	callBytes, err := os.ReadFile(filepath.Join(cleanDir, "call"))
	if err != nil {
		return nil, fmt.Errorf("missing call in %s: %w", cleanDir, err)
	}
	var call MCPCall
	if err := json.Unmarshal(callBytes, &call); err != nil {
		return nil, fmt.Errorf("invalid call in %s: %w", cleanDir, err)
	}
	if call.Tool == "" {
		return nil, fmt.Errorf("empty tool in %s", cleanDir)
	}
	// The protocol carries an object, and a call that names none would travel
	// as a null. A tool without arguments is called with an empty object, and
	// that is what a recording has to write down.
	if len(call.Arguments) == 0 {
		return nil, fmt.Errorf("missing arguments in %s", cleanDir)
	}

	resultBytes, err := os.ReadFile(filepath.Join(cleanDir, "result"))
	if err != nil {
		return nil, fmt.Errorf("missing result in %s: %w", cleanDir, err)
	}
	var result MCPResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, fmt.Errorf("invalid result in %s: %w", cleanDir, err)
	}

	compare := CompareText
	if data, err := os.ReadFile(filepath.Join(cleanDir, "compare")); err == nil {
		compare = strings.TrimSpace(string(data))
		if compare != CompareText && compare != CompareOutcome {
			return nil, fmt.Errorf("unknown compare %q in %s", compare, cleanDir)
		}
	}

	var notes string
	if data, err := os.ReadFile(filepath.Join(cleanDir, "notes.md")); err == nil {
		notes = string(data)
	}

	return &MCPCase{
		Verb:    filepath.Base(filepath.Dir(cleanDir)),
		Name:    filepath.Base(cleanDir),
		Path:    cleanDir,
		Call:    call,
		Result:  result,
		Notes:   notes,
		Compare: compare,
	}, nil
}

// DiscoverMCPCases scans root recursively for MCP cases. A directory holding a
// `call` file is one, and nothing below it is looked at.
func DiscoverMCPCases(root string) ([]*MCPCase, error) {
	var results []*MCPCase
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if info, err := os.Stat(filepath.Join(path, "call")); err != nil || info.IsDir() {
			return nil
		}
		c, err := LoadMCPCase(path)
		if err != nil {
			return err
		}
		results = append(results, c)
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Verb != results[j].Verb {
			return results[i].Verb < results[j].Verb
		}
		return results[i].Name < results[j].Name
	})
	return results, nil
}

// MCPRunFunc is the entry point under test: one tool call in one staged world.
type MCPRunFunc func(call MCPCall, world string) (MCPOutcome, error)

// MCPRunOutcome describes the outcome of running a single MCP case.
type MCPRunOutcome struct {
	Case       *MCPCase
	Passed     bool
	Actual     MCPOutcome
	Mismatches []string
}

// RunMCPCase runs a single MCP case in an isolated staged world.
func RunMCPCase(c *MCPCase, run MCPRunFunc) (*MCPRunOutcome, error) {
	tmpDir, err := mkdirTemp("", "mcp-case-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	if err := StageWorld(filepath.Join(c.Path, "world"), tmpDir); err != nil {
		return nil, err
	}
	world := filepath.ToSlash(tmpDir)

	call := c.Call
	call.Arguments = json.RawMessage(strings.ReplaceAll(string(call.Arguments), WorldToken, world))
	actual, err := run(call, tmpDir)
	if err != nil {
		return nil, err
	}
	actual.Text = string(Normalize([]byte(actual.Text), tmpDir))
	actual.RPCError = string(Normalize([]byte(actual.RPCError), tmpDir))

	mismatches := compareMCP(c, actual)
	return &MCPRunOutcome{
		Case:       c,
		Passed:     len(mismatches) == 0,
		Actual:     actual,
		Mismatches: mismatches,
	}, nil
}

// compareMCP is the comparison itself: what a run answered against what the
// reference answered, at the grade the case asks for.
func compareMCP(c *MCPCase, actual MCPOutcome) []string {
	var mismatches []string
	if (c.Result.RPCError != "") != (actual.RPCError != "") {
		mismatches = append(mismatches, fmt.Sprintf("protocol error: expected %q, got %q",
			c.Result.RPCError, actual.RPCError))
	}
	if c.Result.IsError != actual.IsError {
		mismatches = append(mismatches, fmt.Sprintf("isError: expected %t, got %t",
			c.Result.IsError, actual.IsError))
	}
	// An outcome case pins isError alone: its wording is loomux's own.
	if c.Compare != CompareOutcome && c.Result.Text != actual.Text {
		mismatches = append(mismatches, fmt.Sprintf("text mismatch:\nexpected %q\ngot      %q",
			c.Result.Text, actual.Text))
	}
	return mismatches
}
