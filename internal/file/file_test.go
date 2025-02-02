package file_test

import (
	"os"
	"testing"

	"github.com/nakamasato/aicoder/internal/file"
)

func TestUpdateFuncInMemory(t *testing.T) {
	originalContent := []byte(`package main

func SampleFunction() {
	// Sample function content
}
`)

	newFunctionContent := `
func SampleFunction() {
	// Updated function content
}
`

	updatedContent, err := file.UpdateFuncInMemory(originalContent, "SampleFunction", newFunctionContent)
	if err != nil {
		t.Fatalf("Error updating function in memory: %v", err)
	}

	expectedContent := `package main


func SampleFunction() {
	// Updated function content
}

`
	if string(updatedContent) != expectedContent {
		t.Errorf("Expected updated content:\n```\n%s\n```\nGot:\n```\n%s\n```", expectedContent, string(updatedContent))
	}

	// Test for a non-existent function
	_, err = file.UpdateFuncInMemory(originalContent, "NonExistentFunction", newFunctionContent)
	if err == nil {
		t.Error("Expected error for non-existent function, got nil")
	}
}

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

func TestDefaultFileReader_ReadContent(t *testing.T) {
	// Setup: Create a temporary file with some content
	tempFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	content := "Hello, World!"
	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tempFile.Close()

	// Test: Use DefaultFileReader to read the content
	reader := file.DefaultFileReader{}
	readContent, err := reader.ReadContent(tempFile.Name())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify: Check if the content read is as expected
	if readContent != content {
		t.Errorf("expected %q, got %q", content, readContent)
	}
}

func TestExists(t *testing.T) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up

	// Test case: File exists
	if !file.Exists(tempFile.Name()) {
		t.Errorf("Expected file to exist, but it does not")
	}

	// Test case: File does not exist
	nonExistentFile := "non_existent_file.txt"
	if file.Exists(nonExistentFile) {
		t.Errorf("Expected file to not exist, but it does")
	}
}
