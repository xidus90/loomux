# GDScript im Code-Graphen (G5c) — Implementierungsplan

> **Für ausführende Agenten:** PFLICHT-SKILL: superpowers:subagent-driven-development (empfohlen) oder superpowers:executing-plans, Task für Task. Schritte sind Checkboxen (`- [ ]`).

**Ziel:** `loomux graph build` liest GDScript, Szenen, Ressourcen und `project.godot` und löst ihre Kanten im Godot-Projekt auf.

**Architektur:** Ein neues Extraktorpaket `internal/code/extract/gdscript` beansprucht `.gd`, `.godot`, `.tres` und `.tscn` als eine Sprache `gdscript`. Es liest `.gd` mit der Grammatik `gdscript`, den Rest mit `godot_resource`. Ein neuer Arm `resolveGDScript` in `internal/code/resolve` löst die Rohkanten auf. Die `res://`-Wurzel und die Autoloads kommen aus `project.godot`; Pfadziele dürfen Dateien jeder Sprache des Builds sein.

**Tech-Stack:** Go, gotreesitter v0.55.1 (reines Go), bestehender Kern `internal/code/extract/treesitter`.

**Spec:** `docs/.superpowers/specs/2026-10-06-loomux-gdscript-graph-design.md`. Wer einen Task ausführt, liest Spec und Plan.

## Globale Vorgaben

- Keine neue Abhängigkeit. gotreesitter wird auf `v0.55.1` gehoben (Task 1), sonst nichts.
- Coverage 100 % je Funktion. Eine Ausnahme nur mit `//coverage:exempt <grund>` direkt über `func`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst.
- Code, Bezeichner, Kommentare, Fehlermeldungen und Commits sind englisch; dieser Plan und die Spec deutsch.
- Commits nach Conventional Commits, ohne Verweis auf Plan, Spec, Task oder Stufe. Kein `Co-Authored-By`, keine Werbezeile. Autor ist der Nutzer.
- Commit-Nachrichten per Write in eine Datei im Scratchpad und `git commit -F <datei>`. Die Ausgabe des Commits (das Tor läuft im pre-commit) per `> <scratchpad>/commit-<n>.log 2>&1` in eine Datei, nie nach `/dev/null`.
- Gearbeitet wird im Worktree `C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/graph-gdscript` auf `feat/graph-gdscript`. Vor jedem Commit: `git rev-parse --show-toplevel` und `git branch --show-current` als Bedingung prüfen.
- Für Implementierer gilt:
  - Kein `python -`, `python3 -` und kein Heredoc an einen Interpreter.
  - Kein Beenden von Prozessen nach Namen.
  - Kein Hintergrundprozess, der nach dem Bericht weiterläuft.
  - Code mit Backslashes nur per Write oder Edit, nie per `sed`.
- RED-Nachweis: Jede neue Testzeile läuft vor dem Code rot. Der Bericht zeigt Befehl und `--- FAIL`-Zeilen. Ein reiner Build-Fehler zählt nicht als RED; dann mit Stub nachweisen.
- Mutationsrunde je Task per `go test -overlay` (Overlay-Pfade in der Form `C:/…`, ohne `-cover`).
  - Die genannten Mutanten sind der Boden: Zusätzlich jede Teilbedingung einzeln streichen.
  - Je Mutant eine Ausgabedatei `<scratchpad>/t<N>-mut-<k>.txt` mit Befehl, Ausgabe und dem Test, der ihn tötet.
  - Ein Mutant, der nicht baut, zählt nicht als getötet.
- Die Hooks in `.claude/settings.json` rufen `bin/loomux.exe`. Im Worktree baut Task 1 es einmal: `go build -o bin/loomux.exe ./cmd/loomux` (git-ignoriert).

## Review-Fokus

1. **CRLF-Dateien:** Godot unter Windows schreibt `.gd` und `.tscn` je nach Einstellung mit `\r\n`. Erwartet: dieselben Knoten und Kanten wie mit `\n`, Signaturen ohne `\r`. Test in Task 2 und Task 3.
2. **`class_name` ohne Namen auf der Zeile:** `class_name` und dann `func f():` in der nächsten Zeile machen in v0.55.1 ein `class_name_statement` mit dem Namen `func` (geprobt). Erwartet: kein Klassenknoten. Test in Task 2.
3. **Verschachtelte Godot-Projekte:** Ein `project.godot` unter dem Ordner eines anderen; `res://` gilt dem nächsten. Erwartet: das innere Projekt gewinnt. Test in Task 4.
4. **Namenskollisionen:**
   - Zwei gleiche `class_name` in einem Projekt ergeben keine Kante, auch wenn ein gleichnamiger Autoload existiert.
   - Eine innere Klasse schlägt ein gleichnamiges `class_name`.
   - Test in Task 4.
5. **Zyklische Vererbung:** `class_name A extends B` und `class_name B extends A` dürfen die Methodensuche nicht endlos laufen lassen. Erwartet: keine Kante, schnelles Ende. Test in Task 4 mit `-timeout 30s`.

---

### Task 1: gotreesitter auf v0.55.1

**Files:**
- Modify: `go.mod`, `go.sum`
- Modify: `internal/code/extract/treesitter/doc.go:42` (Konstante `Parser`)

**Interfaces:**
- Produces: `treesitter.Parser == "gotreesitter/v0.55.1"`.

- [ ] **Step 1: Arbeitsbaum und Hook-Binary**

```bash
git rev-parse --show-toplevel   # muss .../.claude/worktrees/graph-gdscript sein
git branch --show-current       # feat/graph-gdscript
go build -o bin/loomux.exe ./cmd/loomux
```

- [ ] **Step 2: RED — der Pin-Test meldet die Abweichung**

```bash
go get github.com/odvcencio/gotreesitter@v0.55.1
go mod tidy
go test ./internal/code/extract/treesitter/ -run TestParserMatchesGoMod
```

Expected: FAIL. `TestParserMatchesGoMod` nennt `v0.55.1` gegen `gotreesitter/v0.55.0`.

- [ ] **Step 3: Konstante heben**

In `internal/code/extract/treesitter/doc.go`:

```go
const Parser = "gotreesitter/v0.55.1"
```

- [ ] **Step 4: Alles grün**

```bash
go test ./internal/code/... ./internal/hooks/... > <scratchpad>/t1-test.txt 2>&1; echo $?
```

Expected: 0. Die Literale `go/1+python/1@gotreesitter/v0.55.0` in `internal/hooks/blast_monitor_test.go:194` und `internal/code/model/graph_test.go:56,65` sind frei gewählte Stempel und bleiben stehen, solange ihre Tests grün sind. Fällt ein Python-Replay-Fall wegen anderer Bäume, anhalten und dem Controller den Diff melden. Die Release-Notes sagen gleiche Bäume zu, die Grammatik-Blobs sind gleich.

- [ ] **Step 5: Commit**

Nachricht (Datei `<scratchpad>/msg-t1.txt`):

```
build(deps): bump gotreesitter to v0.55.1

A runtime-only release: the grammar blobs of every language loomux reads
are unchanged, and the Python parser drops back to two initial stacks
where no unpacking can occur. The runtime is part of every tree-sitter
extractor's version, so each graph is rebuilt once.
```

```bash
git add go.mod go.sum internal/code/extract/treesitter/doc.go
git commit -F <scratchpad>/msg-t1.txt > <scratchpad>/commit-1.log 2>&1; echo $?
```

---

### Task 2: Extraktor für `.gd`

**Files:**
- Create: `internal/code/extract/gdscript/gdscript.go` (Paket, `Language`, `File`, Knoten)
- Create: `internal/code/extract/gdscript/edges.go` (extends, imports, calls, references)
- Create: `internal/code/extract/gdscript/gdscript_test.go`
- Create: `internal/code/extract/gdscript/internal_test.go`

**Interfaces:**
- Consumes: `treesitter.Parse`, `(*treesitter.Doc).Symbol/Type/Field/Text/FileNode/ParseErrors`, `treesitter.Walk`, `extract.Result/RawEdge/Import`.
- Produces:
  - `gdscript.Language{}` mit `Name() == "gdscript"`, `Version() == "gdscript/1@" + treesitter.Parser` und `Extensions() == []string{".gd"}`. Task 3 erweitert die Endungen auf vier.
  - `gdscript.File(rel, source string) (extract.Result, error)`.
- `Result`-Form eines Skripts:
  - `Language: "gdscript"`; `Package`: der `class_name` oder `""`.
  - `Imports`: je `const X = preload("p")` oder `var X = load("p")` auf Skriptebene ein `Import{Alias: "X", Path: "p"}`.
- Knoten:
  - Datei;
  - `class` für `class_name` (ID `rel#Name`, über die ganze Datei);
  - innere `class` (qualifiziert);
  - `method` (Owner = qualifizierte Klasse) oder `function` (Skript ohne `class_name`, Owner `""`);
  - `signal`.
- Rohkanten:
  - `contains` (Eltern → Kind);
  - `extends`: Quelle ist Klasse oder Datei, mit `Name`/`Receiver` oder `Specifier`;
  - `imports`: Quelle ist die Datei, `Specifier` = Pfad wie geschrieben;
  - `calls`: `Name` allein, `Name` mit `Receiver` (Kette, `self.` abgeschnitten), `Receiver: "super"`, oder `Name: "emit"` mit `Receiver: <signal>` für `emit_signal("x")`;
  - `references`: `Name`, je Quelle und Name einmal.

- [ ] **Step 1: Failing tests (Knoten)**

`internal/code/extract/gdscript/gdscript_test.go`:

```go
package gdscript_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/gdscript"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

const rel = "a/s.gd"

// extractSrc calls the package function and not Language{}.File: the code
// graph follows a package selector, and so sees these tests reach the
// extractor.
func extractSrc(t *testing.T, path, src string) extract.Result {
	t.Helper()
	r, err := gdscript.File(path, src)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// views is every node as most tests compare it: everything but its body.
func views(r extract.Result) []string {
	var out []string
	for _, n := range r.Nodes {
		out = append(out, fmt.Sprintf("%s %s owner=%s %s exported=%v sig=%q", n.ID, n.Kind, n.Owner, n.Span, n.Exported, n.Signature))
	}
	return out
}

func edgesOf(r extract.Result, rel model.Relation) []extract.RawEdge {
	var out []extract.RawEdge
	for _, e := range r.Edges {
		if e.Relation == rel {
			out = append(out, e)
		}
	}
	return out
}

func contains(src, dst model.NodeID) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationContains, TargetID: dst, File: rel}
}

// same compares two lists and treats nil and empty as equal.
func same[T any](t *testing.T, what string, got, want []T) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n got  %v\n want %v", what, got, want)
	}
}

const named = "@tool\nclass_name Sample\nextends Node\n\nsignal changed(value: int)\n\nstatic func make() -> Sample:\n\treturn null\n\nfunc _ready() -> void:\n\tpass\n\nclass Inner extends Node:\n\tfunc inner_fn():\n\t\tpass\n\n\tclass Deeper:\n\t\tfunc deep():\n\t\t\tpass\n"

func TestANamedScriptIsOneClassOverTheWholeFile(t *testing.T) {
	r := extractSrc(t, rel, named)
	same(t, "nodes", views(r), []string{
		`a/s.gd file owner= L1-L20 exported=true sig=""`,
		`a/s.gd#Sample class owner= L1-L19 exported=true sig="class_name Sample"`,
		`a/s.gd#Sample.changed signal owner=Sample L5-L5 exported=true sig="signal changed(value: int)"`,
		`a/s.gd#Sample.make method owner=Sample L7-L8 exported=true sig="static func make() -> Sample"`,
		`a/s.gd#Sample._ready method owner=Sample L10-L11 exported=false sig="func _ready() -> void"`,
		`a/s.gd#Sample.Inner class owner=Sample L13-L19 exported=true sig="class Inner extends Node"`,
		`a/s.gd#Sample.Inner.inner_fn method owner=Sample.Inner L14-L15 exported=true sig="func inner_fn()"`,
		`a/s.gd#Sample.Inner.Deeper class owner=Sample.Inner L17-L19 exported=true sig="class Deeper"`,
		`a/s.gd#Sample.Inner.Deeper.deep method owner=Sample.Inner.Deeper L18-L19 exported=true sig="func deep()"`,
	})
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("a/s.gd", "a/s.gd#Sample"),
		contains("a/s.gd#Sample", "a/s.gd#Sample.changed"),
		contains("a/s.gd#Sample", "a/s.gd#Sample.make"),
		contains("a/s.gd#Sample", "a/s.gd#Sample._ready"),
		contains("a/s.gd#Sample", "a/s.gd#Sample.Inner"),
		contains("a/s.gd#Sample.Inner", "a/s.gd#Sample.Inner.inner_fn"),
		contains("a/s.gd#Sample.Inner", "a/s.gd#Sample.Inner.Deeper"),
		contains("a/s.gd#Sample.Inner.Deeper", "a/s.gd#Sample.Inner.Deeper.deep"),
	})
	if r.Package != "Sample" || r.Language != "gdscript" || r.Path != rel || r.ParseErrors != 0 {
		t.Errorf("Package %q, Language %q, Path %q, ParseErrors %d", r.Package, r.Language, r.Path, r.ParseErrors)
	}
}

