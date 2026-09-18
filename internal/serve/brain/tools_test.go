package brain_test

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/privacy"
	servebrain "github.com/xidus90/loomux/internal/serve/brain"
)

// connect wires a server with the five tools to an in-memory client. No socket,
// no port, no waiting.
func connect(t *testing.T, channel privacy.Channel, deps servebrain.Deps) *mcp.ClientSession {
	t.Helper()
	return connectWith(t, channel, deps, nil)
}

// connectWith is connect with client options, so that a test can listen for the
// progress notifications the hints become.
func connectWith(t *testing.T, channel privacy.Channel, deps servebrain.Deps, opts *mcp.ClientOptions) *mcp.ClientSession {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "loomux", Version: "test"}, nil)
	servebrain.Register(server, channel, deps)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	// Connect returns the server session too; the test needs only the error.
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, opts)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

// progressSink collects the progress notifications of one session.
type progressSink struct {
	messages chan string
}

func newProgressSink() *progressSink {
	return &progressSink{messages: make(chan string, 8)}
}

func (s *progressSink) options() *mcp.ClientOptions {
	return &mcp.ClientOptions{
		ProgressNotificationHandler: func(_ context.Context, req *mcp.ProgressNotificationClientRequest) {
			s.messages <- req.Params.Message
		},
	}
}

// next waits for one notification. A notification travels on its own, so it can
// arrive after the answer did.
func (s *progressSink) next(t *testing.T) string {
	t.Helper()
	select {
	case message := <-s.messages:
		return message
	case <-time.After(5 * time.Second):
		t.Fatal("no progress notification arrived")
		return ""
	}
}

func TestTheFiveToolsAreListed(t *testing.T) {
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(answer.Request, string, string, func(string)) (string, []string, error) {
			return "", nil, nil
		},
	})
	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	// As a set, not in order: the SDK keeps its tools in a featureSet and lists
	// them sorted by name (features.go, all/sortKeys), so the canonical order of
	// mcptools.Tools is not observable through tools/list.
	want := []string{"brain_catalog", "brain_neighbors", "brain_read", "brain_search", "brain_status"}
	got := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		got = append(got, tool.Name)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("the tools are %v, want %v", got, want)
	}
}

func TestTheListenersChannelReachesTheAnswer(t *testing.T) {
	var seen privacy.Channel
	session := connect(t, privacy.ChannelCloud, servebrain.Deps{
		Answer: func(req answer.Request, _, _ string, _ func(string)) (string, []string, error) {
			seen = req.Channel
			return "ok", nil, nil
		},
	})
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if seen != privacy.ChannelCloud {
		t.Errorf("the answer saw channel %q, want cloud", seen)
	}
}

func TestArgumentsReachTheAnswer(t *testing.T) {
	var seen answer.Request
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(req answer.Request, _, _ string, _ func(string)) (string, []string, error) {
			seen = req
			return "ok", nil, nil
		},
	})
	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "brain_search",
		Arguments: map[string]any{
			"query":   "zettel",
			"scope":   "wiki",
			"profile": "full",
			"n":       3,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if seen.Command != "search" || seen.Query != "zettel" || seen.Scope != "wiki" ||
		seen.Profile != "full" || seen.Count != 3 {
		t.Errorf("the answer saw %+v", seen)
	}
}

func TestTheDirectoriesReachTheAnswer(t *testing.T) {
	var registryDir, legacyDir string
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		RegistryDir: "registry",
		LegacyDir:   "legacy",
		Answer: func(_ answer.Request, registry, legacy string, _ func(string)) (string, []string, error) {
			registryDir, legacyDir = registry, legacy
			return "ok", nil, nil
		},
	})
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if registryDir != "registry" || legacyDir != "legacy" {
		t.Errorf("the answer saw %q and %q", registryDir, legacyDir)
	}
}

// The channel is the listener's, and an argument that names one is a claim: a
// cloud listener stays cloud however the caller asks.
func TestACallerCannotNameTheChannel(t *testing.T) {
	var seen privacy.Channel
	session := connect(t, privacy.ChannelCloud, servebrain.Deps{
		Answer: func(req answer.Request, _, _ string, _ func(string)) (string, []string, error) {
			seen = req.Channel
			return "ok", nil, nil
		},
	})
	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "brain_catalog",
		Arguments: map[string]any{"channel": "local"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if seen != privacy.ChannelCloud {
		t.Errorf("the answer saw channel %q, want cloud", seen)
	}
}

// The relative path is the same positional as the query, so read and neighbors
// must arrive in Query.
func TestTheRelativePathIsThePositional(t *testing.T) {
	var seen answer.Request
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(req answer.Request, _, _ string, _ func(string)) (string, []string, error) {
			seen = req
			return "ok", nil, nil
		},
	})
	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "brain_read",
		Arguments: map[string]any{"scope": "wiki", "relative": "note.md", "section": "Intro"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if seen.Command != "read" || seen.Query != "note.md" || seen.Section != "Intro" {
		t.Errorf("the answer saw %+v", seen)
	}
}

