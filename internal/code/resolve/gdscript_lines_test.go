package resolve_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
)

// Godot 3 calls the base's version of a function as `.f()`; Godot 4 as
// `super.f()` or `super()`. Both land on the base.
func TestGodotBaseCallsOfBothLines(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"p/project.godot": "config_version=4\n",
		"p/base.gd":       "class_name Base\nextends Node\n\nfunc _ready():\n\tpass\n\nfunc greet(n):\n\tpass\n",
		"p/three.gd":      "extends Base\n\nfunc _ready():\n\t._ready()\n\t.greet(1)\n",
		"p/four.gd":       "extends Base\n\nfunc _ready():\n\tsuper()\n\tsuper.greet(1)\n",
	})
	for _, f := range []model.NodeID{"p/three.gd#_ready", "p/four.gd#_ready"} {
		wantCalls(t, g, f, "p/base.gd#Base._ready extracted", "p/base.gd#Base.greet extracted")
	}
}

// Every way of naming a signal by a string literal references the signal:
// connect, disconnect and is_connected in both lines, emit_signal on another
// object, Godot 3's yield and Godot 4's await.
func TestGodotSignalsNamedByStringsOrAwaited(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"s/project.godot": "config_version=4\n\n[autoload]\n\nBus=\"*res://bus.gd\"\n",
		"s/bus.gd":        "signal ping(n)\n",
		"s/user.gd": "signal died\n\n" +
			"func a():\n\tconnect(\"died\", self, \"_on_died\")\n\n" +
			"func b():\n\tdisconnect(\"died\", self, \"_on_died\")\n\n" +
			"func c():\n\tis_connected(\"died\", self, \"_on_died\")\n\n" +
			"func d():\n\tconnect(\"died\", _on_died)\n\n" +
			"func e():\n\tyield(self, \"died\")\n\n" +
			"func f():\n\tawait died\n\n" +
			"func g():\n\tdied.is_connected(_on_died)\n\n" +
			"func _on_died():\n\tpass\n",
		"s/other.gd": "func a():\n\tBus.connect(\"ping\", self, \"_on_ping\")\n\n" +
			"func b():\n\tBus.emit_signal(\"ping\", 1)\n\n" +
			"func c():\n\tyield(Bus, \"ping\")\n\n" +
			"func d():\n\tawait Bus.ping\n\n" +
			"func e(sig):\n\tBus.connect(sig, self, \"_on_ping\")\n\n" +
			"func _on_ping(n):\n\tpass\n",
	})
	for _, f := range []string{"a", "b", "c", "d", "g"} {
		wantTargets(t, g, model.NodeID("s/user.gd#"+f), model.RelationReferences, "s/user.gd#_on_died", "s/user.gd#died")
	}
	for _, f := range []string{"e", "f"} {
		wantTargets(t, g, model.NodeID("s/user.gd#"+f), model.RelationReferences, "s/user.gd#died")
	}
	wantTargets(t, g, "s/other.gd#a", model.RelationReferences, "s/bus.gd", "s/bus.gd#ping", "s/other.gd#_on_ping")
	for _, f := range []string{"b", "c", "d"} {
		wantTargets(t, g, model.NodeID("s/other.gd#"+f), model.RelationReferences, "s/bus.gd", "s/bus.gd#ping")
	}
	// A signal name in a variable names no signal; the handler still counts.
	wantTargets(t, g, "s/other.gd#e", model.RelationReferences, "s/bus.gd", "s/other.gd#_on_ping")
}

// A handler connected on a node the script reaches by path -- $Node,
// get_node("x"), a call's result -- is referenced like one connected on a
// name, in Godot 3's `connect("s", self, "m")` and Godot 4's `.s.connect(m)`.
func TestGodotHandlersConnectedOnNodesByPath(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"n/project.godot": "config_version=4\n",
		"n/ui.gd": "func a():\n\t$Anim.connect(\"animation_finished\", self, \"_on_done\")\n\n" +
			"func b():\n\tget_node(\"Button\").connect(\"pressed\", self, \"_on_done\", [1])\n\n" +
			"func c():\n\t$Button.pressed.connect(_on_done)\n\n" +
			"func d():\n\t$Anim.disconnect(\"animation_finished\", self, \"_on_done\")\n\n" +
			"func e():\n\t$Anim.emit_signal(\"animation_finished\", self, \"_on_done\")\n\n" +
			"func _on_done(x = null):\n\tpass\n",
	})
	for _, f := range []string{"a", "b", "c", "d"} {
		wantTargets(t, g, model.NodeID("n/ui.gd#"+f), model.RelationReferences, "n/ui.gd#_on_done")
	}
	// emit_signal's further arguments are the signal's, not a handler.
	wantTargets(t, g, "n/ui.gd#e", model.RelationReferences)
}

