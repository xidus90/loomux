#include "calc.hpp"

#include <gtest/gtest.h>

TEST(Calc, DoublesAValue) { EXPECT_EQ(calc::twice(2), 4); }
