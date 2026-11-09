package syntax

// Stack has a method with type parameters of its own (Go 1.27).
type Stack[T any] struct{ items []T }

func (s *Stack[T]) Map[U any](f func(T) U) *Stack[U] {
	out := &Stack[U]{}
	for _, it := range s.items {
		out.items = append(out.items, f(it))
	}
	return out
}

func h(n int) int { return n }

func use(s *Stack[int]) *Stack[int] { return s.Map[int](h) }

type Inner struct{ B int }

type Outer struct{ Inner }

func v() int { return 2 }

// build keys a struct literal by a promoted field (Go 1.27).
func build() Outer { return Outer{B: v()} }
