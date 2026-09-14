package gitenv_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func repoRoot() string {
	_, here, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(here), "..", "..")
}

// gitCall reports whether expr starts git: exec.Command("git", ...) or
// exec.CommandContext(ctx, "git", ...).
func gitCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if pkg, ok := selector.X.(*ast.Ident); !ok || pkg.Name != "exec" {
		return false
	}
	index := 0
	switch selector.Sel.Name {
	case "Command":
	case "CommandContext":
		index = 1
	default:
		return false
	}
	if len(call.Args) <= index {
		return false
	}
	literal, ok := call.Args[index].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return false
	}
	value, err := strconv.Unquote(literal.Value)
	return err == nil && value == "git"
}

// scrubbing reports whether expr calls gitenv.Clean, gitenv.Environ or a package-local
// cleanEnv -- directly, or through a variable assigned from one in body.
func scrubbing(expr ast.Expr, body *ast.BlockStmt, seen map[string]bool) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.CallExpr:
			if name, ok := n.Fun.(*ast.Ident); ok && name.Name == "cleanEnv" {
				found = true
			}
			if selector, ok := n.Fun.(*ast.SelectorExpr); ok && (selector.Sel.Name == "Clean" || selector.Sel.Name == "Environ") {
				if pkg, ok := selector.X.(*ast.Ident); ok && pkg.Name == "gitenv" {
					found = true
				}
			}
		case *ast.Ident:
			if !seen[n.Name] && assignedScrubbing(n.Name, body, seen) {
				found = true
			}
		}
		return !found
	})
	return found
}

func assignedScrubbing(name string, body *ast.BlockStmt, seen map[string]bool) bool {
	seen[name] = true
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok || found {
			return !found
		}
		for i, lhs := range assign.Lhs {
			target, ok := lhs.(*ast.Ident)
			if ok && target.Name == name && i < len(assign.Rhs) && scrubbing(assign.Rhs[i], body, seen) {
				found = true
			}
		}
		return !found
	})
	return found
}

// envScrubbed reports whether one of the statements that follow a
// `name := exec.Command("git", ...)` sets name.Env from a scrubbed
// environment before name is assigned again.
func envScrubbed(following []ast.Stmt, name string, body *ast.BlockStmt) bool {
	for _, stmt := range following {
		assign, ok := stmt.(*ast.AssignStmt)
		if !ok {
			continue
		}
		for i, lhs := range assign.Lhs {
			switch target := lhs.(type) {
			case *ast.SelectorExpr:
				owner, ok := target.X.(*ast.Ident)
				if ok && owner.Name == name && target.Sel.Name == "Env" && i < len(assign.Rhs) {
					return scrubbing(assign.Rhs[i], body, map[string]bool{})
				}
			case *ast.Ident:
				if target.Name == name {
					return false
				}
			}
		}
	}
	return false
}

func unscrubbedInFunction(fset *token.FileSet, root string, body *ast.BlockStmt) []string {
	protected := map[ast.Expr]bool{}
	ast.Inspect(body, func(node ast.Node) bool {
		block, ok := node.(*ast.BlockStmt)
		if !ok {
			return true
		}
		for i, stmt := range block.List {
			assign, ok := stmt.(*ast.AssignStmt)
			if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 || !gitCall(assign.Rhs[0]) {
				continue
			}
			if name, ok := assign.Lhs[0].(*ast.Ident); ok && envScrubbed(block.List[i+1:], name.Name, body) {
				protected[assign.Rhs[0]] = true
			}
		}
		return true
	})

	var offenders []string
	ast.Inspect(body, func(node ast.Node) bool {
		if expr, ok := node.(ast.Expr); ok && gitCall(expr) && !protected[expr] {
			position := fset.Position(expr.Pos())
			relative, _ := filepath.Rel(root, position.Filename)
			offenders = append(offenders, fmt.Sprintf("%s:%d", filepath.ToSlash(relative), position.Line))
		}
		return true
	})
	return offenders
}

// Read off the source rather than trusted to review, as
// tests/test_git_env.py does for the Python suite: an inherited environment
// is invisible at the call site, and it costs the real repository's index,
// config or branch when it comes back.
func TestNoGitCallRunsWithAnInheritedEnvironment(t *testing.T) {
	root := repoRoot()
	fset := token.NewFileSet()
	var offenders []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			if function, ok := decl.(*ast.FuncDecl); ok && function.Body != nil {
				offenders = append(offenders, unscrubbedInFunction(fset, root, function.Body)...)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) > 0 {
		t.Fatalf("git runs with an inherited environment at:\n%s", strings.Join(offenders, "\n"))
	}
}
