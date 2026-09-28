package model

import (
	"context"
	_ "embed"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// DescribeVersion names the prompt that asked for a file head's sentence.
const DescribeVersion = "beschreibung-v1"

//go:embed prompts/beschreibung-v1.md
var describePrompt string

// The cut and the length the rig measured beschreibung-v1 on (local.py:24-36).
const (
	describeMaxChars = 1800
	describeMaxWords = 22
)

// Describe is the one sentence for a file head, or none, and the head keeps
// four lines. Five rules, each enough to refuse; there is no fallback to the
// document's first line, which would stand in the head as if measured.
func (p *Proposer) Describe(ctx context.Context, text string) (string, bool) {
	answer, ok := p.client.Ask(ctx, strings.ReplaceAll(describePrompt, "{text}", pytext.FirstRunes(text, describeMaxChars)))
	if !ok {
		return "", false
	}
	sentence := pytext.Strip(answer)
	if !IsOneSentence(sentence) || WordCount(sentence) > describeMaxWords || !IsGerman(sentence) ||
		len(ChoppedWords(sentence)) > 0 || !readsBack(sentence) {
		return "", false
	}
	return sentence, true
}