// Nothing validates arguments against the schema on this path, so a read that
// also carries a query really can arrive. The command picks the positional.
func TestAStrayQueryDoesNotOverrideTheRelativePath(t *testing.T) {
	var seen answer.Request
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(req answer.Request, _, _ string, _ func(string)) (string, []string, error) {
			seen = req
			return "ok", nil, nil
		},
	})
	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "brain_read",
		Arguments: map[string]any{"scope": "wiki", "relative": "note.md", "query": "../../etc/passwd"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if seen.Query != "note.md" {
		t.Errorf("the read used %q as its positional", seen.Query)
	}
}

// Arguments that are not a JSON object end as an empty map. For a tool that
// needs no argument of its own that is an ordinary call, never an outage.
func TestArgumentsThatAreNotAnObjectEndEmpty(t *testing.T) {
	var seen answer.Request
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(req answer.Request, _, _ string, _ func(string)) (string, []string, error) {
			seen = req
			return "ok", nil, nil
		},
	})
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "brain_catalog",
		Arguments: []any{1},
	})
	if err != nil {
		t.Fatalf("broken arguments must not be an outage: %v", err)
	}
	if res.IsError {
		t.Error("broken arguments must not be a refusal either")
	}
	// Without a scope the root catalog: "all" is the default of both fronts.
	// The ten is count's default, which every tool call carries and only
	// search reads.
	if seen.Command != "catalog" || seen.Scope != "all" || seen.Count != 10 {
		t.Errorf("the answer saw %+v", seen)
	}
}

// The one argument a tool cannot do without is refused when it is not there,
// rather than answered as the empty string. A search for "" came back "no
// matches", which reads to a model as "nothing found" for a question nobody
// asked; the reference refuses the same call.
func TestACallWithoutItsOneRequiredArgumentIsRefused(t *testing.T) {
	for _, tc := range []struct{ tool, want string }{
		{"brain_search", "query is required"},
		{"brain_read", "relative is required"},
		{"brain_neighbors", "relative is required"},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			asked := false
			session := connect(t, privacy.ChannelLocal, servebrain.Deps{
				Answer: func(answer.Request, string, string, func(string)) (string, []string, error) {
					asked = true
					return "ok", nil, nil
				},
			})
			res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      tc.tool,
				Arguments: map[string]any{"scope": "notes"},
			})
			if err != nil {
				t.Fatalf("a missing argument must not be an outage: %v", err)
			}
			if !res.IsError {
				t.Fatal("a missing argument must be a refusal")
			}
			if text, ok := res.Content[0].(*mcp.TextContent); !ok || text.Text != tc.want {
				t.Errorf("said %+v", res.Content[0])
			}
			if asked {
				t.Error("the answer was asked although the call was incomplete")
			}
		})
	}
}

// The three answers a formatter ends for stdout lose that one terminator on the
// way into a CallToolResult; a document keeps its own.
func TestTheStdoutTerminatorIsNotPartOfAToolResult(t *testing.T) {
	for _, tc := range []struct{ tool, answered, want string }{
		{"brain_status", "one\ntwo\n", "one\ntwo"},
		{"brain_neighbors", "incoming: -\noutgoing: -\n", "incoming: -\noutgoing: -"},
		{"brain_search", "no matches\n", "no matches"},
		{"brain_read", "# Title\nbody\n", "# Title\nbody\n"},
		{"brain_catalog", "# brain\n\n", "# brain\n\n"},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			session := connect(t, privacy.ChannelLocal, servebrain.Deps{
				Answer: func(answer.Request, string, string, func(string)) (string, []string, error) {
					return tc.answered, nil, nil
				},
			})
			res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      tc.tool,
				Arguments: map[string]any{"scope": "notes", "relative": "a.md", "query": "q"},
			})
			if err != nil {
				t.Fatal(err)
			}
			if text, ok := res.Content[0].(*mcp.TextContent); !ok || text.Text != tc.want {
				t.Errorf("said %q, want %q", res.Content[0], tc.want)
			}
		})
	}
}

func TestACoreRefusalIsContentForTheModel(t *testing.T) {
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(answer.Request, string, string, func(string)) (string, []string, error) {
			return "", nil, errors.New("unknown scope: nonesuch")
		},
	})
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_catalog"})
	if err != nil {
		t.Fatalf("a core refusal must not be an MCP error: %v", err)
	}
	if !res.IsError {
		t.Error("a core refusal must set IsError")
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "unknown scope") {
		t.Errorf("the refusal does not carry its reason: %q", text)
	}
}

