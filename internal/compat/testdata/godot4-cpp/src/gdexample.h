#pragma once

#include <godot_cpp/classes/sprite2d.hpp>

namespace godot {

class GDExample : public Sprite2D {
  GDCLASS(GDExample, Sprite2D)

protected:
  static void _bind_methods();

public:
  double get_speed() const { return speed; }

private:
  double speed = 1.0;
};

} // namespace godot
