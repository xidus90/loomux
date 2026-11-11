#include "gdexample.h"

#include <godot_cpp/core/class_db.hpp>

using namespace godot;

void GDExample::_bind_methods() {
  ClassDB::bind_method(D_METHOD("get_speed"), &GDExample::get_speed);
  ADD_SIGNAL(MethodInfo("position_changed"));
}
