namespace syn {

int twice(int value) { return value * 2; }

struct Counter {
  int n = 0;
  template <typename Self> auto &&get(this Self &&self) { return self.n; }
  int doubled(this const Counter &self) { return twice(self.n); }
};

constexpr int pick(int value) {
  if consteval {
    return value;
  } else {
    return twice(value);
  }
}

int after() { return twice(pick(1)); }

int last() { return after(); }

} // namespace syn
