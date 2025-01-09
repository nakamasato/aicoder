package file

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"path/filepath"
)

type Function struct {
	Name      string
	StartLine int
	EndLine   int
}

type Var struct {
	Name      string
	StartLine int
	EndLine   int
}

// ParseGo parses a go file and returns the functions and variables with their line ranges.
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
			startPos := fs.Position(fn.Pos())
			endPos := fs.Position(fn.End())
			functions = append(functions, Function{
				Name:      fn.Name.Name,
				StartLine: startPos.Line,
				EndLine:   endPos.Line,
			})
		}
		// vars
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
			for _, spec := range genDecl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range valueSpec.Names {
						startPos := fs.Position(name.Pos())
						endPos := fs.Position(name.End())
						variables = append(variables, Var{
							Name:      name.Name,
							StartLine: startPos.Line,
							EndLine:   endPos.Line,
						})
					}
				}
			}
		}
		return true
	})

	return functions, variables, nil
}
