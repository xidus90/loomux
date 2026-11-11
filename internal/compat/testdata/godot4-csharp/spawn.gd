extends Node


func spawn() -> Node2D:
	var mob := Mob.new()
	add_child(mob)
	return mob
