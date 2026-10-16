extends GdUnitTestSuite


func test_fails() -> void:
	assert_int(A.new().double(2)).is_equal(5)
