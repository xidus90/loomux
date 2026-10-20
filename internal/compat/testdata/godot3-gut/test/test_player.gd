extends "res://addons/gut/test.gd"

const Player = preload("res://scripts/player.gd")


func test_f_adds_one_to_the_base():
	var player = Player.new()
	assert_eq(player.f(), 2)
	player.free()


func test_walk_counts_twice():
	var player = Player.new()
	add_child(player)
	player.walk()
	assert_eq(player.steps, 2)
	player.free()
