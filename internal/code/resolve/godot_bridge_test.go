package resolve_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/gdscript"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// godotWith extracts a Godot tree with the real extractor and adds results
// of other languages as a future C# or C++ extractor would hand them over.
func godotWith(t *testing.T, tree map[string]string, foreign ...extract.Result) (*model.Graph, []resolve.Unresolved) {
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
	g, un := resolve.GraphWith(append(files, foreign...), nil, stamp)
	return validated(t, g), un
}

// native is a file of another language offering one definition under a
// Godot namespace.
func native(lang, file, name string, kind model.Kind, ns extract.Namespace) extract.Result {
	n := model.Node{ID: model.NodeID(file + "#" + name), Name: name, Kind: kind, Path: file, BodyHash: "h"}
	return extract.Result{
		Path: file, Language: lang,
		Nodes:   []model.Node{{ID: model.NodeID(file), Name: file, Kind: model.KindFile, Path: file, BodyHash: "h"}, n},
		Edges:   []extract.RawEdge{{Source: model.NodeID(file), Relation: model.RelationContains, TargetID: n.ID, File: file}},
		Exports: []extract.Export{{Namespace: ns, Name: name, Target: n.ID}},
	}
}

func cppClass(file, name string) extract.Result {
	return native("cpp", file, name, "class", extract.NamespaceGodotClass)
}

func wantEdge(t *testing.T, g *model.Graph, from, to model.NodeID, rel model.Relation) {
	t.Helper()
	e := edgeBetween(g, from, to, rel)
	if e == nil || e.Confidence != model.ConfidenceExtracted {
		t.Errorf("%s -%s-> %s = %+v, want an extracted edge; edges %v", from, rel, to, e, g.Edges)
	}
}

func wantNoEdgeTo(t *testing.T, g *model.Graph, to model.NodeID) {
	t.Helper()
	for _, e := range g.Edges {
		if e.Target == to && e.Relation != model.RelationContains {
			t.Errorf("unexpected edge %+v", e)
		}
	}
}

func wantUnresolved(t *testing.T, un []resolve.Unresolved, want ...string) {
	t.Helper()
	var got []string
	for _, u := range un {
		got = append(got, u.Name+" "+string(u.Reason))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unresolved %q, want %q", got, want)
	}
}

const scene = "[gd_scene format=3]\n\n" +
	"[ext_resource type=\"Script\" path=\"res://Player.cs\" id=\"1\"]\n" +
	"[ext_resource type=\"Script\" path=\"res://hud.gd\" id=\"2\"]\n\n" +
	"[node name=\"P\" type=\"Node\"]\nscript = ExtResource(\"1\")\n\n" +
	"[node name=\"HUD\" type=\"Node\" parent=\".\"]\nscript = ExtResource(\"2\")\n\n"

func TestGodotConnectionSignalOnCSharp(t *testing.T) {
	g, _ := godotWith(t, map[string]string{
		"game/project.godot": "config_version=5\n",
		"game/hud.gd":        "extends Node\n\nfunc on_hit():\n\tpass\n",
		"game/main.tscn":     scene + "[connection signal=\"Hit\" from=\".\" to=\"HUD\" method=\"on_hit\"]\n",
	}, native("csharp", "game/Player.cs", "Hit", "signal", extract.NamespaceGodotSignal))
	wantEdge(t, g, "game/main.tscn", "game/Player.cs#Hit", model.RelationReferences)
	wantEdge(t, g, "game/main.tscn", "game/hud.gd#on_hit", model.RelationReferences)
}

func TestGodotConnectionHandlerOnCSharp(t *testing.T) {
	g, _ := godotWith(t, map[string]string{
		"game/project.godot": "config_version=5\n",
		"game/hud.gd":        "extends Node\n\nsignal pressed\n",
		"game/main.tscn":     scene + "[connection signal=\"pressed\" from=\"HUD\" to=\".\" method=\"OnPressed\"]\n",
	}, native("csharp", "game/Player.cs", "OnPressed", "method", extract.NamespaceGodotMethod))
	wantEdge(t, g, "game/main.tscn", "game/Player.cs#OnPressed", model.RelationReferences)
	wantEdge(t, g, "game/main.tscn", "game/hud.gd#pressed", model.RelationReferences)
}

