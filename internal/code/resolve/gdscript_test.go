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
		"p/project.godot":  "[autoload]\nTool=\"*res://tool_auto.gd\"\nTwin=\"*res://twin_auto.gd\"\n",
		"p/tool_auto.gd":   "func use():\n\tpass\n",
		"p/twin_auto.gd":   "static func make():\n\tpass\n",
		"p/tool_class.gd":  "class_name Tool\n\nstatic func use():\n\tpass\n",
		"p/widget.gd":      "class_name Widget\n\nstatic func use():\n\tpass\n",
		"p/gadget.gd":      "class_name Gadget\n\nstatic func use():\n\tpass\n",
		"p/gadget_impl.gd": "static func use():\n\tpass\n",
		"p/twin_a.gd":      "class_name Twin\n\nstatic func make():\n\tpass\n",
		"p/twin_b.gd":      "class_name Twin\n\nstatic func make():\n\tpass\n",
		"p/outer.gd":       "class_name Outer\n\nclass Inner:\n\tstatic func go():\n\t\tpass\n",
		"p/lost.gd":        "const Tool = load(\"uid://x\")\n\nfunc b():\n\tTool.use()\n",
		"p/user.gd":        "const Gadget = preload(\"res://gadget_impl.gd\")\n\nfunc a():\n\tTool.use()\n\tWidget.use()\n\tGadget.use()\n\tTwin.make()\n\tOuter.Inner.go()\n\tOuter.Nope.go()\n\nclass Widget:\n\tstatic func use():\n\t\tpass\n",
	})
	wantCalls(t, g, "p/user.gd#a",
		"p/gadget_impl.gd#use extracted", "p/outer.gd#Outer.Inner.go extracted",
		"p/tool_class.gd#Tool.use extracted", "p/user.gd#Widget.use extracted")
	// An alias whose target is not in the build still decides: the name does
	// not fall through to the class_name or the autoload of the same name.
	wantCalls(t, g, "p/lost.gd#b")
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

// A class_name and an autoload belong to their project alone: a sibling
// project does not see them, a nested project does not see the one around it,
// and a same-named class_name binds in each project to its own.
func TestGodotClassNamesAndAutoloadsStayInTheirProject(t *testing.T) {
	const calls = "func a():\n\tOnly.use()\n\tSvc.use()\n\tTwin.make()\n"
	g := gdGraph(t, map[string]string{
		"a/project.godot":     "[autoload]\nSvc=\"*res://svc.gd\"\n",
		"a/only.gd":           "class_name Only\n\nstatic func use():\n\tpass\n",
		"a/svc.gd":            "static func use():\n\tpass\n",
		"a/twin.gd":           "class_name Twin\n\nstatic func make():\n\tpass\n",
		"a/user.gd":           calls,
		"a/sub/project.godot": "config_version=5\n",
		"a/sub/twin.gd":       "class_name Twin\n\nstatic func make():\n\tpass\n",
		"a/sub/user.gd":       calls,
		"b/project.godot":     "config_version=5\n",
		"b/user.gd":           calls,
		// The same path as the autoload's, so a leaked autoload name would
		// resolve to a file here instead of ending at a missing one.
		"a/sub/svc.gd": "static func use():\n\tpass\n",
		"b/svc.gd":     "static func use():\n\tpass\n",
	})
	wantCalls(t, g, "a/user.gd#a",
		"a/only.gd#Only.use extracted", "a/svc.gd#use extracted", "a/twin.gd#Twin.make extracted")
	wantCalls(t, g, "a/sub/user.gd#a", "a/sub/twin.gd#Twin.make extracted")
	wantCalls(t, g, "b/user.gd#a")
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

func TestGodotInnerClassesScenesAndFilesOutsideAnyProject(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"q/project.godot": "config_version=5\n",
		"q/helper.gd":     "class_name Helper\n",
		"q/outer.gd":      "class_name Outer\n\nclass Inner:\n\tvar h: Helper\n",
		"q/sub.gd":        "extends Outer.Inner\n",
		"q/user.gd":       "const Scn = preload(\"res://s.tscn\")\n\nfunc f(s: Scn):\n\tpass\n",
		"q/s.tscn":        "[gd_scene format=3]\n",
		"loose/y.gd":      "var a: Nope\n\nfunc f():\n\tNope.Deep.go()\n",
	})
	wantTargets(t, g, "q/outer.gd#Outer.Inner", model.RelationReferences, "q/helper.gd#Helper")
	wantTargets(t, g, "q/sub.gd", model.RelationExtends, "q/outer.gd#Outer.Inner")
	wantTargets(t, g, "q/user.gd#f", model.RelationReferences, "q/s.tscn")
	wantTargets(t, g, "loose/y.gd", model.RelationReferences)
	wantCalls(t, g, "loose/y.gd#f")
}

