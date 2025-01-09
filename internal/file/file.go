package file

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type File struct {
	Path    string
	Content string
}


// GetFunctionLines returns the start and end line numbers of a function in a file.
// Naive implementation for Golang or Java. TODO: improve the mechanism to get lines.
func GetBlockBaseFunctionLines(filePath string, functionName string) (int, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	var startLine, endLine int
	insideFunction := false
	openBraces := 0

	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// func start
		if strings.HasPrefix(trimmedLine, "func "+functionName) || strings.Contains(trimmedLine, " "+functionName+"(") {
			startLine = lineNumber
			insideFunction = true
			openBraces = strings.Count(trimmedLine, "{") - strings.Count(trimmedLine, "}")
			continue
		}

		// func end
		if insideFunction {
			openBraces += strings.Count(trimmedLine, "{")
			openBraces -= strings.Count(trimmedLine, "}")
			if openBraces == 0 {
				endLine = lineNumber
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}

	if startLine == 0 {
		return 0, 0, fmt.Errorf("function %s not found in %s", functionName, filePath)
	}
	return startLine, endLine, nil
}
