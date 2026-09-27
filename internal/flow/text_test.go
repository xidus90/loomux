package flow_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/xidus90/loomux/internal/flow"
)

func TestReadTextTakesAFileAmongTheFlowsTexts(t *testing.T) {
	texts := fstest.MapFS{"instructions/draft.md": {Data: []byte("Draft {{topic}}.\n")}}
	raw, err := flow.ReadText(texts, flow.Text{Key: "instruction", Path: "instructions/draft.md"})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "Draft {{topic}}.\n" {
		t.Fatalf("got %q", raw)
	}
}

func TestReadTextTakesAnInlineText(t *testing.T) {
	raw, err := flow.ReadText(fstest.MapFS{}, flow.Text{Key: "message", Inline: "done in {{count}}"})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "done in {{count}}" {
		t.Fatalf("got %q", raw)
	}
}

func TestReadTextNamesTheKeyOfAMissingFile(t *testing.T) {
	_, err := flow.ReadText(fstest.MapFS{}, flow.Text{Key: "question", Path: "ask.md"})
	if err == nil || !strings.HasPrefix(err.Error(), "reading question ask.md: ") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadTextTurnsCRLFIntoLF(t *testing.T) {
	texts := fstest.MapFS{"questions/q.md": {Data: []byte("Ship it?\r\nReally?\r\n")}}
	got, err := flow.ReadText(texts, flow.Text{Key: "question", Path: "questions/q.md"})
	if err != nil || string(got) != "Ship it?\nReally?\n" {
		t.Fatalf("got %q, %v", got, err)
	}
}

// A multi-line inline text keeps the carriage returns of a flow.toml saved
// with CRLF: the TOML decoder hands them over as written.
func TestReadTextTurnsCRLFIntoLFInAnInlineText(t *testing.T) {
	got, err := flow.ReadText(fstest.MapFS{}, flow.Text{Key: "message", Inline: "first\r\nsecond\r\n"})
	if err != nil || string(got) != "first\nsecond\n" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestReadTextNamesTheMissingFile(t *testing.T) {
	_, err := flow.ReadText(fstest.MapFS{}, flow.Text{Key: "instruction", Path: "instructions/x.md"})
	if err == nil || !strings.Contains(err.Error(), "reading instruction instructions/x.md") {
		t.Fatalf("err = %v", err)
	}
}

func TestRenderSeesFieldsAndParams(t *testing.T) {
	state := flow.State{Fields: map[string]flow.Value{"notes": []string{"one", "two"}}}
	params := flow.Params{"max_rounds": 5}
	got, err := flow.Render([]byte("{{max_rounds}} rounds:\n{{notes}}"), state, params)
	if err != nil {
		t.Fatal(err)
	}
	if got != "5 rounds:\n- one\n- two" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderPassesOnWhatIsNoPlaceholder(t *testing.T) {
	_, err := flow.Render([]byte("{{ name }}"), flow.State{}, nil)
	if err == nil {
		t.Fatal("want an error")
	}
}
