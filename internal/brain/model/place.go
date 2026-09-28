package model

import (
	"context"
	_ "embed"
	"encoding/json"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// PlaceVersion names the prompt that asked for a file's target area.
const PlaceVersion = "ablage-v1"

//go:embed prompts/ablage-v1.md
var placePrompt string

// placeMaxChars is the head the rig measured ablage-v1 on (local.py:48-51).
const placeMaxChars = 1800

// placeSchema is the rig's _ABLAGE_SCHEMA; the endpoint enforces it, the
// prompt only names it.
func placeSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scope": map[string]any{"type": "string"},
			"grund": map[string]any{"type": "string"},
		},
		"required": []string{"scope", "grund"},
	}
}

// Place is one scope from scopes, or none, and the file stays in its inbox.
// A well-formed scope the register does not know is no answer.
func (p *Proposer) Place(ctx context.Context, text string, scopes []string) (string, bool) {
	listed := make([]string, len(scopes))
	for i, scope := range scopes {
		listed[i] = "- " + scope
	}
	prompt := strings.NewReplacer("{scopes}", strings.Join(listed, "\n"), "{text}", pytext.FirstRunes(text, placeMaxChars)).Replace(placePrompt)
	answer, ok := p.client.AskFormat(ctx, prompt, placeSchema())
	if !ok {
		return "", false
	}
	var read any
	if err := json.Unmarshal([]byte(answer), &read); err != nil {
		return "", false
	}
	object, _ := read.(map[string]any)
	scope, _ := object["scope"].(string)
	if !slices.Contains(scopes, scope) {
		return "", false
	}
	return scope, true
}
