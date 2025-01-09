package file

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// ParsePython parses a Python file and returns the functions and variables in the file.
func ParsePython(path string) ([]Function, []Var, error) {

	pythonScript := `
import ast
import json
import sys

def parse_file(file_path):
    with open(file_path, "r") as f:
        code = f.read()
    parsed = ast.parse(code)
    functions = []
    variables = []
    for node in ast.walk(parsed):
        if isinstance(node, ast.FunctionDef):
            functions.append({"name": node.name, "line": node.lineno})
        elif isinstance(node, ast.Assign):
            for target in node.targets:
                if isinstance(target, ast.Name):
                    variables.append({"name": target.id, "line": node.lineno})
    return {"functions": functions, "variables": variables}

if __name__ == "__main__":
    file_path = sys.argv[1]
    result = parse_file(file_path)
    print(json.dumps(result))
`

	// Pythonスクリプトを実行
	cmd := exec.Command("python3", "-c", pythonScript, path)
	var out bytes.Buffer
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, nil, fmt.Errorf("error running Python script: %s", stderr.String())
	}

	// 結果を解析
	var result struct {
		Functions []Function `json:"functions"`
		Variables []Var      `json:"variables"`
	}
	err = json.Unmarshal(out.Bytes(), &result)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing Python script output: %v", err)
	}

	return result.Functions, result.Variables, nil
}