func TestAnUnnamedScriptHangsItsDefinitionsOnTheFile(t *testing.T) {
	r := extractSrc(t, rel, "extends Node\n\nsignal done\n\nfunc run():\n\tpass\n\nclass Helper:\n\tfunc help():\n\t\tpass\n")
	same(t, "nodes", views(r), []string{
		`a/s.gd file owner= L1-L11 exported=true sig=""`,
		`a/s.gd#done signal owner= L3-L3 exported=true sig="signal done"`,
		`a/s.gd#run function owner= L5-L6 exported=true sig="func run()"`,
		`a/s.gd#Helper class owner= L8-L10 exported=true sig="class Helper"`,
		`a/s.gd#Helper.help method owner=Helper L9-L10 exported=true sig="func help()"`,
	})
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("a/s.gd", "a/s.gd#done"),
		contains("a/s.gd", "a/s.gd#run"),
		contains("a/s.gd", "a/s.gd#Helper"),
		contains("a/s.gd#Helper", "a/s.gd#Helper.help"),
	})
	if r.Package != "" {
		t.Errorf("Package = %q, want none", r.Package)
	}
}

// Godot takes class_name anywhere at the top of the script, after extends
// too; the class still spans the file and owns every function.
func TestAClassNameAfterExtendsStillNamesTheScript(t *testing.T) {
	r := extractSrc(t, rel, "extends Node\nclass_name Late\nfunc f():\n\tpass\n")
	same(t, "nodes", views(r), []string{
		`a/s.gd file owner= L1-L5 exported=true sig=""`,
		`a/s.gd#Late class owner= L1-L4 exported=true sig="class_name Late"`,
		`a/s.gd#Late.f method owner=Late L3-L4 exported=true sig="func f()"`,
	})
}

// gotreesitter v0.55.1 reads `class_name` with no name on its line as a
// class named after the first word of the next line: here "func". Godot
// wants the name on the keyword's line, and so does the extractor.
func TestAClassNameWithoutANameOnItsLineNamesNothing(t *testing.T) {
	r := extractSrc(t, rel, "class_name\nfunc f():\n\tpass\n")
	same(t, "nodes", views(r), []string{`a/s.gd file owner= L1-L4 exported=true sig=""`})
	if r.Package != "" || r.ParseErrors == 0 {
		t.Errorf("Package %q, ParseErrors %d; want none and some", r.Package, r.ParseErrors)
	}
}

// A function the parser rebuilt around text it rejected gets no node; the
// next one it swallowed goes with it, the one after that is whole.
func TestABrokenHeaderGetsNoNode(t *testing.T) {
	r := extractSrc(t, rel, "func broken(:\n\tpass\n\nfunc after():\n\tpass\n\nfunc ok():\n\tpass\n")
	same(t, "nodes", views(r), []string{
		`a/s.gd file owner= L1-L9 exported=true sig=""`,
		`a/s.gd#ok function owner= L7-L8 exported=true sig="func ok()"`,
	})
	if r.ParseErrors == 0 {
		t.Error("ParseErrors = 0, want the rejected text counted")
	}
}

func TestAnEmptyScriptIsItsFileNode(t *testing.T) {
	r := extractSrc(t, rel, "")
	same(t, "nodes", views(r), []string{`a/s.gd file owner= L1-L1 exported=true sig=""`})
}

// Godot on Windows may save with CRLF; the nodes must not change with it.
func TestCRLFGivesTheSameNodes(t *testing.T) {
	lf := extractSrc(t, rel, named)
	crlf := extractSrc(t, rel, strings.ReplaceAll(named, "\n", "\r\n"))
	same(t, "nodes", views(crlf), views(lf))
	same(t, "edges", crlf.Edges, lf.Edges)
}

func TestLanguageDescribesTheExtractor(t *testing.T) {
	l := gdscript.Language{}
	if l.Name() != "gdscript" || l.Version() != "gdscript/1@"+treesitter.Parser || !reflect.DeepEqual(l.Extensions(), []string{".gd"}) {
		t.Fatalf("Language = %q %q %v", l.Name(), l.Version(), l.Extensions())
	}
	r, err := l.File(rel, "func f():\n\tpass\n")
	if err != nil || len(r.Nodes) != 2 {
		t.Fatalf("Language.File = %v, %v", views(r), err)
	}
}
```

`internal/code/extract/gdscript/internal_test.go`:

```go
package gdscript

import (
	"strings"
	"testing"
)

func TestScriptReportsAParseFailure(t *testing.T) {
	// The grammar builds a tree from any input; only a missing grammar makes
	// the parse itself fail.
	_, err := script(nil, "a.gd", "func f():\n\tpass\n")
	if err == nil || !strings.Contains(err.Error(), "parse a.gd") {
		t.Errorf("script(nil grammar) = %v, want an error naming the file", err)
	}
}
```

- [ ] **Step 2: Run, expect FAIL**

Damit RED eine Assertion ist und kein Build-Fehler: zuerst `gdscript.go` nur mit `Language`, `File` und `script`, die ein `extract.Result{Path: rel, Language: "gdscript"}` ohne Knoten liefern.

```bash
go test ./internal/code/extract/gdscript/ > <scratchpad>/t2-red.txt 2>&1
```

Expected: `--- FAIL` bei jedem Knotentest.

- [ ] **Step 3: `gdscript.go` schreiben**

```go
// Package gdscript extracts the nodes and raw edges of one file of a Godot 4
// project on the shared tree-sitter core: a GDScript file (.gd), a scene or a
// resource (.tscn, .tres), or the project file (project.godot).
//
// One language for all four, because the resolver resolves a language
// against its own files alone, and a scene reaches the script it attaches,
// a script the scene it preloads, only inside one group.
//
// Raw, as in the other extractors: a base class, a call or a name a script
// uses names its target, and resolving it needs every file of the project,
// which is internal/code/resolve's job.
//
// Every script is a class. A script with `class_name X` gets a class node X
// over the whole file, and its functions and signals are X's members; a
// script without one gets no class node, and its functions hang on the file.
// Inner classes get nodes at every depth, qualified by the classes around
// them. var, const and enum get none; their text stays in the body of the
// class or the file.
//
// A definition the parser rebuilt around text it rejected gets no node, and
// nothing inside it counts, as in the Python extractor (header).
//
// The node types and fields read here, as gotreesitter builds them
// (treesitter.Parser), GDScript first:
//
//	source                the statements as children
//	class_name_statement  field name, on the keyword's line; field extends in `class_name X extends Y`
//	extends_statement     a type child (an identifier, or an attribute of identifiers) or a string child
//	function_definition   fields name, parameters, return_type, body; a ":" child closes the header
//	class_definition      fields name, extends, body (a class_body); a ":" child closes the header
//	signal_statement      field name
//	const_statement       fields name, value; so is variable_statement
//	call                  the callee identifier first, field arguments
//	attribute             identifier and "." children; an attribute_call last for a member call
//	attribute_call        the member identifier first, field arguments
//	string                its text, quotes included
package gdscript

