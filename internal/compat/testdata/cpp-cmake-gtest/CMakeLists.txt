cmake_minimum_required(VERSION 3.25)
project(calc LANGUAGES CXX)

# The standard comes from the configure call (-DCMAKE_CXX_STANDARD=<v>); a
# compiler without it fails instead of falling back to an older one.
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

include(FetchContent)
# SYSTEM keeps GoogleTest's headers out of clang-tidy's findings.
FetchContent_Declare(
  googletest
  URL https://github.com/google/googletest/releases/download/v1.18.0/googletest-1.18.0.tar.gz
  URL_HASH SHA256=6e3191c1455468b3fc35a417fb565c1c5071aee1b7e7f85e30cf48a98d37d8b5
  SYSTEM
)
set(INSTALL_GTEST OFF CACHE BOOL "" FORCE)
set(BUILD_GMOCK OFF CACHE BOOL "" FORCE)
FetchContent_MakeAvailable(googletest)
# Third-party sources: neither linted nor measured.
set_target_properties(gtest gtest_main PROPERTIES EXPORT_COMPILE_COMMANDS OFF)

enable_testing()

add_library(calc src/calc.cpp)
target_include_directories(calc PUBLIC src)
target_compile_options(calc PRIVATE --coverage)
target_link_options(calc PUBLIC --coverage)

add_executable(calc_test tests/calc_test.cpp)
target_link_libraries(calc_test PRIVATE calc GTest::gtest_main)
add_test(NAME calc_test COMMAND calc_test)