// A method passed as a value is referenced: by name, through self, in a
// funcref or a Callable. A parameter or a local of that name hides it.
func TestGodotMethodsAsValues(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"m/project.godot": "config_version=5\n",
		"m/m.gd": "func a():\n\tvar f = funcref(self, \"t\")\n\n" +
			"func b():\n\tvar c = Callable(self, \"t\")\n\n" +
			"func c():\n\tvar f := t\n\n" +
			"func d():\n\tvar f = self.t\n\n" +
			"func e(t):\n\tprint(t)\n\n" +
			"func f():\n\tvar t = 1\n\tprint(t)\n\n" +
			"func g():\n\tvar h = func(t): return t\n\n" +
			"func t():\n\tpass\n",
	})
	for _, f := range []string{"a", "b", "c", "d"} {
		wantTargets(t, g, model.NodeID("m/m.gd#"+f), model.RelationReferences, "m/m.gd#t")
	}
	for _, f := range []string{"e", "f", "g"} {
		wantTargets(t, g, model.NodeID("m/m.gd#"+f), model.RelationReferences)
	}
}

// A method called by its name as a string, or a Callable called, is a call.
func TestGodotCallsByNameAndCallables(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"c/project.godot": "config_version=4\n\n[autoload]\n\nNet=\"*res://net.gd\"\n",
		"c/net.gd":        "func sync_all():\n\tpass\n",
		"c/c.gd": "func a():\n\tcall(\"t\")\n\tcall_deferred(\"u\")\n\tcallv(\"v\", [])\n\n" +
			"func b():\n\trpc(\"t\")\n\trpc_unreliable(\"u\")\n\trpc_id(1, \"v\")\n\trpc_unreliable_id(1, \"w\")\n\n" +
			"func c():\n\tNet.call_deferred(\"sync_all\")\n\tself.call(\"t\")\n\n" +
			"func d():\n\tt.call()\n\tu.call_deferred()\n\tself.v.callv([])\n\tw.rpc()\n\tt.rpc_id(1)\n\n" +
			"func e(t):\n\tt.call()\n\n" +
			"func t():\n\tpass\n\nfunc u():\n\tpass\n\nfunc v():\n\tpass\n\nfunc w():\n\tpass\n",
	})
	wantCalls(t, g, "c/c.gd#a", "c/c.gd#t extracted", "c/c.gd#u extracted", "c/c.gd#v extracted")
	wantCalls(t, g, "c/c.gd#b", "c/c.gd#t extracted", "c/c.gd#u extracted", "c/c.gd#v extracted", "c/c.gd#w extracted")
	wantCalls(t, g, "c/c.gd#c", "c/c.gd#t extracted", "c/net.gd#sync_all extracted")
	wantCalls(t, g, "c/c.gd#d", "c/c.gd#t extracted", "c/c.gd#u extracted", "c/c.gd#v extracted", "c/c.gd#w extracted")
	wantCalls(t, g, "c/c.gd#e")
}

// Godot 3's setget and Godot 4's get/set name their accessors.
func TestGodotPropertyAccessorsOfBothLines(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"a/project.godot": "config_version=4\n",
		"a/three.gd": "var hp = 1 setget set_hp, get_hp\nvar mp setget , get_mp\n" +
			"export(int) var xp = 0 setget set_xp\nonready var lp = 2 setget set_lp\n\n" +
			"func set_hp(v):\n\tpass\nfunc get_hp():\n\tpass\nfunc get_mp():\n\tpass\nfunc set_xp(v):\n\tpass\nfunc set_lp(v):\n\tpass\n",
		"a/four.gd": "var hp: int:\n\tget = get_hp, set = set_hp\n\n" +
			"func set_hp(v):\n\tpass\nfunc get_hp():\n\tpass\n",
	})
	wantTargets(t, g, "a/three.gd", model.RelationReferences,
		"a/three.gd#get_hp", "a/three.gd#get_mp", "a/three.gd#set_hp", "a/three.gd#set_lp", "a/three.gd#set_xp")
	wantTargets(t, g, "a/four.gd", model.RelationReferences, "a/four.gd#get_hp", "a/four.gd#set_hp")
}