import (
	"slices"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
	gdgrammar "github.com/odvcencio/gotreesitter/grammars/gdscript"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// version is this extractor's identity. Bump it by hand whenever the nodes or
// edges below change shape or meaning; the extract cache and the graph's meta
// key on it.
const version = "gdscript/1"

// langName is the language as the graph's meta lists it and as a Result carries it.
const langName = "gdscript"

// Language is the GDScript extractor as extract.Language describes one.
type Language struct{}

// Name is "gdscript".
func (Language) Name() string { return langName }

// Version is this package's version plus the tree-sitter runtime: a new
// runtime can build other trees from the same source.
func (Language) Version() string { return version + "@" + treesitter.Parser }

// Extensions are the files of a Godot project this package reads.
func (Language) Extensions() []string { return []string{".gd"} }

// File is this package's File.
func (Language) File(rel, source string) (extract.Result, error) { return File(rel, source) }

// File extracts one file. rel is its repo-relative, slash-separated path;
// source is its contents.
func File(rel, source string) (extract.Result, error) {
	return script(gdgrammar.Language(), rel, source)
}

// script extracts one GDScript file with the grammar handed in: the grammar
// builds a tree from any input, and only a missing grammar makes the parse
// fail.
func script(lang *gts.Language, rel, source string) (extract.Result, error) {
	doc, err := treesitter.Parse(lang, rel, []byte(source))
	if err != nil {
		return extract.Result{}, err
	}
	defer doc.Close()

	x := &extractor{doc: doc, fileID: model.NodeID(rel), referenced: map[[2]string]bool{}}
	x.script()
	return extract.Result{
		Path:     rel,
		Language: langName,
		Package:  x.className,
		Imports:  x.imports,
		// script has run every Symbol by now, as FileNode requires.
		Nodes:       append([]model.Node{doc.FileNode()}, x.nodes...),
		Edges:       slices.Concat(x.contains, x.importEdges, x.extends, x.calls, x.refs),
		ParseErrors: doc.ParseErrors(),
	}, nil
}

// extractor gathers one script's nodes and edges in source order.
type extractor struct {
	doc       *treesitter.Doc
	fileID    model.NodeID
	className string

	nodes       []model.Node
	contains    []extract.RawEdge
	extends     []extract.RawEdge
	calls       []extract.RawEdge
	refs        []extract.RawEdge
	importEdges []extract.RawEdge
	imports     []extract.Import
	referenced  map[[2]string]bool // source and name of every reference so far
}

// scope is what a definition or a call at one place belongs to.
type scope struct {
	source model.NodeID // the innermost definition with a node, or the script's class
	class  string       // the qualified class around it; "" at the top of a script without class_name
	fn     string       // the function it is in, for super(); "" outside one
}

// script reads the whole file: the class_name first, wherever it stands,
// because every definition of the script belongs to it, then the statements.
func (x *extractor) script() {
	root := x.doc.Root
	top := scope{source: x.fileID}
	if stmt := x.classNameStatement(root); stmt != nil {
		name := x.doc.Text(x.doc.Field(stmt, "name"))
		n := x.doc.Symbol(treesitter.Sym{
			Outer: root, Head: stmt.StartByte(), HeaderEnd: stmt.EndByte(),
			Name: name, Qualified: name, Kind: "class", Exported: exported(name),
		})
		x.add(n, x.fileID)
		x.className = name
		top = scope{source: n.ID, class: name}
		x.extendsOf(x.doc.Field(stmt, "extends"), n.ID)
	}
	x.body(root, top, true)
}

// classNameStatement is the script's class_name, or nil: the first one at
// the top whose name stands on the keyword's line and that holds no ERROR.
func (x *extractor) classNameStatement(root *gts.Node) *gts.Node {
	for _, stmt := range root.Children() {
		if x.doc.Type(stmt) != "class_name_statement" || holdsError(stmt) {
			continue
		}
		if name := x.doc.Field(stmt, "name"); name.StartPoint().Row == stmt.StartPoint().Row {
			return stmt
		}
	}
	return nil
}

// body reads the statements of a script (top) or of a class body.
func (x *extractor) body(n *gts.Node, sc scope, top bool) {
	for _, stmt := range n.Children() {
		switch x.doc.Type(stmt) {
		case "class_name_statement":
			// Read by script, its extends with it.
		case "extends_statement":
			x.extendsOf(stmt, sc.source)
		case "function_definition":
			x.function(stmt, sc)
		case "class_definition":
			x.class(stmt, sc)
		case "signal_statement":
			x.signal(stmt, sc)
		default:
			if top {
				x.alias(stmt)
			}
			x.collect(stmt, sc)
		}
	}
}

// function makes the node of a func and collects what its header and body
// call and name.
func (x *extractor) function(def *gts.Node, sc scope) {
	end, ok := x.header(def)
	if !ok {
		return
	}
	name := x.doc.Text(x.doc.Field(def, "name"))
	kind := model.Kind("method")
	if sc.class == "" {
		kind = "function"
	}
	n := x.symbol(def, end, name, kind, sc)
	inner := scope{source: n.ID, class: sc.class, fn: name}
	for _, c := range def.Children() {
		x.collect(c, inner)
	}
}

// class makes the node of an inner class, reads its base and its body.
func (x *extractor) class(def *gts.Node, sc scope) {
	end, ok := x.header(def)
	if !ok {
		return
	}
	name := x.doc.Text(x.doc.Field(def, "name"))
	n := x.symbol(def, end, name, "class", sc)
	x.extendsOf(x.doc.Field(def, "extends"), n.ID)
	x.body(x.doc.Field(def, "body"), scope{source: n.ID, class: qualify(sc.class, name)}, false)
}

// signal makes the node of a signal declaration; its signature is the whole
// statement.
func (x *extractor) signal(stmt *gts.Node, sc scope) {
	if holdsError(stmt) {
		return
	}
	x.symbol(stmt, stmt.EndByte(), x.doc.Text(x.doc.Field(stmt, "name")), "signal", sc)
}

// symbol mints the node of one definition, whose header ends at end, and the
// contains edge from the scope's source to it.
func (x *extractor) symbol(def *gts.Node, end uint32, name string, kind model.Kind, sc scope) model.Node {
	n := x.doc.Symbol(treesitter.Sym{
		Outer: def, Head: def.StartByte(), HeaderEnd: end,
		Name: name, Qualified: qualify(sc.class, name), Kind: kind,
		Owner: sc.class, Exported: exported(name),
	})
	x.add(n, sc.source)
	return n
}

// add records a node and the contains edge from parent to it.
func (x *extractor) add(n model.Node, parent model.NodeID) {
	x.nodes = append(x.nodes, n)
	x.contains = append(x.contains, extract.RawEdge{
		Source: parent, Relation: model.RelationContains, TargetID: n.ID, File: x.doc.Rel,
	})
}

// header is where the signature of def ends, the end of its first direct ":"
// child, and false when the parser rebuilt the definition around text it
// rejected: an ERROR in or under a child ahead of that colon, or no colon.
// `func broken(:` swallows the next definition into its parameters, and its
// span would claim lines of another; no node beats a wrong one.
func (x *extractor) header(def *gts.Node) (uint32, bool) {
	for _, c := range def.Children() {
		if holdsError(c) {
			break
		}
		if x.doc.Type(c) == ":" {
			return c.EndByte(), true
		}
	}
	return 0, false
}

// holdsError reports whether n is an ERROR node or has one below it.
func holdsError(n *gts.Node) bool {
	found := false
	treesitter.Walk(n, func(c *gts.Node) bool {
		found = found || c.IsError()
		return !found
	})
	return found
}

// qualify is name inside the class owner: "Outer.Inner", or name alone at
// the top of a script without class_name.
func qualify(owner, name string) string {
	if owner == "" {
		return name
	}
	return owner + "." + name
}

// exported is whether a name is public by GDScript's convention: it does not
// start with an underscore. The engine's callbacks (_ready, _process) follow
// the same convention and count as not exported.
func exported(name string) bool { return !strings.HasPrefix(name, "_") }
```

- [ ] **Step 4: Failing tests (Kanten)**

An `gdscript_test.go` anhängen:

```go
// edgeSrc holds every edge form of a script, probed on gotreesitter v0.55.1.
const edgeSrc = "class_name Sample extends \"res://base.gd\"\n\nconst Helper := preload(\"res://lib/helper.gd\")\nvar Scene = load(\"scenes/x.tscn\")\nvar nothing = preload()\n\nsignal changed\n\nfunc run(a: Foo = Foo.new()) -> Bar:\n\tbump()\n\tself.bump()\n\tHelper.make(1)\n\tOuter.Inner.go()\n\tself.box.open()\n\tselfish.go()\n\tsuper.run(a)\n\tsuper()\n\tchanged.emit()\n\temit_signal(\"changed\")\n\tget_node(\"x\").poke()\n\tvar r = load(\"res://y.gd\")\n\tif a is Qux:\n\t\tpass\n\treturn a as Quux\n\nclass Inner extends Outer.Base:\n\tpass\n"

func edge(src model.NodeID, r model.Relation, name, receiver, spec string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: r, Name: name, Receiver: receiver, Specifier: spec, File: rel}
}

func TestEveryEdgeFormOfAScript(t *testing.T) {
	r := extractSrc(t, rel, edgeSrc)
	const run = "a/s.gd#Sample.run"
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{
		edge("a/s.gd#Sample", model.RelationExtends, "", "", "res://base.gd"),
		edge("a/s.gd#Sample.Inner", model.RelationExtends, "Base", "Outer", ""),
	})
	same(t, "imports", edgesOf(r, model.RelationImports), []extract.RawEdge{
		edge("a/s.gd", model.RelationImports, "", "", "res://lib/helper.gd"),
		edge("a/s.gd", model.RelationImports, "", "", "scenes/x.tscn"),
		edge("a/s.gd", model.RelationImports, "", "", "res://y.gd"),
	})
	same(t, "aliases", r.Imports, []extract.Import{
		{Alias: "Helper", Path: "res://lib/helper.gd"},
		{Alias: "Scene", Path: "scenes/x.tscn"},
	})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		edge(run, model.RelationCalls, "new", "Foo", ""),
		edge(run, model.RelationCalls, "bump", "", ""),
		edge(run, model.RelationCalls, "bump", "", ""),
		edge(run, model.RelationCalls, "make", "Helper", ""),
		edge(run, model.RelationCalls, "go", "Outer.Inner", ""),
		edge(run, model.RelationCalls, "open", "box", ""),
		// Only "self." is cut off; a name that starts with self is a name.
		edge(run, model.RelationCalls, "go", "selfish", ""),
		edge(run, model.RelationCalls, "run", "super", ""),
		edge(run, model.RelationCalls, "run", "super", ""),
		edge(run, model.RelationCalls, "emit", "changed", ""),
		edge(run, model.RelationCalls, "emit", "changed", ""),
		edge(run, model.RelationCalls, "get_node", "", ""),
	})
	var refs []string
	for _, e := range edgesOf(r, model.RelationReferences) {
		refs = append(refs, string(e.Source)+" "+e.Name)
	}
	same(t, "references", refs, []string{
		run + " a", run + " Foo", run + " Bar", run + " self", run + " Helper",
		run + " Outer", run + " selfish", run + " super", run + " changed", run + " Qux", run + " Quux",
	})
}

// A script without class_name has the file as the source of its base, its
// top-level calls and its references.
func TestAnUnnamedScriptIsTheSourceOfItsTopLevel(t *testing.T) {
	r := extractSrc(t, rel, "extends Base\n\nvar x: Thing = make()\n")
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{edge("a/s.gd", model.RelationExtends, "Base", "", "")})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{edge("a/s.gd", model.RelationCalls, "make", "", "")})
	same(t, "references", edgesOf(r, model.RelationReferences), []extract.RawEdge{edge("a/s.gd", model.RelationReferences, "Thing", "", "")})
}

// super() outside a function has no name to look for, and an alias only
// counts at the top of the script.
func TestSuperOutsideAFunctionAndAnAliasInAClassGiveNothing(t *testing.T) {
	r := extractSrc(t, rel, "var v = super()\n\nclass C:\n\tconst H = preload(\"res://h.gd\")\n")
	same(t, "calls", edgesOf(r, model.RelationCalls), nil)
	same(t, "aliases", r.Imports, nil)
	same(t, "imports", edgesOf(r, model.RelationImports), []extract.RawEdge{edge("a/s.gd", model.RelationImports, "", "", "res://h.gd")})
}

// emit_signal and preload with no string literal name nothing.
func TestCallsWithoutALiteralNameNothing(t *testing.T) {
	r := extractSrc(t, rel, "func f(p):\n\temit_signal(p)\n\tload(p)\n\tpreload()\n")
	same(t, "calls", edgesOf(r, model.RelationCalls), nil)
	same(t, "imports", edgesOf(r, model.RelationImports), nil)
}

func TestCRLFGivesTheSameEdges(t *testing.T) {
	lf := extractSrc(t, rel, edgeSrc)
	crlf := extractSrc(t, rel, strings.ReplaceAll(edgeSrc, "\n", "\r\n"))
	same(t, "edges", crlf.Edges, lf.Edges)
	same(t, "aliases", crlf.Imports, lf.Imports)
}
```

- [ ] **Step 5: Run, expect FAIL**

Damit `gdscript.go` schon baut, legt dieser Schritt `edges.go` zunächst mit leeren Rümpfen der fünf Methoden an: `extendsOf`, `alias`, `collect`, `call` und `attribute`.

```bash
go test ./internal/code/extract/gdscript/ > <scratchpad>/t2-red2.txt 2>&1
```

Expected: `--- FAIL` in den vier neuen Kantentests, die Knotentests grün.

- [ ] **Step 6: `edges.go` schreiben**

```go
package gdscript

