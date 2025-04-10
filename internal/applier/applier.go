package applier

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/nakamasato/aicoder/internal/planner"
)

// Applier defines the interface for applying changes to different types of files.
// Implementations should handle reading from r, applying the change c, and returning
// the modified content as bytes.
type Applier interface {
	Apply(r io.Reader, c planner.BlockChange) ([]byte, error)
}

// ApplyChanges applies changes based on the provided changesPlan.
// If dryrun is true, it displays the diffs without modifying the actual files.
func ApplyChanges(changesPlan *planner.ChangesPlan, dryrun bool) error {

	goAplr := &goApplier{}
	hclAplr := &hclApplier{}

	var diffs []string

	for _, change := range changesPlan.Changes {
		// Capture the current value of change to avoid closure issues
		if !isValidFileType(change.Block.Path, change.Block.TargetType) {
			return fmt.Errorf("unsupported file type: %s", change.Block.Path)
		}

		change := change
		targetPath := change.Block.Path
		f, err := os.OpenFile(targetPath, os.O_RDWR, 0644)
		if err != nil {
			return fmt.Errorf("failed to open file (%s): %w", targetPath, err)
		}
		defer f.Close()

		var data []byte
		if filepath.Ext(change.Block.Path) == ".go" {
			data, err = goAplr.Apply(f, change)
			if err != nil {
				return fmt.Errorf("failed to apply change to go file (%s): %w", targetPath, err)
			}
		} else if filepath.Ext(change.Block.Path) == ".hcl" || filepath.Ext(change.Block.Path) == ".tf" {
			data, err = hclAplr.Apply(f, change)
			if err != nil {
				return fmt.Errorf("failed to apply change to hcl file (%s): %w", targetPath, err)
			}
		} else if change.Block.TargetType == "file" {
			// Apply change to file
			data = []byte(change.NewContent)
		}

		if dryrun {
			// Generate diff
			originalContent, err := os.ReadFile(change.Block.Path)
			if err != nil {
				return fmt.Errorf("failed to read original file (%s): %w", change.Block.Path, err)
			}
			diff, err := generateDiff(originalContent, data)
			if err != nil {
				return fmt.Errorf("failed to generate diff: %w", err)
			}
			diffs = append(diffs, diff)
		} else {
			// Reset the file pointer and truncate the file
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("failed to seek to the beginning of file (%s): %w", targetPath, err)
			}
			if err := f.Truncate(0); err != nil {
				return fmt.Errorf("failed to truncate file (%s): %w", targetPath, err)
			}
			if _, err := f.Write(data); err != nil {
				return fmt.Errorf("failed to write to file (%s): %w", targetPath, err)
			}
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

func isValidFileType(path string, targetType string) bool {
	ext := filepath.Ext(path)
	return ext == ".go" || ext == ".hcl" || ext == ".tf" || targetType == "file"
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
		fmt.Println("There is diff")
	}
	return string(output), nil
}

// GetFileContent retrieves the content of the specified file.
func GetFileContent(path string) ([]byte, error) {
	return os.ReadFile(path)
}
