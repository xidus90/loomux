# ulflow M1, Welle 0 und 1 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Welle 1 läuft mit superpowers:dispatching-parallel-agents: sieben Subagenten in einer Nachricht, jeder in seinem eigenen Worktree. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die Verträge von `ulflow` stehen als Go-Typen, und die sieben Blattpakete darunter — Bedingungen, Platzhalter, Journal-Schreiber mit Hashes, Fake-Modell, geänderte Dateien, `[agent]`-Konfiguration, Laufdateien — sind gebaut, voll gedeckt und in `feature/agent-harness` zusammengeführt.

**Architecture:** Welle 0 legt in `internal/flow`, `internal/model` und `internal/flowcfg` nur Typen und Schnittstellen fest, dazu je neues Paket eine `doc.go`. Welle 1 baut sieben Pakete, von denen keines eine Datei eines anderen berührt und jedes nur gegen die Verträge aus Welle 0 baut. Deshalb laufen die sieben Lanes gleichzeitig, jede in `.worktrees/ulflow-<lane>` auf `ulflow/<lane>`, und werden danach mit `--no-ff` in `feature/agent-harness` zusammengeführt.

**Tech Stack:** Go mit dem Sprachstand `go 1.22` aus `go.mod` (installiert ist go1.27.0, maßgeblich ist `go.mod`), `github.com/BurntSushi/toml` v1.6.0, `encoding/json`, `crypto/sha256`, git auf dem PATH.

**Spec:** `docs/.superpowers/specs/2026-09-11-ulflow-laufzeit-design.md`, Abschnitte „Verhaltensvertrag", „Das Flow-Format", „Modellwahl und Modell-Port", „Journal", „Pakete" und „Wellen" (Welle 0 und 1).

## Global Constraints

- Kommentare, Bezeichner, Fehlermeldungen und Commit-Nachrichten englisch.
- Sprachstand `go 1.22`: kein range-over-func, kein `maps.Keys`, kein `slices.Collect`, keine API ab Go 1.23.
- TDD: jeder Code-Schritt beginnt mit einem Test, der vorher rot ist.
- 100 % Coverage in jedem Paket, das der Task anlegt oder ändert; `go test -cover` zeigt `coverage: 100.0% of statements` (oder `[no statements]` bei reinen Typpaketen).
- Vor jedem Commit `gofmt -l` auf die Dateien des Tasks: leere Ausgabe. Sonst `gofmt -w` und neu prüfen.
- Kein `Co-Authored-By` und keine Nennung eines Modells im Commit. Mehrzeilige Nachrichten über eine Datei und `git commit -F`.
- Vor jedem Commit `git rev-parse --abbrev-ref HEAD`, `git log --oneline -1` und `git diff --cached --stat` lesen; nur die Dateien stagen, die der Task nennt.
- Kein Push. Kein `--no-verify`.
- Keine Änderung an `go.mod` oder `go.sum`.
- Eine Lane fasst nur die Dateien an, die ihr Task unter **Files** nennt.
- Kein Paket ruft `time.Now`.
- Ein frischer Worktree bekommt vor seinem ersten Commit `env -u VIRTUAL_ENV uv sync`. Sonst scheitert der Pre-Commit-Hook an `.venv/Scripts/ultraloom.exe` („Zugriff verweigert"): Das `#` im Pfad lässt uv das Projekt bei jedem `uv run` neu installieren.
- Die Paketdokumentation neuer Pakete steht in `doc.go`; keine zweite `// Package …`-Zeile in anderen Dateien.
- Ein Shell-Befehl je Schritt. Jeder Befehl nennt sein Verzeichnis mit `cd "<Worktree>" && …`.

## Entscheidungen, die dieser Plan trifft

Die Spec lässt diese Punkte offen; am 2026-09-11 gemessen oder aus dem Python-Code nachgerechnet:

1. **`NextID` kennt keinen Fehler.** Unter Windows meldet `os.ReadDir` auf einer *Datei* `fs.ErrNotExist` (gemessen). Ein Fehlerzweig wäre also nur für Rechtefehler da und nicht testbar. Python kommt über `glob` zum selben Ergebnis: Ein Laufverzeichnis, das nicht lesbar ist, ergibt `0001`. Scheitert danach das Schreiben des Journals, meldet das `journal.Append`.
2. **`flowcfg` liest das Dokument als Map.** Ins Struct-Feld decodiert, verschluckt `github.com/BurntSushi/toml` ein `agent = 3` ohne Fehler (gemessen). Die Schlüssel `cli_path` und `settings` gehören dem Python-Adapter bis M2. Sie gelten als bekannt und werden nicht gelesen; jeder andere unbekannte Schlüssel ist ein Fehler.
3. **`ChangedFiles` fragt das Präfix vor dem Status**, anders als `worktree.py`, das den zweiten Prozess bei sauberem Baum spart. So hat jeder Fehler einen Test: außerhalb eines Repositorys scheitert `rev-parse --show-prefix`, mit beschädigtem Index nur `status` (beides gemessen: Exit 0 gegen Exit 128).
4. **Die Laufmarke schreibt Optionen nach Namen sortiert**, Python in Einfügereihenfolge. Die Werte sind JSON-Texte wie in Python; die Bytes weichen ab (Go schreibt Nicht-ASCII roh), die Bedeutung nicht.
5. **Texte in Bedingungen kennen genau zwei Escapes**, `\"` und `\\`. Ohne sie ließe sich kein Anführungszeichen vergleichen.
6. **Platzhalter sind streng.** Zwischen `{{` und `}}` steht ein Name ohne Leerraum; alles andere ist ein Ladefehler und muss als `\{{` geschrieben werden. Eine nachsichtige Lesart machte aus JSON in Anweisungen Namen, die niemand deklariert hat.
7. **`max_visits = 0` ist ein Fehler**, ebenso `"<param> + -1"`. Ein Knoten, der nie laufen darf, ist ein Tippfehler.

## Ablauf

1. **Task 0** in `.worktrees/agent-harness` durch einen Subagenten. Orchestrator prüft: `go build ./...`, `go vet ./...`, `git log --oneline -1`.
2. **Lane-Worktrees anlegen**, sieben Befehle, je einer (Orchestrator):
   - `git -C "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" worktree add "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-expr" -b ulflow/expr feature/agent-harness`
   - dasselbe für `tmpl`, `journal`, `model`, `gitwork`, `flowcfg`, `runs`
3. **In jedem Lane-Worktree** `cd "…/.worktrees/ulflow-<lane>" && env -u VIRTUAL_ENV uv sync` (Orchestrator, vor dem Dispatch).
4. **Tasks 1–7 parallel**: eine Nachricht, sieben Agent-Aufrufe, jeder mit `model: "opus"`, dem vollständigen Text seines Tasks, den Global Constraints und seinem Worktree-Pfad. Einen `effort` nimmt der Agent-Aufruf nicht an, und das Repo hat keine `.claude/agents/*.md`, die ihn setzen könnte; der Subagent läuft also mit dem Effort der Sitzung. Nicht `isolation: "worktree"` benutzen; das legt unter `.claude/worktrees/` an. Jeder Brief enthält den Satz: „Meldet ein Hook einen Befund über ein anderes Verzeichnis als deinen Worktree, berichte ihn und behebe ihn nicht."
5. **Task 8**: Prüfen und Zusammenführen durch den Orchestrator.

**Warum der Orchestrator den Diff jeder Lane liest:** `ulguard` prüft Pfadregeln relativ zu `--root` der Sitzung (`relativePath` in `cmd/guard/guard.go:114`). Ein Pfad in einem Lane-Worktree liegt außerhalb davon, bleibt absolut und trifft keine Regel wie `.ultraloom/runs/*`. In Lane-Worktrees schützen also nur die Befehlsregeln. Deshalb ist `git diff --stat` gegen die Dateiliste der Lane Teil von Task 8.

---

### Task 0: Verträge

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness`

**Files:**
- Create: `internal/flow/doc.go`, `internal/flow/types.go`, `internal/flow/graph.go`, `internal/flow/block.go`
- Create: `internal/model/model.go`
- Create: `internal/flowcfg/flowcfg.go`
- Create: `internal/flow/expr/doc.go`, `internal/flow/tmpl/doc.go`, `internal/blocks/doc.go`, `internal/runner/doc.go`, `internal/runs/doc.go`
- Test: `internal/flow/contract_test.go`, `internal/model/model_test.go`

**Interfaces:**
- Consumes: nichts.
- Produces: alle Typen dieser Dateien, wörtlich wie unten. Welle 1 importiert `flow.Type`, `flow.String`, `flow.Int`, `flow.Bool`, `flow.StringList`, `flow.Value`, `flow.State`, `flow.Params`, `flow.Cap`, `flow.Condition`, `flow.Predicate`, `flow.Baseline`, `model.Request`, `model.Reply`, `model.Model`, `flowcfg.Config`, `flowcfg.ModelSpec`, `flowcfg.Resolved`.

- [ ] **Step 1: Write the failing tests**

`internal/flow/contract_test.go`:

```go
package flow_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/xidus90/ultra-loom/internal/flow"
)

// Doubles for every contract. They exist so that a signature that moves breaks
// the build here, in the package that owns it, and not in four lanes at once.
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
```

`internal/model/model_test.go`:

```go
package model_test

import (
	"context"

	"github.com/xidus90/ultra-loom/internal/model"
)

// A double for the port, so a moved signature breaks the build here.
type stubModel struct{}

func (stubModel) Ask(context.Context, model.Request) (model.Reply, error) {
	return model.Reply{}, nil
}

var _ model.Model = stubModel{}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test ./internal/flow/... ./internal/model/...`
Expected: FAIL — `no non-test Go files` bzw. `undefined: flow.Block`.

- [ ] **Step 3: Write the contracts**

`internal/flow/doc.go`:

```go
// Package flow holds what a flow is made of: typed state, the loaded graph,
// and the contracts every block, condition and registry is built against.
//
// The contracts are fixed before any package that uses them is written, so the
// expression language, the loader, the blocks and the runner can be built side
// by side. The loader that fills a Graph from TOML lives here as well, in a
// later wave.
package flow
```

`internal/flow/types.go`:

```go
package flow

// Type is the declared type of a state field or a parameter.
type Type string

// The four types a flow can declare.
const (
	String     Type = "string"
	Int        Type = "int"
	Bool       Type = "bool"
	StringList Type = "list[string]"
)

// Value is one typed value. Its dynamic type is exactly one of string, int,
// bool or []string, matching the Type it was declared with; no other Go type
// ever appears. TOML's int64 and JSON's float64 are converted before a value
// reaches a State.
type Value = any

// Field declares a state field or a parameter.
type Field struct {
	Type    Type
	Default Value
}

// State is what a node sees: every declared field, and how often each node has
// run so far.
//
// A State is never changed in place. Blocks return a Delta and the runner
// builds the next State from it; otherwise a resume could not reconstruct what
// a node saw.
type State struct {
	Fields map[string]Value
	Visits map[string]int
}

// Delta is what one node changes: field name to new value.
type Delta map[string]Value

// Params are a run's parameters after defaults and --option were applied.
type Params map[string]Value

// Baseline is what a run starts from: the HEAD commit and the paths that were
// already changed at that moment, relative to the project root.
type Baseline struct {
	Commit string
	Dirty  []string
}
```

`internal/flow/graph.go`:

```go
package flow

// End is the pseudo node an edge points to when the run is done.
const End = "END"

// Cap is a node's visit ceiling: the value of the int parameter Param plus Add,
// or Add alone when Param is empty. A node without max_visits has Cap{Add: 1}.
type Cap struct {
	Param string
	Add   int
}

// Node is one [[node]] entry of a flow file.
type Node struct {
	Name      string
	Kind      string
	Model     string // a name under [agent.models]; empty when the node names none
	MaxVisits Cap
	// Keys are the kind's own keys as TOML decoded them. The block for Kind
	// checks and reads them; nothing else does.
	Keys map[string]any
}

// Condition decides whether an edge holds for a state.
type Condition interface {
	Holds(state State, params Params) bool
}

// Predicate is a condition written in Go and registered by name.
type Predicate func(state State, params Params) bool

// Edge is one [[edge]] entry. A nil When always holds. An error edge carries no
// When and is taken only when its source node failed.
type Edge struct {
	From    string
	To      string
	When    Condition
	OnError bool
}

// Graph is a loaded flow. Nodes and Edges keep file order, because the first
// edge whose condition holds is the one a run takes.
type Graph struct {
	Name   string
	File   string // the flow file, as messages name it
	Dir    string // the directory instruction and question files are relative to
	Start  string
	Model  string // the flow's model name; empty when it names none
	Params map[string]Field
	State  map[string]Field
	Nodes  []Node
	Edges  []Edge
}
```

`internal/flow/block.go`:

```go
package flow

import (
	"context"
	"errors"

	"github.com/xidus90/ultra-loom/internal/flowcfg"
	"github.com/xidus90/ultra-loom/internal/model"
)

// ErrInvalidAnswer marks a gate answer that matches none of the gate's choices.
// The runner refuses such a resume and leaves the gate open; it is not a node
// failure and takes no error edge.
var ErrInvalidAnswer = errors.New("the answer matches none of the choices")

// Text is one text a node renders: a file next to the flow, or an inline text.
type Text struct {
	Key    string // the node key the text came from, e.g. "instruction"
	Path   string // relative to Graph.Dir; empty for an inline text
	Inline string // the text itself when Path is empty
}