import (
	"strings"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// extendsOf records the base an extends_statement names, for the class or
// script src: a class name or a chain of them (Name, Receiver), or a path
// (Specifier). ext may be nil -- a class without a base.
func (x *extractor) extendsOf(ext *gts.Node, src model.NodeID) {
	for _, c := range ext.Children() {
		e := extract.RawEdge{Source: src, Relation: model.RelationExtends, File: x.doc.Rel}
		switch x.doc.Type(c) {
		case "string":
			e.Specifier = unquote(x.doc.Text(c))
		case "type":
			chain, ok := x.chain(c.Child(0))
			if !ok {
				continue
			}
			e.Name = chain
			if i := strings.LastIndexByte(chain, '.'); i >= 0 {
				e.Receiver, e.Name = chain[:i], chain[i+1:]
			}
		default:
			continue
		}
		x.extends = append(x.extends, e)
	}
}

// chain is an identifier, or an attribute made of identifiers alone, as
// written: "Outer.Inner". False for anything else.
func (x *extractor) chain(n *gts.Node) (string, bool) {
	switch x.doc.Type(n) {
	case "identifier":
		return x.doc.Text(n), true
	case "attribute":
		var parts []string
		for _, c := range n.Children() {
			switch x.doc.Type(c) {
			case "identifier":
				parts = append(parts, x.doc.Text(c))
			case ".":
			default:
				return "", false
			}
		}
		return strings.Join(parts, "."), true
	}
	return "", false
}

// alias records `const X = preload("p")` and `var X = load("p")` at the top
// of a script: X binds the file p for the rest of it. The import edge itself
// comes from collect, which walks the statement after this.
func (x *extractor) alias(stmt *gts.Node) {
	v := x.doc.Field(stmt, "value")
	if x.doc.Type(v) != "call" {
		return
	}
	switch x.doc.Text(v.Child(0)) {
	case "preload", "load":
		if p, ok := x.firstArg(v); ok {
			x.imports = append(x.imports, extract.Import{Alias: x.doc.Text(x.doc.Field(stmt, "name")), Path: p})
		}
	}
}

// collect walks a subtree that holds no definition with a node and records
// its calls and references for sc and its imports for the file. An ERROR
// node is skipped whole: a call found there sits in text the grammar
// rejected.
func (x *extractor) collect(n *gts.Node, sc scope) {
	treesitter.Walk(n, func(c *gts.Node) bool {
		if c.IsError() {
			return false
		}
		switch x.doc.Type(c) {
		case "call":
			x.call(c, sc)
			return false
		case "attribute":
			x.attribute(c, sc)
			return false
		case "identifier":
			x.reference(c, sc)
		}
		return true
	})
}

// call records a call of a plain name: foo(), and the forms that are no
// call of a function by that name -- preload and load, whose literal is a
// file the script needs; super(), the base's version of the function it
// stands in; emit_signal("x"), which is x.emit(). The callee is no
// reference; the arguments are walked.
func (x *extractor) call(c *gts.Node, sc scope) {
	e := extract.RawEdge{Source: sc.source, Relation: model.RelationCalls, Name: x.doc.Text(c.Child(0)), File: x.doc.Rel}
	switch e.Name {
	case "preload", "load":
		if p, ok := x.firstArg(c); ok {
			x.importEdges = append(x.importEdges, extract.RawEdge{
				Source: x.fileID, Relation: model.RelationImports, Specifier: p, File: x.doc.Rel,
			})
		}
		e.Name = ""
	case "super":
		e.Name, e.Receiver = sc.fn, "super"
	case "emit_signal":
		signal, ok := x.firstArg(c)
		e.Name, e.Receiver = "emit", signal
		if !ok {
			e.Name = ""
		}
	}
	if e.Name != "" {
		x.calls = append(x.calls, e)
	}
	for _, k := range c.Children()[1:] {
		x.collect(k, sc)
	}
}

// attribute reads a.b.c and a.b(): the leading name is a reference, a name
// after a "." is a member and none. Each attribute_call is a call whose
// receiver is the chain of plain names before it, `self.` cut off, as long
// as nothing else -- a call, $Node, a subscript -- stands there.
func (x *extractor) attribute(n *gts.Node, sc scope) {
	var chain []string
	plain := true
	for i, c := range n.Children() {
		switch x.doc.Type(c) {
		case ".":
		case "identifier":
			if i == 0 {
				x.reference(c, sc)
			}
			chain = append(chain, x.doc.Text(c))
		case "attribute_call":
			if plain {
				recv := strings.Join(chain, ".")
				if recv == "self" {
					recv = ""
				} else {
					recv = strings.TrimPrefix(recv, "self.")
				}
				x.calls = append(x.calls, extract.RawEdge{
					Source: sc.source, Relation: model.RelationCalls,
					Name: x.doc.Text(c.Child(0)), Receiver: recv, File: x.doc.Rel,
				})
			}
			plain = false
			for _, k := range c.Children()[1:] {
				x.collect(k, sc)
			}
		default:
			plain = false
			x.collect(c, sc)
		}
	}
}

// reference records a name a definition uses, once per definition. Most are
// locals and parameters; the resolver keeps those that name a class.
func (x *extractor) reference(c *gts.Node, sc scope) {
	key := [2]string{string(sc.source), x.doc.Text(c)}
	if x.referenced[key] {
		return
	}
	x.referenced[key] = true
	x.refs = append(x.refs, extract.RawEdge{Source: sc.source, Relation: model.RelationReferences, Name: key[1], File: x.doc.Rel})
}

// firstArg is the first argument of a call, unquoted, and whether it is a
// string literal. The arguments are "(", the arguments with their commas,
// ")"; `preload()` has none.
func (x *extractor) firstArg(c *gts.Node) (string, bool) {
	args := x.doc.Field(c, "arguments").Children()
	if len(args) < 3 {
		return "", false
	}
	return unquote(x.doc.Text(args[1])), x.doc.Type(args[1]) == "string"
}

// unquote strips the quotes of a string literal: "a", 'a'.
func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}
```

- [ ] **Step 7: Grün und Coverage**

```bash
go test -coverprofile=<scratchpad>/t2.cov ./internal/code/extract/gdscript/ > <scratchpad>/t2-green.txt 2>&1; echo $?
go tool cover -func=<scratchpad>/t2.cov | grep -v '100.0%'
```

Expected: Exit 0; außer der Gesamtzeile keine Funktion unter 100 %. Bleibt ein Zweig offen, mit einer Testeingabe schließen, nicht mit einer Ausnahme.

- [ ] **Step 8: Mutationsrunde**

Boden, je Mutant ein Overlay:
1. `classNameStatement`: Zeilenvergleich `==` → `!=`.
2. `classNameStatement`: `|| holdsError(stmt)` streichen.
3. `function`: `kind = "function"` streichen.
4. `header`: `break` → `continue`.
5. `body`: `if top {` → `if true {`.
6. `alias`: `case "preload", "load":` → nur `"preload"`.
7. `call`: Zweig `case "super":` streichen.
8. `call`: `if !ok { e.Name = "" }` streichen.
9. `attribute`: `if i == 0` → `if true`.
10. `attribute`: `plain = false` im `default`-Zweig streichen.
11. `reference`: Dedupe-Rückgabe streichen.
12. `firstArg`: `< 3` → `< 2`.
13. `unquote`: `s[len(s)-1] == s[0]` streichen.

Zusätzlich jede weitere Teilbedingung einzeln.

- [ ] **Step 9: Commit**

```
feat(graph): extract GDScript files into the code graph

A script with class_name becomes a class over the whole file and owns its
functions and signals; inner classes are qualified by the classes around
them. Bases, preload and load literals, calls by receiver chain, super and
emit_signal, and every name a definition uses come out raw for the
resolver. The extractor is not registered yet.
```

```bash
git add internal/code/extract/gdscript/
git commit -F <scratchpad>/msg-t2.txt > <scratchpad>/commit-2.log 2>&1; echo $?
```

---

### Task 3: Szenen, Ressourcen und `project.godot`

**Files:**
- Create: `internal/code/extract/gdscript/resource.go`
- Modify: `internal/code/extract/gdscript/gdscript.go` (`Extensions`, `File`)
- Modify: `internal/code/extract/gdscript/gdscript_test.go` (`TestLanguageDescribesTheExtractor`, neue Tests)
- Modify: `internal/code/extract/gdscript/internal_test.go`

**Interfaces:**
- Produces:
  - `Extensions() == []string{".gd", ".godot", ".tres", ".tscn"}`. `File` liest `.gd` als Skript, alles andere als Ressource.
- Ressource:
  - Dateiknoten und je `[ext_resource … path="p"]` eine Rohkante `imports` mit `Specifier: "p"`.
- `project.godot`:
  - Je Property der Sektion `[autoload]` ein `Import{Alias: <key>, Path: <wert ohne führendes *>}` plus Rohkante `imports`.
  - `run/main_scene` der Sektion `[application]` als Rohkante `imports`.
  - Andere `*.godot`-Dateien bekommen nur den Dateiknoten.

- [ ] **Step 1: Failing tests**

An `gdscript_test.go` anhängen; in `TestLanguageDescribesTheExtractor` die Erwartung auf `[]string{".gd", ".godot", ".tres", ".tscn"}` ändern:

```go
func TestAResourceImportsWhatItsExtResourcesName(t *testing.T) {
	const p = "data/stats.tres"
	r := extractSrc(t, p, "[gd_resource type=\"Resource\" script_class=\"Stats\" load_steps=2 format=3]\n\n[ext_resource type=\"Script\" path=\"res://stats.gd\" id=\"1_s\"]\n[ext_resource type=\"Texture2D\" uid=\"uid://abc\" id=\"2_t\"]\n\n[resource]\nscript = ExtResource(\"1_s\")\nhp = 3\n")
	if len(r.Nodes) != 1 || r.Nodes[0].ID != p || r.Language != "gdscript" || len(r.Imports) != 0 {
		t.Fatalf("nodes %v, language %q, imports %v", views(r), r.Language, r.Imports)
	}
	same(t, "imports", r.Edges, []extract.RawEdge{
		{Source: p, Relation: model.RelationImports, Specifier: "res://stats.gd", File: p},
	})
}

func TestTheProjectFileNamesItsAutoloadsAndMainScene(t *testing.T) {
	const p = "game/project.godot"
	r := extractSrc(t, p, "; comment\nconfig_version=5\n\n[autoload]\n\nClock=\"*res://autoload/clock.gd\"\nGame=\"res://autoload/game.tscn\"\n\n[application]\n\nconfig/name=\"X\"\nrun/main_scene=\"res://main.tscn\"\n")
	same(t, "autoloads", r.Imports, []extract.Import{
		{Alias: "Clock", Path: "res://autoload/clock.gd"},
		{Alias: "Game", Path: "res://autoload/game.tscn"},
	})
	same(t, "imports", r.Edges, []extract.RawEdge{
		{Source: p, Relation: model.RelationImports, Specifier: "res://autoload/clock.gd", File: p},
		{Source: p, Relation: model.RelationImports, Specifier: "res://autoload/game.tscn", File: p},
		{Source: p, Relation: model.RelationImports, Specifier: "res://main.tscn", File: p},
	})
}

// Only project.godot is the project file; another *.godot file, and an
// [autoload] section in a scene, name nothing.
func TestOnlyTheProjectFileHasAutoloads(t *testing.T) {
	const body = "[autoload]\n\nClock=\"*res://clock.gd\"\n\n[application]\n\nrun/main_scene=\"res://main.tscn\"\n"
	for _, p := range []string{"game/other.godot", "game/x.tscn"} {
		r := extractSrc(t, p, body)
		if len(r.Imports) != 0 || len(r.Edges) != 0 || len(r.Nodes) != 1 {
			t.Errorf("%s: imports %v, edges %v, nodes %v", p, r.Imports, r.Edges, views(r))
		}
	}
}

// gotreesitter v0.55.1 leaves an unclosed section header as loose tokens of
// the resource, with no ERROR node (probed): it is no section and names
// nothing, and the section after it is whole.
func TestAnUnclosedSectionNamesNothing(t *testing.T) {
	const p = "x.tscn"
	r := extractSrc(t, p, "[ext_resource path=\"res://a.gd\"\n[ext_resource type=\"Script\" path=\"res://b.gd\" id=\"1\"]\n")
	same(t, "imports", r.Edges, []extract.RawEdge{
		{Source: p, Relation: model.RelationImports, Specifier: "res://b.gd", File: p},
	})
}
```

`internal_test.go` ergänzen:

```go
func TestResourceReportsAParseFailure(t *testing.T) {
	_, err := resource(nil, "x.tscn", "[gd_scene]\n")
	if err == nil || !strings.Contains(err.Error(), "parse x.tscn") {
		t.Errorf("resource(nil grammar) = %v, want an error naming the file", err)
	}
}
```

- [ ] **Step 2: Run, expect FAIL**

Für den RED-Lauf wird `resource.go` zunächst mit einem `resource`, der `extract.Result{}` liefert, angelegt.

```bash
go test ./internal/code/extract/gdscript/ > <scratchpad>/t3-red.txt 2>&1
```

Expected: `--- FAIL` in den neuen Tests und in `TestLanguageDescribesTheExtractor`.

- [ ] **Step 3: `resource.go` und Dispatch**

`gdscript.go`, Package-Kommentar um die Ressourcengrammatik ergänzen (an die Tabelle anhängen):

```go
// and the resource grammar of scenes, resources and the project file:
//
//	resource   the sections as children, and properties ahead of the first
//	section    "[", its identifier, attribute children, "]", then property children
//	attribute  key identifier, "=", value
//	property   key path, "=", value
```

`gdscript.go`, `Extensions` und `File`:

```go
// Extensions are the files of a Godot project this package reads: scripts,
// the project file, resources and scenes.
func (Language) Extensions() []string { return []string{".gd", ".godot", ".tres", ".tscn"} }

// File extracts one file. rel is its repo-relative, slash-separated path;
// source is its contents. A script is read as GDScript, anything else as a
// Godot resource.
func File(rel, source string) (extract.Result, error) {
	if path.Ext(rel) == ".gd" {
		return script(gdgrammar.Language(), rel, source)
	}
	return resource(resgrammar.Language(), rel, source)
}
```

Imports in `gdscript.go` um `"path"` und `resgrammar "github.com/odvcencio/gotreesitter/grammars/godot_resource"` ergänzen.

`resource.go`:

```go
package gdscript

import (
	"path"
	"strings"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// resource extracts a scene, a resource or the project file: its file node,
// an import of every external resource it names by path, and for
// project.godot the autoloads and the main scene. A reference by uid alone
// names no file the graph could point at.
func resource(lang *gts.Language, rel, source string) (extract.Result, error) {
	doc, err := treesitter.Parse(lang, rel, []byte(source))
	if err != nil {
		return extract.Result{}, err
	}
	defer doc.Close()

	r := extract.Result{
		Path: rel, Language: langName,
		Nodes: []model.Node{doc.FileNode()}, ParseErrors: doc.ParseErrors(),
	}
	edge := func(spec string) {
		r.Edges = append(r.Edges, extract.RawEdge{
			Source: model.NodeID(rel), Relation: model.RelationImports, Specifier: spec, File: rel,
		})
	}
	project := path.Base(rel) == "project.godot"
	for _, s := range doc.Root.Children() {
		if doc.Type(s) != "section" {
			continue
		}
		switch doc.Text(s.Child(1)) {
		case "ext_resource":
			if p, ok := pairs(doc, s, "attribute")["path"]; ok {
				edge(p)
			}
		case "autoload":
			for _, kv := range ordered(doc, s, "property") {
				if project {
					p := strings.TrimPrefix(kv[1], "*")
					r.Imports = append(r.Imports, extract.Import{Alias: kv[0], Path: p})
					edge(p)
				}
			}
		case "application":
			if p, ok := pairs(doc, s, "property")["run/main_scene"]; ok && project {
				edge(p)
			}
		}
	}
	return r, nil
}

// ordered is the key and unquoted string value of every child of s of type
// typ, in source order: attribute in a section's header, property below it.
// A value that is no string literal is left out.
func ordered(doc *treesitter.Doc, s *gts.Node, typ string) [][2]string {
	var out [][2]string
	for _, c := range s.Children() {
		if doc.Type(c) == typ && doc.Type(c.Child(2)) == "string" {
			out = append(out, [2]string{doc.Text(c.Child(0)), unquote(doc.Text(c.Child(2)))})
		}
	}
	return out
}

// pairs is ordered as a map, for a section read by key.
func pairs(doc *treesitter.Doc, s *gts.Node, typ string) map[string]string {
	m := map[string]string{}
	for _, kv := range ordered(doc, s, typ) {
		m[kv[0]] = kv[1]
	}
	return m
}
```

`TestTheProjectFileNamesItsAutoloadsAndMainScene` erwartet die Kanten in Quellreihenfolge: erst die Autoloads, dann die Hauptszene. Die Schleife liefert genau diese Reihenfolge.

- [ ] **Step 4: Grün, Coverage 100 %**

```bash
go test -coverprofile=<scratchpad>/t3.cov ./internal/code/extract/gdscript/ > <scratchpad>/t3-green.txt 2>&1; echo $?
go tool cover -func=<scratchpad>/t3.cov | grep -v '100.0%'
```

- [ ] **Step 5: Mutationsrunde**

Boden:
1. `project :=` → `true`.
2. `TrimPrefix(kv[1], "*")` → `kv[1]`.
3. `ok && project` → `ok`.
4. `doc.Type(c.Child(2)) == "string"` streichen.
5. `doc.Type(s) != "section"` → `false`.
6. `path.Ext(rel) == ".gd"` → `!=`.

- [ ] **Step 6: Commit**

```
feat(graph): read scenes, resources and project.godot for the code graph

A scene or resource imports every external resource it names by path, so
a script attached only in a scene is no island. project.godot names its
autoloads, which bind a global name to a file, and its main scene.
```

---

### Task 4: Auflösung `resolveGDScript`

**Files:**
- Create: `internal/code/resolve/gdscript.go`
- Create: `internal/code/resolve/gdscript_test.go`
- Modify: `internal/code/resolve/resolve.go` (`Graph`, `edgesOf`)

**Interfaces:**
- Consumes: `gdscript.File`, `extract.Result` aus Task 2/3 (`Package` = class_name; `Imports` mit Alias); `one`, `dirOf` aus `resolve`.
- Produces: `edgesOf(lang string, files []extract.Result, mods []Module, paths map[string]bool) []model.Edge`. `Graph` füllt `paths` mit dem `Path` jedes Results aller Sprachen.

- [ ] **Step 1: Failing tests**

`internal/code/resolve/gdscript_test.go`:

```go
package resolve_test

import (
	"sort"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/gdscript"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// gdGraph extracts a tree of Godot files with the real extractor, in byte
// order, adds the foreign files as bare file nodes of another language, and
// resolves them.
func gdGraph(t *testing.T, tree map[string]string, foreign ...string) *model.Graph {
	t.Helper()
	paths := make([]string, 0, len(tree))
	for p := range tree {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var files []extract.Result
	for _, p := range paths {
		r, err := gdscript.File(p, tree[p])
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, r)
	}
	for _, p := range foreign {
		files = append(files, extract.Result{Path: p, Language: "csharp", Nodes: []model.Node{
			{ID: model.NodeID(p), Name: p, Kind: model.KindFile, Path: p, BodyHash: "h"},
		}})
	}
	return validated(t, resolve.Graph(files, nil, stamp))
}

func TestGodotPathsResolveAgainstTheProjectAndAcrossLanguages(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"game/project.godot":    "[autoload]\nGame=\"*res://autoload/game.gd\"\n\n[application]\nrun/main_scene=\"res://main.tscn\"\n",
		"game/autoload/game.gd": "extends Node\n\nfunc start():\n\tpass\n",
		"game/main.tscn":        "[gd_scene format=3]\n\n[ext_resource type=\"Script\" path=\"res://main.gd\" id=\"1\"]\n[ext_resource type=\"Texture2D\" path=\"res://icon.png\" id=\"2\"]\n[ext_resource type=\"Script\" path=\"res://player.cs\" id=\"3\"]\n",
		"game/main.gd":          "extends Node\n\nconst Lib = preload(\"lib/lib.gd\")\n\nfunc _ready():\n\tLib.helper()\n\tGame.start()\n\tpreload(\"res://main.gd\")\n",
		"game/lib/lib.gd":       "static func helper():\n\tpass\n",
		"loose/x.gd":            "const L = preload(\"res://game/lib/lib.gd\")\nconst M = preload(\"../game/lib/lib.gd\")\nconst U = load(\"uid://abc\")\n",
	}, "game/player.cs")
	wantTargets(t, g, "game/project.godot", model.RelationImports, "game/autoload/game.gd", "game/main.tscn")
	wantTargets(t, g, "game/main.tscn", model.RelationImports, "game/main.gd", "game/player.cs")
	wantTargets(t, g, "game/main.gd", model.RelationImports, "game/lib/lib.gd")
	wantTargets(t, g, "loose/x.gd", model.RelationImports, "game/lib/lib.gd")
	wantCalls(t, g, "game/main.gd#_ready", "game/autoload/game.gd#start extracted", "game/lib/lib.gd#helper extracted")
}

func TestGodotInheritanceSuperSignalsAndConstructors(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"p/project.godot": "config_version=5\n",
		"p/base.gd":       "class_name Base\nextends Node\n\nsignal hit\n\nfunc _init():\n\tpass\n\nfunc greet():\n\tpass\n",
		"p/mid.gd":        "class_name Mid\nextends Base\n",
		"p/leaf.gd":       "extends \"res://mid.gd\"\n\nfunc greet():\n\tsuper()\n\thit.emit()\n\temit_signal(\"hit\")\n\nfunc run():\n\tgreet()\n\tself.greet()\n\tsuper.greet()\n\tvar b: Base = Base.new()\n\tvar m = Mid.new()\n\thit.is_connected(greet)\n\tprint(b)\n",
	})
	wantTargets(t, g, "p/base.gd#Base", model.RelationExtends)
	wantTargets(t, g, "p/mid.gd#Mid", model.RelationExtends, "p/base.gd#Base")
	wantTargets(t, g, "p/leaf.gd", model.RelationExtends, "p/mid.gd#Mid")
	wantCalls(t, g, "p/leaf.gd#greet", "p/base.gd#Base.greet extracted")
	wantTargets(t, g, "p/leaf.gd#greet", model.RelationReferences, "p/base.gd#Base.hit")
	wantCalls(t, g, "p/leaf.gd#run",
		"p/base.gd#Base._init extracted", "p/base.gd#Base.greet extracted",
		"p/leaf.gd#greet extracted", "p/mid.gd#Mid extracted")
	wantTargets(t, g, "p/leaf.gd#run", model.RelationReferences, "p/base.gd#Base", "p/mid.gd#Mid")
}

