extends GutTest

func test_passes():
	assert_eq(1, 1)

func test_fails():
	assert_eq(1, 2, "one is not two")
