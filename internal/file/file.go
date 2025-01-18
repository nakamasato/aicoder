package file

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type File struct {
	Path    string
	Content string
}

// UpdateFuncInMemory updates a specific function's content in memory.
func UpdateFuncInMemory(originalContent []byte, functionName, newFunctionContent string) ([]byte, error) {
	originalStr := string(originalContent)
	startMarker := fmt.Sprintf("func %s(", functionName)

	startIndex := strings.Index(originalStr, startMarker)
	if startIndex == -1 {
		return nil, fmt.Errorf("function %s not found", functionName)
	}

	// Find the end of the function by counting braces
	openBraces := 0
	endIndex := -1
	for i := startIndex; i < len(originalStr); i++ {
		if originalStr[i] == '{' {
			openBraces++
		} else if originalStr[i] == '}' {
			openBraces--
			if openBraces == 0 {
				endIndex = i + 1
				break
			}
		}
	}

	if endIndex == -1 {
		return nil, fmt.Errorf("could not determine the end of function %s", functionName)
	}

	// Replace the function content
	var buffer bytes.Buffer
	buffer.WriteString(originalStr[:startIndex])
	buffer.WriteString(newFunctionContent)
	buffer.WriteString(originalStr[endIndex:])

	return buffer.Bytes(), nil
}

// WriteFile writes the content to the file at the given path.
func SaveObject(obj interface{}, outputFile string) error {
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal obj: %w", err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}
	return nil
}

// ReadObject reads the content of the file at the given path and unmarshals it into the given object.
func ReadObject(filePath string, obj interface{}) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if err := json.Unmarshal(data, obj); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}
	return nil
}
