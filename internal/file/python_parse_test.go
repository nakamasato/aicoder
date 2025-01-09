package file

import (
	"os"
	"testing"
)

func TestParsePython(t *testing.T) {
	pythonCode := `
def foo():
    pass

x = 10
`
	filePath := "test.py"
	err := os.WriteFile(filePath, []byte(pythonCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create test Python file: %v", err)
	}
	defer os.Remove(filePath)

	functions, variables, err := ParsePython(filePath)
	if err != nil {
		t.Fatalf("ParsePython returned an error: %v", err)
	}

	if len(functions) != 1 || functions[0].Name != "foo" {
		t.Errorf("Expected one function named 'foo', got: %v", functions)
	}

	if len(variables) != 1 || variables[0].Name != "x" {
		t.Errorf("Expected one variable named 'x', got: %v", variables)
	}
}
