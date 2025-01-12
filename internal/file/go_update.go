package file

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
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

// UpdateFuncGo updates the specified function in a Go file with new content and comment.
// The function is identified by its name and the file is updated in place.
// The function comment is updated if specified. Set comment to an empty string to keep the existing comment.
func UpdateFuncGo(path, function, content, comment string) error {
	// Read the original file content
	_, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the Go file
	fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, path, nil, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("failed to parse Go file: %w", err)
	}

	// Update the function body and comment
	functionFound := UpdateFuncBody(node, function, content)
	if !functionFound {
		return fmt.Errorf("function %s not found in file %s", function, path)
	}

	// Update the function comment if specified
	if comment != "" {
		UpdateFuncComment(node, function, comment)
	}

	// Generate the updated code
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fs, node); err != nil {
		return fmt.Errorf("failed to generate updated Go code: %w", err)
	}

	// Format the updated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format updated Go code: %w", err)
	}

	// Write the updated code back to the file
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return fmt.Errorf("failed to write updated Go file: %w", err)
	}

	return nil
}

// UpdateFuncBody updates the body of the specified function in the AST.
func UpdateFuncBody(node *ast.File, function, content string) bool {
	functionFound := false

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != function {
			return true // Continue traversal
		}

		// Parse the new function body as an expression
		newExpr, err := parser.ParseExpr(fmt.Sprintf("func() { %s }", content))
		if err != nil {
			return false // Stop traversal on parse error
		}

		// Convert the parsed expression to a function declaration
		funcLit, ok := newExpr.(*ast.FuncLit)
		if !ok {
			return false // Stop traversal if not a function literal
		}

		// Extract the block statement from the function literal
		newBodyBlock := funcLit.Body

		fn.Body = newBodyBlock
		functionFound = true
		return false // Stop traversal after finding the function
	})

	return functionFound
}

// UpdateFuncComment updates the comment of the specified function in the AST.
func UpdateFuncComment(node *ast.File, function, comment string) {
	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != function {
			return true // Continue traversal
		}

		// Update the comment above the function
		fn.Doc = &ast.CommentGroup{
			List: []*ast.Comment{
				{
					Text: fmt.Sprintf("// %s", comment),
				},
			},
		}

		return false // Stop traversal after finding the function
	})
}
