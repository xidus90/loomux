class_name Player
extends Base

signal moved

var steps := 0


func _ready() -> void:
	moved.connect(_on_moved)


func f() -> int:
	return super.f() + 1


func walk() -> void:
	steps += 1
	moved.emit()


func _on_moved() -> void:
	steps += 1
