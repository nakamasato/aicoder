package file

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"path/filepath"
)

// ParseGo parses a go file and returns the functions in the file.
func ParseGo(path string) ([]Function, []Var, error) {

	fs := token.NewFileSet()
	var functions []Function
	var variables []Var

	if filepath.Ext(path) != ".go" {
		return functions, variables, fmt.Errorf("file is not a go file: %s", path)
	}

	node, err := parser.ParseFile(fs, path, nil, parser.AllErrors)
	if err != nil {
		log.Printf("Failed to parse file: %s, error: %v", path, err)
		return functions, variables, err
	}

	ast.Inspect(node, func(n ast.Node) bool {
		// funcs
		if fn, ok := n.(*ast.FuncDecl); ok {
			pos := fs.Position(fn.Pos())
			functions = append(functions, Function{
				Name: fn.Name.Name,
				Line: pos.Line,
			})
		}
		// vars
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
			for _, spec := range genDecl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range valueSpec.Names {
						pos := fs.Position(name.Pos())
						variables = append(variables, Var{
							Name: name.Name,
							Line: pos.Line,
						})
					}
				}
			}
		}
		return true
	})

	return functions, variables, nil
}
