// Package errcheck Analyzer for unhandled/ignored return errors
package errcheck

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "errcheck",
	Doc:  "check for unchecked errors",
	Run:  run,
}

var errorType = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

func isErrorType(t types.Type) bool {
	return types.Implements(t, errorType)
}

// resultErrors returns boolean array indicating if any return value is an error
func resultErrors(pass *analysis.Pass, call *ast.CallExpr) []bool {
	switch t := pass.TypesInfo.Types[call].Type.(type) {
	case *types.Named: // value
		return []bool{isErrorType(t)}
	case *types.Pointer: // pointer
		return []bool{isErrorType(t)}
	case *types.Tuple: // multiple return variables
		s := make([]bool, t.Len())
		for i := 0; i < t.Len(); i++ {
			switch mt := t.At(i).Type().(type) {
			case *types.Named:
				s[i] = isErrorType(mt)
			case *types.Pointer:
				s[i] = isErrorType(mt)
			}
		}
		return s
	}
	return []bool{false}
}

// isPrintFunction reutrns true if expresion is print function
func isPrintFunction(c *ast.CallExpr) bool {
	const prefix = "Print"
	if id, ok := c.Fun.(*ast.Ident); ok {
		return strings.HasPrefix(id.Name, prefix)
	}

	if sel, ok := c.Fun.(*ast.SelectorExpr); ok {
		return strings.HasPrefix(sel.Sel.Name, prefix)
	}

	return false
}

// isReturnError returns true, if any of return values is an error.
func isReturnError(pass *analysis.Pass, call *ast.CallExpr) bool {
	for _, isError := range resultErrors(pass, call) {
		if isError {
			return true
		}
	}
	return false
}

func run(pass *analysis.Pass) (interface{}, error) {
	expr := func(x *ast.ExprStmt) {
		// Check that statement is a function call that
		// has unchecked error
		if call, ok := x.X.(*ast.CallExpr); ok {
			if isPrintFunction(call) {
				return
			}
			if isReturnError(pass, call) {
				pass.Reportf(x.Pos(), "expression returns unchecked error")
			}
		}
	}
	tuplefunc := func(x *ast.AssignStmt) {
		// analysing assignment that ignores error with '_'
		// e.g: a, b, _ := tuplefunc()
		// check that it's a function call
		if call, ok := x.Rhs[0].(*ast.CallExpr); ok {
			if isPrintFunction(call) {
				return
			}
			results := resultErrors(pass, call)
			for i := 0; i < len(x.Lhs); i++ {
				// Check all identificators on the left hand side of assignment
				if id, ok := x.Lhs[i].(*ast.Ident); ok && id.Name == "_" && results[i] {
					pass.Reportf(id.NamePos, "assignment with unchecked error")
				}
			}
		}
	}
	errfunc := func(x *ast.AssignStmt) {
		// multiple assignment: a, _ := b, myfunc()
		// analysing assignment that ignores error with '_'
		for i := 0; i < len(x.Lhs); i++ {
			if id, ok := x.Lhs[i].(*ast.Ident); ok {
				// function call on the right hand side
				if call, ok := x.Rhs[i].(*ast.CallExpr); ok {
					if isPrintFunction(call) {
						return
					}
					if id.Name == "_" && isReturnError(pass, call) {
						pass.Reportf(id.NamePos, "assignment with unchecked error")
					}
				}
			}
		}
	}

	deferFunc := func(x *ast.DeferStmt) {
		if isPrintFunction(x.Call) {
			return
		}
		if isReturnError(pass, x.Call) {
			pass.Reportf(x.Pos(), "defer statement with unchecked error")
		}
	}

	goFunc := func(x *ast.GoStmt) {
		if isPrintFunction(x.Call) {
			return
		}
		if isReturnError(pass, x.Call) {
			pass.Reportf(x.Pos(), "go statement with unchecked error")
		}
	}

	for _, file := range pass.Files {
		// Iterating all AST nodes with ast.Inspect
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.ExprStmt: // statement
				expr(x)
			case *ast.AssignStmt: // assigmnent operator
				// one statement on the right hand side: x,y := myfunc()
				if len(x.Rhs) == 1 {
					tuplefunc(x)
				} else {
					// several statements on the right hand side: x,y := z,myfunc()
					errfunc(x)
				}
			case *ast.DeferStmt:
				deferFunc(x)
			case *ast.GoStmt:
				goFunc(x)
			}
			return true
		})
	}
	return nil, nil
}
