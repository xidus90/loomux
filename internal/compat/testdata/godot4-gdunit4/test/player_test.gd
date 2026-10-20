extends GdUnitTestSuite


func test_f_adds_one_to_the_base() -> void:
	var player := Player.new()
	assert_int(player.f()).is_equal(2)
	player.free()


func test_walk_counts_twice() -> void:
	var player := Player.new()
	player._ready()
	player.walk()
	assert_int(player.steps).is_equal(2)
	player.free()
