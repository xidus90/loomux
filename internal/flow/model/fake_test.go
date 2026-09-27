package model_test

import (
	"context"
	"errors"
	"testing"

	"github.com/xidus90/loomux/internal/flow/model"
)

var _ model.Model = (*model.Fake)(nil)

func TestFakeAnswersInOrderAndRecordsEveryRequest(t *testing.T) {
	fake := model.NewFake(
		model.Answer{Reply: model.Reply{Fields: map[string]any{"verdict": "open"}, Tokens: 5}},
		model.Answer{Reply: model.Reply{Fields: map[string]any{"verdict": "done"}, Tokens: 7}},
	)
	first, err := fake.Ask(context.Background(), model.Request{Prompt: "one"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := fake.Ask(context.Background(), model.Request{Prompt: "two"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Fields["verdict"] != "open" || second.Tokens != 7 {
		t.Fatalf("answers out of order: %+v, %+v", first, second)
	}
	seen := fake.Seen()
	if len(seen) != 2 || seen[0].Prompt != "one" || seen[1].Prompt != "two" {
		t.Fatalf("requests not recorded in order: %+v", seen)
	}
}

// A queued error is raised, and the reply beside it is not handed out: error
// paths are as testable as happy ones.
func TestFakeRaisesAQueuedError(t *testing.T) {
	refused := errors.New("refused")
	fake := model.NewFake(model.Answer{Reply: model.Reply{Tokens: 99}, Err: refused})
	reply, err := fake.Ask(context.Background(), model.Request{Prompt: "p"})
	if !errors.Is(err, refused) {
		t.Fatalf("want the queued error, got %v", err)
	}
	if reply.Tokens != 0 {
		t.Fatalf("a failed ask returns no reply, got %+v", reply)
	}
}

func TestFakeWithNothingLeftSaysForWhichPrompt(t *testing.T) {
	_, err := model.NewFake().Ask(context.Background(), model.Request{Prompt: "hello"})
	if err == nil || err.Error() != `no reply left for "hello"` {
		t.Fatalf("got %v", err)
	}
}

func TestSeenIsACopy(t *testing.T) {
	fake := model.NewFake(model.Answer{})
	if _, err := fake.Ask(context.Background(), model.Request{Prompt: "p"}); err != nil {
		t.Fatal(err)
	}
	fake.Seen()[0].Prompt = "changed"
	if fake.Seen()[0].Prompt != "p" {
		t.Fatal("a caller must not rewrite what the fake recorded")
	}
}