func TestGodotNameBindingOrderAndAmbiguity(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"p/project.godot": "[autoload]\nTool=\"*res://tool_auto.gd\"\nTwin=\"*res://twin_auto.gd\"\n",
		"p/tool_auto.gd":  "func use():\n\tpass\n",
		"p/twin_auto.gd":  "static func make():\n\tpass\n",
		"p/tool_class.gd": "class_name Tool\n\nstatic func use():\n\tpass\n",
		"p/widget.gd":     "class_name Widget\n\nstatic func use():\n\tpass\n",
		"p/gadget.gd":     "class_name Gadget\n\nstatic func use():\n\tpass\n",
		"p/gadget_impl.gd": "static func use():\n\tpass\n",
		"p/twin_a.gd":     "class_name Twin\n\nstatic func make():\n\tpass\n",
		"p/twin_b.gd":     "class_name Twin\n\nstatic func make():\n\tpass\n",
		"p/outer.gd":      "class_name Outer\n\nclass Inner:\n\tstatic func go():\n\t\tpass\n",
		"p/user.gd":       "const Gadget = preload(\"res://gadget_impl.gd\")\n\nfunc a():\n\tTool.use()\n\tWidget.use()\n\tGadget.use()\n\tTwin.make()\n\tOuter.Inner.go()\n\tOuter.Nope.go()\n\nclass Widget:\n\tstatic func use():\n\t\tpass\n",
	})
	wantCalls(t, g, "p/user.gd#a",
		"p/gadget_impl.gd#use extracted", "p/outer.gd#Outer.Inner.go extracted",
		"p/tool_class.gd#Tool.use extracted", "p/user.gd#Widget.use extracted")
}

