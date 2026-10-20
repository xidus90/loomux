extends GutTest


func test_f_adds_one_to_the_base() -> void:
	var player := Player.new()
	assert_eq(player.f(), 2)
	player.free()


func test_walk_counts_twice() -> void:
	var player := Player.new()
	player._ready()
	player.walk()
	assert_eq(player.steps, 2)
	player.free()
