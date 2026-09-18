package golang

import (
	"go/ast"
	"strings"
	"unicode"
)

// maxBodyChars caps the searchable body. Graft's figure; a definition longer
// than this is findable by its first 5000 characters or not at all.
const maxBodyChars = 5000

// mintID returns base, or base with the lowest free ordinal appended.
//
// A loop and not a single "~2" guess: a qualified source name may itself end
// in "~2", and only the loop is tight against that.
func mintID(base string, minted map[string]bool) string {
	id := base
	for k := 2; minted[id]; k++ {
		id = base + "~" + itoa(k)
	}
	minted[id] = true
	return id
}

// itoa is strconv.Itoa without the import, so this file stays free of
// anything but the AST.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

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

// collapse turns a definition's text into one searchable line: every run of
// whitespace becomes a single space, and the result is capped.
func collapse(text string) string {
	out := strings.Join(strings.Fields(text), " ")
	if len(out) > maxBodyChars {
		return out[:maxBodyChars]
	}
	return out
}