// extends a path and an inner class of it; an abstract function without a
// body (Godot 4.5) still gets its node.
func TestGodotExtendsAnInnerClassOfAPathAndAbstractFunctions(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"e/project.godot": "config_version=5\n",
		"e/outer.gd":      "class Inner:\n\tfunc f():\n\t\tpass\n",
		"e/sub.gd":        "extends \"res://outer.gd\".Inner\n\nfunc g():\n\tf()\n",
		"e/abs.gd":        "@abstract\nclass_name Shape\n\n@abstract func area() -> float\n\nfunc twice():\n\treturn area() * 2\n",
	})
	wantTargets(t, g, "e/sub.gd", model.RelationExtends, "e/outer.gd#Inner")
	wantCalls(t, g, "e/sub.gd#g", "e/outer.gd#Inner.f extracted")
	wantCalls(t, g, "e/abs.gd#Shape.twice", "e/abs.gd#Shape.area extracted")
}

// A scene's connections reference the handler in the target node's script
// and the signal in the source node's, in format=2 (Godot 3, ids as
// integers) and format=3 (Godot 4, ids as strings). A node that instances a
// scene stands for that scene's root script.
func TestGodotSceneConnectionsOfBothFormats(t *testing.T) {
	const script = "signal hit\n\nfunc _on_pressed():\n\tpass\n\nfunc _on_hit():\n\tpass\n"
	g := gdGraph(t, map[string]string{
		"g/project.godot": "config_version=4\n",
		"g/main.gd":       script,
		"g/child.gd":      "signal hit\n",
		"g/child.tscn":    "[gd_scene load_steps=2 format=2]\n\n[ext_resource path=\"res://child.gd\" type=\"Script\" id=1]\n\n[node name=\"Child\" type=\"Node\"]\nscript = ExtResource( 1 )\n",
		"g/three.tscn": "[gd_scene load_steps=3 format=2]\n\n" +
			"[ext_resource path=\"res://main.gd\" type=\"Script\" id=1]\n[ext_resource path=\"res://child.tscn\" type=\"PackedScene\" id=2]\n\n" +
			"[node name=\"Main\" type=\"Node2D\"]\nscript = ExtResource( 1 )\n\n" +
			"[node name=\"Box\" type=\"Node\" parent=\".\"]\n\n" +
			"[node name=\"B\" type=\"Button\" parent=\"Box\"]\n\n" +
			"[node name=\"C\" parent=\".\" instance=ExtResource( 2 )]\n\n" +
			"[connection signal=\"pressed\" from=\"Box/B\" to=\".\" method=\"_on_pressed\"]\n" +
			"[connection signal=\"hit\" from=\"C\" to=\".\" method=\"_on_hit\" binds= [ 1 ]]\n" +
			"[connection signal=\"hit\" from=\".\" to=\"Box\" method=\"_on_hit\"]\n",
		"g/four.tscn": "[gd_scene load_steps=3 format=3 uid=\"uid://cfour\"]\n\n" +
			"[ext_resource type=\"Script\" uid=\"uid://cmain\" path=\"res://main.gd\" id=\"1_ab\"]\n" +
			"[ext_resource type=\"PackedScene\" path=\"res://child.tscn\" id=\"2_cd\"]\n\n" +
			"[node name=\"Main\" type=\"Node2D\"]\nscript = ExtResource(\"1_ab\")\n\n" +
			"[node name=\"C\" parent=\".\" instance=ExtResource(\"2_cd\")]\n\n" +
			"[connection signal=\"hit\" from=\"C\" to=\".\" method=\"_on_hit\" flags=18]\n" +
			"[connection signal=\"hit\" from=\".\" to=\".\" method=\"_on_pressed\"]\n",
	})
	wantTargets(t, g, "g/three.tscn", model.RelationReferences,
		"g/child.gd#hit", "g/main.gd#_on_hit", "g/main.gd#_on_pressed", "g/main.gd#hit")
	wantTargets(t, g, "g/four.tscn", model.RelationReferences,
		"g/child.gd#hit", "g/main.gd#_on_hit", "g/main.gd#_on_pressed", "g/main.gd#hit")
}

// An autoload that is a scene binds the scene's root script.
func TestGodotASceneAutoloadBindsItsRootScript(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"a/project.godot": "config_version=5\n\n[autoload]\n\nBus=\"*res://bus.tscn\"\n",
		"a/bus.gd":        "signal ping\n\nfunc send():\n\tpass\n",
		"a/bus.tscn":      "[gd_scene format=3]\n\n[ext_resource type=\"Script\" path=\"res://bus.gd\" id=\"1\"]\n\n[node name=\"Bus\" type=\"Node\"]\nscript = ExtResource(\"1\")\n",
		"a/user.gd":       "func a():\n\tBus.send()\n\tBus.ping.emit()\n",
	})
	wantCalls(t, g, "a/user.gd#a", "a/bus.gd#send extracted")
	wantTargets(t, g, "a/user.gd#a", model.RelationReferences, "a/bus.gd", "a/bus.gd#ping")
}

