package syntax

func g() int { return 1 }

// f takes new with an expression operand (Go 1.26).
func f() *int { return new(g()) }
