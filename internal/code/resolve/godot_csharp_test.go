package resolve_test

import (
	"path"
	"sort"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// godotCSharp extracts a mixed Godot tree with the registered extractors, as
// a graph build does, and resolves it.
func godotCSharp(t *testing.T, tree map[string]string) *model.Graph {
	t.Helper()
	paths := make([]string, 0, len(tree))
	for p := range tree {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var files []extract.Result
	for _, p := range paths {
		lang, ok := all.For(p)
		if !ok {
			// project.godot is the one file of the tree no extractor claims.
			if path.Base(p) == "project.godot" {
				continue
			}
			t.Fatalf("no extractor claims %s", p)
		}
		r, err := lang.File(p, tree[p])
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, r)
	}
	g, _ := resolve.GraphWith(files, nil, stamp)
	return validated(t, g)
}

const playerScene = "[gd_scene format=3]\n\n" +
	"[ext_resource type=\"Script\" path=\"res://Player.cs\" id=\"1\"]\n\n" +
	"[node name=\"P\" type=\"Node\"]\nscript = ExtResource(\"1\")\n\n" +
	"[connection signal=\"Hit\" from=\".\" to=\".\" method=\"OnHit\"]\n"

const playerCs = "using Godot;\n\n" +
	"public partial class Player : CharacterBody2D\n{\n" +
	"\t[Signal] public delegate void HitEventHandler();\n" +
	"\tpublic void OnHit() {}\n" +
	"\tpublic void OnHit(int damage) {}\n}\n"

func TestGodotSceneReachesCSharpScript(t *testing.T) {
	g := godotCSharp(t, map[string]string{
		"game/project.godot": "config_version=5\n",
		"game/main.tscn":     playerScene,
		"game/Player.cs":     playerCs,
	})
	wantEdge(t, g, "game/main.tscn", "game/Player.cs", model.RelationImports)
	wantEdge(t, g, "game/main.tscn", "game/Player.cs#Player.OnHit", model.RelationReferences)
	wantEdge(t, g, "game/main.tscn", "game/Player.cs#Player.HitEventHandler", model.RelationReferences)
	// An overload is no second handler: the first declaration answers.
	wantNoEdgeTo(t, g, "game/Player.cs#Player.OnHit~2")
}

func TestGodotGDScriptNewsGlobalClassCSharp(t *testing.T) {
	g := godotCSharp(t, map[string]string{
		"game/project.godot": "config_version=5\n",
		"game/x.gd":          "extends Node\n\nfunc f():\n\tvar m = Mob.new()\n",
		"game/Mob.cs":        "using Godot;\n\n[GlobalClass]\npublic partial class Mob : Node\n{\n}\n",
	})
	wantEdge(t, g, "game/x.gd#f", "game/Mob.cs#Mob", model.RelationCalls)
}

// The scene runs Player.cs; a handler in another part of the class is not
// offered.
func TestGodotSceneMissesHandlerInOtherPartialFile(t *testing.T) {
	g := godotCSharp(t, map[string]string{
		"game/project.godot": "config_version=5\n",
		"game/main.tscn":     playerScene,
		"game/Player.cs":     "using Godot;\n\npublic partial class Player : CharacterBody2D\n{\n}\n",
		"game/Player.Hit.cs": "using Godot;\n\npublic partial class Player\n{\n\tpublic void OnHit() {}\n}\n",
	})
	wantEdge(t, g, "game/main.tscn", "game/Player.cs", model.RelationImports)
	wantNoEdgeTo(t, g, "game/Player.Hit.cs#Player.OnHit")
}

func TestGodotSceneDoesNotReachScriptOfOtherProject(t *testing.T) {
	g := godotCSharp(t, map[string]string{
		"a/project.godot": "config_version=5\n",
		"a/Player.cs":     playerCs,
		"a/main.tscn":     playerScene,
		"b/project.godot": "config_version=5\n",
		"b/main.tscn":     playerScene,
	})
	// Control: the scene of a's own project reaches the script.
	wantEdge(t, g, "a/main.tscn", "a/Player.cs#Player.OnHit", model.RelationReferences)
	for _, e := range g.Edges {
		if e.Source == "b/main.tscn" && strings.HasPrefix(string(e.Target), "a/") {
			t.Errorf("scene of project b reached project a: %+v", e)
		}
	}
}