// Definition is what a node's result depends on besides its input. The runner
// hashes it into the journal's definition_hash.
type Definition struct {
	Text    []byte   // the raw bytes of the instruction or question; nil when there is none
	Model   string   // "<provider>:<model>" or "<provider>:cli-default"; empty without a model
	Effort  string   // empty when the node has none
	Profile string   // the tool profile name the journal records; empty when there is none
	Tools   []string // the resolved tool list; nil when there is none
}

// Env is what a running block may use besides the graph, the node and the state.
type Env struct {
	Root     string      // the project root
	Params   Params      // the run's parameters
	Answer   *string     // the answer a resume brought for this gate; nil otherwise
	Baseline *Baseline   // nil when the run has none
	Model    model.Model // nil when the run has no model
	Agent    flowcfg.Config
}

// Result is what one execution of a block produced.
type Result struct {
	Delta    Delta
	Tokens   int
	Model    string  // the model that answered, as the journal records it; empty without a model
	Question *string // non-nil: the node pauses with this question and the run ends paused
	Exit     *Exit   // non-nil: the run ends with this code and message
}

// Exit is a block's decision to end the run with a code of its own.
type Exit struct {
	Code    int
	Message string
}

// Block implements one node kind. The runtime knows no block by name; it finds
// them through a Registry.
type Block interface {
	// Kind is the value of `kind` this block implements.
	Kind() string
	// Check returns one finding per problem with the node's own keys.
	Check(node Node) []string
	// Texts names every text the node renders, so placeholders can be checked
	// before a run.
	Texts(node Node) []Text
	// Writes names the state fields the node writes, with their types.
	Writes(node Node) map[string]Type
	// NeedsBaseline reports whether a node of this kind needs the run's baseline.
	NeedsBaseline() bool
	// Define describes what the node's result depends on besides its input.
	Define(graph *Graph, node Node, env Env) (Definition, error)
	// Run executes the node once.
	Run(ctx context.Context, graph *Graph, node Node, state State, env Env) (Result, error)
}

// Registry finds blocks and predicates by name.
type Registry interface {
	Block(kind string) (Block, bool)
	Predicate(name string) (Predicate, bool)
}
```

`internal/model/model.go`:

```go
// Package model is the port every agent node reaches a model through.
//
// The runner never talks to a provider directly. Tests hand it the Fake; M2
// brings the adapters for `claude -p` and `agy -p`.
package model

import "context"

// ReplyType is the type of one field a reply must carry.
type ReplyType string

// The three types a reply field can have; replies are flat on purpose.
const (
	ReplyString ReplyType = "string"
	ReplyInt    ReplyType = "int"
	ReplyBool   ReplyType = "bool"
)

// Request is one model call, fully described.
type Request struct {
	Prompt   string
	Tools    []string
	Effort   string
	Provider string
	Model    string               // empty: the provider CLI's own default
	Reply    map[string]ReplyType // every field the answer must carry, and no other
}

// Reply is a model's answer and what it cost.
type Reply struct {
	Fields map[string]any // a string, int or bool per field of Request.Reply
	Tokens int
	Model  string // the model that actually answered; empty when the adapter cannot tell
}

// Model answers requests.
type Model interface {
	Ask(ctx context.Context, request Request) (Reply, error)
}
```

`internal/flowcfg/flowcfg.go`:

```go
// Package flowcfg reads the [agent] table of .ultraloom/config.toml: which
// models a flow may name, which one is the default, and which MCP servers the
// mcp tool profile opens.
//
// It is deliberately small. internal/ulconfig from Go hooks stage 3 will read
// the whole file, and this package folds into it then.
package flowcfg

// ModelSpec is one entry under [agent.models].
type ModelSpec struct {
	Provider string
	Model    string // empty: the provider CLI's own default
}

// Config is the [agent] table.
type Config struct {
	Default    string // empty when unset
	Models     map[string]ModelSpec
	MCPServers []string
}

// Resolved is the model a node ends up with after the model chain.
type Resolved struct {
	Name     string // the name under [agent.models]; empty when no stage named one
	Provider string
	Model    string // empty: the provider CLI's own default
}
```

`internal/flow/expr/doc.go`:

```go
// Package expr parses and evaluates what a flow file says in `when` and
// `max_visits`: conditions over state fields and parameters, and visit caps.
package expr
```

`internal/flow/tmpl/doc.go`:

```go
// Package tmpl renders {{name}} placeholders in instructions, questions and
// messages, in one pass and without logic.
package tmpl
```

`internal/blocks/doc.go`:

```go
// Package blocks holds the node kinds M1 builds: gate, exit and agent.
package blocks
```

`internal/runner/doc.go`:

```go
// Package runner walks a loaded graph, journals every step, retraces a journal
// on resume, and pauses at gates.
package runner
```

`internal/runs/doc.go`:

```go
// Package runs names a run: its number, its marker file, and the two files it
// writes itself.
package runs
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test ./internal/flow/... ./internal/model/...`
Expected: `ok` für `internal/flow` und `internal/model`; `internal/flow/expr` und `internal/flow/tmpl` melden `[no test files]`.

- [ ] **Step 5: Build and vet the whole tree**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go vet ./...`
Expected: keine Ausgabe, Exit 0.

- [ ] **Step 6: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && gofmt -l internal/flow internal/model internal/flowcfg internal/blocks internal/runner internal/runs`
Expected: keine Ausgabe.

- [ ] **Step 7: Commit**

```bash
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && git add internal/flow internal/model internal/flowcfg internal/blocks internal/runner internal/runs
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && git diff --cached --stat
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && git commit -m "Lay down the contracts every ulflow package builds against"
```

---

### Task 1: Lane `expr` — Bedingungen und Deckel

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-expr`, Zweig `ulflow/expr`

**Files:**
- Create: `internal/flow/expr/condition.go`, `internal/flow/expr/cap.go`
- Test: `internal/flow/expr/condition_test.go`, `internal/flow/expr/cap_test.go`

**Interfaces:**
- Consumes: `flow.Type`, `flow.String`, `flow.Int`, `flow.Bool`, `flow.StringList`, `flow.Value`, `flow.State`, `flow.Params`, `flow.Condition`, `flow.Predicate`, `flow.Cap` (Task 0).
- Produces:
  - `type Names struct { Fields map[string]flow.Type; Params map[string]flow.Type; Predicates map[string]flow.Predicate }`
  - `func ParseCondition(source string, names Names) (flow.Condition, error)`
  - `func ParseCap(raw any, params map[string]flow.Type) (flow.Cap, error)` — `raw` ist der Wert, wie TOML ihn decodiert: `nil`, `int64` oder `string`.
  - Fehlertexte ohne Datei- und Knotenpräfix; das setzt der Lader davor.

- [ ] **Step 1: Write the failing condition tests**

`internal/flow/expr/condition_test.go`:

