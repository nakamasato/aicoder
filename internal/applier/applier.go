package applier

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/planner"
)

// ApplyChanges applies changes based on the provided changesPlan.
// If dryrun is true, it displays the diffs without modifying the actual files.
func ApplyChanges(changesPlan *planner.ChangesPlan, dryrun bool) error {
	var diffs []string

	for _, change := range changesPlan.Changes {
		// Capture the current value of change to avoid closure issues
		change := change
		targetPath := change.Block.Path
		if dryrun {
			originalContent, err := os.ReadFile(change.Block.Path)
			if err != nil {
				return fmt.Errorf("failed to read original file (%s): %w", change.Block.Path, err)
			}
			// Apply change in memory
			modifiedContent, err := file.UpdateFuncInMemory(originalContent, change.Block.TargetName, change.NewContent)
			if err != nil {
				return fmt.Errorf("failed to apply change in memory: %w", err)
			}

			// Generate diff
			diff, err := generateDiff(originalContent, modifiedContent)
			if err != nil {
				return fmt.Errorf("failed to generate diff: %w", err)
			}
			diffs = append(diffs, diff)
		} else if filepath.Ext(change.Block.Path) == ".go" {
			// Apply change to temp file
			err := file.UpdateFuncGo(targetPath, change.Block.TargetName, change.NewContent, "")
			if err != nil {
				return fmt.Errorf("failed to apply change to temp file (%s): %w", targetPath, err)
			}
		} else if filepath.Ext(change.Block.Path) == ".hcl" || filepath.Ext(change.Block.Path) == ".tf" {
			// Apply change to HCL file
			// TODO: improve hcl_parse.go and hcl_update.go
			src, err := os.ReadFile(targetPath)
			if err != nil {
				return fmt.Errorf("failed to read original file (%s): %w", targetPath, err)
			}
			f, diag := hclwrite.ParseConfig(src, targetPath, hcl.InitialPos)
			if diag.HasErrors() {
				return fmt.Errorf("failed to parse HCL file: %s, error: %v", targetPath, diag.Error())
			}
			err = file.UpdateBlock(f, change.Block.TargetType, change.Block.TargetName, change.NewContent) // targetname is strings.Join(block.Labels(), ",")
			if err != nil {
				return fmt.Errorf("failed to update block (%s): %w", targetPath, err)
			}
			if err := os.WriteFile(targetPath, f.Bytes(), 0644); err != nil {
				return fmt.Errorf("failed to write updated file: %w", err)
			}
		} else {
			return fmt.Errorf("unsupported file type: %s", change.Block.Path)
		}
	}

	if dryrun {
		for _, diff := range diffs {
			scanner := bufio.NewScanner(strings.NewReader(diff))
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "+") {
					color.New(color.FgGreen).Println(line)
				} else if strings.HasPrefix(line, "-") {
					color.New(color.FgRed).Println(line)
				} else {
					fmt.Println(line)
				}
			}

			if err := scanner.Err(); err != nil {
				fmt.Printf("Error reading diff output: %v", err)
			}
		}
	}

	return nil
}

// generateDiff generates a unified diff between the original and modified content.
func generateDiff(original, modified []byte) (string, error) {
	// Create temporary files for original and modified content
	originalTempFile, err := os.CreateTemp("", "original-*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file for original content: %w", err)
	}
	defer os.Remove(originalTempFile.Name()) // Clean up

	modifiedTempFile, err := os.CreateTemp("", "modified-*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file for modified content: %w", err)
	}
	defer os.Remove(modifiedTempFile.Name()) // Clean up

	// Write contents to the temporary files
	if _, err := originalTempFile.Write(original); err != nil {
		return "", fmt.Errorf("failed to write original content to temp file: %w", err)
	}
	if _, err := modifiedTempFile.Write(modified); err != nil {
		return "", fmt.Errorf("failed to write modified content to temp file: %w", err)
	}

	// Close the files to flush the content
	originalTempFile.Close()
	modifiedTempFile.Close()

	// Use diff command on the temporary files
	cmd := exec.Command("diff", "-u", originalTempFile.Name(), modifiedTempFile.Name())
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Println("There is diff")
	}
	return string(output), nil
}

// func applyChange(change planner.Change, targetPath string) (originalContent, modifiedContent []byte, err error) {
// 	if _, err := os.Stat(change.Path); os.IsNotExist(err) {
// 		return createNewFile(change, targetPath)
// 	}
// 	return updateExistingFile(change, targetPath)
// }

// func createNewFile(change planner.Change, targetPath string) (originalContent, modifiedContent []byte, err error) {

// 	if change.LineNum != 0 {
// 		return originalContent, modifiedContent, fmt.Errorf("line number must be 0 for new files")
// 	}

// 	if change.Add == "" {
// 		return originalContent, modifiedContent, fmt.Errorf("add content is required for new files")
// 	}

// 	if err := os.WriteFile(targetPath, []byte(change.Add), 0644); err != nil {
// 		return originalContent, modifiedContent, fmt.Errorf("failed to create new file: %w", err)
// 	}
// 	fmt.Printf("Successfully created new file: %s\n", targetPath)
// 	return originalContent, []byte(change.Add), nil
// }

// applyChange applies a single change to the specified file.
// If targetPath is a temporary file (during dryrun), it writes the modified content there.
// Otherwise, it writes directly to the original file.
// func updateExistingFile(change planner.Change, targetPath string) (originalContent, modifiedContent []byte, err error) {
// 	// Open the original file
// 	file, err := os.Open(change.Path)
// 	if err != nil {
// 		return originalContent, modifiedContent, fmt.Errorf("failed to open file: %w", err)
// 	}
// 	defer file.Close()

// 	var lines []string
// 	scanner := bufio.NewScanner(file)
// 	for scanner.Scan() {
// 		bytes := scanner.Bytes()
// 		originalContent = append(originalContent, bytes...)
// 		lines = append(lines, string(bytes))
// 	}

// 	if err := scanner.Err(); err != nil {
// 		return originalContent, modifiedContent, fmt.Errorf("failed to read file: %w", err)
// 	}

// 	// Apply the change
// 	if change.LineNum > 0 && change.LineNum <= len(lines) {
// 		if change.Delete != "" {
// 			lines[change.LineNum-1] = strings.Replace(lines[change.LineNum-1], change.Delete, "", 1)
// 		}
// 		if change.Add != "" {
// 			lines[change.LineNum-1] = lines[change.LineNum-1] + change.Add
// 		}
// 	} else {
// 		return originalContent, modifiedContent, fmt.Errorf("line number %d out of range", change.LineNum)
// 	}

// 	// Join the lines back into a single string
// 	output := strings.Join(lines, "\n")

// 	// Write the changes back to the target file
// 	if err := os.WriteFile(targetPath, []byte(output), 0644); err != nil {
// 		return originalContent, modifiedContent, fmt.Errorf("failed to write file: %w", err)
// 	}
// 	fmt.Printf("Successfully updated existing file: %s\n", targetPath)

// 	return originalContent, []byte(output), nil
// }

// GetFileContent retrieves the content of the specified file.
func GetFileContent(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func ApplyChangeFilePlan(change *planner.ChangeFilePlan, targetPath string) error {
	if err := os.WriteFile(targetPath, []byte(change.NewContent), 0644); err != nil {
		return fmt.Errorf("failed to create new file: %w", err)
	}
	fmt.Printf("Successfully created new file: %s\n", targetPath)
	return nil
}
