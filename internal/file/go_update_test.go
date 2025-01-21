package file

import (
	"go/ast"
	"go/format"
	"go/token"
	"strings"
	"testing"
)

func TestExtractFuncBodyStatements(t *testing.T) {
	source := `
		package main

		func TestFunction() {
			a := 1
			b := 2
			c := a + b
		}

		func AnotherFunction() {
			x := 10
		}
	`

	tests := []struct {
		funcName     string
		expectedStmt []string
		expectError  bool
	}{
		{
			funcName: "TestFunction",
			expectedStmt: []string{
				"1",
				"2",
				"b",
			},
			expectError: false,
		},
		{
			funcName: "AnotherFunction",
			expectedStmt: []string{
				"10",
			},
			expectError: false,
		},
		{
			funcName:     "NonExistentFunction",
			expectedStmt: nil,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.funcName, func(t *testing.T) {
			stmts, err := ExtractFuncBodyStatements(source, tt.funcName)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}
			if len(stmts) != len(tt.expectedStmt) {
				t.Errorf("expected %d statements, got %d", len(tt.expectedStmt), len(stmts))
			}

			for i, stmt := range stmts {
				fset := token.NewFileSet()
				var actualStmt string
				ast.Inspect(stmt, func(n ast.Node) bool {
					if n != nil {
						actualStmt = nodeToString(fset, n)
					}
					return true
				})

				if actualStmt != tt.expectedStmt[i] {
					t.Errorf("expected statement: %q, got: %q", tt.expectedStmt[i], actualStmt)
				}
			}
		})
	}
}

// nodeToString converts an AST node to its string representation.
func nodeToString(fset *token.FileSet, node ast.Node) string {
	var sb strings.Builder
	err := format.Node(&sb, fset, node)
	if err != nil {
		return ""
	}
	return sb.String()
}
