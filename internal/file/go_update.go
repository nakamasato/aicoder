package file

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// ExtractFuncBodyStatements は、Goコードのソースコンテンツから特定の関数のステートメントを返します。
func ExtractFuncBodyStatements(source, funcName string) ([]ast.Stmt, error) {
	// ソースコードを解析
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", source, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}

	// 指定された関数名を探してその本体を返す
	for _, decl := range node.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == funcName {
			if funcDecl.Body != nil {
				return funcDecl.Body.List, nil
			}
			return nil, fmt.Errorf("function %q has no body", funcName)
		}
	}

	return nil, fmt.Errorf("function %q not found", funcName)
}
