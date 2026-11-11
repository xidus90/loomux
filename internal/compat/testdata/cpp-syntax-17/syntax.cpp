#include <utility>

namespace syn::inner {

template <typename T> int pick(T value) {
  if constexpr (sizeof(T) > 4) {
    return 1;
  } else {
    return static_cast<int>(value);
  }
}

int sum() {
  auto [a, b] = std::pair{1, 2};
  return a + b + pick(a);
}

int after() { return sum(); }

} // namespace syn::inner