```go
package expr_test

import (
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/flow"
	"github.com/xidus90/ultra-loom/internal/flow/expr"
)

var names = expr.Names{
	Fields: map[string]flow.Type{
		"verdict": flow.String,
		"count":   flow.Int,
		"done":    flow.Bool,
		"notes":   flow.StringList,
	},
	Params: map[string]flow.Type{
		"max_rounds": flow.Int,
		"target":     flow.String,
		"strict":     flow.Bool,
		"tags":       flow.StringList,
	},
	Predicates: map[string]flow.Predicate{
		"green": func(state flow.State, _ flow.Params) bool { return state.Fields["verdict"] == "green" },
	},
}

var params = flow.Params{"max_rounds": 3, "target": "done", "strict": true, "tags": []string{"a"}}

func state(verdict string, count int, done bool, notes []string) flow.State {
	return flow.State{Fields: map[string]flow.Value{
		"verdict": verdict, "count": count, "done": done, "notes": notes,
	}}
}

func TestConditionsHoldAsWritten(t *testing.T) {
	cases := []struct {
		source string
		state  flow.State
		want   bool
	}{
		{`verdict == "done"`, state("done", 0, false, nil), true},
		{`verdict == "done"`, state("open", 0, false, nil), false},
		{`verdict != "done"`, state("open", 0, false, nil), true},
		{`verdict == target`, state("done", 0, false, nil), true},
		{`verdict == "say \"hi\" \\ bye"`, state(`say "hi" \ bye`, 0, false, nil), true},
		{`count < max_rounds`, state("", 2, false, nil), true},
		{`count < max_rounds`, state("", 3, false, nil), false},
		{`count <= 3`, state("", 3, false, nil), true},
		{`count > -1`, state("", 0, false, nil), true},
		{`count >= 4`, state("", 3, false, nil), false},
		{`count == 3`, state("", 3, false, nil), true},
		{`count != 3`, state("", 3, false, nil), false},
		{`  count==3  `, state("", 3, false, nil), true},
		{`done == true`, state("", 0, true, nil), true},
		{`done != false`, state("", 0, false, nil), false},
		{`done == strict`, state("", 0, true, nil), true},
		{`notes == []`, state("", 0, false, nil), true},
		{`notes == []`, state("", 0, false, []string{"x"}), false},
		{`notes != []`, state("", 0, false, []string{"x"}), true},
		{`green`, state("green", 0, false, nil), true},
		{`green`, state("red", 0, false, nil), false},
		{`green | count > 5`, state("red", 6, false, nil), true},
		{`green | count > 5`, state("red", 1, false, nil), false},
		{`green & count > 5`, state("green", 6, false, nil), true},
		{`green & count > 5`, state("green", 1, false, nil), false},
	}
	for _, c := range cases {
		condition, err := expr.ParseCondition(c.source, names)
		if err != nil {
			t.Fatalf("%s: %v", c.source, err)
		}
		if got := condition.Holds(c.state, params); got != c.want {
			t.Errorf("%s: Holds = %v, want %v", c.source, got, c.want)
		}
	}
}

// Every refusal names what it found and, where there is a list to choose from,
// the list. The loader puts the file and the edge in front.
func TestConditionsAreRefusedWithAReason(t *testing.T) {
	cases := []struct{ source, want string }{
		{``, "the condition is empty"},
		{`   `, "the condition is empty"},
		{`verdit == "done"`, `reads "verdit"; known fields: count, done, notes, verdict`},
		{`green | done == true & count > 1`, "the condition mixes | and &; use only one of them"},
		{`| green`, "| has no term before it"},
		{`green | | green`, "| has no term before it"},
		{`green &`, "& has no term after it"},
		{`nothing`, `names "nothing", which is neither a field nor a predicate; known predicates: green`},
		{`count`, `field "count" needs a comparison`},
		{`count == 3 3`, `cannot read "count == 3 3"; expected <field> <op> <value> or a predicate name`},
		{`"a" == verdict`, `cannot read "\"a\" == verdict"`},
		{`count = 3`, "unknown operator at column 7; operators are ==, !=, <, <=, >, >="},
		{`count ! 3`, "unknown operator at column 7"},
		{`count == (3)`, "unexpected '(' at column 10"},
		{`count == -`, "a lone - at column 10 is not a number"},
		{`verdict == "open`, "has no closing quote"},
		{`verdict == "open\`, "has no closing quote"},
		{`count == 99999999999999999999`, "99999999999999999999 is too large for an int"},
		{`count == nope`, `compares "count" with "nope", which is no parameter; known parameters: max_rounds, strict, tags, target`},
		{`count == "3"`, `compares int field "count" with a string`},
		{`notes == tags`, `list field "notes" compares only with []`},
		{`verdict < "b"`, `< needs an int, but "verdict" is string`},
		{`notes <= []`, `<= needs an int, but "notes" is list[string]`},
		{`count == ==`, `cannot compare "count" with ==`},
	}
	for _, c := range cases {
		_, err := expr.ParseCondition(c.source, names)
		if err == nil {
			t.Errorf("%q: want an error containing %q, got none", c.source, c.want)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: error %q does not contain %q", c.source, err, c.want)
		}
	}
}

// With nothing declared the list is not left empty: "known predicates: " with
// nothing behind it reads like a message that was cut off.
func TestAnEmptyListOfNamesSaysNone(t *testing.T) {
	_, err := expr.ParseCondition(`anything`, expr.Names{})
	if err == nil || !strings.Contains(err.Error(), "known predicates: none") {
		t.Fatalf("want known predicates: none, got %v", err)
	}
}
```

- [ ] **Step 2: Run the condition tests to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-expr" && go test ./internal/flow/expr/...`
Expected: FAIL — `undefined: expr.Names`.

- [ ] **Step 3: Implement conditions**

`internal/flow/expr/condition.go`:

```go
package expr

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/xidus90/ultra-loom/internal/flow"
)

// Names tells the parser what a condition may refer to.
type Names struct {
	Fields     map[string]flow.Type
	Params     map[string]flow.Type
	Predicates map[string]flow.Predicate
}

// ParseCondition reads one `when` and checks every name and type in it.
//
// The language is small on purpose: comparisons `<field> <op> <value>` and
// predicate names, joined only by `|` or only by `&`, without brackets.
// Everything it can say is checked here, before a run spends a model call.
func ParseCondition(source string, names Names) (flow.Condition, error) {
	tokens, err := scan(source)
	if err != nil {
		return nil, err
	}
	terms, joiner, err := split(tokens)
	if err != nil {
		return nil, err
	}
	conditions := make([]flow.Condition, 0, len(terms))
	for _, term := range terms {
		condition, err := parseTerm(term, names)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, condition)
	}
	if joiner == pipe {
		return anyOf(conditions), nil
	}
	return allOf(conditions), nil
}

type kind int

const (
	ident kind = iota
	number
	text
	operator
	pipe
	amp
	emptyList
)

type token struct {
	kind  kind
	value string // for text, the content without quotes and escapes
}

func scan(source string) ([]token, error) {
	var tokens []token
	for i := 0; i < len(source); {
		c := source[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case c == '|':
			tokens = append(tokens, token{pipe, "|"})
			i++
		case c == '&':
			tokens = append(tokens, token{amp, "&"})
			i++
		case strings.HasPrefix(source[i:], "[]"):
			tokens = append(tokens, token{emptyList, "[]"})
			i += 2
		case c == '"':
			value, width, err := quoted(source[i:])
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token{text, value})
			i += width
		case strings.IndexByte("=!<>", c) >= 0:
			op := operatorAt(source[i:])
			if op == "" {
				return nil, fmt.Errorf("unknown operator at column %d; operators are ==, !=, <, <=, >, >=", i+1)
			}
			tokens = append(tokens, token{operator, op})
			i += len(op)
		case c == '-' || isDigit(c):
			end := i + 1
			for end < len(source) && isDigit(source[end]) {
				end++
			}
			if source[i:end] == "-" {
				return nil, fmt.Errorf("a lone - at column %d is not a number", i+1)
			}
			tokens = append(tokens, token{number, source[i:end]})
			i = end
		case isLetter(c):
			end := i + 1
			for end < len(source) && (isLetter(source[end]) || isDigit(source[end])) {
				end++
			}
			tokens = append(tokens, token{ident, source[i:end]})
			i = end
		default:
			return nil, fmt.Errorf("unexpected %q at column %d", c, i+1)
		}
	}
	return tokens, nil
}

// operatorAt names the operator at the start of rest. The two-character ones
// are tried first, so "<=" is never read as "<" followed by "=".
func operatorAt(rest string) string {
	for _, op := range []string{"==", "!=", "<=", ">=", "<", ">"} {
		if strings.HasPrefix(rest, op) {
			return op
		}
	}
	return ""
}

// quoted reads the double-quoted text at the start of rest and reports how many
// bytes it took. A backslash takes the next byte literally, so \" and \\ are
// the way to write those two.
func quoted(rest string) (string, int, error) {
	var value strings.Builder
	for i := 1; i < len(rest); i++ {
		switch rest[i] {
		case '"':
			return value.String(), i + 1, nil
		case '\\':
			i++
			if i < len(rest) {
				value.WriteByte(rest[i])
			}
		default:
			value.WriteByte(rest[i])
		}
	}
	return "", 0, fmt.Errorf("text %s has no closing quote", rest)
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isLetter(c byte) bool { return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

// split cuts the tokens at | or & and refuses a condition that uses both:
// without brackets, "a | b & c" has two readings and neither is obviously meant.
func split(tokens []token) ([][]token, kind, error) {
	if len(tokens) == 0 {
		return nil, 0, fmt.Errorf("the condition is empty")
	}
	joiner := kind(-1)
	var terms [][]token
	var current []token
	for _, tok := range tokens {
		if tok.kind != pipe && tok.kind != amp {
			current = append(current, tok)
			continue
		}
		if joiner != -1 && joiner != tok.kind {
			return nil, 0, fmt.Errorf("the condition mixes | and &; use only one of them")
		}
		joiner = tok.kind
		if len(current) == 0 {
			return nil, 0, fmt.Errorf("%s has no term before it", tok.value)
		}
		terms = append(terms, current)
		current = nil
	}
	if len(current) == 0 {
		return nil, 0, fmt.Errorf("%s has no term after it", tokens[len(tokens)-1].value)
	}
	return append(terms, current), joiner, nil
}

func parseTerm(term []token, names Names) (flow.Condition, error) {
	if len(term) == 1 && term[0].kind == ident {
		return predicateTerm(term[0].value, names)
	}
	if len(term) == 3 && term[0].kind == ident && term[1].kind == operator {
		return comparisonTerm(term[0].value, term[1].value, term[2], names)
	}
	return nil, fmt.Errorf("cannot read %q; expected <field> <op> <value> or a predicate name", render(term))
}

func render(term []token) string {
	parts := make([]string, len(term))
	for i, tok := range term {
		parts[i] = tok.value
		if tok.kind == text {
			parts[i] = strconv.Quote(tok.value)
		}
	}
	return strings.Join(parts, " ")
}

func predicateTerm(name string, names Names) (flow.Condition, error) {
	if predicate, ok := names.Predicates[name]; ok {
		return predicateCondition{predicate}, nil
	}
	if _, ok := names.Fields[name]; ok {
		return nil, fmt.Errorf("field %q needs a comparison, e.g. %s == ...", name, name)
	}
	return nil, fmt.Errorf("names %q, which is neither a field nor a predicate; known predicates: %s",
		name, known(names.Predicates))
}

func comparisonTerm(field, op string, value token, names Names) (flow.Condition, error) {
	fieldType, ok := names.Fields[field]
	if !ok {
		return nil, fmt.Errorf("reads %q; known fields: %s", field, known(names.Fields))
	}
	c := comparison{field: field, op: op, kind: fieldType}
	var valueType flow.Type
	switch value.kind {
	case text:
		valueType, c.literal = flow.String, value.value
	case number:
		n, err := strconv.Atoi(value.value)
		if err != nil {
			return nil, fmt.Errorf("%s is too large for an int", value.value)
		}
		valueType, c.literal = flow.Int, n
	case emptyList:
		valueType = flow.StringList
	case ident:
		if value.value == "true" || value.value == "false" {
			valueType, c.literal = flow.Bool, value.value == "true"
			break
		}
		paramType, ok := names.Params[value.value]
		if !ok {
			return nil, fmt.Errorf("compares %q with %q, which is no parameter; known parameters: %s",
				field, value.value, known(names.Params))
		}
		valueType, c.param = paramType, value.value
	default:
		return nil, fmt.Errorf("cannot compare %q with %s", field, value.value)
	}
	if valueType != fieldType {
		return nil, fmt.Errorf("compares %s field %q with a %s", fieldType, field, valueType)
	}
	if fieldType == flow.StringList && c.param != "" {
		return nil, fmt.Errorf("list field %q compares only with []", field)
	}
	if op != "==" && op != "!=" && fieldType != flow.Int {
		return nil, fmt.Errorf("%s needs an int, but %q is %s", op, field, fieldType)
	}
	return c, nil
}

type comparison struct {
	field   string
	op      string
	kind    flow.Type
	literal flow.Value
	param   string // set: compare with this parameter instead of literal
}

func (c comparison) Holds(state flow.State, params flow.Params) bool {
	left := state.Fields[c.field]
	right := c.literal
	if c.param != "" {
		right = params[c.param]
	}
	switch c.kind {
	case flow.Int:
		return compareInts(left.(int), c.op, right.(int))
	case flow.StringList:
		empty := len(left.([]string)) == 0
		return empty == (c.op == "==")
	default:
		return (left == right) == (c.op == "==")
	}
}

func compareInts(left int, op string, right int) bool {
	switch op {
	case "==":
		return left == right
	case "!=":
		return left != right
	case "<":
		return left < right
	case "<=":
		return left <= right
	case ">":
		return left > right
	default:
		return left >= right
	}
}

type predicateCondition struct{ holds flow.Predicate }

func (p predicateCondition) Holds(state flow.State, params flow.Params) bool {
	return p.holds(state, params)
}

type anyOf []flow.Condition

func (terms anyOf) Holds(state flow.State, params flow.Params) bool {
	for _, term := range terms {
		if term.Holds(state, params) {
			return true
		}
	}
	return false
}

type allOf []flow.Condition

func (terms allOf) Holds(state flow.State, params flow.Params) bool {
	for _, term := range terms {
		if !term.Holds(state, params) {
			return false
		}
	}
	return true
}

// known lists the names a message offers, sorted, or says there are none.
func known[T any](names map[string]T) string {
	if len(names) == 0 {
		return "none"
	}
	list := make([]string, 0, len(names))
	for name := range names {
		list = append(list, name)
	}
	slices.Sort(list)
	return strings.Join(list, ", ")
}
```

- [ ] **Step 4: Run the condition tests to verify they pass**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-expr" && go test ./internal/flow/expr/...`
Expected: `ok`.

- [ ] **Step 5: Write the failing cap tests**

`internal/flow/expr/cap_test.go`:

```go
package expr_test

import (
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/flow"
	"github.com/xidus90/ultra-loom/internal/flow/expr"
)

var capParams = map[string]flow.Type{"max_rounds": flow.Int, "topic": flow.String}

func TestCapsReadEveryForm(t *testing.T) {
	cases := []struct {
		raw  any
		want flow.Cap
	}{
		{nil, flow.Cap{Add: 1}},
		{int64(4), flow.Cap{Add: 4}},
		{"max_rounds", flow.Cap{Param: "max_rounds"}},
		{"max_rounds + 1", flow.Cap{Param: "max_rounds", Add: 1}},
		{"max_rounds+2", flow.Cap{Param: "max_rounds", Add: 2}},
	}
	for _, c := range cases {
		got, err := expr.ParseCap(c.raw, capParams)
		if err != nil {
			t.Fatalf("%v: %v", c.raw, err)
		}
		if got != c.want {
			t.Errorf("%v: got %+v, want %+v", c.raw, got, c.want)
		}
	}
}

func TestCapsAreRefusedWithAReason(t *testing.T) {
	cases := []struct {
		raw  any
		want string
	}{
		{int64(0), "max_visits must be at least 1, got 0"},
		{int64(-2), "max_visits must be at least 1, got -2"},
		{1.5, `max_visits must be an integer or "<parameter> + <integer>", got 1.5`},
		{"max_rounds + x", `max_visits "max_rounds + x" is not "<parameter> + <integer>"`},
		{"max_rounds + -1", `max_visits "max_rounds + -1" is not "<parameter> + <integer>"`},
		{"rounds", `max_visits "rounds" names no parameter; known parameters: max_rounds, topic`},
		{"topic + 1", `max_visits parameter "topic" is string, not int`},
	}
	for _, c := range cases {
		_, err := expr.ParseCap(c.raw, capParams)
		if err == nil {
			t.Errorf("%v: want an error containing %q, got none", c.raw, c.want)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%v: error %q does not contain %q", c.raw, err, c.want)
		}
	}
}
```

- [ ] **Step 6: Run the cap tests to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-expr" && go test ./internal/flow/expr/...`
Expected: FAIL — `undefined: expr.ParseCap`.

- [ ] **Step 7: Implement caps**

`internal/flow/expr/cap.go`:

```go
package expr

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xidus90/ultra-loom/internal/flow"
)

// ParseCap reads max_visits as TOML decoded it: absent, an integer, or a text
// naming an int parameter, optionally plus a non-negative integer. It is the
// only arithmetic a flow file has.
func ParseCap(raw any, params map[string]flow.Type) (flow.Cap, error) {
	switch value := raw.(type) {
	case nil:
		return flow.Cap{Add: 1}, nil
	case int64:
		if value < 1 {
			return flow.Cap{}, fmt.Errorf("max_visits must be at least 1, got %d", value)
		}
		return flow.Cap{Add: int(value)}, nil
	case string:
		return capText(value, params)
	default:
		return flow.Cap{}, fmt.Errorf("max_visits must be an integer or \"<parameter> + <integer>\", got %v", raw)
	}
}

func capText(source string, params map[string]flow.Type) (flow.Cap, error) {
	name, add, hasAdd := strings.Cut(source, "+")
	name = strings.TrimSpace(name)
	limit := flow.Cap{Param: name}
	if hasAdd {
		n, err := strconv.Atoi(strings.TrimSpace(add))
		if err != nil || n < 0 {
			return flow.Cap{}, fmt.Errorf("max_visits %q is not \"<parameter> + <integer>\"", source)
		}
		limit.Add = n
	}
	paramType, ok := params[name]
	if !ok {
		return flow.Cap{}, fmt.Errorf("max_visits %q names no parameter; known parameters: %s", source, known(params))
	}
	if paramType != flow.Int {
		return flow.Cap{}, fmt.Errorf("max_visits parameter %q is %s, not int", name, paramType)
	}
	return limit, nil
}
```

- [ ] **Step 8: Run all tests with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-expr" && go test -cover ./internal/flow/expr/...`
Expected: `ok … coverage: 100.0% of statements`.

- [ ] **Step 9: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-expr" && gofmt -l internal/flow/expr`
Expected: keine Ausgabe.

- [ ] **Step 10: Commit**

```bash
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-expr" && git add internal/flow/expr
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-expr" && git diff --cached --stat
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-expr" && git commit -m "Parse and evaluate flow conditions and visit caps"
```

---

### Task 2: Lane `tmpl` — Platzhalter

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-tmpl`, Zweig `ulflow/tmpl`

**Files:**
- Create: `internal/flow/tmpl/tmpl.go`
- Test: `internal/flow/tmpl/tmpl_test.go`

**Interfaces:**
- Consumes: nichts aus Task 0 außer `doc.go`. Werte kommen als `map[string]any` mit `string`, `int`, `bool` oder `[]string`.
- Produces:
  - `type Template struct{ /* unexported */ }`
  - `func Parse(text string) (Template, error)`
  - `func (t Template) Names() []string` — sortiert, jeder Name einmal
  - `func (t Template) Render(values map[string]any) (string, error)`

- [ ] **Step 1: Write the failing tests**

`internal/flow/tmpl/tmpl_test.go`:

```go
package tmpl_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/flow/tmpl"
)