func TestGraphDropsARelationGoDoesNotResolve(t *testing.T) {
	g := validated(t, resolve.Graph([]extract.Result{result("a/a.go",
		[]model.Node{fileNode("a/a.go")},
		[]extract.RawEdge{{Source: "a/a.go", Relation: model.RelationReferences, Name: "X", File: "a/a.go"}})}, nil, stamp))
	wantTargets(t, g, "a/a.go", model.RelationReferences)
}

// An _init node is where X.new() lands, and only its own: one declared in a
// base does not count.
func TestGodotNewTargetsTheClassesOwnInit(t *testing.T) {
	proj, err := gdscript.File("s/project.godot", "config_version=5\n")
	if err != nil {
		t.Fatal(err)
	}
	class := func(name string, base string, withInit bool) extract.Result {
		file := "s/" + name + ".gd"
		cls := model.NodeID(file + "#" + name)
		r := extract.Result{Path: file, Language: "gdscript", Package: name,
			Nodes: []model.Node{
				{ID: model.NodeID(file), Name: name + ".gd", Kind: model.KindFile, Path: file, BodyHash: "h"},
				{ID: cls, Name: name, Kind: "class", Path: file, BodyHash: "h"},
			},
			Edges: []extract.RawEdge{{Source: model.NodeID(file), Relation: model.RelationContains, TargetID: cls, File: file}},
		}
		if withInit {
			id := model.NodeID(string(cls) + "._init")
			r.Nodes = append(r.Nodes, model.Node{ID: id, Name: "_init", Kind: "method", Path: file, BodyHash: "h"})
			r.Edges = append(r.Edges, extract.RawEdge{Source: cls, Relation: model.RelationContains, TargetID: id, File: file})
		}
		if base != "" {
			r.Edges = append(r.Edges, extract.RawEdge{Source: cls, Relation: model.RelationExtends, Name: base, File: file})
		}
		return r
	}
	user := class("User", "", false)
	userGo := model.NodeID("s/User.gd#User.go")
	user.Nodes = append(user.Nodes, model.Node{ID: userGo, Name: "go", Kind: "method", Path: "s/User.gd", BodyHash: "h"})
	user.Edges = append(user.Edges,
		extract.RawEdge{Source: "s/User.gd#User", Relation: model.RelationContains, TargetID: userGo, File: "s/User.gd"},
		extract.RawEdge{Source: userGo, Relation: model.RelationCalls, Name: "new", Receiver: "Own", File: "s/User.gd"},
		extract.RawEdge{Source: userGo, Relation: model.RelationCalls, Name: "new", Receiver: "Child", File: "s/User.gd"})
	// Two constructors are no one constructor: the class itself is the target.
	dup := class("Dup", "", true)
	dup.Nodes = append(dup.Nodes, model.Node{ID: "s/Dup.gd#Dup._init#2", Name: "_init", Kind: "method", Path: "s/Dup.gd", BodyHash: "h"})
	dup.Edges = append(dup.Edges, extract.RawEdge{Source: "s/Dup.gd#Dup", Relation: model.RelationContains, TargetID: "s/Dup.gd#Dup._init#2", File: "s/Dup.gd"})
	user.Edges = append(user.Edges,
		extract.RawEdge{Source: userGo, Relation: model.RelationCalls, Name: "new", Receiver: "Dup", File: "s/User.gd"})
	g := validated(t, resolve.Graph([]extract.Result{proj, class("Own", "", true), class("Child", "Own", false), dup, user}, nil, stamp))
	wantCalls(t, g, userGo, "s/Child.gd#Child extracted", "s/Dup.gd#Dup extracted", "s/Own.gd#Own._init extracted")
}

func TestGodotBindingOrderInnerClassesAndSignalVerbs(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"b/project.godot": "config_version=5\n",
		"b/target.gd":     "static func use():\n\tpass\n",
		"b/u.gd":          "const Both = preload(\"res://target.gd\")\n\nfunc a():\n\tBoth.use()\n\nclass Both:\n\tstatic func use():\n\t\tpass\n",
		"b/host.gd":       "class_name Host\n\nclass A:\n\tclass B:\n\t\tpass\n\tvar x: B\n",
		"b/nc.gd":         "var a: Inner\n\nclass Inner:\n\tpass\n",
		"b/selfie.gd":     "class_name Selfie\nextends \"res://selfie.gd\"\n",
		"b/lib.gd":        "class_name Lib\n\nstatic func connect():\n\tpass\n",
		"b/sig.gd":        "class_name Sig\n\nsignal hit\n\nfunc off(f):\n\thit.disconnect(f)\n\nfunc on(f):\n\thit.connect(f)\n\nfunc via_lib():\n\tLib.connect()\n",
	})
	wantCalls(t, g, "b/u.gd#a", "b/u.gd#Both.use extracted")
	wantTargets(t, g, "b/host.gd#Host.A", model.RelationReferences, "b/host.gd#Host.A.B")
	wantTargets(t, g, "b/nc.gd", model.RelationReferences, "b/nc.gd#Inner")
	wantTargets(t, g, "b/selfie.gd#Selfie", model.RelationExtends)
	wantTargets(t, g, "b/sig.gd#Sig.off", model.RelationReferences, "b/sig.gd#Sig.hit")
	wantTargets(t, g, "b/sig.gd#Sig.on", model.RelationReferences, "b/sig.gd#Sig.hit")
	wantCalls(t, g, "b/sig.gd#Sig.via_lib", "b/lib.gd#Lib.connect extracted")
}

