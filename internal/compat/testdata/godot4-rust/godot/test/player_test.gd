extends GdUnitTestSuite


func test_player_reports_its_speed() -> void:
	var player := Player.new()
	assert_float(player.get_speed()).is_equal(2.0)
	player.free()


func test_player_moves_by_a_delta() -> void:
	var player := Player.new()
	assert_float(player.do_move(1.5)).is_equal(3.5)
	player.free()