func parse(t *testing.T, text string) tmpl.Template {
	t.Helper()
	template, err := tmpl.Parse(text)
	if err != nil {
		t.Fatalf("%q: %v", text, err)
	}
	return template
}

func TestRenderFillsEveryKindOfValue(t *testing.T) {
	template := parse(t, "Review {{topic}} in round {{round}} (strict: {{strict}}):\n{{notes}}")
	got, err := template.Render(map[string]any{
		"topic": "the spec", "round": 2, "strict": true, "notes": []string{"one", "two"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "Review the spec in round 2 (strict: true):\n- one\n- two"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAnEmptyListRendersAsNothing(t *testing.T) {
	got, err := parse(t, "[{{notes}}]").Render(map[string]any{"notes": []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "[]" {
		t.Fatalf("got %q, want []", got)
	}
}

// One pass: a value that looks like a placeholder is written as it is. A second
// pass would let a model's answer pull other fields into the next prompt.
func TestRenderIsOnePass(t *testing.T) {
	got, err := parse(t, "{{answer}}").Render(map[string]any{"answer": "{{secret}}"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "{{secret}}" {
		t.Fatalf("got %q, want the value unread", got)
	}
}

func TestEscapedBracesAreLiteral(t *testing.T) {
	template := parse(t, `JSON: \{{"a": 1}} and {{name}}`)
	if names := template.Names(); !slices.Equal(names, []string{"name"}) {
		t.Fatalf("names = %v, want [name]", names)
	}
	got, err := template.Render(map[string]any{"name": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if got != `JSON: {{"a": 1}} and x` {
		t.Fatalf("got %q", got)
	}
}

func TestNamesAreSortedAndUnique(t *testing.T) {
	names := parse(t, "{{b}} {{a_1}} {{b}}").Names()
	if !slices.Equal(names, []string{"a_1", "b"}) {
		t.Fatalf("names = %v, want [a_1 b]", names)
	}
}

func TestATextWithoutPlaceholdersStaysAsItIs(t *testing.T) {
	template := parse(t, "plain } { text")
	if len(template.Names()) != 0 {
		t.Fatalf("names = %v, want none", template.Names())
	}
	got, err := template.Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "plain } { text" {
		t.Fatalf("got %q", got)
	}
}

func TestParseRefusesWhatIsNoPlaceholder(t *testing.T) {
	cases := []struct{ text, want string }{
		{"Hello {{name", "unclosed {{ at byte 6"},
		{"{{ name }}", "{{ name }} is not a placeholder"},
		{"{{}}", "{{}} is not a placeholder"},
		{"{{1st}}", "{{1st}} is not a placeholder"},
		{`{"a": {{"b": 1}}}`, "is not a placeholder"},
	}
	for _, c := range cases {
		_, err := tmpl.Parse(c.text)
		if err == nil {
			t.Errorf("%q: want an error containing %q, got none", c.text, c.want)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: error %q does not contain %q", c.text, err, c.want)
		}
	}
}

func TestRenderNamesAMissingValue(t *testing.T) {
	_, err := parse(t, "{{topic}}").Render(map[string]any{})
	if err == nil || err.Error() != "no value for {{topic}}" {
		t.Fatalf("got %v", err)
	}
}

func TestRenderRefusesAValueNoTextCanShow(t *testing.T) {
	_, err := parse(t, "{{n}}").Render(map[string]any{"n": int64(3)})
	if err == nil || err.Error() != "{{n}} holds a int64, which a text cannot show" {
		t.Fatalf("got %v", err)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-tmpl" && go test ./internal/flow/tmpl/...`
Expected: FAIL — `undefined: tmpl.Parse`.

- [ ] **Step 3: Implement**

`internal/flow/tmpl/tmpl.go`:

```go
package tmpl

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Template is a parsed text: literal runs and {{name}} placeholders.
type Template struct {
	parts []part
}

type part struct {
	literal string
	name    string // set: this part is a placeholder
}

// Parse reads the {{name}} placeholders out of a text; \{{ writes {{ literally.
//
// A placeholder holds a name and nothing else, not even whitespace.
// Instructions carry JSON and Go code, and a lenient reading would turn their
// braces into names nobody declared.
func Parse(text string) (Template, error) {
	var t Template
	var literal strings.Builder
	flush := func() {
		if literal.Len() > 0 {
			t.parts = append(t.parts, part{literal: literal.String()})
			literal.Reset()
		}
	}
	for i := 0; i < len(text); {
		switch {
		case strings.HasPrefix(text[i:], `\{{`):
			literal.WriteString("{{")
			i += 3
		case strings.HasPrefix(text[i:], "{{"):
			end := strings.Index(text[i+2:], "}}")
			if end < 0 {
				return Template{}, fmt.Errorf("unclosed {{ at byte %d", i)
			}
			name := text[i+2 : i+2+end]
			if !isName(name) {
				return Template{}, fmt.Errorf(
					"{{%s}} is not a placeholder; a name is letters, digits and _, and \\{{ writes literal braces", name)
			}
			flush()
			t.parts = append(t.parts, part{name: name})
			i += end + 4
		default:
			literal.WriteByte(text[i])
			i++
		}
	}
	flush()
	return t, nil
}

func isName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		letter := c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		digit := c >= '0' && c <= '9'
		if !letter && !(digit && i > 0) {
			return false
		}
	}
	return true
}

// Names are the placeholders of the text, sorted and each once.
func (t Template) Names() []string {
	var names []string
	for _, p := range t.parts {
		if p.name != "" && !slices.Contains(names, p.name) {
			names = append(names, p.name)
		}
	}
	slices.Sort(names)
	return names
}

// Render fills every placeholder in one pass: a value that itself contains
// {{...}} is written as it is and never read again.
func (t Template) Render(values map[string]any) (string, error) {
	var out strings.Builder
	for _, p := range t.parts {
		if p.name == "" {
			out.WriteString(p.literal)
			continue
		}
		value, ok := values[p.name]
		if !ok {
			return "", fmt.Errorf("no value for {{%s}}", p.name)
		}
		shown, err := show(p.name, value)
		if err != nil {
			return "", err
		}
		out.WriteString(shown)
	}
	return out.String(), nil
}

// show writes one value the way a prompt reads it: a list becomes one "- "
// line per entry.
func show(name string, value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	case bool:
		return strconv.FormatBool(v), nil
	case []string:
		lines := make([]string, len(v))
		for i, item := range v {
			lines[i] = "- " + item
		}
		return strings.Join(lines, "\n"), nil
	default:
		return "", fmt.Errorf("{{%s}} holds a %T, which a text cannot show", name, value)
	}
}
```

- [ ] **Step 4: Run the tests with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-tmpl" && go test -cover ./internal/flow/tmpl/...`
Expected: `ok … coverage: 100.0% of statements`.

- [ ] **Step 5: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-tmpl" && gofmt -l internal/flow/tmpl`
Expected: keine Ausgabe.

- [ ] **Step 6: Commit**

```bash
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-tmpl" && git add internal/flow/tmpl
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-tmpl" && git diff --cached --stat
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-tmpl" && git commit -m "Render placeholders in flow texts in one pass"
```

---

### Task 3: Lane `journal` — Schreiber, Hashes, optionale Felder

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-journal`, Zweig `ulflow/journal`

**Files:**
- Modify: `internal/journal/journal.go` (Paketkommentar Zeilen 1–6, `Entry`, `entryKeys`-Umfeld, Schleife über unbekannte Schlüssel in `decode`)
- Create: `internal/journal/hash.go`, `internal/journal/write.go`, `internal/journal/lookup.go`
- Test: `internal/journal/journal_test.go` (ergänzen), `internal/journal/hash_test.go`, `internal/journal/write_test.go`, `internal/journal/write_internal_test.go`, `internal/journal/lookup_test.go`

**Interfaces:**
- Consumes: nichts aus Task 0.
- Produces:
  - `Entry` bekommt `Model *string` (`json:"model"`) und `DefinitionHash *string` (`json:"definition_hash"`).
  - `func Canonical(value any) ([]byte, error)`
  - `func InputHash(node string, data any) (string, error)` — sha256 hex über `Canonical({"data": data, "node": node})`
  - `func DefinitionHash(node any, text []byte, model, effort string, tools []string) (string, error)`
  - `func Append(path string, entry Entry) error`
  - `func Lookup(entries []Entry, node, inputHash, outcome string) (Entry, bool)` — leeres `outcome` passt auf jedes

- [ ] **Step 1: Write the failing reader tests**

An `internal/journal/journal_test.go` anhängen:

```go
const goLine = `{"definition_hash":"d1","delta":{},"detail":null,"effort":"high","input_hash":"h2","kind":"agent","model":"claude:cli-default","node":"draft","outcome":"ok","seconds":1,"tokens":7,"tools":"edit"}`

// ulflow writes two keys the Python runtime never did, and a reader that only
// knew ten would call every Go run damaged -- ulguard's session-start included.
func TestEntriesReadsTheTwoKeysOnlyUlflowWrites(t *testing.T) {
	got, err := journal.Entries(write(t, goLine+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Model == nil || *got[0].Model != "claude:cli-default" {
		t.Fatalf("model not read: %+v", got[0].Model)
	}
	if got[0].DefinitionHash == nil || *got[0].DefinitionHash != "d1" {
		t.Fatalf("definition_hash not read: %+v", got[0].DefinitionHash)
	}
}

// Absent is valid for these two, unlike the ten: a Python journal has neither.
func TestEntriesReadsAPythonLineWithoutThem(t *testing.T) {
	got, err := journal.Entries(write(t, okLine+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Model != nil || got[0].DefinitionHash != nil {
		t.Fatalf("absent keys carry no value, got %+v / %+v", got[0].Model, got[0].DefinitionHash)
	}
}

// A gate has no model, and ulflow writes that as null rather than leaving the
// key out.
func TestEntriesReadsANullModel(t *testing.T) {
	gate := strings.Replace(pausedLine, `{"delta"`, `{"model":null,"definition_hash":"d2","delta"`, 1)
	got, err := journal.Entries(write(t, gate+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Model != nil {
		t.Fatalf("a null model carries no value, got %q", *got[0].Model)
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-journal" && go test ./internal/journal/...`
Expected: FAIL — `got[0].Model undefined`.

- [ ] **Step 3: Teach the reader the two optional keys**

In `internal/journal/journal.go` den Paketkommentar (Zeilen 1–6) ersetzen durch:

```go
// Package journal is the run journal: one JSONL line per node.
//
// The same file is the log a person reads to evaluate a run and the only
// source a resume reads from. `session-start` in ulguard only reads it; ulflow
// writes it and fingerprints what each node saw and was told.
package journal
```

In `Entry` nach dem Feld `Detail` einfügen:

```go

	// Model and DefinitionHash are written by ulflow alone. A journal of the
	// Python runtime has neither key, and it stays readable.
	Model          *string `json:"model"`
	DefinitionHash *string `json:"definition_hash"`
```

Direkt nach der Variablen `entryKeys` einfügen:

```go

// optionalKeys may be absent. ulflow writes both on every line; a journal
// written before ulflow existed has neither, and reading it is the point.
var optionalKeys = []string{"model", "definition_hash"}
```

In `decode` die Bedingung der Schleife über unbekannte Schlüssel ersetzen:

```go
		if !slices.Contains(entryKeys, key) && !slices.Contains(optionalKeys, key) {
```

- [ ] **Step 4: Run the reader tests to verify they pass**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-journal" && go test ./internal/journal/...`
Expected: `ok`.

- [ ] **Step 5: Write the failing hash tests**

`internal/journal/hash_test.go`:

```go
package journal_test

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/journal"
)

// encoding/json sorts map keys but writes struct fields in declaration order;
// a hash that turned on that order would redo finished work on resume.
func TestCanonicalSortsKeysAtEveryDepthAndKeepsHTML(t *testing.T) {
	value := struct {
		Zeta  int            `json:"zeta"`
		Alpha map[string]any `json:"alpha"`
	}{Zeta: 2, Alpha: map[string]any{"y": []any{1, "<&>"}, "x": nil}}

	got, err := journal.Canonical(value)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"alpha":{"x":null,"y":[1,"<&>"]},"zeta":2}`; string(got) != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestCanonicalKeepsLargeNumbersExact(t *testing.T) {
	got, err := journal.Canonical(map[string]any{"big": uint64(18446744073709551615), "small": 0.1})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"big":18446744073709551615,"small":0.1}`; string(got) != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestCanonicalRefusesWhatJSONCannotSay(t *testing.T) {
	_, err := journal.Canonical(math.Inf(1))
	if err == nil || !strings.Contains(err.Error(), "cannot serialize") {
		t.Fatalf("got %v", err)
	}
}

func TestInputHashIsSha256OfTheCanonicalPayload(t *testing.T) {
	got, err := journal.InputHash("draft", map[string]any{"b": 2, "a": 1})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(`{"data":{"a":1,"b":2},"node":"draft"}`))
	if want := hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	other, err := journal.InputHash("review", map[string]any{"a": 1, "b": 2})
	if err != nil {
		t.Fatal(err)
	}
	if other == got {
		t.Fatal("the node name is part of the hash")
	}
}

func TestInputHashRefusesUnserializableData(t *testing.T) {
	if _, err := journal.InputHash("n", map[string]any{"f": func() {}}); err == nil {
		t.Fatal("a func has no JSON form, so it has no hash")
	}
}

func TestDefinitionHashMovesWithEveryPart(t *testing.T) {
	node := map[string]any{"name": "draft"}
	text := []byte("Write it.")
	tools := []string{"Read"}
	base, err := journal.DefinitionHash(node, text, "claude:cli-default", "high", tools)
	if err != nil {
		t.Fatal(err)
	}
	again, err := journal.DefinitionHash(node, text, "claude:cli-default", "high", tools)
	if err != nil {
		t.Fatal(err)
	}
	if again != base {
		t.Fatal("the same definition must hash the same")
	}
	variants := map[string]func() (string, error){
		"node": func() (string, error) {
			return journal.DefinitionHash(map[string]any{"name": "review"}, text, "claude:cli-default", "high", tools)
		},
		"text": func() (string, error) {
			return journal.DefinitionHash(node, []byte("Write it well."), "claude:cli-default", "high", tools)
		},
		"model": func() (string, error) {
			return journal.DefinitionHash(node, text, "claude:claude-opus-5", "high", tools)
		},
		"effort": func() (string, error) {
			return journal.DefinitionHash(node, text, "claude:cli-default", "low", tools)
		},
		"tools": func() (string, error) {
			return journal.DefinitionHash(node, text, "claude:cli-default", "high", []string{"Edit", "Read"})
		},
	}
	for part, hash := range variants {
		got, err := hash()
		if err != nil {
			t.Fatal(err)
		}
		if got == base {
			t.Errorf("changing the %s left the hash unchanged", part)
		}
	}
}

func TestDefinitionHashRefusesAnUnserializableNode(t *testing.T) {
	if _, err := journal.DefinitionHash(map[string]any{"x": math.NaN()}, nil, "", "", nil); err == nil {
		t.Fatal("NaN has no JSON form, so it has no hash")
	}
}
```

- [ ] **Step 6: Run them to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-journal" && go test ./internal/journal/...`
Expected: FAIL — `undefined: journal.Canonical`.

- [ ] **Step 7: Implement the hashes**

`internal/journal/hash.go`:

```go
package journal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Canonical is the one JSON spelling the writer and both hashes use: object
// keys sorted at every depth, no HTML escaping, no trailing newline.
//
// encoding/json sorts map keys but writes struct fields in declaration order,
// so the value is marshalled once, read back as generic JSON with numbers kept
// as written, and marshalled again. Every object is a map by then.
func Canonical(value any) ([]byte, error) {
	first, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("cannot serialize: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(first))
	// Without UseNumber a large integer would pass through a float64 and come
	// out as a different number.
	decoder.UseNumber()
	var generic any
	// json.Marshal's own output always decodes; there is no error to handle.
	_ = decoder.Decode(&generic)

	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	// Only maps, slices, strings, bools, nil and json.Number are left, and all
	// of them encode.
	_ = encoder.Encode(generic)
	return bytes.TrimSuffix(out.Bytes(), []byte("\n")), nil
}

// InputHash is journal.py's input_hash: a stable fingerprint of what a node saw.
// A resume finds a node's earlier result by it.
func InputHash(node string, data any) (string, error) {
	return digest(map[string]any{"node": node, "data": data})
}

// DefinitionHash fingerprints what a node was told: its entry in the flow file,
// the raw bytes of its instruction or question, the resolved model, effort and
// tool list. A replay compares it to say which nodes ran on an older definition.
func DefinitionHash(node any, text []byte, model, effort string, tools []string) (string, error) {
	return digest(map[string]any{"node": node, "text": text, "model": model, "effort": effort, "tools": tools})
}

func digest(value any) (string, error) {
	blob, err := Canonical(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:]), nil
}
```

- [ ] **Step 8: Run the hash tests to verify they pass**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-journal" && go test ./internal/journal/...`
Expected: `ok`.

- [ ] **Step 9: Write the failing writer and lookup tests**

`internal/journal/write_test.go`:

```go
package journal_test

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/journal"
)

func text(s string) *string { return &s }

func TestAppendWritesOneCanonicalLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs", "0001.jsonl")
	entry := journal.Entry{
		Node: "draft", Kind: "agent", InputHash: "h",
		Delta:   map[string]any{"z": 1, "a": "<b>"},
		Outcome: "ok", Tools: text("edit"), Effort: text("low"), Tokens: 3, Seconds: 1.5,
		Model: text("claude:cli-default"), DefinitionHash: text("d"),
	}
	if err := journal.Append(path, entry); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"definition_hash":"d","delta":{"a":"<b>","z":1},"detail":null,"effort":"low","input_hash":"h",` +
		`"kind":"agent","model":"claude:cli-default","node":"draft","outcome":"ok","seconds":1.5,"tokens":3,"tools":"edit"}` + "\n"
	if string(got) != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

// What Append writes, Entries reads back -- including the null a gate writes
// for its model.
func TestAppendThenEntriesRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0002.jsonl")
	gate := journal.Entry{Node: "approve", Kind: "gate", InputHash: "h1", Delta: map[string]any{},
		Outcome: "paused", Detail: text("ship it?"), DefinitionHash: text("d1")}
	agent := journal.Entry{Node: "draft", Kind: "agent", InputHash: "h2", Delta: map[string]any{"n": 2},
		Outcome: "ok", Tokens: 9, Seconds: 0.25, Model: text("claude:claude-opus-5"), DefinitionHash: text("d2")}
	for _, entry := range []journal.Entry{gate, agent} {
		if err := journal.Append(path, entry); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"model":null`) {
		t.Fatalf("a gate writes its model as null: %s", raw)
	}
	got, err := journal.Entries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Model != nil || *got[0].Detail != "ship it?" ||
		*got[1].Model != "claude:claude-opus-5" || got[1].Delta["n"] != float64(2) {
		t.Fatalf("round trip lost something: %+v", got)
	}
}

// Refused before the file is touched, so a bad entry leaves no half line.
func TestAppendRefusesAnUnserializableEntryBeforeTouchingTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs", "0003.jsonl")
	err := journal.Append(path, journal.Entry{Node: "draft", Delta: map[string]any{"x": math.NaN()}})
	if err == nil || !strings.Contains(err.Error(), "draft") {
		t.Fatalf("want an error naming the node, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Dir(path)); !os.IsNotExist(statErr) {
		t.Fatalf("nothing may be created for a refused entry, stat says %v", statErr)
	}
}

func TestAppendReportsAParentThatIsAFile(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := journal.Append(filepath.Join(blocker, "0001.jsonl"), journal.Entry{Node: "n"})
	if err == nil || !strings.Contains(err.Error(), "creating") {
		t.Fatalf("got %v", err)
	}
}

func TestAppendReportsAPathThatIsADirectory(t *testing.T) {
	err := journal.Append(t.TempDir(), journal.Entry{Node: "n"})
	if err == nil || !strings.Contains(err.Error(), "opening") {
		t.Fatalf("got %v", err)
	}
}
```

`internal/journal/write_internal_test.go`:

```go
package journal

import (
	"errors"
	"testing"
)

// A disk that refuses a write to an open file cannot be arranged from a test,
// so the wrapping is checked on its own.
func TestClosedWrapsAFailedWrite(t *testing.T) {
	if err := closed("run.jsonl", nil); err != nil {
		t.Fatalf("no failure, no error: %v", err)
	}
	err := closed("run.jsonl", errors.New("disk full"))
	if err == nil || err.Error() != "writing run.jsonl: disk full" {
		t.Fatalf("got %v", err)
	}
}
```

`internal/journal/lookup_test.go`:

```go
package journal_test

import (
	"testing"

	"github.com/xidus90/ultra-loom/internal/journal"
)

func TestLookupFindsTheLatestMatch(t *testing.T) {
	entries := []journal.Entry{
		{Node: "draft", InputHash: "h1", Outcome: "ok", Tokens: 1},
		{Node: "draft", InputHash: "h1", Outcome: "ok", Tokens: 2},
		{Node: "draft", InputHash: "h1", Outcome: "error", Tokens: 3},
		{Node: "draft", InputHash: "h2", Outcome: "ok", Tokens: 4},
		{Node: "review", InputHash: "h1", Outcome: "ok", Tokens: 5},
	}
	// With an outcome the search skips a later decoy instead of stopping at it:
	// a visit limit or a pause writes non-ok entries under a key that succeeded.
	if got, ok := journal.Lookup(entries, "draft", "h1", "ok"); !ok || got.Tokens != 2 {
		t.Fatalf("latest ok: got %+v, %v", got, ok)
	}
	if got, ok := journal.Lookup(entries, "draft", "h1", ""); !ok || got.Tokens != 3 {
		t.Fatalf("latest of any outcome: got %+v, %v", got, ok)
	}
	if _, ok := journal.Lookup(entries, "draft", "h3", ""); ok {
		t.Fatal("an unknown input has no entry")
	}
	if _, ok := journal.Lookup(entries, "review", "h1", "paused"); ok {
		t.Fatal("no paused entry for review")
	}
}
```

- [ ] **Step 10: Run them to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-journal" && go test ./internal/journal/...`
Expected: FAIL — `undefined: journal.Append`.

- [ ] **Step 11: Implement the writer and the lookup**

`internal/journal/write.go`:

```go
package journal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Append writes one entry as one line, creating the file and its parents.
//
// The line is built before the file is touched, so an entry JSON cannot express
// leaves nothing behind. LF on every platform: the journal is a data format
// whose bytes a resume and the golden-journal test compare.
func Append(path string, entry Entry) error {
	line, err := Canonical(record(entry))
	if err != nil {
		return fmt.Errorf("journal entry for node %s: %w", entry.Node, err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	_, writeErr := file.Write(append(line, '\n'))
	return closed(path, errors.Join(writeErr, file.Close()))
}

// closed reports a failed write or close. Apart from Append, because a disk
// that refuses an open file is the one failure no test can provoke.
func closed(path string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("writing %s: %w", path, err)
}

// record spells an entry with every key, so model and definition_hash are
// written as null rather than left out.
func record(entry Entry) map[string]any {
	return map[string]any{
		"node":            entry.Node,
		"kind":            entry.Kind,
		"input_hash":      entry.InputHash,
		"delta":           entry.Delta,
		"outcome":         entry.Outcome,
		"tools":           entry.Tools,
		"effort":          entry.Effort,
		"tokens":          entry.Tokens,
		"seconds":         entry.Seconds,
		"detail":          entry.Detail,
		"model":           entry.Model,
		"definition_hash": entry.DefinitionHash,
	}
}
```

`internal/journal/lookup.go`:

```go
package journal

// Lookup is journal.py's lookup: the latest entry for this node and this input.
//
// An empty outcome matches any. A non-empty one skips entries that ended
// otherwise instead of stopping at the first match, because a visit limit and a
// gate's pause both write non-ok entries under the key of an entry that
// succeeded.
func Lookup(entries []Entry, node, inputHash, outcome string) (Entry, bool) {
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if entry.Node != node || entry.InputHash != inputHash {
			continue
		}
		if outcome == "" || entry.Outcome == outcome {
			return entry, true
		}
	}
	return Entry{}, false
}
```

- [ ] **Step 12: Run the package with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-journal" && go test -cover ./internal/journal/...`
Expected: `ok … coverage: 100.0% of statements`.

- [ ] **Step 13: The session-start hook still reads journals**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-journal" && go test ./cmd/guard/...`
Expected: `ok`.

- [ ] **Step 14: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-journal" && gofmt -l internal/journal`
Expected: keine Ausgabe.

- [ ] **Step 15: Commit**

```bash
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-journal" && git add internal/journal
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-journal" && git diff --cached --stat
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-journal" && git commit -m "Write the journal and fingerprint what each node saw and was told"
```

---

### Task 4: Lane `model` — Fake-Modell

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-model`, Zweig `ulflow/model`

**Files:**
- Create: `internal/model/fake.go`
- Test: `internal/model/fake_test.go`

**Interfaces:**
- Consumes: `model.Request`, `model.Reply`, `model.Model` (Task 0).
- Produces:
  - `type Answer struct { Reply Reply; Err error }`
  - `type Fake struct{ /* unexported */ }`
  - `func NewFake(answers ...Answer) *Fake`
  - `func (f *Fake) Ask(ctx context.Context, request Request) (Reply, error)`
  - `func (f *Fake) Seen() []Request`

- [ ] **Step 1: Write the failing tests**

`internal/model/fake_test.go`:

```go
package model_test

import (
	"context"
	"errors"
	"testing"

	"github.com/xidus90/ultra-loom/internal/model"
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
```

- [ ] **Step 2: Run them to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-model" && go test ./internal/model/...`
Expected: FAIL — `undefined: model.Fake`.

- [ ] **Step 3: Implement**

`internal/model/fake.go`:

```go
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

// Fake answers from a queue and records every request it was handed. It is
// reachable from tests only; no configuration selects it.
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
```

- [ ] **Step 4: Run with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-model" && go test -cover ./internal/model/...`
Expected: `ok … coverage: 100.0% of statements`.

- [ ] **Step 5: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-model" && gofmt -l internal/model`
Expected: keine Ausgabe.

- [ ] **Step 6: Commit**

```bash
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-model" && git add internal/model/fake.go internal/model/fake_test.go
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-model" && git diff --cached --stat
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-model" && git commit -m "Add a fake model that answers from a queue"
```

---

### Task 5: Lane `gitwork` — geänderte Dateien

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-gitwork`, Zweig `ulflow/gitwork`

**Files:**
- Modify: `internal/gitwork/gitwork.go` (anhängen: `ChangedFiles`, `parseStatus`)
- Test: `internal/gitwork/changed_test.go` (nutzt die Helfer `repo`, `commit`, `run` aus `gitwork_test.go`)

**Interfaces:**
- Consumes: `ignored`, `git`, `ErrIgnoredRoot` aus `internal/gitwork/gitwork.go`.
- Produces: `func ChangedFiles(root string) ([]string, error)` — Pfade relativ zu `root`, mit `/`, in der Reihenfolge von git.

- [ ] **Step 1: Write the failing tests**

`internal/gitwork/changed_test.go`:

```go
package gitwork_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/gitwork"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestChangedFilesOfACleanTreeIsEmpty(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	got, err := gitwork.ChangedFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("a clean tree has no changes, got %v", got)
	}
}

// -uall, so an untracked directory comes back as the files in it and not as
// one entry that is no file's path.
func TestChangedFilesListsModifiedAndUntrackedFiles(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	writeFile(t, filepath.Join(root, "a.txt"), "changed")
	writeFile(t, filepath.Join(root, "new", "deep", "b.txt"), "b")
	got, err := gitwork.ChangedFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	if want := []string{"a.txt", "new/deep/b.txt"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// A rename is two fields and only the first carries the status prefix; the
// old path is a change too, or a test moved out of the way would go unseen.
func TestChangedFilesReportsBothSidesOfARename(t *testing.T) {
	root := repo(t)
	writeFile(t, filepath.Join(root, "old.txt"), "o")
	run(t, root, "add", "old.txt")
	run(t, root, "commit", "-m", "first")
	run(t, root, "mv", "old.txt", "renamed.txt")
	got, err := gitwork.ChangedFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	if want := []string{"old.txt", "renamed.txt"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// git answers relative to the repository root; a root below it gets its own
// spelling, and changes outside it are not its project's.
func TestChangedFilesBelowTheRepositoryRoot(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	writeFile(t, filepath.Join(root, "top.txt"), "t")
	writeFile(t, filepath.Join(root, "sub", "x.txt"), "x")
	got, err := gitwork.ChangedFiles(filepath.Join(root, "sub"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"x.txt"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// -z, so a non-ASCII path is not C-quoted into a string no rule matches.
func TestChangedFilesKeepsANonASCIIPathVerbatim(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	writeFile(t, filepath.Join(root, "grün.txt"), "g")
	got, err := gitwork.ChangedFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"grün.txt"}; !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestChangedFilesOutsideARepository(t *testing.T) {
	if _, err := gitwork.ChangedFiles(t.TempDir()); err == nil {
		t.Fatal("a directory that is not a repository has no changes to report")
	}
}

func TestChangedFilesRefusesAnIgnoredRoot(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	writeFile(t, filepath.Join(root, ".gitignore"), "parked/\n")
	parked := filepath.Join(root, "parked")
	if err := os.MkdirAll(parked, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := gitwork.ChangedFiles(parked)
	if !errors.Is(err, gitwork.ErrIgnoredRoot) {
		t.Fatalf("want ErrIgnoredRoot, got %v", err)
	}
}

// Measured on 2026-09-11: with a damaged index, rev-parse --show-prefix still
// answers and status fails with exit 128.
func TestChangedFilesReportsAStatusThatFails(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	writeFile(t, filepath.Join(root, ".git", "index"), "garbage")
	_, err := gitwork.ChangedFiles(root)
	if err == nil || !strings.Contains(err.Error(), "git status") {
		t.Fatalf("want the failed status named, got %v", err)
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-gitwork" && go test ./internal/gitwork/...`
Expected: FAIL — `undefined: gitwork.ChangedFiles`.

- [ ] **Step 3: Implement**

An `internal/gitwork/gitwork.go` anhängen:

```go

// ChangedFiles is worktree.py's changed_files: every path git reports as
// changed, added or untracked below root, spelled relative to root.
//
// `status` and not `diff`, because an untracked file is invisible to diff.
// `-z`, because a non-ASCII path comes back quoted otherwise, and `-uall`,
// because the default collapses an untracked directory into one entry that is
// no file's path. git answers relative to the repository root whatever the
// working directory, so root's prefix is cut off and anything outside root is
// dropped: it is not this project's change.
//
// The prefix is asked before the status, unlike worktree.py, which spares that
// process when nothing changed. The order gives every failure a test: outside a
// repository the prefix fails, with a damaged index only the status does.
func ChangedFiles(root string) ([]string, error) {
	if ignored(root) {
		return nil, fmt.Errorf("%s: %w -- run ultraloom in a working tree of its own", root, ErrIgnoredRoot)
	}
	prefix, err := git(root, "rev-parse", "--show-prefix")
	if err != nil {
		return nil, err
	}
	status, err := git(root, "status", "--porcelain", "-z", "-uall")
	if err != nil {
		return nil, err
	}
	prefix = strings.TrimSpace(prefix)
	var paths []string
	for _, path := range parseStatus(status) {
		if relative, below := strings.CutPrefix(path, prefix); below {
			paths = append(paths, relative)
		}
	}
	return paths, nil
}

// parseStatus reads the paths out of a `--porcelain -z` answer, field by field.
//
// Most fields are "XY path". A rename or a copy is two fields, and only the
// first carries the three-character prefix; cutting it off the second too
// would turn "tests/test_cli.py" into "s/test_cli.py".
func parseStatus(output string) []string {
	var fields []string
	for _, field := range strings.Split(output, "\x00") {
		if field != "" {
			fields = append(fields, field)
		}
	}
	var paths []string
	for index := 0; index < len(fields); index++ {
		field := fields[index]
		paths = append(paths, field[3:])
		if (field[0] == 'R' || field[0] == 'C') && index+1 < len(fields) {
			index++
			paths = append(paths, fields[index])
		}
	}
	return paths
}
```

- [ ] **Step 4: Run with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-gitwork" && go test -cover ./internal/gitwork/...`
Expected: `ok … coverage: 100.0% of statements`.

- [ ] **Step 5: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-gitwork" && gofmt -l internal/gitwork`
Expected: keine Ausgabe.

- [ ] **Step 6: Commit**

```bash
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-gitwork" && git add internal/gitwork/gitwork.go internal/gitwork/changed_test.go
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-gitwork" && git diff --cached --stat
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-gitwork" && git commit -m "List the files a run finds changed when it starts"
```

---

### Task 6: Lane `flowcfg` — `[agent]`, Modellkette, Werkzeugprofile

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-flowcfg`, Zweig `ulflow/flowcfg`

**Files:**
- Create: `internal/flowcfg/load.go`, `internal/flowcfg/resolve.go`, `internal/flowcfg/tools.go`
- Test: `internal/flowcfg/load_test.go`, `internal/flowcfg/resolve_test.go`, `internal/flowcfg/tools_test.go`

**Interfaces:**
- Consumes: `flowcfg.Config`, `flowcfg.ModelSpec`, `flowcfg.Resolved` (Task 0).
- Produces:
  - `const File = ".ultraloom/config.toml"`
  - `const DefaultProvider = "claude"`
  - `func Load(root string) (Config, error)`
  - `func (c Config) Resolve(node, flow string) (Resolved, error)`
  - `func (r Resolved) Label() string` — `"<provider>:<model>"` oder `"<provider>:cli-default"`
  - `func Tools(profile string, servers []string) ([]string, error)`

- [ ] **Step 1: Write the failing load tests**

`internal/flowcfg/load_test.go`:

```go
package flowcfg_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/flowcfg"
)

func project(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, flowcfg.File)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestLoadWithoutAFileIsEmpty(t *testing.T) {
	got, err := flowcfg.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, flowcfg.Config{}) {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadWithoutAnAgentTableIsEmpty(t *testing.T) {
	got, err := flowcfg.Load(project(t, "[verify]\ntests = [\"tests\"]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, flowcfg.Config{}) {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadReadsTheWholeTable(t *testing.T) {
	root := project(t, `
[verify]
tests = ["tests"]

[agent]
default = "writer"
mcp_servers = ["brain", "github"]
cli_path = "C:/tools/claude.exe"
settings = "project"

[agent.models.writer]
provider = "claude"
model = "claude-opus-5"

[agent.models.researcher]
provider = "agy"
`)
	got, err := flowcfg.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	want := flowcfg.Config{
		Default:    "writer",
		MCPServers: []string{"brain", "github"},
		Models: map[string]flowcfg.ModelSpec{
			"writer":     {Provider: "claude", Model: "claude-opus-5"},
			"researcher": {Provider: "agy"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
}

func TestLoadRefusesWithEveryFinding(t *testing.T) {
	cases := []struct{ body, want string }{
		{"agent = 3\n", "[agent] must be a table"},
		{"[agent]\nmodle = \"x\"\n", `[agent] has unknown key "modle"; known keys: cli_path, default, mcp_servers, models, settings`},
		{"[agent]\ndefault = 3\n", "[agent].default must be a non-empty string"},
		{"[agent]\ndefault = \"\"\n", "[agent].default must be a non-empty string"},
		{"[agent]\nmcp_servers = \"brain\"\n", "[agent].mcp_servers must be a list of strings"},
		{"[agent]\nmcp_servers = [\"brain\", 3]\n", "[agent].mcp_servers must be a list of strings"},
		{"[agent]\nmodels = 3\n", "[agent.models] must be a table of models"},
		{"[agent.models]\nwriter = 3\n", "[agent.models.writer] must be a table"},
		{"[agent.models.writer]\nmodel = \"x\"\n", "[agent.models.writer] needs provider as a non-empty string"},
		{"[agent.models.writer]\nprovider = \"claude\"\nmodel = \"\"\n", "[agent.models.writer].model must be a non-empty string"},
		{"[agent.models.writer]\nprovider = \"claude\"\ntemperature = 1\n", `[agent.models.writer] has unknown key "temperature"; known keys: model, provider`},
		{"[agent\n", "config.toml"},
	}
	for _, c := range cases {
		_, err := flowcfg.Load(project(t, c.body))
		if err == nil {
			t.Errorf("%q: want an error containing %q, got none", c.body, c.want)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: error %q does not contain %q", c.body, err, c.want)
		}
	}
}

// All findings at once, not the first: a config with two mistakes should not
// take two runs to fix.
func TestLoadNamesEveryFindingInOneError(t *testing.T) {
	_, err := flowcfg.Load(project(t, "[agent]\ndefault = 3\n[agent.models.writer]\nmodel = \"x\"\n"))
	if err == nil {
		t.Fatal("want an error")
	}
	for _, want := range []string{"[agent].default must be", "[agent.models.writer] needs provider"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
}

func TestLoadReportsAConfigPathThatIsADirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, flowcfg.File), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := flowcfg.Load(root)
	if err == nil || !strings.Contains(err.Error(), "reading") {
		t.Fatalf("got %v", err)
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-flowcfg" && go test ./internal/flowcfg/...`
Expected: FAIL — `undefined: flowcfg.Load`.

- [ ] **Step 3: Implement loading**

`internal/flowcfg/load.go`:

```go
package flowcfg

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

// File is where the configuration lives, relative to the project root.
const File = ".ultraloom/config.toml"

// knownKeys are the keys [agent] may hold. cli_path and settings belong to the
// Python adapter until M2 replaces it; they are accepted and not read here.
var knownKeys = []string{"cli_path", "default", "mcp_servers", "models", "settings"}

// Load reads the [agent] table. A project without the file or without the table
// has an empty Config, and that is valid: every node then runs on the CLI's own
// default. Every finding is reported at once.
//
// The document is decoded into a map and not into a struct field, because
// BurntSushi/toml leaves such a field nil without an error when the file says
// `agent = 3` (measured on 2026-09-11).
func Load(root string) (Config, error) {
	path := filepath.Join(root, File)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("reading %s: %w", path, err)
	}
	var document map[string]any
	if _, err := toml.Decode(string(data), &document); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	value, present := document["agent"]
	if !present {
		return Config{}, nil
	}
	agent, isTable := value.(map[string]any)
	if !isTable {
		return Config{}, fmt.Errorf("%s: [agent] must be a table", path)
	}
	config, findings := parseAgent(agent)
	if len(findings) > 0 {
		return Config{}, fmt.Errorf("%s: %s", path, strings.Join(findings, "; "))
	}
	return config, nil
}

func parseAgent(agent map[string]any) (Config, []string) {
	var config Config
	var findings []string
	for _, key := range sortedKeys(agent) {
		if !slices.Contains(knownKeys, key) {
			findings = append(findings, fmt.Sprintf("[agent] has unknown key %q; known keys: %s",
				key, strings.Join(knownKeys, ", ")))
		}
	}
	if value, present := agent["default"]; present {
		name, isString := value.(string)
		if !isString || name == "" {
			findings = append(findings, "[agent].default must be a non-empty string")
		}
		config.Default = name
	}
	if value, present := agent["mcp_servers"]; present {
		servers, valid := stringList(value)
		if !valid {
			findings = append(findings, "[agent].mcp_servers must be a list of strings")
		}
		config.MCPServers = servers
	}
	if value, present := agent["models"]; present {
		models, modelFindings := parseModels(value)
		findings = append(findings, modelFindings...)
		config.Models = models
	}
	return config, findings
}

func parseModels(value any) (map[string]ModelSpec, []string) {
	table, isTable := value.(map[string]any)
	if !isTable {
		return nil, []string{"[agent.models] must be a table of models"}
	}
	models := make(map[string]ModelSpec, len(table))
	var findings []string
	for _, name := range sortedKeys(table) {
		entry, isTable := table[name].(map[string]any)
		if !isTable {
			findings = append(findings, fmt.Sprintf("[agent.models.%s] must be a table", name))
			continue
		}
		for _, key := range sortedKeys(entry) {
			if key != "model" && key != "provider" {
				findings = append(findings, fmt.Sprintf("[agent.models.%s] has unknown key %q; known keys: model, provider", name, key))
			}
		}
		provider, _ := entry["provider"].(string)
		if provider == "" {
			findings = append(findings, fmt.Sprintf("[agent.models.%s] needs provider as a non-empty string", name))
		}
		model, isString := entry["model"].(string)
		if _, present := entry["model"]; present && (!isString || model == "") {
			findings = append(findings, fmt.Sprintf("[agent.models.%s].model must be a non-empty string", name))
		}
		models[name] = ModelSpec{Provider: provider, Model: model}
	}
	return models, findings
}

func stringList(value any) ([]string, bool) {
	items, isList := value.([]any)
	if !isList {
		return nil, false
	}
	list := make([]string, 0, len(items))
	for _, item := range items {
		text, isString := item.(string)
		if !isString {
			return nil, false
		}
		list = append(list, text)
	}
	return list, true
}

func sortedKeys[T any](table map[string]T) []string {
	keys := make([]string, 0, len(table))
	for key := range table {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
```

- [ ] **Step 4: Run the load tests to verify they pass**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-flowcfg" && go test ./internal/flowcfg/...`
Expected: `ok`.

- [ ] **Step 5: Write the failing resolve and tools tests**

`internal/flowcfg/resolve_test.go`:

```go
package flowcfg_test

import (
	"testing"

	"github.com/xidus90/ultra-loom/internal/flowcfg"
)

var config = flowcfg.Config{
	Default: "writer",
	Models: map[string]flowcfg.ModelSpec{
		"writer":     {Provider: "claude", Model: "claude-opus-5"},
		"researcher": {Provider: "agy"},
		"reviewer":   {Provider: "claude", Model: "claude-sonnet-5"},
	},
}

func TestResolveTakesTheFirstNameInTheChain(t *testing.T) {
	cases := []struct {
		node, flow string
		want       flowcfg.Resolved
	}{
		{"researcher", "reviewer", flowcfg.Resolved{Name: "researcher", Provider: "agy"}},
		{"", "reviewer", flowcfg.Resolved{Name: "reviewer", Provider: "claude", Model: "claude-sonnet-5"}},
		{"", "", flowcfg.Resolved{Name: "writer", Provider: "claude", Model: "claude-opus-5"}},
	}
	for _, c := range cases {
		got, err := config.Resolve(c.node, c.flow)
		if err != nil {
			t.Fatal(err)
		}
		if got != c.want {
			t.Errorf("node %q, flow %q: got %+v, want %+v", c.node, c.flow, got, c.want)
		}
	}
}

// No stage names a model: the adapter passes none and the CLI decides.
func TestResolveWithNoNameAnywhereIsTheClaudeDefault(t *testing.T) {
	got, err := flowcfg.Config{}.Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != (flowcfg.Resolved{Provider: flowcfg.DefaultProvider}) || got.Label() != "claude:cli-default" {
		t.Fatalf("got %+v, label %q", got, got.Label())
	}
}

func TestResolveRefusesAnUnknownName(t *testing.T) {
	_, err := config.Resolve("writre", "")
	want := `model "writre" is not under [agent.models]; known models: researcher, reviewer, writer`
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %s", err, want)
	}
	_, err = flowcfg.Config{Default: "ghost"}.Resolve("", "")
	if err == nil || err.Error() != `model "ghost" is not under [agent.models]; known models: none` {
		t.Fatalf("got %v", err)
	}
}

func TestLabelNamesTheModelOrTheCLIDefault(t *testing.T) {
	if got := (flowcfg.Resolved{Provider: "claude", Model: "claude-opus-5"}).Label(); got != "claude:claude-opus-5" {
		t.Fatalf("got %q", got)
	}
	if got := (flowcfg.Resolved{Provider: "agy"}).Label(); got != "agy:cli-default" {
		t.Fatalf("got %q", got)
	}
}
```

`internal/flowcfg/tools_test.go`:

```go
package flowcfg_test

import (
	"slices"
	"testing"

	"github.com/xidus90/ultra-loom/internal/flowcfg"
)

func TestToolsOfEveryProfile(t *testing.T) {
	cases := map[string][]string{
		"read_only": {"Glob", "Grep", "Read"},
		"edit":      {"Edit", "Glob", "Grep", "Read", "Write"},
		"shell":     {"Bash", "Glob", "Grep", "Read"},
		"mcp":       {"Glob", "Grep", "Read"},
	}
	for profile, want := range cases {
		got, err := flowcfg.Tools(profile, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: got %v, want %v", profile, got, want)
		}
	}
}

func TestOnlyTheMCPProfileTakesServersSortedAndOnce(t *testing.T) {
	got, err := flowcfg.Tools("mcp", []string{"github", "brain", "brain"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Glob", "Grep", "Read", "mcp__brain", "mcp__github"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	edit, err := flowcfg.Tools("edit", []string{"brain"})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(edit, "mcp__brain") {
		t.Fatalf("edit takes no servers, got %v", edit)
	}
}

func TestToolsRefusesAnUnknownProfile(t *testing.T) {
	_, err := flowcfg.Tools("admin", nil)
	want := `unknown tool profile "admin"; known profiles: edit, mcp, read_only, shell`
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %s", err, want)
	}
}

// The profiles are the only ceiling between an agent node and Bash or Write;
// a caller that edits its copy must not widen the next node's.
func TestToolsHandsOutACopy(t *testing.T) {
	first, err := flowcfg.Tools("read_only", nil)
	if err != nil {
		t.Fatal(err)
	}
	first[0] = "Bash"
	again, err := flowcfg.Tools("read_only", nil)
	if err != nil {
		t.Fatal(err)
	}
	if again[0] != "Glob" {
		t.Fatalf("a profile was widened through a returned slice: %v", again)
	}
}
```

- [ ] **Step 6: Run them to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-flowcfg" && go test ./internal/flowcfg/...`
Expected: FAIL — `config.Resolve undefined`.

- [ ] **Step 7: Implement the chain and the profiles**

`internal/flowcfg/resolve.go`:

```go
package flowcfg

import "fmt"

// DefaultProvider is the provider of a node for which no stage of the chain
// names a model.
const DefaultProvider = "claude"

// Resolve is the model chain: the node's model, else the flow's, else
// [agent].default. The first name that is set must be under [agent.models].
// With no name at all the node runs on the claude CLI's own default.
func (c Config) Resolve(node, flow string) (Resolved, error) {
	for _, name := range []string{node, flow, c.Default} {
		if name == "" {
			continue
		}
		spec, ok := c.Models[name]
		if !ok {
			return Resolved{}, fmt.Errorf("model %q is not under [agent.models]; known models: %s", name, knownModels(c.Models))
		}
		return Resolved{Name: name, Provider: spec.Provider, Model: spec.Model}, nil
	}
	return Resolved{Provider: DefaultProvider}, nil
}

// Label is how the journal records a resolved model.
func (r Resolved) Label() string {
	if r.Model == "" {
		return r.Provider + ":cli-default"
	}
	return r.Provider + ":" + r.Model
}

func knownModels(models map[string]ModelSpec) string {
	if len(models) == 0 {
		return "none"
	}
	return joinKeys(models)
}
```

`internal/flowcfg/tools.go`:

```go
package flowcfg

import (
	"fmt"
	"slices"
	"strings"
)

// profiles are tools.py's PROFILES, each sorted and without duplicates: the
// list becomes part of a prompt and of a journal entry, so its order must not
// depend on anything.
var profiles = map[string][]string{
	"read_only": {"Glob", "Grep", "Read"},
	"edit":      {"Edit", "Glob", "Grep", "Read", "Write"},
	"shell":     {"Bash", "Glob", "Grep", "Read"},
	"mcp":       {"Glob", "Grep", "Read"},
}

// Tools is tools.py's resolve_tools: the tools a node with this profile may
// use. Only the mcp profile takes servers, one mcp__<server> each.
//
// The result is always a fresh slice. The profiles are the only ceiling between
// an agent node and Bash or Write, and a caller that edited a shared slice would
// widen it for every node after.
func Tools(profile string, servers []string) ([]string, error) {
	base, ok := profiles[profile]
	if !ok {
		return nil, fmt.Errorf("unknown tool profile %q; known profiles: %s", profile, joinKeys(profiles))
	}
	tools := slices.Clone(base)
	if profile == "mcp" {
		for _, server := range servers {
			tools = append(tools, "mcp__"+server)
		}
		slices.Sort(tools)
		tools = slices.Compact(tools)
	}
	return tools, nil
}

func joinKeys[T any](table map[string]T) string {
	return strings.Join(sortedKeys(table), ", ")
}
```

- [ ] **Step 8: Run with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-flowcfg" && go test -cover ./internal/flowcfg/...`
Expected: `ok … coverage: 100.0% of statements`.

- [ ] **Step 9: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-flowcfg" && gofmt -l internal/flowcfg`
Expected: keine Ausgabe.

- [ ] **Step 10: Commit**

```bash
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-flowcfg" && git add internal/flowcfg/load.go internal/flowcfg/load_test.go internal/flowcfg/resolve.go internal/flowcfg/resolve_test.go internal/flowcfg/tools.go internal/flowcfg/tools_test.go
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-flowcfg" && git diff --cached --stat
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-flowcfg" && git commit -m "Read the agent table, resolve the model chain and the tool profiles"
```

---

### Task 7: Lane `runs` — Laufnummer, Laufmarke, eigene Dateien

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-runs`, Zweig `ulflow/runs`

**Files:**
- Create: `internal/runs/id.go`, `internal/runs/marker.go`
- Test: `internal/runs/id_test.go`, `internal/runs/marker_test.go`

**Interfaces:**
- Consumes: `flow.Baseline` (Task 0).
- Produces:
  - `const Dir = ".ultraloom/runs"`
  - `func NextID(root string) string`
  - `func JournalPath(root, id string) string`, `func MarkerPath(root, id string) string`
  - `func Files(id string) []string` — `[".ultraloom/runs/<id>.flow", ".ultraloom/runs/<id>.jsonl"]`
  - `type Marker struct { Flow string; Options map[string]string; Baseline *flow.Baseline; Runtime string; Version string }`
  - `func WriteMarker(path string, marker Marker) error`
  - `func ReadMarker(path string) (*Marker, error)` — fehlende Datei: `nil, nil`

- [ ] **Step 1: Write the failing id tests**

`internal/runs/id_test.go`:

```go
package runs_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xidus90/ultra-loom/internal/runs"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNextIDOfAProjectWithoutRunsIsTheFirst(t *testing.T) {
	if got := runs.NextID(t.TempDir()); got != "0001" {
		t.Fatalf("got %s", got)
	}
}

// A counter over the journals and never a clock. Markers, names that are not
// numbers, and a number no int holds do not count.
func TestNextIDCountsPastTheHighestNumericJournal(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"0001.jsonl", "0009.jsonl", "0012.flow", "notes.jsonl", "12a.jsonl", "99999999999999999999.jsonl"} {
		touch(t, filepath.Join(root, runs.Dir, name))
	}
	if got := runs.NextID(root); got != "0010" {
		t.Fatalf("got %s, want 0010", got)
	}
}

// Measured on 2026-09-11: on Windows, reading a file as a directory reports
// "does not exist", so a runs path that is a file is a project without runs --
// the answer Python's glob gives too.
func TestNextIDOfARunsPathThatIsAFile(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, runs.Dir))
	if got := runs.NextID(root); got != "0001" {
		t.Fatalf("got %s", got)
	}
}

func TestTheFilesOfARun(t *testing.T) {
	root := t.TempDir()
	if got, want := runs.JournalPath(root, "0007"), filepath.Join(root, ".ultraloom", "runs", "0007.jsonl"); got != want {
		t.Fatalf("journal: got %s, want %s", got, want)
	}
	if got, want := runs.MarkerPath(root, "0007"), filepath.Join(root, ".ultraloom", "runs", "0007.flow"); got != want {
		t.Fatalf("marker: got %s, want %s", got, want)
	}
	want := []string{".ultraloom/runs/0007.flow", ".ultraloom/runs/0007.jsonl"}
	if got := runs.Files("0007"); !slices.Equal(got, want) {
		t.Fatalf("files: got %v, want %v", got, want)
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-runs" && go test ./internal/runs/...`
Expected: FAIL — `undefined: runs.NextID`.

- [ ] **Step 3: Implement ids and paths**

`internal/runs/id.go`:

```go
package runs

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Dir is where runs live, relative to the project root.
const Dir = ".ultraloom/runs"

// NextID is cli.py's next_run_id: one more than the highest numeric journal
// name, four digits. A counter and never a clock.
//
// A directory that cannot be read counts as one without runs, as Python's glob
// has it. On Windows even a file in its place reads as absent (measured on
// 2026-09-11), and a write into it fails loudly in journal.Append.
func NextID(root string) string {
	entries, _ := os.ReadDir(filepath.Join(root, Dir))
	highest := 0
	for _, entry := range entries {
		stem, isJournal := strings.CutSuffix(entry.Name(), ".jsonl")
		if !isJournal || !digits(stem) {
			continue
		}
		number, err := strconv.Atoi(stem)
		if err != nil {
			continue
		}
		highest = max(highest, number)
	}
	return fmt.Sprintf("%04d", highest+1)
}

func digits(text string) bool {
	if text == "" {
		return false
	}
	for i := 0; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return false
		}
	}
	return true
}

// JournalPath is where run id writes its journal.
func JournalPath(root, id string) string {
	return filepath.Join(root, Dir, id+".jsonl")
}

// MarkerPath is where run id writes its marker.
func MarkerPath(root, id string) string {
	return filepath.Join(root, Dir, id+".flow")
}

// Files is cli.py's _run_files: the two files a run writes itself, spelled as
// changed-file lists spell paths, so a guard can subtract exactly these two and
// not every other run's files along with them.
func Files(id string) []string {
	return []string{Dir + "/" + id + ".flow", Dir + "/" + id + ".jsonl"}
}
```

- [ ] **Step 4: Run the id tests to verify they pass**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-runs" && go test ./internal/runs/...`
Expected: `ok`.

- [ ] **Step 5: Write the failing marker tests**

`internal/runs/marker_test.go`:

```go
package runs_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/flow"
	"github.com/xidus90/ultra-loom/internal/runs"
)

func marker() runs.Marker {
	return runs.Marker{
		Flow:     "spec_to_board",
		Options:  map[string]string{"zeta": "x\ny", "topic": "a b"},
		Baseline: &flow.Baseline{Commit: "abc", Dirty: []string{"b.txt", "a.txt"}},
		Runtime:  "go",
		Version:  "0.1.0",
	}
}

func TestWriteMarkerSpellsTheFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs", "0001.flow")
	if err := runs.WriteMarker(path, marker()); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `spec_to_board
baseline="a.txt\nb.txt"
baseline_commit="abc"
runtime="go"
topic="a b"
ulflow_version="0.1.0"
zeta="x\ny"
`
	if string(got) != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestAWrittenMarkerReadsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0001.flow")
	if err := runs.WriteMarker(path, marker()); err != nil {
		t.Fatal(err)
	}
	got, err := runs.ReadMarker(path)
	if err != nil {
		t.Fatal(err)
	}
	want := marker()
	want.Baseline.Dirty = []string{"a.txt", "b.txt"}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("got  %+v\nwant %+v", *got, want)
	}
}

func TestWriteMarkerRefuses(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	touch(t, blocker)
	reserved := marker()
	reserved.Options = map[string]string{"runtime": "python"}
	nameless := marker()
	nameless.Flow = ""
	cases := []struct {
		name   string
		path   string
		marker runs.Marker
		want   string
	}{
		{"reserved option", filepath.Join(dir, "a.flow"), reserved, `option "runtime" is reserved for the marker itself`},
		{"no flow", filepath.Join(dir, "b.flow"), nameless, "a marker needs the name of its flow"},
		{"parent is a file", filepath.Join(blocker, "c.flow"), marker(), "creating"},
		{"path is a directory", dir, marker(), "writing"},
	}
	for _, c := range cases {
		err := runs.WriteMarker(c.path, c.marker)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: want an error containing %q, got %v", c.name, c.want, err)
		}
	}
}

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "0001.flow")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadMarkerOfAMissingFileIsNothing(t *testing.T) {
	got, err := runs.ReadMarker(filepath.Join(t.TempDir(), "0001.flow"))
	if err != nil || got != nil {
		t.Fatalf("got %+v, %v", got, err)
	}
}

// Markers the Python runtime wrote carry bare values and no runtime line, and
// a run already on disk should stay resumable.
func TestReadMarkerReadsAPythonMarker(t *testing.T) {
	got, err := runs.ReadMarker(write(t, "verify_until_green\r\nchecks=ruff pytest\r\nmax_rounds=123\r\n\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := runs.Marker{Flow: "verify_until_green", Options: map[string]string{"checks": "ruff pytest", "max_rounds": "123"}}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("got %+v, want %+v", *got, want)
	}
}

// The commit decides alone: a path set without a commit was written before the
// commit existed and is no baseline; a commit without a path set is one with no
// dirty files.
func TestReadMarkerTakesTheBaselineFromItsCommit(t *testing.T) {
	onlyDirty, err := runs.ReadMarker(write(t, "f\nbaseline=\"a.txt\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if onlyDirty.Baseline != nil || len(onlyDirty.Options) != 0 {
		t.Fatalf("a path set without a commit is no baseline: %+v", onlyDirty)
	}
	onlyCommit, err := runs.ReadMarker(write(t, "f\nbaseline_commit=\"abc\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if onlyCommit.Baseline == nil || onlyCommit.Baseline.Commit != "abc" || len(onlyCommit.Baseline.Dirty) != 0 {
		t.Fatalf("a commit alone is a baseline without dirty files: %+v", onlyCommit.Baseline)
	}
}

func TestReadMarkerRefuses(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"empty", "", "says nothing -- not even which flow it belongs to"},
		{"blank first line", "  \nx=\"1\"\n", "says nothing"},
		{"line without =", "f\nnot an option\n", `option line without '=': "not an option"`},
	}
	for _, c := range cases {
		_, err := runs.ReadMarker(write(t, c.body))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: want an error containing %q, got %v", c.name, c.want, err)
		}
	}
	_, err := runs.ReadMarker(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "reading") {
		t.Errorf("a directory: got %v", err)
	}
}
```

- [ ] **Step 6: Run them to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-runs" && go test ./internal/runs/...`
Expected: FAIL — `undefined: runs.Marker`.

- [ ] **Step 7: Implement the marker**

`internal/runs/marker.go`:

```go
package runs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/ultra-loom/internal/flow"
)

// Marker is a run's marker file: which flow the run belongs to and how it was
// started. The journal records what nodes did, not which graph they came from,
// so resume and replay would have nothing to load without it.
type Marker struct {
	Flow     string
	Options  map[string]string
	Baseline *flow.Baseline
	Runtime  string // "go" for runs of ulflow; empty in markers the Python runtime wrote
	Version  string // the ulflow version that started the run; empty when unknown
}

// The option names the marker uses for itself.
const (
	keyBaseline       = "baseline"
	keyBaselineCommit = "baseline_commit"
	keyRuntime        = "runtime"
	keyVersion        = "ulflow_version"
)

var reserved = []string{keyBaseline, keyBaselineCommit, keyRuntime, keyVersion}

// WriteMarker writes cli.py's marker format: the flow name on the first line,
// then one name=<JSON string> line per option, sorted by name. JSON keeps a
// value with a newline -- a baseline's path list -- on its own line.
func WriteMarker(path string, marker Marker) error {
	if marker.Flow == "" {
		return fmt.Errorf("%s: a marker needs the name of its flow", path)
	}
	options := make(map[string]string, len(marker.Options)+len(reserved))
	for name, value := range marker.Options {
		if slices.Contains(reserved, name) {
			return fmt.Errorf("%s: option %q is reserved for the marker itself", path, name)
		}
		options[name] = value
	}
	if marker.Baseline != nil {
		dirty := slices.Clone(marker.Baseline.Dirty)
		slices.Sort(dirty)
		options[keyBaseline] = strings.Join(dirty, "\n")
		options[keyBaselineCommit] = marker.Baseline.Commit
	}
	if marker.Runtime != "" {
		options[keyRuntime] = marker.Runtime
	}
	if marker.Version != "" {
		options[keyVersion] = marker.Version
	}
	names := make([]string, 0, len(options))
	for name := range options {
		names = append(names, name)
	}
	slices.Sort(names)
	lines := []string{marker.Flow}
	for _, name := range names {
		// A string always marshals.
		value, _ := json.Marshal(options[name])
		lines = append(lines, name+"="+string(value))
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// ReadMarker is cli.py's _recorded_run. An absent marker is (nil, nil).
func ReadMarker(path string) (*Marker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if strings.TrimSpace(lines[0]) == "" {
		return nil, fmt.Errorf("%s: says nothing -- not even which flow it belongs to", path)
	}
	marker := &Marker{Flow: strings.TrimSpace(lines[0]), Options: map[string]string{}}
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		name, raw, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("%s: option line without '=': %q", path, line)
		}
		marker.Options[name] = decodeOption(raw)
	}
	dirty := pop(marker.Options, keyBaseline)
	if commit, present := marker.Options[keyBaselineCommit]; present {
		marker.Baseline = &flow.Baseline{Commit: commit, Dirty: decodeDirty(dirty)}
	}
	pop(marker.Options, keyBaselineCommit)
	marker.Runtime = pop(marker.Options, keyRuntime)
	marker.Version = pop(marker.Options, keyVersion)
	return marker, nil
}

// decodeOption reads a value as JSON when it is a JSON string. Markers written
// before this encoding carry the value bare, and `max_rounds=123` means the
// text "123" either way.
func decodeOption(raw string) string {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw
	}
	if text, isString := value.(string); isString {
		return text
	}
	return raw
}

func decodeDirty(recorded string) []string {
	var dirty []string
	for _, line := range strings.Split(recorded, "\n") {
		if line != "" {
			dirty = append(dirty, line)
		}
	}
	return dirty
}

func pop(options map[string]string, name string) string {
	value := options[name]
	delete(options, name)
	return value
}
```

- [ ] **Step 8: Run with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-runs" && go test -cover ./internal/runs/...`
Expected: `ok … coverage: 100.0% of statements`.

- [ ] **Step 9: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-runs" && gofmt -l internal/runs`
Expected: keine Ausgabe.

- [ ] **Step 10: Commit**

```bash
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-runs" && git add internal/runs/id.go internal/runs/id_test.go internal/runs/marker.go internal/runs/marker_test.go
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-runs" && git diff --cached --stat
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-runs" && git commit -m "Number runs and read and write their markers"
```

---

### Task 8: Prüfen und Zusammenführen (Orchestrator)

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness`

**Files:** keine eigenen; Merge-Commits.

Für jede Lane `<lane>` aus `expr`, `tmpl`, `journal`, `model`, `gitwork`, `flowcfg`, `runs`, in dieser Reihenfolge:

- [ ] **Step 1: Nur eigene Dateien?**

Run: `git -C "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" diff --stat feature/agent-harness...ulflow/<lane>`
Expected: nur die Dateien unter **Files** des Tasks der Lane. Jede andere Datei — besonders unter `.ultraloom/`, `.claude/`, `go.mod`, `go.sum` — heißt: nicht mergen, Lane zurückgeben.

- [ ] **Step 2: Grün und voll gedeckt im Lane-Worktree?**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-<lane>" && go test -cover ./internal/...`
Expected: `ok` überall; das Paket der Lane mit `coverage: 100.0% of statements`.

- [ ] **Step 3: Review**

Zweistufiges Review nach superpowers:subagent-driven-development: erst Spec-Treue gegen den Task, dann Codequalität. Befunde gehen an den Subagenten der Lane zurück, der in seinem Worktree nachbessert und neu committet.

- [ ] **Step 4: Merge**

Run: `git -C "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" merge --no-ff ulflow/<lane> -m "Merge ulflow lane <lane>"`
Expected: Merge ohne Konflikt. Ein Konflikt heißt, dass zwei Lanes dieselbe Datei angefasst haben; abbrechen mit `git merge --abort` und die Dateilisten prüfen.

Nach dem letzten Merge:

- [ ] **Step 5: Das ganze Modul baut**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go vet ./...`
Expected: keine Ausgabe.

- [ ] **Step 6: Alle Tests, alle neuen Pakete voll gedeckt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -cover ./...`
Expected: `ok` überall; `internal/flow/expr`, `internal/flow/tmpl`, `internal/journal`, `internal/model`, `internal/gitwork`, `internal/flowcfg`, `internal/runs` mit `100.0%`; `internal/flow` mit `[no statements]` oder `100.0%`.

- [ ] **Step 7: Nichts ist beim Remote angekommen**

Run: `git -C "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" ls-remote origin`
Expected: kein `refs/heads/ulflow/*`, kein `refs/heads/feature/agent-harness`.

- [ ] **Step 8: Lane-Worktrees abbauen**

Je Lane: `ulguard worktree-remove "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-<lane>"`, danach `git -C "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" branch -d ulflow/<lane>`.
Expected: beide ohne Fehler. Verweigert git das Entfernen, **nicht** `--force`: dem Nutzer melden, was im Worktree liegt. `.worktrees/agent-harness` bleibt stehen.

## Außerhalb dieses Plans

- Welle 2 (Lader mit den sieben Prüfstufen und Auffindung, Bausteine `gate`, `exit`, `agent`, Runner) und Welle 3 (`cmd/flow`, Golden-Journal, Install-Skripte, Neubau beider Stände). Ihre Pläne entstehen nach Task 8, gegen die dann zusammengeführten Verträge.
- Der Planungs-Flow als Testdatei; er gehört zur Lane `loader` in Welle 2.
- Die Umwandlung von TOML-`int64` und JSON-`float64` in die `Value`-Typen einer `State`; das tun Lader und Runner in Welle 2.
