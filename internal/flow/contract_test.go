package flow_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/xidus90/loomux/internal/flow"
)

// Doubles for every contract. They exist so that a signature that moves breaks
// the build here, in the package that owns it, and not in every package that
// implements it at once.
type stubBlock struct{}

func (stubBlock) Kind() string                          { return "stub" }
func (stubBlock) Check(flow.Node) []string              { return nil }
func (stubBlock) Texts(flow.Node) []flow.Text           { return nil }
func (stubBlock) Writes(flow.Node) map[string]flow.Type { return nil }
func (stubBlock) NeedsBaseline() bool                   { return false }

func (stubBlock) Define(*flow.Graph, flow.Node, flow.Env) (flow.Definition, error) {
	return flow.Definition{}, nil
}

func (stubBlock) Run(context.Context, *flow.Graph, flow.Node, flow.State, flow.Env) (flow.Result, error) {
	return flow.Result{}, nil
}

type stubRegistry struct{}

func (stubRegistry) Block(string) (flow.Block, bool)         { return stubBlock{}, true }
func (stubRegistry) Predicate(string) (flow.Predicate, bool) { return nil, false }

type always struct{}

func (always) Holds(flow.State, flow.Params) bool { return true }

var (
	_ flow.Block     = stubBlock{}
	_ flow.Registry  = stubRegistry{}
	_ flow.Condition = always{}
	_ flow.Predicate = func(flow.State, flow.Params) bool { return true }
)

// A refused gate answer has to stay recognisable through wrapping: the runner
// tells it apart from a node failure, which would take an error edge instead of
// leaving the gate open.
func TestInvalidAnswerSurvivesWrapping(t *testing.T) {
	wrapped := fmt.Errorf("%w: pick one of yes, no", flow.ErrInvalidAnswer)
	if !errors.Is(wrapped, flow.ErrInvalidAnswer) {
		t.Fatal("a wrapped invalid answer must still read as one")
	}
}

// Flow files spell the end of a run END; the loader and the runner compare
// against this constant and nothing else.
func TestEndIsSpelledAsInFlowFiles(t *testing.T) {
	if flow.End != "END" {
		t.Fatalf("End = %q, flow files write END", flow.End)
	}
}
