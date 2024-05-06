// Package exitcheck Analyzer for usage of os.Exit function directly in main function.
package exitcheck

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "main_os_exit_check",
	Doc:  "check if main function uses os.exit",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Map to store package names based on import paths.
	packages := make(map[string]string)
	dotPackages := make(map[string]bool)

	expr := func(x *ast.ExprStmt) {
		if call, ok := x.X.(*ast.CallExpr); ok {
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				if fun.Name == "Exit" && dotPackages["os"] {
					pass.Reportf(fun.NamePos, "os.Exit call in main function via dot import")
				}
			case *ast.SelectorExpr:
				if fun.Sel.Name != "Exit" {
					return
				}

				if ident, ok := fun.X.(*ast.Ident); ok {
					if pkg, ok := packages[ident.Name]; ok && pkg == "os" {
						if ident.Name == "os" {
							pass.Reportf(ident.NamePos, "os.Exit call in main function")
						} else {
							pass.Reportf(ident.NamePos, "os.Exit call in main function via alias")
						}
					}
				}
			}
		}
	}

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
				for _, spec := range genDecl.Specs {
					if importSpec, ok := spec.(*ast.ImportSpec); ok {
						path := importSpec.Path.Value
						packageName := path[1 : len(path)-1]
						if importSpec.Name != nil && importSpec.Name.Name == "." {
							// if it's a dot import save package
							dotPackages[packageName] = true
						} else if importSpec.Name != nil {
							// If the import has an alias, use the alias as the package name.
							packages[importSpec.Name.Name] = packageName
						} else {
							// If there's no alias, extract the package name from the import path.
							packages[packageName] = packageName
						}
					}
				}
			}
		}

		// Iterating all AST nodes with ast.Inspect
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.FuncDecl: // statement
				if x.Name.Name == "main" {
					ast.Inspect(x.Body, func(stmtNode ast.Node) bool {
						// Check if the statement is found.
						if stmt, ok := stmtNode.(*ast.ExprStmt); ok {
							expr(stmt)
							// Return false to stop further traversal.
							return false
						}
						return true // Continue traversing
					})
				}
			}
			return true
		})
	}
	return nil, nil
}
