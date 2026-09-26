package main

// Run is the Go one; tool/run.py defines a Python Run as well.
func Run() {}

// T has a method Run, and so does the Python class T.
type T struct{}

// Run is T's.
func (T) Run() {}