// A handler offered by another file of C# is not the one the scene runs.
func TestGodotConnectionLooksOnlyInTheScriptTheNodeRuns(t *testing.T) {
	g, _ := godotWith(t, map[string]string{
		"game/project.godot": "config_version=5\n",
		"game/hud.gd":        "extends Node\n\nsignal pressed\n",
		"game/main.tscn":     scene + "[connection signal=\"pressed\" from=\"HUD\" to=\".\" method=\"OnPressed\"]\n",
	},
		extract.Result{Path: "game/Player.cs", Language: "csharp", Nodes: []model.Node{{ID: "game/Player.cs", Name: "p", Kind: model.KindFile, Path: "game/Player.cs", BodyHash: "h"}}},
		native("csharp", "game/Other.cs", "OnPressed", "method", extract.NamespaceGodotMethod))
	wantNoEdgeTo(t, g, "game/Other.cs#OnPressed")
}

func TestGodotExtendsCppSameProject(t *testing.T) {
	g, un := godotWith(t, map[string]string{
		"game/project.godot": "config_version=5\n",
		"game/x.gd":          "extends MyNode\n",
	}, cppClass("game/native/my.cpp", "MyNode"))
	wantEdge(t, g, "game/x.gd", "game/native/my.cpp#MyNode", model.RelationExtends)
	wantUnresolved(t, un)
}

func TestGodotExtendsCppBesideProject(t *testing.T) {
	g, un := godotWith(t, map[string]string{
		"game/project.godot": "config_version=5\n",
		"game/x.gd":          "extends MyNode\n",
	}, cppClass("src/my.cpp", "MyNode"))
	wantEdge(t, g, "game/x.gd", "src/my.cpp#MyNode", model.RelationExtends)
	wantUnresolved(t, un)
}

func TestGodotExtendsCppAtRepoRoot(t *testing.T) {
	g, un := godotWith(t, map[string]string{
		"project.godot": "config_version=5\n",
		"sub/x.gd":      "extends MyNode\n",
	}, cppClass("native/my.cpp", "MyNode"))
	wantEdge(t, g, "sub/x.gd", "native/my.cpp#MyNode", model.RelationExtends)
	wantUnresolved(t, un)
}

func TestGodotExtendsCppOtherProject(t *testing.T) {
	g, un := godotWith(t, map[string]string{
		"a/project.godot": "config_version=5\n",
		"b/project.godot": "config_version=5\n",
		"b/x.gd":          "extends MyNode\n",
	}, cppClass("a/my.cpp", "MyNode"))
	wantNoEdgeTo(t, g, "a/my.cpp#MyNode")
	wantUnresolved(t, un, "MyNode none")
}

func TestGodotExtendsCppAmbiguous(t *testing.T) {
	tree := map[string]string{"game/project.godot": "config_version=5\n", "game/x.gd": "extends MyNode\n"}
	for name, cpp := range map[string][]extract.Result{
		"beside and inside": {cppClass("src/my.cpp", "MyNode"), cppClass("game/my.cpp", "MyNode")},
		"twice inside":      {cppClass("game/a.cpp", "MyNode"), cppClass("game/b.cpp", "MyNode")},
	} {
		g, un := godotWith(t, tree, cpp...)
		for _, r := range cpp {
			wantNoEdgeTo(t, g, r.Exports[0].Target)
		}
		t.Run(name, func(t *testing.T) { wantUnresolved(t, un, "MyNode ambiguous") })
	}
}

func TestGodotHandoverStaysInProject(t *testing.T) {
	g, un := godotWith(t, map[string]string{
		"a/project.godot": "config_version=5\n",
		"a/only.gd":       "class_name Only\n",
		"b/project.godot": "config_version=5\n",
		"b/x.gd":          "extends Only\n",
	})
	wantNoEdgeTo(t, g, "a/only.gd#Only")
	wantUnresolved(t, un, "Only none")
}

