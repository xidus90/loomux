package hosts

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// readAntigravity decodes an Antigravity hook payload.
//
// Antigravity (agy 1.2.2) emits protojson payloads for command hooks.
// The session identifier is delivered in `conversationId` (standard protojson camelCase)
// or `conversation_id`.
//
// Like readClaude, it refuses what is not an object (non-JSON, array, string, number, null).
// Fields are read by type assertion so mistyped values read as absent ("").
//
// For SubagentStart/SubagentStop: Antigravity passes toolCall in PreToolUse,
// but only stepIdx, error, and conversationId in PostToolUse -- no agent_id.
// Thus AgentID and AgentType are returned as empty (""), causing subagent-start/stop
// to reject with exit 1 ("payload carries no agent_id").
func readAntigravity(r io.Reader) (Payload, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return Payload{}, fmt.Errorf("reading stdin: %w", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		var wrongType *json.UnmarshalTypeError
		if errors.As(err, &wrongType) {
			return Payload{}, errNotAnObject
		}
		return Payload{}, fmt.Errorf("stdin is not JSON: %w", err)
	}
	if payload == nil {
		return Payload{}, errNotAnObject
	}
	sessionID, _ := payload["conversationId"].(string)
	if sessionID == "" {
		sessionID, _ = payload["conversation_id"].(string)
	}
	return Payload{SessionID: sessionID}, nil
}

type antigravityAnswer struct {
	InjectSteps []antigravityStep `json:"injectSteps"`
}

type antigravityStep struct {
	EphemeralMessage string `json:"ephemeralMessage"`
}

// writeAntigravityContext writes lines as an ephemeral message in an
// injectSteps array, the format Antigravity reads from a hook on stdout.
func writeAntigravityContext(w io.Writer, _ string, lines []string) error {
	answer := antigravityAnswer{
		InjectSteps: []antigravityStep{
			{EphemeralMessage: strings.Join(lines, "\n")},
		},
	}
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(answer); err != nil {
		return fmt.Errorf("antigravity: writing context: %w", err)
	}
	return nil
}