func TestTheAnswerTextComesBackAsTextContent(t *testing.T) {
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(answer.Request, string, string, func(string)) (string, []string, error) {
			return "the catalog", nil, nil
		},
	})
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_catalog"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatal("a good answer must not set IsError")
	}
	if got := res.Content[0].(*mcp.TextContent).Text; got != "the catalog" {
		t.Errorf("text is %q", got)
	}
}

// Both hint paths end as progress: the warming hint while connecting, the notes
// with the answer. They are two paths and stay two messages.
//
// A search's note also stands in the answer -- see
// TestASearchCarriesItsFindingsInTheAnswer for why -- so the text below is the
// hits and the note, while the two messages are still two.
func TestBothHintPathsBecomeProgress(t *testing.T) {
	sink := newProgressSink()
	session := connectWith(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(_ answer.Request, _, _ string, notice func(string)) (string, []string, error) {
			notice("warming the engine")
			return "hits", []string{"the note"}, nil
		},
	}, sink.options())
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "brain_search",
		Meta:      mcp.Meta{"progressToken": "token-1"},
		Arguments: map[string]any{"query": "zettel"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if got := res.Content[0].(*mcp.TextContent).Text; got != `hits`+"\n"+`! the note` {
		t.Errorf("text is %q", got)
	}
	if got := sink.next(t); got != "warming the engine" {
		t.Errorf("first hint is %q, want the warming hint", got)
	}
	if got := sink.next(t); got != "the note" {
		t.Errorf("second hint is %q, want the note", got)
	}
}

// Without a progress token the hints lapse: a message with no recipient is not
// one, and it must not fail the answer.
//
// What lapses is the *message*. A search's finding is not lost with it: it
// stands in the answer as well, which is exactly why it was put there.
func TestWithoutAProgressTokenTheHintsLapse(t *testing.T) {
	sink := newProgressSink()
	call := 0
	// Two calls on one session, the second one with a token: what arrives first
	// tells whether the first call's hints lapsed, and waiting for a message that
	// does come beats waiting a while for one that does not.
	session := connectWith(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(_ answer.Request, _, _ string, notice func(string)) (string, []string, error) {
			call++
			notice("warming " + string(rune('0'+call)))
			return "hits", []string{"note " + string(rune('0'+call))}, nil
		},
	}, sink.options())
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "brain_search",
		Arguments: map[string]any{"query": "zettel"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if got := res.Content[0].(*mcp.TextContent).Text; got != `hits`+"\n"+`! note 1` {
		t.Errorf("text is %q", got)
	}
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "brain_search",
		Meta:      mcp.Meta{"progressToken": "token-1"},
		Arguments: map[string]any{"query": "zettel"},
	}); err != nil {
		t.Fatalf("second CallTool: %v", err)
	}
	if got := sink.next(t); got != "warming 2" {
		t.Errorf("the first hint to arrive is %q; the untokened call did not lapse", got)
	}
	if got := sink.next(t); got != "note 2" {
		t.Errorf("the second hint to arrive is %q", got)
	}
	select {
	case message := <-sink.messages:
		t.Errorf("a further hint arrived: %q", message)
	default:
	}
}

// A profile the engine does not know is refused rather than searched on. The
// command line is refused by its parser; a tool call has no parser.
func TestAProfileTheEngineDoesNotKnowIsRefused(t *testing.T) {
	asked := false
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(answer.Request, string, string, func(string)) (string, []string, error) {
			asked = true
			return "ok", nil, nil
		},
	})
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "brain_search",
		Arguments: map[string]any{"query": "q", "profile": "deep"},
	})
	if err != nil {
		t.Fatalf("an unknown profile must not be an outage: %v", err)
	}
	if !res.IsError {
		t.Fatal("an unknown profile must be a refusal")
	}
	want := `invalid profile "deep"; choose from fast, full, keyword`
	if text, ok := res.Content[0].(*mcp.TextContent); !ok || text.Text != want {
		t.Errorf("said %+v", res.Content[0])
	}
	if asked {
		t.Error("the answer was asked although the profile was unknown")
	}
}

// The three the schema names travel through untouched.
func TestTheKnownProfilesTravelOn(t *testing.T) {
	for _, profile := range []string{"fast", "full", "keyword"} {
		t.Run("profile="+profile, func(t *testing.T) {
			var seen answer.Request
			session := connect(t, privacy.ChannelLocal, servebrain.Deps{
				Answer: func(req answer.Request, _, _ string, _ func(string)) (string, []string, error) {
					seen = req
					return "ok", nil, nil
				},
			})
			res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      "brain_search",
				Arguments: map[string]any{"query": "q", "profile": profile},
			})
			if err != nil || res.IsError {
				t.Fatalf("a known profile must travel on: %v %+v", err, res)
			}
			if seen.Profile != profile {
				t.Errorf("the answer saw profile %q", seen.Profile)
			}
		})
	}
}

