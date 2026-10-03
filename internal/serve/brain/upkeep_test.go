package brain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/privacy"
	servebrain "github.com/xidus90/loomux/internal/serve/brain"
)

// fakeUpkeep is an upkeep whose gate the test opens and whose trailer it
// names.
type fakeUpkeep struct {
	gate     chan struct{}
	settled  bool
	lines    []string
	err      error
	waitErr  error
	asked    []string
	channels []privacy.Channel
}

func (f *fakeUpkeep) Settled() bool { return f.settled }

func (f *fakeUpkeep) CaughtUp(ctx context.Context) error {
	if f.waitErr != nil {
		return f.waitErr
	}
	if f.gate != nil {
		<-f.gate
	}
	return nil
}

func (f *fakeUpkeep) Trailer(cmd string, ch privacy.Channel) ([]string, error) {
	f.asked = append(f.asked, cmd)
	f.channels = append(f.channels, ch)
	return f.lines, f.err
}

func answering(text string, err error) func(answer.Request, string, func(string)) (string, []string, error) {
	return func(answer.Request, string, func(string)) (string, []string, error) {
		return text, nil, err
	}
}

func callRead(t *testing.T, session *mcp.ClientSession, meta mcp.Meta) *mcp.CallToolResult {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "brain_read", Meta: meta, Arguments: map[string]any{"relative": "a.md"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func TestTheTrailerIsAppendedToTheAnswersText(t *testing.T) {
	upkeep := &fakeUpkeep{settled: true, lines: []string{"! one", "! two"}}
	session := connect(t, privacy.ChannelCloud, servebrain.Deps{Answer: answering("page", nil), Upkeep: upkeep})
	res := callRead(t, session, nil)
	if got := res.Content[0].(*mcp.TextContent).Text; got != "page\n! one\n! two" || len(res.Content) != 1 {
		t.Fatalf("text %q", got)
	}
	if len(upkeep.asked) != 1 || upkeep.asked[0] != "read" || upkeep.channels[0] != privacy.ChannelCloud {
		t.Fatalf("asked %v on %v", upkeep.asked, upkeep.channels)
	}
}

func TestAnErrorAnswerCarriesNoTrailer(t *testing.T) {
	upkeep := &fakeUpkeep{settled: true, lines: []string{"! one"}}
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{Answer: answering("", errors.New("unknown scope")), Upkeep: upkeep})
	res := callRead(t, session, nil)
	if got := res.Content[0].(*mcp.TextContent).Text; !res.IsError || got != "unknown scope" || len(upkeep.asked) != 0 {
		t.Fatalf("error %v, text %q, asked %v", res.IsError, got, upkeep.asked)
	}
}

// A trailer that cannot be built costs the trailer and not the answer: the
// answer is ready and right, and the notes are only what surrounds it. Its
// cause names host paths more often than not, so only the local channel reads
// it, as with the failure line of the upkeep itself.
func TestATrailerThatCannotBeBuiltKeepsTheAnswer(t *testing.T) {
	for ch, want := range map[privacy.Channel]string{
		privacy.ChannelLocal: "page\n! the notes of the daily reconciliation could not be read: " +
			`C:\state\registry.toml: not valid TOML`,
		privacy.ChannelCloud: "page\n! the notes of the daily reconciliation could not be read (the cause is named on the local channel)",
	} {
		upkeep := &fakeUpkeep{settled: true, err: errors.New(`C:\state\registry.toml: not valid TOML`)}
		session := connect(t, ch, servebrain.Deps{Answer: answering("page", nil), Upkeep: upkeep})
		res := callRead(t, session, nil)
		if got := res.Content[0].(*mcp.TextContent).Text; res.IsError || got != want {
			t.Errorf("%s: error %v, text %q", ch, res.IsError, got)
		}
	}
}

func TestTheFirstAnswerWaitsForTheCatchUpAndSaysSo(t *testing.T) {
	sink := newProgressSink()
	upkeep := &fakeUpkeep{gate: make(chan struct{})}
	session := connectWith(t, privacy.ChannelLocal, servebrain.Deps{Answer: answering("page", nil), Upkeep: upkeep}, sink.options())
	done := make(chan *mcp.CallToolResult)
	go func() { done <- callRead(t, session, mcp.Meta{"progressToken": "t"}) }()
	if got := sink.next(t); got != servebrain.CatchUpNotice {
		t.Fatalf("notice %q", got)
	}
	select {
	case <-done:
		t.Fatal("the answer came before the catch-up")
	default:
	}
	close(upkeep.gate)
	if res := <-done; res.Content[0].(*mcp.TextContent).Text != "page" {
		t.Fatalf("answer %q", res.Content[0].(*mcp.TextContent).Text)
	}
}

func TestASettledUpkeepSaysNothingBeforeTheAnswer(t *testing.T) {
	sink := newProgressSink()
	upkeep := &fakeUpkeep{settled: true}
	session := connectWith(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(_ answer.Request, _ string, notice func(string)) (string, []string, error) {
			notice("from the answer")
			return "page", nil, nil
		},
		Upkeep: upkeep,
	}, sink.options())
	callRead(t, session, mcp.Meta{"progressToken": "t"})
	if got := sink.next(t); got != "from the answer" {
		t.Fatalf("first notice %q, want the answer's own", got)
	}
}

func TestAWaitThatEndsWithoutTheCatchUpFailsTheCall(t *testing.T) {
	upkeep := &fakeUpkeep{waitErr: context.Canceled}
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{Answer: answering("page", nil), Upkeep: upkeep})
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "brain_read", Arguments: map[string]any{"relative": "a.md"},
	}); err == nil {
		t.Fatal("a call answered without the catch-up")
	}
}
