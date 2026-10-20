// Package calc is the smallest module that has every kind of check to run.
package calc

// Base is what Child builds on.
func Base() int { return 1 }

// Child adds one to Base.
func Child() int { return Base() + 1 }

// Double scales Child.
func Double(value int) int { return Child() * value }
