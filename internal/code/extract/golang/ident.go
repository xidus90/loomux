package golang

import (
	"go/ast"
	"strings"
	"unicode"
)

// exported reports Go visibility: the first letter of a symbol's OWN name is
// uppercase. For a receiver-qualified name the own name is the part after the
// last dot.
func exported(name string) bool {
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}

// receiverType is the base type name of a method's receiver, with a pointer
// unwrapped: `func (u *User)` owns "User". Empty when it cannot be read.
func receiverType(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	expr := recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	// A generic receiver: `func (c *Cache[K]) Get()` -- the name is the index
	// expression's own, not its type argument.
	switch t := expr.(type) {
	case *ast.IndexExpr:
		expr = t.X
	case *ast.IndexListExpr:
		expr = t.X
	}
	if id, ok := expr.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// receiverVar is the name a method binds its receiver to -- the `u` in
// `func (u *User) Save()`. Empty for an unnamed receiver.
func receiverVar(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 || len(recv.List[0].Names) == 0 {
		return ""
	}
	return recv.List[0].Names[0].Name
}
