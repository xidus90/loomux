class_name Player
extends "res://scripts/base.gd"

signal moved

var steps := 0


func _ready():
	connect("moved", self, "_on_moved")


func f():
	return .f() + 1


func walk():
	steps += 1
	emit_signal("moved")


func _on_moved():
	steps += 1