func TestGodotTheNearestProjectOwnsResPaths(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"a/project.godot":     "config_version=5\n",
		"a/x.gd":              "",
		"a/s.gd":              "const X = preload(\"res://x.gd\")\n",
		"a/sub/project.godot": "config_version=5\n",
		"a/sub/x.gd":          "",
		"a/sub/s.gd":          "const X = preload(\"res://x.gd\")\n",
	})
	wantTargets(t, g, "a/s.gd", model.RelationImports, "a/x.gd")
	wantTargets(t, g, "a/sub/s.gd", model.RelationImports, "a/sub/x.gd")
}

func TestGodotACycleOfBasesEndsAndSelfReferencesDrop(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"c/project.godot": "config_version=5\n",
		"c/a.gd":          "class_name A\nextends B\n\nfunc f():\n\tmissing()\n",
		"c/b.gd":          "class_name B\nextends A\n",
		"c/me.gd":         "class_name Me\nextends Me\n",
		"c/k.gd":          "class_name K\n\nvar other: K\n\nfunc f(x: K, y: L) -> L:\n\treturn L.new()\n",
		"c/l.gd":          "class_name L\n",
	})
	wantTargets(t, g, "c/a.gd#A", model.RelationExtends, "c/b.gd#B")
	wantTargets(t, g, "c/b.gd#B", model.RelationExtends, "c/a.gd#A")
	wantTargets(t, g, "c/me.gd#Me", model.RelationExtends)
	wantCalls(t, g, "c/a.gd#A.f")
	wantTargets(t, g, "c/k.gd#K", model.RelationReferences)
	wantTargets(t, g, "c/k.gd#K.f", model.RelationReferences, "c/l.gd#L")
	wantCalls(t, g, "c/k.gd#K.f", "c/l.gd#L extracted")
}
```

Die zyklische Probe läuft mit Zeitgrenze: `go test -timeout 30s`.

- [ ] **Step 2: Run, expect FAIL**

```bash
go test -timeout 60s ./internal/code/resolve/ -run Godot > <scratchpad>/t4-red.txt 2>&1
```

Expected: `--- FAIL` in allen fünf. Der Arm fehlt, nur `contains` kommt durch.

- [ ] **Step 3: `resolve.go` anpassen**

In `Graph`, vor der Schleife über die Sprachen:

```go
	// Every file of the build, of any language: a Godot path may name a file
	// another language extracted, a C# script attached in a scene.
	paths := map[string]bool{}
	for _, f := range files {
		paths[f.Path] = true
	}
	for _, lang := range g.Meta.Languages {
		g.Edges = append(g.Edges, edgesOf(lang, groups[lang], mods, paths)...)
	}
