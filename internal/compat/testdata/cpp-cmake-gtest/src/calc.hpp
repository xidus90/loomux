#pragma once

namespace calc {

class Base {
public:
  int f() const;
};

class Child : public Base {
public:
  int f() const;
};

int twice(int value);

} // namespace calc