func TestGodotLooseClassNameIsNoExport(t *testing.T) {
	g, un := godotWith(t, map[string]string{
		"loose/only.gd":   "class_name Only\n",
		"b/project.godot": "config_version=5\n",
		"b/x.gd":          "extends Only\n",
	})
	wantNoEdgeTo(t, g, "loose/only.gd#Only")
	wantUnresolved(t, un, "Only none")
}

func TestGodotClassExportServesOtherLanguages(t *testing.T) {
	q := extract.Result{
		Path: "game/q.cs", Language: "csharp",
		Nodes: []model.Node{{ID: "game/q.cs", Name: "q", Kind: model.KindFile, Path: "game/q.cs", BodyHash: "h"}},
		Bridges: []extract.Bridge{{Source: "game/q.cs", Relation: model.RelationReferences,
			Namespace: extract.NamespaceGodotClass, Name: "Only", File: "game/q.cs", Line: 2}},
	}
	g, _ := godotWith(t, map[string]string{
		"game/project.godot": "config_version=5\n",
		"game/only.gd":       "class_name Only\n",
	}, q)
	wantEdge(t, g, "game/q.cs", "game/only.gd#Only", model.RelationReferences)
}

func TestGodotExtendsMissingInnerAfterPathIsNoHandover(t *testing.T) {
	g, un := godotWith(t, map[string]string{
		"game/project.godot": "config_version=5\n",
		"game/a.gd":          "class Present:\n\tpass\n",
		"game/x.gd":          "extends \"res://a.gd\".Missing\n",
	}, cppClass("game/my.cpp", "Missing"))
	wantNoEdgeTo(t, g, "game/my.cpp#Missing")
	wantUnresolved(t, un, "Missing none")
}

func TestGodotExtendsDottedNameIsNoHandover(t *testing.T) {
	g, un := godotWith(t, map[string]string{
		"game/project.godot": "config_version=5\n",
		"game/x.gd":          "extends Outer.MyNode\n",
	}, cppClass("game/my.cpp", "MyNode"), cppClass("game/outer.cpp", "Outer"))
	wantNoEdgeTo(t, g, "game/my.cpp#MyNode")
	wantNoEdgeTo(t, g, "game/outer.cpp#Outer")
	wantUnresolved(t, un, "Outer.MyNode none")
}

func TestGodotNewOnCppClass(t *testing.T) {
	g, un := godotWith(t, map[string]string{
		"game/project.godot": "config_version=5\n",
		"game/x.gd":          "extends Node\n\nfunc f():\n\tvar n = MyNode.new()\n\tvar m = Outer.In.new()\n",
	}, cppClass("game/my.cpp", "MyNode"), cppClass("game/outer.cpp", "Outer"))
	wantEdge(t, g, "game/x.gd#f", "game/my.cpp#MyNode", model.RelationCalls)
	wantNoEdgeTo(t, g, "game/outer.cpp#Outer")
	if e := edgeBetween(g, "game/x.gd#f", "game/my.cpp#MyNode", model.RelationReferences); e != nil {
		t.Errorf("the references edge of .new() was handed over: %+v", e)
	}
	wantUnresolved(t, un, "Node none")
}

func TestGodotExtendsEngineClass(t *testing.T) {
	_, un := godotWith(t, map[string]string{
		"game/project.godot": "config_version=5\n",
		"game/x.gd":          "extends Node2D\n\nfunc f():\n\tvar n = Node2D.new()\n",
	})
	wantUnresolved(t, un, "Node2D none")
}

func TestGodotScriptWithoutProject(t *testing.T) {
	g, un := godotWith(t, map[string]string{
		"game/project.godot": "config_version=5\n",
		"loose/x.gd":         "extends MyNode\n",
	}, cppClass("src/my.cpp", "MyNode"))
	wantNoEdgeTo(t, g, "src/my.cpp#MyNode")
	wantUnresolved(t, un, "MyNode none")
}
