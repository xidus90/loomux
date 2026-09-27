package hosts

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// readAntigravity decodes an Antigravity hook payload.
//
// Antigravity (agy 1.2.2) emits protojson payloads for command hooks.
// invocationNum was measured 2026-09-27 on agy 1.2.11: present on
// PreInvocation as a JSON number, 0 on the first model call, +1 per call.
// protojson writes a 64-bit integer as a decimal string and a 32-bit one as a
// number; the field's width, and with it the string form, stay unmeasured, so
// invocationOf takes both. The session identifier is delivered in
// `conversationId` (standard protojson camelCase) or `conversation_id`.
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
	return Payload{SessionID: sessionID, Repeat: invocationOf(payload["invocationNum"]) > 0}, nil
}

// invocationOf reads invocationNum in either of protojson's spellings: a JSON
// number, or a decimal string, which is how it writes a 64-bit integer. Any
// other value, and a string that is no whole number or overflows 64 bits, is
// 0; a Repeat read false by mistake only does the first start's work once
// more: its announcements, and sessions.Revive, harmless on a conversation
// nothing retired.
func invocationOf(v any) int64 {
	switch v := v.(type) {
	case float64:
		return int64(v)
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}
		return n
	}
	return 0
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