// The declarations of a Godot 3 script -- tool, class_name with an icon,
// export(...) and onready vars, the network keywords -- keep every node, and
// a preload in an export or onready var binds its alias like const does.
func TestGodotThreeDeclarationsKeepTheirNodesAndAliases(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"t/project.godot": "config_version=4\n",
		"t/lib.gd":        "static func make():\n\tpass\n",
		"t/p.gd": "tool\nextends Node\nclass_name P, \"res://icon.svg\"\n\n" +
			"export(Script) var Lib = preload(\"res://lib.gd\")\nonready var Lib2 = preload(\"res://lib.gd\")\n" +
			"export(int, 0, 10) var n = 1\nonready var label = $Label\n\n" +
			"remote func a():\n\tLib.make()\nmaster func b():\n\tLib2.make()\npuppet func c():\n\tpass\n" +
			"remotesync func d():\n\tpass\nmastersync func e():\n\tpass\npuppetsync func f():\n\tpass\nsync func g():\n\tpass\n",
	})
	for _, f := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		if !hasNode(g, model.NodeID("t/p.gd#P."+f)) {
			t.Errorf("no node for P.%s", f)
		}
	}
	wantCalls(t, g, "t/p.gd#P.a", "t/lib.gd#make extracted")
	wantCalls(t, g, "t/p.gd#P.b", "t/lib.gd#make extracted")
}

func hasNode(g *model.Graph, id model.NodeID) bool {
	for _, n := range g.Nodes {
		if n.ID == id {
			return true
		}
	}
	return false
}

// Where a path, an inner class or a root script is missing from the build,
// the new forms give no edge rather than a wrong one.
func TestGodotLinesDeadEnds(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"x/project.godot": "config_version=5\n\n[autoload]\n\nBus=\"*res://bus.tscn\"\n",
		"x/bus.tscn": "[gd_scene format=3]\n\n[ext_resource type=\"Script\" path=\"res://gone.gd\" id=\"1\"]\n[ext_resource type=\"Script\" path=\"res://plain.gd\"]\n\n" +
			"[node name=\"Bus\" type=\"Node\"]\nscript = ExtResource(\"1\")\n\n[connection signal=\"s\" from=\".\" to=\".\" method=\"m\"]\n",
		"x/plain.gd": "",
		"x/outer.gd": "class Inner:\n\tpass\n",
		"x/a.gd":     "extends \"res://outer.gd\".Nope\n",
		"x/b.gd":     "extends \"res://gone.gd\".Inner\n",
		"x/user.gd":  "func a():\n\tBus.send()\n",
	})
	wantTargets(t, g, "x/a.gd", model.RelationExtends)
	wantTargets(t, g, "x/b.gd", model.RelationExtends)
	wantTargets(t, g, "x/bus.tscn", model.RelationReferences)
	wantTargets(t, g, "x/bus.tscn", model.RelationImports, "x/plain.gd")
	wantCalls(t, g, "x/user.gd#a")
	wantTargets(t, g, "x/user.gd#a", model.RelationReferences, "x/bus.tscn")
}

// uid:// paths resolve through a script's .uid file (Godot 4.4+) and through
// the uid in a scene's or resource's header, in scripts, autoloads and the
// main scene.
func TestGodotUIDPaths(t *testing.T) {
	g := gdGraph(t, map[string]string{
		"u/project.godot": "config_version=5\n\n[application]\n\nrun/main_scene=\"uid://cmain\"\n\n[autoload]\n\nBus=\"*uid://cbus\"\n",
		"u/bus.gd":        "func send():\n\tpass\n",
		"u/bus.gd.uid":    "uid://cbus\n",
		"u/main.tscn":     "[gd_scene format=3 uid=\"uid://cmain\"]\n\n[ext_resource type=\"Script\" uid=\"uid://cbus\" id=\"1\"]\n",
		"u/stats.tres":    "[gd_resource type=\"Resource\" format=3 uid=\"uid://cstats\"]\n",
		"u/user.gd":       "const S = preload(\"uid://cstats\")\nconst L = preload(\"uid://cnone\")\n\nfunc a():\n\tBus.send()\n",
	})
	wantTargets(t, g, "u/project.godot", model.RelationImports, "u/bus.gd", "u/main.tscn")
	wantTargets(t, g, "u/main.tscn", model.RelationImports, "u/bus.gd")
	wantTargets(t, g, "u/user.gd", model.RelationImports, "u/stats.tres")
	wantCalls(t, g, "u/user.gd#a", "u/bus.gd#send extracted")
}
