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
			if change.Block.TargetType == "function" {
				err := file.UpdateFuncGo(targetPath, change.Block.TargetName, change.NewContent, change.NewComment)
				if err != nil {
					return fmt.Errorf("failed to apply change to temp file (%s): %w", targetPath, err)
				}
			} else {
				return fmt.Errorf("unsupported target type: %s", change.Block.TargetType)
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
			err = file.UpdateBlock(f, change.Block.TargetType, change.Block.TargetName, change.NewContent, nil) // targetname is strings.Join(block.Labels(), ",") and newComments is not implemented yet
			if err != nil {
				return fmt.Errorf("failed to update block (%s): %w", targetPath, err)
			}
			if err := os.WriteFile(targetPath, f.Bytes(), 0644); err != nil {
				return fmt.Errorf("failed to write updated file: %w", err)
			}
		} else if change.Block.TargetType == "file" {
			// Apply change to file
			err := ApplyChangeFilePlan(&change, targetPath)
			if err != nil {
				return fmt.Errorf("failed to apply change to file (%s): %w", targetPath, err)
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

// GetFileContent retrieves the content of the specified file.
func GetFileContent(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func ApplyChangeFilePlan(change *planner.BlockChange, targetPath string) error {
	if err := os.WriteFile(targetPath, []byte(change.NewContent), 0644); err != nil {
		return fmt.Errorf("failed to create new file: %w", err)
	}
	fmt.Printf("Successfully created new file: %s\n", targetPath)
	return nil
}
