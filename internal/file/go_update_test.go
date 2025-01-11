package file

import (
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"strings"
	"testing"
)

func TestUpdateFuncGo(t *testing.T) {
	// テスト用Goファイルの内容
	originalContent := `
package main

import "fmt"

func greet(name string) {
	fmt.Printf("Hello, %s!\n", name)
}
`

	newContent := `fmt.Printf("Hi, %s! Welcome back.\n", name)`

	// temporary file
	tmpFile, err := os.CreateTemp("", "*.go")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) // remove file after test

	if _, err := tmpFile.Write([]byte(originalContent)); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temporary file: %v", err)
	}

	err = UpdateFuncGo(tmpFile.Name(), "greet", newContent)
	if err != nil {
		t.Fatalf("UpdateFuncGo returned an error: %v", err)
	}

	// read updated file
	updatedContent, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	// not formatting well
	expectedContent := `
        package main

        import "fmt"

        func greet(name string) {
        	fmt.Printf("Hi, %s! Welcome back.\n",

        			name)
        }
`

	// 更新後の期待される結果をフォーマット
	formattedExpectedContent, err := format.Source([]byte(expectedContent))
	if err != nil {
		t.Fatalf("Failed to format expected content: %v", err)
	}

	// 更新後のコードをフォーマット
	formattedUpdatedContent, err := format.Source(updatedContent)
	if err != nil {
		t.Fatalf("Failed to format updated content: %v", err)
	}

	// フォーマット後の結果を比較
	updatedContentStr := strings.TrimSpace(string(formattedUpdatedContent))
	expectedContentStr := strings.TrimSpace(string(formattedExpectedContent))

	if updatedContentStr != expectedContentStr {
		t.Errorf("Updated content does not match expected content.\nExpected:\n%s\nGot:\n%s", expectedContentStr, updatedContentStr)
	}
}

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