package load

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/xidus90/loomux/internal/flow"
)

// The stage tests load against blocks of their own. These behave like the
// real ones in the three things the loader asks of a block -- which of its own
// keys it objects to, which texts a node renders, and which fields it writes --
// and in nothing else, so a finding a stage test pins is the loader's and does
// not move when a real block tightens its own rules. blocks_test.go loads the
// example and planning flows against the real blocks.
type testBlock struct{ kind string }

func (b testBlock) Kind() string { return b.kind }

func (b testBlock) Check(node flow.Node) []string {
	if b.kind != "exit" {
		return nil
	}
	code, ok := node.Keys["code"].(int64)
	if ok && (code == 0 || code == 1 || code == 3) {
		return []string{fmt.Sprintf("code %d is the runtime's own; a block may not choose 0, 1 or 3", code)}
	}
	return nil
}

func (b testBlock) Texts(node flow.Node) []flow.Text {
	switch b.kind {
	case "agent":
		return []flow.Text{{Key: "instruction", Path: node.StringKey("instruction")}}
	case "gate":
		return []flow.Text{{Key: "question", Path: node.StringKey("question")}}
	default:
		return []flow.Text{{Key: "message", Inline: node.StringKey("message")}}
	}
}

func (b testBlock) Writes(node flow.Node) map[string]flow.Type {
	switch b.kind {
	case "agent":
		reply, _ := node.Keys["reply"].(map[string]any)
		written := make(map[string]flow.Type, len(reply))
		for name, kind := range reply {
			text, _ := kind.(string)
			written[name] = flow.Type(text)
		}
		return written
	case "gate":
		answer := node.StringKey("answer")
		return map[string]flow.Type{answer: flow.String, answer + "_text": flow.String}
	default:
		return nil
	}
}

func (testBlock) NeedsBaseline() bool { return false }

func (testBlock) Define(*flow.Graph, flow.Node, flow.Env) (flow.Definition, error) {
	return flow.Definition{}, nil
}

func (testBlock) Run(context.Context, *flow.Graph, flow.Node, flow.State, flow.Env) (flow.Result, error) {
	return flow.Result{}, nil
}

