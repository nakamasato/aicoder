package applier

import (
	"bytes"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestUpdateFuncGo(t *testing.T) {
	// テスト用Goファイルの内容
	originalContent := `
package main

import "fmt"

// greet receives a name and prints a greeting message.
func greet(name string) {
	fmt.Printf("Hello, %s!\n", name)
}
`

	newContent := `fmt.Printf("Hi, %s! Welcome back.\n", name)`

	// Use a buffer to simulate file I/O
	var inputBuffer bytes.Buffer
	inputBuffer.WriteString(originalContent)

	b, err := updateFuncGo(&inputBuffer, "greet", newContent, "greet function greet with welcome message")
	if err != nil {
		t.Fatalf("UpdateFuncGo returned an error: %v", err)
	}

	// read updated content from the output buffer
	updatedContent := b

	// not formatting well
	expectedContent := `
        package main

        import "fmt"

		// greet function greet with welcome message
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

func TestUpdateFuncBody(t *testing.T) {
	source := `
package main

func greet(name string) {
	fmt.Printf("Hello, %s!\n", name)
}
`
	newContent := `
fmt.Printf("Hi, %s! Welcome back.\n", name)
`

	expectedContent := `package main

func greet(name string) {
	fmt.
		Printf("Hi, %s! Welcome back.\n",
			name)
}`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", source, parser.AllErrors)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	functionFound := updateFuncBody(node, "greet", newContent)
	if !functionFound {
		t.Fatalf("Function greet not found")
	}

	var buf strings.Builder
	if err := format.Node(&buf, fset, node); err != nil {
		t.Fatalf("Failed to format updated node: %v", err)
	}

	updatedContent := buf.String()
	if !strings.Contains(updatedContent, expectedContent) {
		t.Errorf("Updated function body does not contain expected content.\nExpected:\n%s\nGot:\n%s", expectedContent, updatedContent)
	}
}

func TestUpdateFuncComment(t *testing.T) {
	source := `
package main

// greet receives a name and prints a greeting message.
func greet(name string) {
	fmt.Printf("Hello, %s!\n", name)
}
`
	expectedComment := "greet function greet with welcome message"

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", source, parser.AllErrors)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	updateFuncComment(node, "greet", expectedComment)

	var buf strings.Builder
	if err := format.Node(&buf, fset, node); err != nil {
		t.Fatalf("Failed to format updated node: %v", err)
	}

	updatedContent := buf.String()
	if !strings.Contains(updatedContent, "// "+expectedComment) {
		t.Errorf("Updated function comment does not contain expected comment.\nExpected:\n%s\nGot:\n%s", expectedComment, updatedContent)
	}
}
