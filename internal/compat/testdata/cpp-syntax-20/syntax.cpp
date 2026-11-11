#include <compare>
#include <concepts>
#include <coroutine>

namespace syn {

template <typename T>
concept Small = sizeof(T) <= 4;

template <Small T>
  requires std::integral<T>
int twice(T value) {
  return value * 2;
}

struct Point {
  int x;
  auto operator<=>(const Point &) const = default;
};

struct Task {
  struct promise_type {
    Task get_return_object() { return {}; }
    std::suspend_never initial_suspend() noexcept { return {}; }
    std::suspend_never final_suspend() noexcept { return {}; }
    void return_void() {}
    void unhandled_exception() {}
  };
};

Task run() {
  co_await std::suspend_never{};
  co_return;
}

bool less(Point a, Point b) { return (a <=> b) < 0; }

int after() {
  run();
  return twice(1) + (less({1}, {2}) ? 1 : 0);
}

} // namespace syn
