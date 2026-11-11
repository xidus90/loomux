package resolve_test

import (
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// A GDExtension class: GDCLASS offers the class, its bindings its method
// and its signal.
const extH = "#include <godot_cpp/classes/sprite2d.hpp>\nnamespace godot {\n" +
	"class GDExample : public Sprite2D {\n  GDCLASS(GDExample, Sprite2D)\nprotected:\n  static void _bind_methods();\n" +
	"public:\n  double get_speed() const { return 1.0; }\n};\n}\n"

const extCpp = "#include \"gdexample.h\"\nusing namespace godot;\n" +
	"void GDExample::_bind_methods() {\n" +
	"  ClassDB::bind_method(D_METHOD(\"get_speed\"), &GDExample::get_speed);\n" +
	"  ADD_SIGNAL(MethodInfo(\"position_changed\"));\n}\n"

// extScene runs GDExample at its root and a script on HUD, and connects
// members of both.
const extScene = "[gd_scene format=3]\n\n" +
	"[ext_resource type=\"Script\" path=\"res://hud.gd\" id=\"1\"]\n\n" +
	"[node name=\"Main\" type=\"GDExample\"]\n\n" +
	"[node name=\"HUD\" type=\"Control\" parent=\".\"]\nscript = ExtResource(\"1\")\n\n" +
	"[connection signal=\"position_changed\" from=\".\" to=\"HUD\" method=\"on_moved\"]\n" +
	"[connection signal=\"done\" from=\"HUD\" to=\".\" method=\"get_speed\"]\n"

// lookScene asks GDExample's members of a built-in Sprite2D, which binds
// none of them.
const lookScene = "[gd_scene format=3]\n\n" +
	"[node name=\"Look\" type=\"Sprite2D\"]\n\n" +
	"[connection signal=\"position_changed\" from=\".\" to=\".\" method=\"get_speed\"]\n"

const extHud = "extends Control\n\nsignal done\n\nfunc on_moved():\n\tpass\n"

// gdextensionTree is a Godot project in project/ with its GDExtension
// sources beside it in src/, under no project.
func gdextensionTree() map[string]string {
	return map[string]string{
		"project/project.godot": "config_version=5\n",
		"project/main.tscn":     extScene,
		"project/look.tscn":     lookScene,
		"project/hud.gd":        extHud,
		"project/player.gd":     "extends GDExample\n\nfunc f():\n\tvar g = GDExample.new()\n",
		"src/gdexample.h":       extH,
		"src/gdexample.cpp":     extCpp,
	}
}

// wantNothingFrom fails on any edge but containment out of from.
func wantNothingFrom(t *testing.T, g *model.Graph, from model.NodeID) {
	t.Helper()
	for _, e := range g.Edges {
		if e.Source == from && e.Relation != model.RelationContains {
			t.Errorf("unexpected edge %+v", e)
		}
	}
}

func TestGDScriptReachesGDExtensionClass(t *testing.T) {
	g, un := godotTree(t, gdextensionTree())
	wantEdge(t, g, "project/player.gd", "src/gdexample.h#GDExample", model.RelationExtends)
	wantEdge(t, g, "project/player.gd#f", "src/gdexample.h#GDExample", model.RelationCalls)
	for _, u := range un {
		if u.File == "project/player.gd" {
			t.Errorf("GDExample left an unresolved name: %+v", u)
		}
	}
}

// A node's type references its class; a built-in type is offered by no one
// and leaves neither an edge nor an unresolved name.
func TestSceneNodeTypeReferencesGDExtensionClass(t *testing.T) {
	g, un := godotTree(t, gdextensionTree())
	wantEdge(t, g, "project/main.tscn", "src/gdexample.h#GDExample", model.RelationReferences)
	wantNothingFrom(t, g, "project/look.tscn")
	for _, u := range un {
		if strings.HasSuffix(u.File, ".tscn") {
			t.Errorf("node type left an unresolved name: %+v", u)
		}
	}
}

// A connection end on a node without a script names the member on the
// node's class. Every typed node also references its class, so the signal's
// own edge shows only with the type references taken out.
func TestSceneConnectionReachesGDExtensionMembers(t *testing.T) {
	var files []extract.Result
	for _, f := range godotFiles(t, gdextensionTree()) {
		var kept []extract.RawEdge
		for _, e := range f.Edges {
			if e.Member != "class" {
				kept = append(kept, e)
			}
		}
		f.Edges = kept
		files = append(files, f)
	}
	g, _ := resolve.GraphWith(files, nil, stamp)
	validated(t, g)
	wantEdge(t, g, "project/main.tscn", "src/gdexample.h#GDExample", model.RelationReferences)
	wantEdge(t, g, "project/main.tscn", "src/gdexample.h#GDExample.get_speed", model.RelationReferences)
	// Control: the ends on HUD land on its script.
	wantEdge(t, g, "project/main.tscn", "project/hud.gd#on_moved", model.RelationReferences)
	wantEdge(t, g, "project/main.tscn", "project/hud.gd#done", model.RelationReferences)
	wantNothingFrom(t, g, "project/look.tscn")
}

// A GDExtension inside one project is no class of another.
func TestOtherProjectDoesNotReachGDExtensionClass(t *testing.T) {
	g, _ := godotTree(t, map[string]string{
		"a/project.godot":     "config_version=5\n",
		"a/src/gdexample.h":   extH,
		"a/src/gdexample.cpp": extCpp,
		"a/main.tscn":         extScene,
		"a/hud.gd":            extHud,
		"b/project.godot":     "config_version=5\n",
		"b/main.tscn":         extScene,
		"b/hud.gd":            extHud,
	})
	// Control: a's own scene reaches the class and its method.
	wantEdge(t, g, "a/main.tscn", "a/src/gdexample.h#GDExample", model.RelationReferences)
	wantEdge(t, g, "a/main.tscn", "a/src/gdexample.h#GDExample.get_speed", model.RelationReferences)
	for _, e := range g.Edges {
		if e.Source == "b/main.tscn" && strings.HasPrefix(string(e.Target), "a/") {
			t.Errorf("scene of project b reached project a: %+v", e)
		}
	}
}
