package model

import (
	"context"
	"fmt"
	"slices"
)

// Answer is one prepared outcome for the Fake: a reply, or an error to raise.
// When Err is set, Reply is not handed out.
type Answer struct {
	Reply Reply
	Err   error
}

// Fake answers from a queue and records every request it was handed. Tests
// construct it directly; no configuration selects it.
type Fake struct {
	pending []Answer
	seen    []Request
}

// NewFake returns a Fake that hands out answers in order.
func NewFake(answers ...Answer) *Fake {
	return &Fake{pending: answers}
}

// Ask records the request and returns the next prepared answer.
func (f *Fake) Ask(_ context.Context, request Request) (Reply, error) {
	f.seen = append(f.seen, request)
	if len(f.pending) == 0 {
		return Reply{}, fmt.Errorf("no reply left for %q", request.Prompt)
	}
	next := f.pending[0]
	f.pending = f.pending[1:]
	if next.Err != nil {
		return Reply{}, next.Err
	}
	return next.Reply, nil
}

// Seen is every request so far, in order, as a copy.
func (f *Fake) Seen() []Request {
	return slices.Clone(f.seen)
}