func TestGodotSignalVerbsOnASignalOfABoundClass(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"s/project.godot": "[autoload]\nBus=\"*res://bus.gd\"\n",
		"s/bus.gd":        "signal ping(n: int)\n\nfunc plain():\n\tpass\n",
		"s/outer.gd":      "class_name Outer\n\nclass Inner:\n\tsignal sig\n",
		"s/kid.gd":        "class_name Kid\nextends \"res://bus.gd\"\n",
		"s/amb.gd":        "class_name Amb\n\nsignal dup\nsignal dup\n",
		"s/user.gd": "func a():\n\tBus.ping.emit(1)\n\nfunc b(cb):\n\tBus.ping.connect(cb)\n\nfunc c(cb):\n\tBus.ping.disconnect(cb)\n\n" +
			"func d():\n\tOuter.Inner.sig.emit()\n\nfunc e():\n\tKid.ping.emit(1)\n\n" +
			"func f(cb):\n\tBus.ping.is_connected(cb)\n\nfunc g():\n\tBus.nosig.emit()\n\nfunc h():\n\tBus.plain.emit()\n\n" +
			"func i():\n\tunknown.ping.emit()\n\nfunc j():\n\tAmb.dup.emit()\n",
	})
	// The receiver's first name is a reference of its own (the class); the
	// signal is the extra edge, and only the extra edge is under test.
	for _, fn := range []string{"a", "b", "c"} {
		wantTargets(t, g, model.NodeID("s/user.gd#"+fn), model.RelationReferences, "s/bus.gd", "s/bus.gd#ping")
	}
	wantTargets(t, g, "s/user.gd#d", model.RelationReferences, "s/outer.gd#Outer", "s/outer.gd#Outer.Inner.sig")
	wantTargets(t, g, "s/user.gd#e", model.RelationReferences, "s/bus.gd#ping", "s/kid.gd#Kid")
	for _, fn := range []string{"f", "g", "h"} {
		wantTargets(t, g, model.NodeID("s/user.gd#"+fn), model.RelationReferences, "s/bus.gd")
	}
	wantTargets(t, g, "s/user.gd#i", model.RelationReferences)
	// Two signals of one name: no edge to either.
	wantTargets(t, g, "s/user.gd#j", model.RelationReferences, "s/amb.gd#Amb")
}

// A build with one Godot file still sees the files of other languages.
func TestGodotAloneFileStillReachesAForeignFile(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"g/s.gd": "const P = preload(\"../p.cs\")\n",
	}, "p.cs")
	wantTargets(t, g, "g/s.gd", model.RelationImports, "p.cs")
}

// An autoload path without a scheme is relative to project.godot, not to the
// script that uses the name.
func TestGodotAnAutoloadPathWithoutASchemeStartsAtTheProject(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"r/project.godot": "[autoload]\nRel=\"*bus2.gd\"\n",
		"r/bus2.gd":       "func f():\n\tpass\n",
		"r/ui/bus2.gd":    "func f():\n\tpass\n",
		"r/ui/user.gd":    "func a():\n\tRel.f()\n",
	})
	wantCalls(t, g, "r/ui/user.gd#a", "r/bus2.gd#f extracted")
}

// A signal name that several declarations hold is no signal hit: the call
// goes on as a call on the receiver chain, here to the inner class of that
// name. Of the class's own signals, two of a name still give no edge.
func TestGodotSeveralSignalsOfOneNameOnABoundClassFallThrough(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"d/project.godot": "config_version=5\n",
		"d/outer.gd":      "class_name Outer\n\nsignal dup\nsignal dup\n\nclass dup:\n\tstatic func emit():\n\t\tpass\n",
		"d/user.gd":       "func a():\n\tOuter.dup.emit()\n",
	})
	wantCalls(t, g, "d/user.gd#a", "d/outer.gd#Outer.dup.emit extracted")
}

func TestGodotAProjectAtTheRepoRoot(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"project.godot": "config_version=5\n",
		"main.gd":       "const Y = preload(\"res://y.gd\")\n",
		"y.gd":          "",
	})
	wantTargets(t, g, "main.gd", model.RelationImports, "y.gd")
}
