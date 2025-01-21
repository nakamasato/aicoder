package applier

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io"

	"github.com/nakamasato/aicoder/internal/planner"
)

type goApplier struct{}

func (a *goApplier) Apply(r io.Reader, w io.Writer, c planner.BlockChange) ([]byte, error) {

	if c.Block.TargetType == "function" {
		return updateFuncGo(r, w, c.Block.TargetName, c.NewContent, c.NewComment)
	}

	return nil, fmt.Errorf("unsupported target type: %s", c.Block.TargetType)
}

// updateFuncGo updates the specified function in a Go file with new content and comment.
// The function is identified by its name and the file is updated in place.
// The function comment is updated if specified. Set comment to an empty string to keep the existing comment.
func updateFuncGo(reader io.Reader, writer io.Writer, function, content, comment string) ([]byte, error) {
	// Read the original file content from the reader
	source, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read from reader: %w", err)
	}

	// Parse the Go file from the source
	fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, "", source, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go source: %w", err)
	}

	// Update the function body and comment
	if content != "" {
		if functionFound := updateFuncBody(node, function, content); !functionFound {
			return nil, fmt.Errorf("function %s not found", function)
		}
	}

	// Update the function comment if specified
	if comment != "" {
		updateFuncComment(node, function, comment)
	}

	// Generate the updated code
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fs, node); err != nil {
		return nil, fmt.Errorf("failed to generate updated Go code: %w", err)
	}

	// Format the updated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to format updated Go code: %w", err)
	}

	return formatted, nil
}

// updateFuncBody updates the body of the specified function in the AST.
func updateFuncBody(node *ast.File, function, content string) bool {
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

// updateFuncComment updates the comment of the specified function in the AST.
func updateFuncComment(node *ast.File, function, comment string) {
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