```

`edgesOf`:

```go
func edgesOf(lang string, files []extract.Result, mods []Module, paths map[string]bool) []model.Edge {
	switch lang {
	case "python":
		return resolvePython(files)
	case "gdscript":
		return resolveGDScript(files, paths)
	}
	// … der bestehende Rest unverändert …
```

Kommentar von `edgesOf` ergänzen: „Python's in resolvePython, Godot's in resolveGDScript“. Den Satz in `Graph` „Each language is resolved against its own files alone“ ergänzen um „; a Godot path alone may land on a file of another language, by its path“.

- [ ] **Step 4: `gdscript.go` in `resolve`**

```go
package resolve

import (
	"path"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// resolveGDScript resolves the raw edges of a Godot project's files: scripts,
// scenes, resources and project.godot.
//
// project.godot is the source of what Godot itself reads from it, not a
// rule loomux rebuilds: its directory is the root res:// paths start from,
// and its [autoload] section binds global names to files. A file belongs
// to the nearest project.godot above it; a file under none has no res://
// path and no autoloads. A path without a scheme is relative to the file's
// directory. A path lands on a file of any language of the build, or gives
// no edge.
//
// Every script is a class: its class_name node, else its file node. A name
// binds, first to last: an inner class of the class the name stands in or
// of one around it; a script-level `const X = preload(...)`; a class_name of
// the project; an autoload of the project. The first stage with a candidate
// decides, and two candidates there give no edge.
//
// Calls follow GDScript: a plain name or self.name is a function of the
// class or one of its bases, and nothing else -- there are no free
// functions in other files, so a name found nowhere is the engine's or a
// builtin and gives no edge. super starts at the base. X.f() binds X; X.new()
// is X's own _init, or X. sig.emit(), connect and disconnect on a signal of
// the class reference the signal. Every edge names exactly one target, or
// there is none, and every edge is extracted.
func resolveGDScript(files []extract.Result, paths map[string]bool) []model.Edge {
	x := gdIndexOf(files, paths)
	var out []model.Edge
	var rest []extract.RawEdge
	// One edge per source, target and relation: self.f() and f() are one
	// dependency, as are a name and the alias that binds the same file.
	seen := map[model.Edge]bool{}
	keep := func(e model.Edge) {
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	for _, f := range files {
		for _, raw := range f.Edges {
			switch raw.Relation {
			case model.RelationContains:
				keep(model.Edge{Source: raw.Source, Target: raw.TargetID, Relation: model.RelationContains, Confidence: model.ConfidenceExtracted})
			case model.RelationExtends:
				if base, ok := x.base(raw); ok {
					x.bases[raw.Source] = base
					keep(gdEdge(raw, base))
				}
			default:
				rest = append(rest, raw)
			}
		}
	}
	// A call in the first file may need the base of a class in the last.
	for _, raw := range rest {
		if e, ok := x.resolve(raw); ok {
			keep(e)
		}
	}
	return out
}
```

```go
// gdIndex is every lookup the Godot resolver makes.
type gdIndex struct {
	paths    map[string]bool                          // every file of the build, any language
	roots    []string                                 // the directories of project.godot files, longest first
	kind     map[model.NodeID]model.Kind              // node -> its kind
	parent   map[model.NodeID]model.NodeID            // node -> the node that contains it
	members  map[model.NodeID]map[string][]model.Node // class or file -> name -> its definitions
	script   map[string]model.NodeID                  // .gd file -> its class: the class_name node, else the file
	aliases  map[string]map[string]string             // file -> alias -> path as written
	classes  map[string]map[string][]model.NodeID     // project root -> class_name -> its classes
	autoload map[string]map[string]string             // project root -> name -> path as written
	bases    map[model.NodeID]model.NodeID            // class -> its base, as resolveGDScript resolves it
}

func gdIndexOf(files []extract.Result, paths map[string]bool) *gdIndex {
	x := &gdIndex{
		paths: paths, kind: map[model.NodeID]model.Kind{}, parent: map[model.NodeID]model.NodeID{},
		members: map[model.NodeID]map[string][]model.Node{}, script: map[string]model.NodeID{},
		aliases: map[string]map[string]string{}, classes: map[string]map[string][]model.NodeID{},
		autoload: map[string]map[string]string{}, bases: map[model.NodeID]model.NodeID{},
	}
	for _, f := range files {
		if path.Base(f.Path) == "project.godot" {
			root := dirOf(f.Path)
			x.roots = append(x.roots, root)
			x.classes[root] = map[string][]model.NodeID{}
			x.autoload[root] = map[string]string{}
			for _, imp := range f.Imports {
				x.autoload[root][imp.Alias] = imp.Path
			}
		}
	}
	// The nearest project first: a/sub/project.godot owns a/sub/x.gd.
	sort.Slice(x.roots, func(i, j int) bool { return len(x.roots[i]) > len(x.roots[j]) })
	for _, f := range files {
		x.aliases[f.Path] = map[string]string{}
		for _, imp := range f.Imports {
			x.aliases[f.Path][imp.Alias] = imp.Path
		}
		if path.Ext(f.Path) == ".gd" {
			x.script[f.Path] = model.NodeID(f.Path)
			if f.Package != "" {
				// Minted first in its file, so it carries no ordinal.
				id := model.NodeID(f.Path + "#" + f.Package)
				x.script[f.Path] = id
				if root, ok := x.project(f.Path); ok {
					x.classes[root][f.Package] = append(x.classes[root][f.Package], id)
				}
			}
		}
		for _, raw := range f.Edges {
			if raw.Relation == model.RelationContains {
				x.parent[raw.TargetID] = raw.Source
			}
		}
		for _, n := range f.Nodes {
			x.kind[n.ID] = n.Kind
			if n.Kind == model.KindFile {
				continue
			}
			p := x.parent[n.ID]
			if x.members[p] == nil {
				x.members[p] = map[string][]model.Node{}
			}
			x.members[p][n.Name] = append(x.members[p][n.Name], n)
		}
	}
	return x
}

// project is the root of the project a file belongs to: the nearest
// project.godot above it.
func (x *gdIndex) project(file string) (string, bool) {
	for _, root := range x.roots {
		if root == "" || strings.HasPrefix(file, root+"/") {
			return root, true
		}
	}
	return "", false
}

// target is the file a path in file names: res:// from the project root, a
// path without a scheme from the file's directory. False when there is no
// project for res://, or no file of the build there -- an icon, a uid://
// or user:// path, a file outside the tree.
func (x *gdIndex) target(file, spec string) (string, bool) {
	var p string
	if rest, ok := strings.CutPrefix(spec, "res://"); ok {
		root, ok := x.project(file)
		if !ok {
			return "", false
		}
		p = path.Join(root, rest)
	} else {
		p = path.Join(dirOf(file), spec)
	}
	return p, x.paths[p]
}

// classOf is what a file stands for as a class: its script's class, or the
// file itself for anything that is no script.
func (x *gdIndex) classOf(file string) model.NodeID {
	if id, ok := x.script[file]; ok {
		return id
	}
	return model.NodeID(file)
}

// home is the class a raw edge's source stands in: a class is its own, a
// function or method the class or script around it, the file its script's.
func (x *gdIndex) home(raw extract.RawEdge) model.NodeID {
	switch {
	case raw.Source == model.NodeID(raw.File):
		return x.script[raw.File]
	case x.kind[raw.Source] == "class":
		return raw.Source
	}
	if p := x.parent[raw.Source]; p != model.NodeID(raw.File) {
		return p
	}
	return x.script[raw.File]
}

// bind is the class or file a name stands for in a class of file, by the
// four stages in order; the first stage with a candidate decides.
func (x *gdIndex) bind(file string, home model.NodeID, name string) (model.NodeID, bool) {
	for c := home; c != ""; c = x.parent[c] {
		if found := ofKind(x.members[c][name], "class"); len(found) > 0 {
			return only(found)
		}
	}
	if spec, ok := x.aliases[file][name]; ok {
		p, ok := x.target(file, spec)
		return x.classOf(p), ok
	}
	root, ok := x.project(file)
	if !ok {
		return "", false
	}
	if ids := x.classes[root][name]; len(ids) > 0 {
		if len(ids) != 1 {
			return "", false
		}
		return ids[0], true
	}
	if spec, ok := x.autoload[root][name]; ok {
		p, ok := x.target(file, spec)
		return x.classOf(p), ok
	}
	return "", false
}

// chain binds the first name of "A.B.C" and walks the rest as inner classes.
func (x *gdIndex) chain(file string, home model.NodeID, names string) (model.NodeID, bool) {
	parts := strings.Split(names, ".")
	c, ok := x.bind(file, home, parts[0])
	for _, p := range parts[1:] {
		if !ok {
			break
		}
		c, ok = only(ofKind(x.members[c][p], "class"))
	}
	return c, ok
}

// base resolves an extends edge: a path, or a name bound from the class
// around the one being defined. A class is never its own base.
func (x *gdIndex) base(raw extract.RawEdge) (model.NodeID, bool) {
	if raw.Specifier != "" {
		p, ok := x.target(raw.File, raw.Specifier)
		return x.classOf(p), ok && x.classOf(p) != raw.Source
	}
	name := raw.Name
	if raw.Receiver != "" {
		name = raw.Receiver + "." + raw.Name
	}
	t, ok := x.chain(raw.File, x.parent[raw.Source], name)
	return t, ok && t != raw.Source
}

// member is the definitions of name of the given kinds in the first class
// of class's line of bases that has any. Each class is visited once, so a
// cycle of bases ends.
func (x *gdIndex) member(class model.NodeID, name string, kinds ...model.Kind) []model.Node {
	seen := map[model.NodeID]bool{}
	for c := class; c != "" && !seen[c]; c = x.bases[c] {
		seen[c] = true
		if found := ofKind(x.members[c][name], kinds...); len(found) > 0 {
			return found
		}
	}
	return nil
}

// resolve resolves an imports, references or calls edge.
func (x *gdIndex) resolve(raw extract.RawEdge) (model.Edge, bool) {
	switch raw.Relation {
	case model.RelationImports:
		p, ok := x.target(raw.File, raw.Specifier)
		return gdEdge(raw, model.NodeID(p)), ok && p != raw.File
	case model.RelationReferences:
		t, ok := x.bind(raw.File, x.home(raw), raw.Name)
		return gdEdge(raw, t), ok && !x.encloses(t, raw.Source)
	default:
		return x.call(raw)
	}
}

// encloses is whether t is src or a definition around it: a name for the
// class one stands in references nothing new.
func (x *gdIndex) encloses(t, src model.NodeID) bool {
	for c := src; c != ""; c = x.parent[c] {
		if c == t {
			return true
		}
	}
	return false
}

// call resolves one call by its shape (see resolveGDScript).
func (x *gdIndex) call(raw extract.RawEdge) (model.Edge, bool) {
	home := x.home(raw)
	fns := []model.Kind{"method", "function"}
	switch {
	case raw.Receiver == "":
		return one(raw, x.member(home, raw.Name, fns...), model.ConfidenceExtracted)
	case raw.Receiver == "super":
		return one(raw, x.member(x.bases[home], raw.Name, fns...), model.ConfidenceExtracted)
	}
	if signalVerb(raw.Name) {
		if sig := x.member(home, raw.Receiver, "signal"); len(sig) > 0 {
			e, ok := one(raw, sig, model.ConfidenceExtracted)
			e.Relation = model.RelationReferences
			return e, ok
		}
	}
	class, ok := x.chain(raw.File, home, raw.Receiver)
	if !ok {
		return model.Edge{}, false
	}
	if raw.Name == "new" {
		if inits := ofKind(x.members[class]["_init"], fns...); len(inits) == 1 {
			return gdEdge(raw, inits[0].ID), true
		}
		return gdEdge(raw, class), true
	}
	return one(raw, x.member(class, raw.Name, fns...), model.ConfidenceExtracted)
}

// signalVerb is whether a member call on a signal uses the signal itself.
func signalVerb(name string) bool {
	return name == "emit" || name == "connect" || name == "disconnect"
}

// ofKind keeps the nodes of the given kinds.
func ofKind(nodes []model.Node, kinds ...model.Kind) []model.Node {
	var out []model.Node
	for _, n := range nodes {
		for _, k := range kinds {
			if n.Kind == k {
				out = append(out, n)
			}
		}
	}
	return out
}

// only is the one node's id, or false for none or several.
func only(nodes []model.Node) (model.NodeID, bool) {
	if len(nodes) != 1 {
		return "", false
	}
	return nodes[0].ID, true
}

// gdEdge is the extracted edge of raw's relation from raw's source to t.
func gdEdge(raw extract.RawEdge, t model.NodeID) model.Edge {
	return model.Edge{Source: raw.Source, Target: t, Relation: raw.Relation, Confidence: model.ConfidenceExtracted}
}
```

Hinweise zur Umsetzung:
- `x.member(x.bases[home], …)` bei einem `home` ohne Basis bekommt `""`; die Schleife läuft dann nicht, und es entsteht keine Kante.
- Bei `case model.RelationReferences` liefert `bind` für einen leeren Namen nichts, weil kein Mitglied, Alias, `class_name` oder Autoload `""` heißt.
- `only`, `ofKind`, `gdEdge` und `signalVerb` gibt es in `resolve` noch nicht (gegreppt beim Planen).

- [ ] **Step 5: Grün, Coverage 100 % auf `resolve`**

```bash
go test -timeout 120s -coverprofile=<scratchpad>/t4.cov ./internal/code/resolve/ > <scratchpad>/t4-green.txt 2>&1; echo $?
go tool cover -func=<scratchpad>/t4.cov | grep -v '100.0%'
```

Jeden offenen Zweig mit einer Eingabe in den fünf Tests schließen. Ein Beispiel ist `home` für eine Kante aus einer inneren Klasse heraus.

- [ ] **Step 6: Mutationsrunde**

Boden:
1. `bind`: Stufe 2 vor Stufe 1 ziehen.
2. `bind`: Stufe 4 vor Stufe 3 ziehen.
3. `bind`: `len(ids) != 1` → `len(ids) == 0`.
4. `project`: die Sortierung der Wurzeln streichen.
5. `member`: `!seen[c]` streichen. Mit `-timeout 30s`; eine Zeitüberschreitung gilt hier als getötet.
6. `resolve`: `&& p != raw.File` streichen.
7. `resolve`: `&& !x.encloses(…)` streichen.
8. `call`: `_init` per `x.member` statt `x.members` suchen, also geerbt.
9. `call`: `signalVerb` → `true`.
10. `base`: `&& t != raw.Source` streichen.
11. `target`: `path.Join(dirOf(file), spec)` → `path.Join("", spec)`.
12. `home`: den `class`-Fall streichen.

- [ ] **Step 7: Commit**

```
feat(graph): resolve Godot paths, class names and autoloads

project.godot gives the res:// root and the autoloads, and a path may land
on a file of any language. A name binds through the inner classes around
it, then a script's preload aliases, the project's class_name and its
autoloads; calls follow the line of bases, super, X.new() and signals.
```

---

### Task 5: Einhängen und Umgebung

**Files:**
- Modify: `internal/code/extract/all/all.go` und `all_test.go`
- Modify: `internal/code/sourceset/sourceset.go` (`extensions`) und der Test, der die Liste prüft (per `grep -n 'Extensions()' internal/code/sourceset/*_test.go`)
- Modify: `internal/code/blast/radius.go` (`IsTestPath`), `radius_test.go:335`
- Modify: `internal/code/blast/resolve.go` (`isFileQuery`) und dessen Test (per `grep -n 'isFileQuery\|IsFileQuery' internal/code/blast/*_test.go`)
- Modify: `internal/code/repomap/map.go` (`langFromPath`) und dessen Test
- Modify: `internal/cli/graph.go` (`report`) und dessen Test
- Modify: `internal/verify/presets.toml`, `internal/verify/presets_test.go:204-215`
- Modify: `internal/code/query/build_test.go` (Ende-zu-Ende)

**Interfaces:**
- Consumes: `gdscript.Language{}` aus Task 3.
- Produces:
  - `all.Languages() == {golang, python, gdscript}`.
  - `sourceset.Extensions() == [".gd", ".go", ".godot", ".py", ".tres", ".tscn"]`.
  - `blast.IsTestPath` kennt `.gd`.
  - Die Zeile von `graph build` nennt `references` wie heute `extends`, nur wenn es welche gibt.

- [ ] **Step 1: Failing tests**

`all_test.go`:
- `TestLanguagesPutGoFirst`: Erwartung `[]string{"go", "python", "gdscript"}`.
- `TestVersionIsSortedJoin`: die zweite Erwartung wird
  ```go
  "gdscript/1@"+treesitter.Parser+"+"+golang.Version+"+python/1@"+treesitter.Parser
  ```
  Der Kommentar dazu bekommt den Satz: „gdscript sorts before go“.
- `TestForPicksByExtension`: in die Map `"g/a.gd": "gdscript", "g/project.godot": "gdscript", "g/x.tscn": "gdscript", "g/x.tres": "gdscript"`; in die Negativliste `"g/A.GD"`, `"g/x.gd.uid"`, `"g/x.import"`.
- `TestExtensionsMatchSourceset`: Erwartung `[]string{".gd", ".go", ".godot", ".py", ".tres", ".tscn"}`.

`radius_test.go`, in die Map von `TestIsTestPathPerLanguage`:

```go
		"test/a_test.gd": true, "x/b_test.gd": true, "godot/test/unit/helper.gd": true, "tests/x.gd": true,
		"core/a.gd": false, "core/testing.gd": false, "contest/x.gd": false, "core/a_test.tscn": false,
```

`presets_test.go`: `TestThePythonGraphLaneIsTheGoOne` umbenennen in `TestTreeSitterGraphLanesAreTheGoOne` und über `[]string{"python", "gdscript"}` laufen lassen:

```go
// The graph belongs to the root, not to a stack, so the graph lanes of the
// tree-sitter languages run the very commands of Go's; the plan keeps one of
// them per run.
func TestTreeSitterGraphLanesAreTheGoOne(t *testing.T) {
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	goLane := p.Stacks["go"].Lanes["graph"]
	for _, stack := range []string{"python", "gdscript"} {
		lane := p.Stacks[stack].Lanes["graph"]
		if len(lane.Commands) == 0 || !slices.Equal(lane.Commands, goLane.Commands) || lane.OnFile != nil || lane.Threaded {
			t.Errorf("%s %+v, go %+v", stack, lane, goLane)
		}
	}
}
```

`internal/code/query/build_test.go`, Ende-zu-Ende:

```go
// A Godot project in a subdirectory: the build reads every file of it, the
// scene reaches its script and the script its autoload through project.godot.
func TestBuildReadsAGodotProject(t *testing.T) {
	root := repo(t, map[string]string{
		"godot/project.godot":       "[autoload]\nGame=\"*res://autoload/game.gd\"\n",
		"godot/autoload/game.gd":    "extends Node\n\nfunc start():\n\tpass\n",
		"godot/main.tscn":           "[gd_scene format=3]\n\n[ext_resource type=\"Script\" path=\"res://main.gd\" id=\"1\"]\n",
		"godot/main.gd":             "class_name Main\nextends Node\n\nfunc _ready():\n\tGame.start()\n",
		"godot/autoload/game.gd.uid": "uid://abc\n",
	})
	g, stats, err := Build(root, ignore)
	if err != nil {
		t.Fatal(err)
	}
	if ls := stats.PerLanguage["gdscript"]; ls.Files != 4 || ls.Parsed != 4 {
		t.Fatalf("gdscript stats %+v, want 4 files parsed (the .uid is no source)", ls)
	}
	want := map[string]bool{
		"godot/main.tscn imports godot/main.gd":                        false,
		"godot/project.godot imports godot/autoload/game.gd":           false,
		"godot/main.gd#Main._ready calls godot/autoload/game.gd#start": false,
	}
	for _, e := range g.Edges {
		k := string(e.Source) + " " + string(e.Relation) + " " + string(e.Target)
		if _, ok := want[k]; ok {
			want[k] = true
		}
	}
	for k, found := range want {
		if !found {
			t.Errorf("no edge %s", k)
		}
	}
}
```

Für `report` (`internal/cli/graph.go`): den bestehenden Test von `report` per `grep -n 'func Test.*Report\|report(' internal/cli/*_test.go` finden. Dort einen Fall ergänzen, in dem ein Graph mit zwei `references`-Kanten `, 2 references)` in der ersten Zeile zeigt, ein Graph ohne keine.

Für `isFileQuery` und `langFromPath` je einen Fall `"main.gd"` ergänzen: `isFileQuery` → `true`, `langFromPath` → `"gdscript"`. Dazu `"main.tscn"` → `"gdscript"`.

- [ ] **Step 2: Run, expect FAIL**

```bash
go test ./internal/code/... ./internal/verify/ ./internal/cli/ > <scratchpad>/t5-red.txt 2>&1
```

- [ ] **Step 3: Umsetzen**

`all.go`:

```go
// Languages is every extractor a build runs, Go first.
func Languages() []extract.Language {
	return []extract.Language{golang.Language{}, python.Language{}, gdscript.Language{}}
}
```

Import `"github.com/xidus90/loomux/internal/code/extract/gdscript"`.

`sourceset.go`:

```go
var extensions = []string{".gd", ".go", ".godot", ".py", ".tres", ".tscn"}
```

Kommentar darüber ergänzen: „.godot, .tres and .tscn are the project file, resources and scenes of a Godot project, read with its scripts“.

`radius.go`, in `IsTestPath`:

```go
	case ".gd":
		// gdUnit4 looks for suites under test/ by default; a project that
		// moves its test_lookup_folder in project.godot is not followed, this
		// predicate sees a path and no project.
		return strings.HasSuffix(p, "_test.gd") || underDir(p, "test") || underDir(p, "tests")
```

`blast/resolve.go`, `isFileQuery`: `".gd", ".godot", ".tres", ".tscn"` an die Liste der `case` anhängen.

`repomap/map.go`, `langFromPath`:

```go
	case ".gd", ".godot", ".tres", ".tscn":
		return "gdscript"
```

`internal/cli/graph.go`, `report`: den Kommentar „Only Python has base classes …“ ersetzen durch „Base classes and references come from the tree-sitter languages; a graph without them reads as the Go graph's report always did.“ Und:

```go
	extends := ""
	if n := byRelation[model.RelationExtends]; n > 0 {
		extends = fmt.Sprintf(", %d extends", n)
	}
	if n := byRelation[model.RelationReferences]; n > 0 {
		extends += fmt.Sprintf(", %d references", n)
	}
```

`presets.toml`, direkt hinter `[stack.gdscript.lint]`:

```toml
# The graph belongs to the root; with Go or Python beside it, one of their
# lanes carries it and this one stands aside.
[stack.gdscript.graph]
commands = ["{loomux} check graph-fresh", "{loomux} check blast-audit --cached --threshold 5"]
```

- [ ] **Step 4: Ganzes Tor**

```bash
go test ./... > <scratchpad>/t5-all.txt 2>&1; echo $?
grep -n -- '--- FAIL' <scratchpad>/t5-all.txt
```

Ein roter Replay-Fall unter `internal/cases` mit einer neuen Lane `graph/gdscript` ist ein Befund für den Controller und kein Anlass, Aufnahmen von Hand zu ändern. Vorab gegreppt: In `testdata/cases` kommt `gdscript` nicht vor.

- [ ] **Step 5: Mutationsrunde**

Boden:
1. `IsTestPath`: `|| underDir(p, "tests")` streichen.
2. `report`: `> 0` → `>= 0`.
3. `langFromPath`: den `.tscn`-Eintrag streichen.

- [ ] **Step 6: Commit**

```
feat(graph): build the code graph of Godot projects

The GDScript extractor joins Go and Python, and the walk takes scripts,
scenes, resources and project.godot. A gdUnit4 suite counts as a test
for blast, and a Godot project's graph lane runs graph-fresh and
blast-audit like Go's and Python's.
```

---

### Task 6: Doku, Roadmap, Messung

**Files:**
- Modify: `README.md`, `README.de.md` (Zeilen 104, 156–158, 292/299)
- Modify: `docs/en/architecture.md:140-145`, `docs/de/architecture.md:~144`
- Modify: `docs/en/cli-reference.md:~483-492`, `docs/de/cli-reference.md:~506-515`
- Modify: `docs/en/configuration.md:~680-686`, `docs/de/configuration.md:~701-707`
- Modify: `docs/wiki/topics/code-graph.md:~40-55`
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md` (neuer Eintrag am Ende)

- [ ] **Step 1: READMEs**

- Zeile 104 (en): „The graph covers Go, read with `go/parser`, and Python, read on `gotreesitter` …“ wird zu
  > The graph covers Go, read with `go/parser`, and Python and Godot projects — GDScript, scenes, resources and `project.godot` — read on `gotreesitter`, a Tree-sitter runtime in pure Go, so the binary stays CGo-free.

  Der Rest des Satzes bleibt. Deutsch entsprechend:
  > Der Graph umfasst Go, gelesen mit `go/parser`, sowie Python und Godot-Projekte — GDScript, Szenen, Ressourcen und `project.godot` —, gelesen auf `gotreesitter` …
- Synopsis von `graph build` (en 292, de 299): „extract Go and Python“ → „extract Go, Python and Godot projects“, deutsch „Extrahiert Go, Python und Godot-Projekte“.
- Roadmap:
  - Die Zeile **GDScript in the code graph** / **GDScript im Code-Graphen** löschen.
  - In der C++-Zeile die Spalte „Depends on“ von `G5c` auf `G5c ✅` setzen.
  - Neue Zeile unter „Coming“, hinter der TypeScript-Zeile (en):
    > `| **Python source roots from a project file** | The Python resolver guesses where absolute imports start (the repository root, the directory above each top-level package, every `manage.py` directory); where a project file states the roots, it should be read instead. Whether `pyproject.toml` does is open: it states them only in tool-specific forms, if at all | G5a follow-up | — | — |`

    Deutsch:
    > `| **Python-Quellwurzeln aus einer Projektdatei** | Der Python-Resolver errät, wo absolute Importe beginnen (Repository-Wurzel, der Ordner über jedem Top-Level-Paket, jeder Ordner mit `manage.py`); wo eine Projektdatei die Wurzeln festlegt, soll sie gelesen werden. Ob `pyproject.toml` das tut, ist offen: nur in werkzeugeigenen Formen, wenn überhaupt | G5a-Folge | — | — |`

- [ ] **Step 2: architecture, cli-reference, configuration, Wiki**

- `architecture.md` (en): „`extract/python` on the shared tree-sitter core …“ → „`extract/python` and `extract/gdscript` on the shared tree-sitter core …“; de entsprechend.
- `cli-reference.md` (en), Abschnitt `graph build`:
  - Endungen: „`.go` and `.py`, matched exactly …, `.pyi` stubs left out“ → „`.go`, `.py`, and a Godot project's `.gd`, `.tscn`, `.tres` and `.godot`, matched exactly and case-sensitively, `.pyi` stubs left out“.
  - „(Go with `go/parser`, Python on `gotreesitter`)“ → „(Go with `go/parser`, Python and GDScript on `gotreesitter`)“.
  - Den Satz „Each language resolves in a name index of its own, so no edge crosses languages.“ ersetzen durch:
    > Each language resolves in a name index of its own; the one edge that crosses languages is a Godot path (`res://` or relative) that names a file another language extracted. A Godot project's `res://` root is the directory of the nearest `project.godot`, whose `[autoload]` section binds global names; without one there are no `res://` edges.
  - Output: „(`extends`, a Python class's base classes, …)“ → „(`extends`, the base classes of Python and GDScript classes, and `references`, the classes, autoloads and signals a GDScript definition names, each only when there are any, so a Go graph's line has neither)“; die Formzeile: `N imports[, N extends][, N references])`.
  - Parse errors: „a Python file with syntax errors“ → „a Python or Godot file with syntax errors“.
  - Exit codes: „also when Python files have parse errors“ → „also when Python or Godot files have parse errors“.
  - de entsprechend, mit denselben Inhalten.
- `configuration.md` (en): im Spiegelpunkt „`internal/code/extract/python` reads `.py` files“ → „`internal/code/extract/python` reads `.py` files and `internal/code/extract/gdscript` a Godot project's `.gd`, `.tscn`, `.tres` and `.godot` files“. Dazu den Abschnitt über Graph-Lanes (en ~365–440) per `grep -n 'graph/python' docs/en/configuration.md` prüfen: Wo „Go and Python“ als Träger der Graph-Lane steht, GDScript nennen. de entsprechend.
- `docs/wiki/topics/code-graph.md`: hinter dem Python-Punkt einen Punkt einfügen:
  > - **Godot-Projekte** liest er seit G5c auf demselben Kern (`internal/code/extract/gdscript`): GDScript, Szenen (`.tscn`), Ressourcen (`.tres`) und `project.godot`, als eine Sprache, damit eine Szene ihr Skript erreicht. `project.godot` legt die `res://`-Wurzel und die Autoloads fest; ohne es gibt es keine `res://`-Kante. Ein Skript mit `class_name` ist ein Klassenknoten über die ganze Datei.

  Den Satz „Jede Sprache löst in einem eigenen Namensindex auf; Kanten über Sprachgrenzen gibt es nicht.“ ersetzen durch:
  > Jede Sprache löst in einem eigenen Namensindex auf. Über eine Sprachgrenze geht nur ein Godot-Pfad, der eine Datei einer anderen Sprache nennt, etwa ein C#-Skript in einer Szene.

  Frontmatter-Datum und Quellen-Pins der Seite nach den Regeln des Wikis nachziehen (`loomux check lint` meldet, was fehlt).
- Danach im ganzen Repo außer `docs/.superpowers` nach altem Verhalten greppen und jede Fundstelle prüfen:
  ```bash
  grep -rn -i 'no edge crosses\|keine Kante überquert\|Kanten über Sprachgrenzen\|Go and Python\|Go und Python\|`.go` and `.py`\|`.go` und `.py`' --include=*.md . | grep -v '^./docs/.superpowers\|^./.claude'
  ```

- [ ] **Step 3: Messung in space**

Das Binary dieses Zweigs bauen, `go build -o <scratchpad>/loomux-g5c.exe ./cmd/loomux`, dann in `C:/Users/micro/Documents/#GIT/space`:

```bash
<scratchpad>/loomux-g5c.exe graph build --root "C:/Users/micro/Documents/#GIT/space" --no-reuse > <scratchpad>/space-cold.txt 2>&1
<scratchpad>/loomux-g5c.exe graph build --root "C:/Users/micro/Documents/#GIT/space" > <scratchpad>/space-warm.txt 2>&1
<scratchpad>/loomux-g5c.exe graph stats --root "C:/Users/micro/Documents/#GIT/space" > <scratchpad>/space-stats.txt 2>&1
```

Achtung: Das schreibt `.loomux/state/graph/` in space. Das ist Maschinenzustand und git-ignoriert. Vorher `git -C "C:/Users/micro/Documents/#GIT/space" status --short` festhalten und danach vergleichen.

Eintrag in `docs/en/benchmarks.md` und `docs/de/benchmarks.md` am Ende, im Format der bestehenden Einträge:
- Überschrift `## 2026-10-0X HH:MM — GDScript Extraction: graph build on space`.
- Zweig und Commit, Befehle, Rohausgabe beider Läufe (kalt und warm), Dateien je Sprache, Knoten, Kanten je Relation.
- Vergleich: vorher 0 Dateien (Spec, Ziel).

Weicht etwas vom Ziel der Spec ab, ist das ein Befund für den Controller und keine Doku-Zeile: 0 Kanten einer Art, `parse errors` > 0 oder ein Abbruch.

- [ ] **Step 4: Tor und Commit**

Zwei Commits: Die Messung ist kein Code.

```
docs: describe the code graph of Godot projects

The extractor list, the build's output and file set, the one edge that
crosses languages, and the roadmap: GDScript leaves it, C++ now depends
on a finished stage, and reading Python's source roots from a project
file joins it.
```

```
docs(benchmarks): measure graph build on a Godot project
```

---

## Abschluss

- Gesamtreview des Zweigs gegen die Spec.
- `release-pr`-Skill für Gruppierung, Label (`release:minor`, `feat`) und Changelog.
- Der Push-Befehl geht an den Nutzer.
- Vor dem PR `git fetch` und `git log HEAD..origin/master --oneline` lesen. `feat/gate-skip-lock-godot` (andere Sitzung) ändert womöglich `presets.toml` und die gdscript-Lanes; nach deren Merge rebasen und `TestTreeSitterGraphLanesAreTheGoOne` sowie den Preset-Block gegen deren Stand prüfen.
