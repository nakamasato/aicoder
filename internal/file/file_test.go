package file_test

import (
	"os"
	"testing"

	"github.com/nakamasato/aicoder/internal/file"
)

type TestStruct struct {
	Name string
	Age  int
}

func TestSaveObject(t *testing.T) {
	obj := TestStruct{Name: "Alice", Age: 30}
	outputFile := "test_output.json"

	err := file.SaveObject(obj, outputFile)
	if err != nil {
		t.Fatalf("SaveObject failed: %v", err)
	}

	// Clean up
	defer os.Remove(outputFile)

	// Check if file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("Expected file %s to exist", outputFile)
	}
}

func TestReadObject(t *testing.T) {
	inputFile := "test_input.json"
	expectedObj := TestStruct{Name: "Bob", Age: 25}

	// Create a test file
	err := file.SaveObject(expectedObj, inputFile)
	if err != nil {
		t.Fatalf("Failed to create test input file: %v", err)
	}

	// Clean up
	defer os.Remove(inputFile)

	var resultObj TestStruct
	err = file.ReadObject(inputFile, &resultObj)
	if err != nil {
		t.Fatalf("ReadObject failed: %v", err)
	}

	if resultObj != expectedObj {
		t.Fatalf("Expected %v, got %v", expectedObj, resultObj)
	}
}
