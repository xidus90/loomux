#include "calc.hpp"

namespace calc {

int Base::f() const { return 1; }

int Child::f() const { return Base::f() + 1; }

int twice(int value) {
  const Child child;
  return child.f() * value;
}

} // namespace calc
