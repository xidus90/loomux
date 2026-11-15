package resolve_test

import (
	"maps"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
)

// rustScene runs PlayerX at its root and a script on HUD, and connects the
// signal and the renamed method of PlayerX.
const rustScene = "[gd_scene format=3]\n\n" +
	"[ext_resource type=\"Script\" path=\"res://hud.gd\" id=\"1\"]\n\n" +
	"[node name=\"Main\" type=\"PlayerX\"]\n\n" +
	"[node name=\"HUD\" type=\"Control\" parent=\".\"]\nscript = ExtResource(\"1\")\n\n" +
	"[connection signal=\"hit\" from=\".\" to=\"HUD\" method=\"on_hit\"]\n" +
	"[connection signal=\"done\" from=\"HUD\" to=\".\" method=\"do_move\"]\n"

// enemyScene connects a class without rename, whose impl sits beside it.
const enemyScene = "[gd_scene format=3]\n\n" +
	"[node name=\"E\" type=\"Enemy\"]\n\n" +
	"[connection signal=\"spotted\" from=\".\" to=\".\" method=\"attack\"]\n"

// gdextRustTree is a Godot project in godot/ with its Rust extension beside
// it in rust/, under no project. Player is registered as PlayerX in one file
// and its Godot API is in another.
func gdextRustTree() map[string]string {
	return map[string]string{
		"godot/project.godot": "config_version=5\n",
		"godot/main.tscn":     rustScene,
		"godot/enemy.tscn":    enemyScene,
		"godot/hud.gd":        "extends Control\n\nsignal done\n\nfunc on_hit():\n\tpass\n",
		"godot/player.gd":     "extends PlayerX\n\nfunc f():\n\tvar p = PlayerX.new()\n",
		"rust/Cargo.toml":     "[package]\nname = \"demo\"\n[lib]\ncrate-type = [\"cdylib\"]\n",
		"rust/src/lib.rs":     "mod player;\nmod player_api;\nmod enemy;\n",
		"rust/src/player.rs": "use godot::prelude::*;\n#[derive(GodotClass)]\n#[class(base=Node2D, rename=PlayerX)]\n" +
			"pub struct Player {\n    base: Base<Node2D>,\n}\n",
		"rust/src/player_api.rs": "use godot::prelude::*;\nuse crate::player::Player;\n#[godot_api]\nimpl Player {\n" +
			"    #[signal]\n    fn hit();\n" +
			"    #[func(rename=do_move)]\n    fn move_to(&mut self) {}\n}\n",
		"rust/src/enemy.rs": "use godot::prelude::*;\n#[derive(GodotClass)]\n#[class(base=Node)]\npub struct Enemy {}\n" +
			"#[godot_api]\nimpl Enemy {\n    #[signal]\n    fn spotted();\n    #[func]\n    fn attack(&self) {}\n}\n",
	}
}

func TestGDScriptReachesRustClass(t *testing.T) {
	g, un := rustTree(t, gdextRustTree())
	wantEdge(t, g, "godot/player.gd", "rust/src/player.rs#Player", model.RelationExtends)
	wantEdge(t, g, "godot/player.gd#f", "rust/src/player.rs#Player", model.RelationCalls)
	wantEdge(t, g, "godot/main.tscn", "rust/src/player.rs#Player", model.RelationReferences)
	wantEdge(t, g, "godot/enemy.tscn", "rust/src/enemy.rs#Enemy", model.RelationReferences)
	for _, u := range un {
		if u.File == "godot/player.gd" {
			t.Errorf("PlayerX left an unresolved name: %+v", u)
		}
	}
}

// A connection end on a node without a script names the member on the
// node's class: the signal by its method's name, the func by its rename,
// offered from the file of the impl although the rename is at the struct.
func TestSceneConnectionReachesRustMembers(t *testing.T) {
	g, _ := rustTree(t, gdextRustTree())
	wantEdge(t, g, "godot/main.tscn", "rust/src/player_api.rs#Player.hit", model.RelationReferences)
	wantEdge(t, g, "godot/main.tscn", "rust/src/player_api.rs#Player.move_to", model.RelationReferences)
	wantEdge(t, g, "godot/enemy.tscn", "rust/src/enemy.rs#Enemy.spotted", model.RelationReferences)
	wantEdge(t, g, "godot/enemy.tscn", "rust/src/enemy.rs#Enemy.attack", model.RelationReferences)
	// Control: the end on HUD lands on its script.
	wantEdge(t, g, "godot/main.tscn", "godot/hud.gd#on_hit", model.RelationReferences)
}

// Two Rust classes under one Godot name in one realm give no edge.
func TestTwoRustClassesOfOneGodotName(t *testing.T) {
	tree := gdextRustTree()
	maps.Copy(tree, map[string]string{
		"rust/src/lib.rs":  "mod player;\nmod player_api;\nmod enemy;\nmod twin;\n",
		"rust/src/twin.rs": "#[derive(GodotClass)]\n#[class(rename=PlayerX)]\npub struct Twin {}\n",
	})
	g, _ := rustTree(t, tree)
	wantNoEdgeTo(t, g, "rust/src/player.rs#Player")
	wantNoEdgeTo(t, g, "rust/src/twin.rs#Twin")
}

// A godot_api impl of a type no Godot class registers offers nothing.
func TestRustImplOfNoGodotClassOffersNothing(t *testing.T) {
	tree := gdextRustTree()
	tree["rust/src/player.rs"] = "pub struct Player {}\n"
	g, _ := rustTree(t, tree)
	wantNoEdgeTo(t, g, "rust/src/player_api.rs#Player.hit")
	wantNoEdgeTo(t, g, "rust/src/player_api.rs#Player.move_to")
}
