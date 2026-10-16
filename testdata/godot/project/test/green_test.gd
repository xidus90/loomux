extends GdUnitTestSuite


func test_double() -> void:
	assert_int(A.new().double(2)).is_equal(4)


func test_autoload() -> void:
	assert_int(Clock.now()).is_equal(42)
