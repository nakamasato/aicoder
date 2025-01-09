// parse_go_test.go
package file

import (
	"os"
	"testing"
)

func TestParseGo(t *testing.T) {
	// Create a temporary Go file for testing
	content := `
		package main

		import "fmt"

		func main() {
			fmt.Println("Hello, World!")
		}

		var x int
		var y = 2
	`
	tmpFile, err := os.CreateTemp("", "*.go")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test ParseGo function
	functions, variables, err := ParseGo(tmpFile.Name())
	if err != nil {
		t.Fatalf("ParseGo returned an error: %v", err)
	}

	// Check functions
	expectedFunctions := []Function{
		{Name: "main", Line: 6},
	}
	if len(functions) != len(expectedFunctions) {
		t.Fatalf("Expected %d functions, got %d", len(expectedFunctions), len(functions))
	}
	for i, fn := range functions {
		if fn.Name != expectedFunctions[i].Name || fn.Line != expectedFunctions[i].Line {
			t.Errorf("Expected function %v, got %v", expectedFunctions[i], fn)
		}
	}

	// Check variables
	expectedVariables := []Var{
		{Name: "x", Line: 10},
		{Name: "y", Line: 11},
	}
	if len(variables) != len(expectedVariables) {
		t.Fatalf("Expected %d variables, got %d", len(expectedVariables), len(variables))
	}
	for i, v := range variables {
		if v.Name != expectedVariables[i].Name || v.Line != expectedVariables[i].Line {
			t.Errorf("Expected variable %v, got %v", expectedVariables[i], v)
		}
	}
}