// A search's findings stand in the answer as well as in the progress notes.
// A host that sends no progress token gets no note at all, so a finding that
// lived only there was lost to it -- and a finding is the reader's only
// warning that the hits are worth less than they look.
func TestASearchCarriesItsFindingsInTheAnswer(t *testing.T) {
	sink := newProgressSink()
	session := connectWith(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(answer.Request, string, string, func(string)) (string, []string, error) {
			return "brain://notes/a.md:1  50%  A\n\n", []string{"the index is stale", "asked twice"}, nil
		},
	}, sink.options())
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Meta:      mcp.Meta{"progressToken": "token-1"},
		Name:      "brain_search",
		Arguments: map[string]any{"query": "q"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "brain://notes/a.md:1  50%  A\n\n! the index is stale\n! asked twice"
	if text, ok := res.Content[0].(*mcp.TextContent); !ok || text.Text != want {
		t.Errorf("said %q, want %q", res.Content[0], want)
	}
	// And still upwards, for the host that sent a token.
	if first := sink.next(t); first != "the index is stale" {
		t.Errorf("the note said %q", first)
	}
}

// The other four tools' notes are not findings about their answer, and nothing
// here guesses what they are: they travel as notes alone.
func TestOnlyASearchPutsItsNotesIntoTheAnswer(t *testing.T) {
	for _, tool := range []string{"brain_status", "brain_catalog", "brain_read", "brain_neighbors"} {
		t.Run(tool, func(t *testing.T) {
			session := connect(t, privacy.ChannelLocal, servebrain.Deps{
				Answer: func(answer.Request, string, string, func(string)) (string, []string, error) {
					return "the answer", []string{"a note"}, nil
				},
			})
			res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      tool,
				Arguments: map[string]any{"scope": "notes", "relative": "a.md"},
			})
			if err != nil {
				t.Fatal(err)
			}
			if text, ok := res.Content[0].(*mcp.TextContent); !ok || text.Text != "the answer" {
				t.Errorf("said %q", res.Content[0])
			}
		})
	}
}

// A search without findings gains nothing and loses nothing.
func TestASearchWithoutFindingsIsTheAnswerAlone(t *testing.T) {
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(answer.Request, string, string, func(string)) (string, []string, error) {
			return "no matches\n", nil, nil
		},
	})
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "brain_search",
		Arguments: map[string]any{"query": "q"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if text, ok := res.Content[0].(*mcp.TextContent); !ok || text.Text != "no matches" {
		t.Errorf("said %q", res.Content[0])
	}
}

// A search without a profile runs the cheap one, as the reference and as
// loomux's own command line do.
//
// Not a nicety: the empty string is no profile the ports know, and both fall
// back to `query` with reranking for it -- the full profile. A caller who named
// none therefore paid for the slowest search there is, on the front a model
// uses most, and nothing said so.
func TestASearchWithoutAProfileRunsTheCheapOne(t *testing.T) {
	var seen answer.Request
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(req answer.Request, _, _ string, _ func(string)) (string, []string, error) {
			seen = req
			return "ok", nil, nil
		},
	})
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "brain_search",
		Arguments: map[string]any{"query": "zettel"},
	})
	if err != nil || res.IsError {
		t.Fatalf("CallTool: %v %+v", err, res)
	}
	if seen.Profile != "fast" {
		t.Errorf("the answer saw profile %q, want fast", seen.Profile)
	}
}

// A brain_search without `n` asks for the reference's ten hits, not for none.
//
// The schema's `default: 10` is decoration on this path: both fronts register
// through the untyped AddTool, which resolves no schema, so nothing applies a
// default before the handler. The reference carries the same ten a second time
// in its code (daemon/tools.py:167), and so must loomux: a count of zero is
// what search.ExecuteSearch truncates every hit away with, and the model reads
// the "no matches" that leaves as an answer to its question.
func TestSearchWithoutACountAsksForTheReferencesTen(t *testing.T) {
	var seen answer.Request
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(req answer.Request, _, _ string, _ func(string)) (string, []string, error) {
			seen = req
			return "wiki/a.md:1 first\nwiki/b.md:1 second", nil, nil
		},
	})
	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "brain_search",
		Arguments: map[string]any{"query": "zettel"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if seen.Count != 10 {
		t.Errorf("the answer saw n=%d, want 10", seen.Count)
	}
}