func catalog(t *testing.T) *flow.Catalog {
	t.Helper()
	made, err := flow.NewCatalog(
		[]flow.Block{testBlock{"agent"}, testBlock{"gate"}, testBlock{"exit"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// testFlow is a flow under testdata, found the way Find finds a project flow.
func testFlow(t *testing.T, name string) Found {
	t.Helper()
	return Found{Name: name, Origin: OriginProject, File: name + "/flow.toml", Files: os.DirFS(filepath.Join("testdata", name))}
}

// inline is a flow of one flow.toml and the files beside it, none on disk.
func inline(flowFile string, files map[string]string) Found {
	texts := fstest.MapFS{"flow.toml": {Data: []byte(flowFile)}}
	for name, body := range files {
		texts[name] = &fstest.MapFile{Data: []byte(body)}
	}
	return Found{Name: "f", Origin: OriginProject, File: "f/flow.toml", Files: texts}
}

func TestTheLoadedGraphSaysWhatTheFileSaid(t *testing.T) {
	graph, err := Load(testFlow(t, "planning"), catalog(t))
	if err != nil {
		t.Fatal(err)
	}
	// The name is the folder's: the file has none of its own to disagree with.
	if graph.Name != "planning" || graph.Start != "intake" {
		t.Fatalf("name = %q, start = %q", graph.Name, graph.Start)
	}
	if len(graph.Nodes) != 12 {
		t.Fatalf("%d nodes", len(graph.Nodes))
	}
	// File order, because the first edge whose condition holds is the one a run
	// takes, and a map would make that order the runtime's business.
	if graph.Nodes[0].Name != "intake" || graph.Edges[0].From != "intake" {
		t.Fatalf("order lost: %q, %q", graph.Nodes[0].Name, graph.Edges[0].From)
	}
	if graph.File != "planning/flow.toml" || graph.Origin != OriginProject || graph.Overlays != nil {
		t.Fatalf("file = %q, origin = %q, overlays = %v", graph.File, graph.Origin, graph.Overlays)
	}
	if _, err := flow.ReadText(graph.Texts, flow.Text{Key: "question", Path: "questions/plan.md"}); err != nil {
		t.Fatalf("the graph cannot read its own texts: %v", err)
	}
}

// Raw is what the definition hash is taken over, so it holds the entry as the
// file wrote it -- max_visits as its text, not as the parsed Cap.
func TestANodeKeepsItsRawEntry(t *testing.T) {
	graph, err := Load(testFlow(t, "planning"), catalog(t))
	if err != nil {
		t.Fatal(err)
	}
	node := graph.Nodes[1] // clarify
	if node.Raw["max_visits"] != "max_rounds + 1" {
		t.Fatalf("raw max_visits = %#v", node.Raw["max_visits"])
	}
	if _, ok := node.Keys["name"]; ok {
		t.Fatal("Keys holds a shared key")
	}
	if node.MaxVisits != (flow.Cap{Param: "max_rounds", Add: 1}) {
		t.Fatalf("cap = %#v", node.MaxVisits)
	}
}

func TestDefaultsArriveInTheDeclaredGoType(t *testing.T) {
	graph, err := Load(testFlow(t, "planning"), catalog(t))
	if err != nil {
		t.Fatal(err)
	}
	// TOML hands out int64; a state field declared int holds an int.
	if graph.Params["max_rounds"].Default != 3 {
		t.Fatalf("max_rounds = %#v", graph.Params["max_rounds"].Default)
	}
	if list, ok := graph.State["notes"].Default.([]string); !ok || len(list) != 0 {
		t.Fatalf("notes = %#v", graph.State["notes"].Default)
	}
}

// An edge with a when carries the parsed condition and the text it came from,
// and one without carries neither: a nil When always holds, and a run reads
// that without asking the file.
func TestAnEdgeCarriesTheConditionItsWhenStandsFor(t *testing.T) {
	graph, err := Load(testFlow(t, "planning"), catalog(t))
	if err != nil {
		t.Fatal(err)
	}
	yes := flow.State{Fields: map[string]flow.Value{"answer": "yes"}}
	no := flow.State{Fields: map[string]flow.Value{"answer": "no"}}
	edge := graph.Edges[2] // gate_clarify -> spec, when answer == "yes"
	if edge.To != "spec" || edge.When == nil || edge.Text != `answer == "yes"` {
		t.Fatalf("edge = %+v", edge)
	}
	if !edge.When.Holds(yes, nil) || edge.When.Holds(no, nil) {
		t.Fatal("the condition does not read the answer")
	}
	if graph.Edges[3].When != nil || graph.Edges[3].Text != "" {
		t.Fatal("an edge without a when carries a condition")
	}
}

func TestLoadNamesAFileItCannotRead(t *testing.T) {
	found := Found{Name: "f", File: "f/flow.toml", Files: fstest.MapFS{}}
	_, err := Load(found, catalog(t))
	if err == nil || !strings.HasPrefix(err.Error(), "f/flow.toml: ") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadRefusesUnknownKeys(t *testing.T) {
	for name, tc := range map[string]struct{ toml, want string }{
		"top level": {"schema_version = 1\ncolour = 1\n[flow]\nstart = \"a\"\n", `unknown key "colour"`},
		"flow":      {"schema_version = 1\n[flow]\nstart = \"a\"\nmodel = \"w\"\n", `[flow] model: a flow names a role, write role = ...`},
		"flow name": {"schema_version = 1\n[flow]\nstart = \"a\"\nname = \"x\"\n", `[flow] name: a flow's name is its folder's name`},
		"node":      {"schema_version = 1\n[flow]\nstart = \"a\"\n[[node]]\nname = \"a\"\nkind = \"exit\"\ncode = 4\nmessage = \"m\"\nmodel = \"w\"\n", `node "a": model: a node names a role, write role = ...`},
		"edge":      {"schema_version = 1\n[flow]\nstart = \"a\"\n[[node]]\nname = \"a\"\nkind = \"exit\"\ncode = 4\nmessage = \"m\"\n[[edge]]\nfrom = \"a\"\nto = \"END\"\nif = \"x\"\n", `edge 1: unknown key "if"`},
		"role name": {"schema_version = 1\n[flow]\nstart = \"a\"\nrole = \"a-b\"\n", `role "a-b" is not a name`},
	} {
		t.Run(name, func(t *testing.T) {
			found := Found{Name: "f", File: "f/flow.toml", Files: fstest.MapFS{"flow.toml": {Data: []byte(tc.toml)}}}
			_, err := Load(found, realCatalog(t))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

// The spec asks for the known keys in the message: the reader of a typo sees
// the word it should have been.
func TestAnUnknownKeyNamesTheKnownOnes(t *testing.T) {
	const source = "schema_version = 1\ncolour = 1\n[flow]\nstart = \"a\"\nstrat = \"a\"\n" +
		"[[node]]\nname = \"a\"\nkind = \"exit\"\nmessage = \"m\"\n[[edge]]\nfrom = \"a\"\nto = \"END\"\nif = \"x\"\n"
	findings(t, loadSource(t, source), []string{
		`unknown key "colour"; known keys: edge, flow, node, params, schema_version, state`,
		`[flow]: unknown key "strat"; known keys: role, start`,
		`edge 1: unknown key "if"; known keys: from, on_error, to, when`,
	})
}

// A node's own keys are its block's to refuse, and the finding shows the keys
// a node of that kind may hold, the shared ones included.
func TestAnUnknownNodeKeyNamesTheKnownOnes(t *testing.T) {
	const source = "schema_version = 1\n[flow]\nstart = \"a\"\n" +
		"[[node]]\nname = \"a\"\nkind = \"exit\"\ncode = 4\nmessage = \"m\"\nmesage = \"m\"\n[[edge]]\nfrom = \"a\"\nto = \"END\"\n"
	_, err := Load(inline(source, nil), realCatalog(t))
	const want = `f/flow.toml: node "a": unknown key "mesage"; known keys: code, kind, max_visits, message, name, role`
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
}

// A node's role follows the rule [agent.roles] holds a role to, and says
// which node it belongs to.
func TestANodeRoleMustBeAName(t *testing.T) {
	const source = "schema_version = 1\n[flow]\nstart = \"a\"\nrole = \"writer\"\n" +
		"[[node]]\nname = \"a\"\nkind = \"exit\"\nmessage = \"m\"\nrole = \"a b\"\n[[edge]]\nfrom = \"a\"\nto = \"END\"\n"
	findings(t, loadSource(t, source), []string{
		`node "a": role "a b" is not a name; a name is [A-Za-z_][A-Za-z0-9_]*`,
	})
}

func TestTheRolesArriveOnTheGraph(t *testing.T) {
	const source = "schema_version = 1\n[flow]\nstart = \"a\"\nrole = \"writer\"\n" +
		"[[node]]\nname = \"a\"\nkind = \"exit\"\nmessage = \"m\"\nrole = \"reviewer\"\n[[edge]]\nfrom = \"a\"\nto = \"END\"\n"
	graph, err := Load(inline(source, nil), catalog(t))
	if err != nil {
		t.Fatal(err)
	}
	if graph.Role != "writer" || graph.Nodes[0].Role != "reviewer" {
		t.Fatalf("flow role = %q, node role = %q", graph.Role, graph.Nodes[0].Role)
	}
	if _, ok := graph.Nodes[0].Keys["role"]; ok {
		t.Fatal("Keys holds the shared key role")
	}
}

func TestLoadRefusesABackslashInATextPath(t *testing.T) {
	flowFile := "schema_version = 1\n[flow]\nstart = \"a\"\n[state]\nanswer = { type = \"string\", default = \"\" }\nanswer_text = { type = \"string\", default = \"\" }\n" +
		"[[node]]\nname = \"a\"\nkind = \"gate\"\nquestion = 'questions\\q.md'\nchoices = [\"yes\", \"no\"]\nanswer = \"answer\"\n[[edge]]\nfrom = \"a\"\nto = \"END\"\n"
	found := Found{Name: "f", File: "f/flow.toml", Files: fstest.MapFS{"flow.toml": {Data: []byte(flowFile)}, "questions/q.md": {Data: []byte("?")}}}
	_, err := Load(found, realCatalog(t))
	if err == nil || !strings.Contains(err.Error(), `question "questions\\q.md" must use / between folders`) {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadHoldsEveryTextToItsFolder(t *testing.T) {
	// An instruction under questions/, a question under instructions/, a path
	// that leaves the folder and an absolute one: all four refused, although
	// the first two files are there to be read.
	const agent = "schema_version = 1\n[flow]\nstart = \"a\"\n[state]\nverdict = { type = \"string\", default = \"\" }\n" +
		"[[node]]\nname = \"a\"\nkind = \"agent\"\ninstruction = %q\nreply = { verdict = \"string\" }\n[[edge]]\nfrom = \"a\"\nto = \"END\"\n"
	const gate = "schema_version = 1\n[flow]\nstart = \"a\"\n[state]\nanswer = { type = \"string\", default = \"\" }\nanswer_text = { type = \"string\", default = \"\" }\n" +
		"[[node]]\nname = \"a\"\nkind = \"gate\"\nquestion = %q\nchoices = [\"yes\", \"no\"]\nanswer = \"answer\"\n[[edge]]\nfrom = \"a\"\nto = \"END\"\n"
	files := map[string]string{"questions/d.md": "Draft.", "instructions/q.md": "Ship?"}
	for _, c := range []struct{ source, path, want string }{
		{agent, "questions/d.md", `node "a": instruction "questions/d.md" must be a file under instructions/`},
		{gate, "instructions/q.md", `node "a": question "instructions/q.md" must be a file under questions/`},
		{agent, "../x.md", `node "a": instruction "../x.md" must be a file under instructions/`},
		{agent, "/abs.md", `node "a": instruction "/abs.md" must be a file under instructions/`},
	} {
		t.Run(c.path, func(t *testing.T) {
			_, err := Load(inline(fmt.Sprintf(c.source, c.path), files), realCatalog(t))
			if err == nil || err.Error() != "f/flow.toml: "+c.want {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
		})
	}
}

// No block renders a file under a key other than instruction or question
// today. One that does is held to instructions/, the folder an overlay of
// its texts would have to reach.
func TestATextOfAnotherKeyLivesUnderInstructions(t *testing.T) {
	if got := textPath(flow.Text{Key: "note", Path: "notes/n.md"}); got != `note "notes/n.md" must be a file under instructions/` {
		t.Fatalf("got %q", got)
	}
	if got := textPath(flow.Text{Key: "note", Path: "instructions/n.md"}); got != "" {
		t.Fatalf("got %q", got)
	}
}

// A flow.toml saved with CRLF carries the carriage returns into a multi-line
// inline text; the hash and the rendered text must not see them.
func TestAnInlineTextOfACRLFFileReadsAsLF(t *testing.T) {
	source := strings.ReplaceAll("schema_version = 1\n[flow]\nstart = \"stop\"\n"+
		"[[node]]\nname = \"stop\"\nkind = \"exit\"\ncode = 4\nmessage = \"\"\"\nfirst\nsecond\n\"\"\"\n"+
		"[[edge]]\nfrom = \"stop\"\nto = \"END\"\n", "\n", "\r\n")
	real := realCatalog(t)
	graph, err := Load(inline(source, nil), real)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := real.Block("exit")
	raw, err := flow.ReadText(graph.Texts, block.Texts(graph.Nodes[0])[0])
	if err != nil || string(raw) != "first\nsecond\n" {
		t.Fatalf("got %q, %v", raw, err)
	}
}
