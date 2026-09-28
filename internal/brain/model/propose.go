package model

import (
	"context"
	_ "embed"
	"strings"

	"github.com/xidus90/loomux/internal/brain/evidence"
	"github.com/xidus90/loomux/internal/config"
)

// ProposeVersion names the prompt in every case it produced: of four measured
// versions two made a tier worse while reading like improvements, so a fallen
// hit rate must be attributable to the prompt or the model later.
const ProposeVersion = "vorschlag-v4"

// proposePrompt is the reference's file byte for byte, its version comment
// included: the whole file goes to the model, as `load().format()` sends it.
//
//go:embed prompts/vorschlag-v4.md
var proposePrompt string

// Proposer is the local model in its roles `propose`, `describe` and
// `place`; the gate hands one out per role.
type Proposer struct {
	client *Client
}

// ProposerFor is the gate: a proposer for this area and role, or none. No
// client -- no address set up -- exists unless the role is on after the
// area's word; only then is the endpoint judged. The privacy mode is not
// asked here: it decides what happens to a proposal, and the caller hands
// out proposers only for closed areas.
func ProposerFor(s config.ModelSettings, m *config.Manifest, role string) (*Proposer, error) {
	narrow := s.Narrowed(m)
	if !narrow.RoleOn(role) {
		return nil, nil
	}
	client, err := NewClient(narrow)
	if err != nil {
		return nil, err
	}
	return &Proposer{client: client}, nil
}

// Propose asks for a proposal on pkg and keeps it only if every claim passes
// the evidence binding `approve` holds it to later. No answer, nothing
// readable as a claim, one invented quote among good ones: all three are no
// proposal, and the caller cannot tell them apart -- on purpose.
func (p *Proposer) Propose(ctx context.Context, pkg string, segments []evidence.Segment) (string, bool) {
	answer, ok := p.client.Ask(ctx, strings.ReplaceAll(proposePrompt, "{paket}", pkg))
	if !ok {
		return "", false
	}
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal(answer), segments)
	if len(passed) == 0 || len(complaints) > 0 {
		return "", false
	}
	return answer, true
}
