package applier

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/nakamasato/aicoder/internal/planner"
	"golang.org/x/sync/errgroup"
)

// ApplyChanges applies changes based on the provided changesPlan.
// If dryrun is true, it displays the diffs without modifying the actual files.
func ApplyChanges(changesPlan planner.ChangesPlan, dryrun bool) error {
	var g errgroup.Group
	var mu sync.Mutex
	var diffs []string

	for _, change := range changesPlan.Changes {
		// Capture the current value of change to avoid closure issues
		change := change
		g.Go(func() error {
			targetPath := change.Path
			if dryrun {
				// Generate temporary file path
				targetPath = change.Path + ".tmp"
			}
			// Apply change to temp file
			_, _, err := applyChange(change, targetPath)
			if err != nil {
				return fmt.Errorf("failed to apply change to temp file (%s): %w", targetPath, err)
			}

			if dryrun {
				cmd := exec.Command("diff", "-u", change.Path, targetPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					log.Println("There is diff")
				}

				mu.Lock()
				diffs = append(diffs, string(output))
				mu.Unlock()
			}

			return nil
		})
	}

	// Wait for all goroutines to finish
	if err := g.Wait(); err != nil {
		return err
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
				log.Printf("Error reading diff output: %v", err)
			}
		}
	}

	return nil
}

func applyChange(change planner.Change, targetPath string) (originalContent, modifiedContent []byte, err error) {
	if _, err := os.Stat(change.Path); os.IsNotExist(err) {
		return createNewFile(change, targetPath)
	}
	return updateExistingFile(change, targetPath)
}

func createNewFile(change planner.Change, targetPath string) (originalContent, modifiedContent []byte, err error) {

	if change.LineNum != 0 {
		return originalContent, modifiedContent, fmt.Errorf("line number must be 0 for new files")
	}

	if change.Add == "" {
		return originalContent, modifiedContent, fmt.Errorf("add content is required for new files")
	}

	if err := os.WriteFile(targetPath, []byte(change.Add), 0644); err != nil {
		return originalContent, modifiedContent, fmt.Errorf("failed to create new file: %w", err)
	}
	fmt.Printf("Successfully created new file: %s\n", targetPath)
	return originalContent, []byte(change.Add), nil
}

// applyChange applies a single change to the specified file.
// If targetPath is a temporary file (during dryrun), it writes the modified content there.
// Otherwise, it writes directly to the original file.
func updateExistingFile(change planner.Change, targetPath string) (originalContent, modifiedContent []byte, err error) {
	// Open the original file
	file, err := os.Open(change.Path)
	if err != nil {
		return originalContent, modifiedContent, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		bytes := scanner.Bytes()
		originalContent = append(originalContent, bytes...)
		lines = append(lines, string(bytes))
	}

	if err := scanner.Err(); err != nil {
		return originalContent, modifiedContent, fmt.Errorf("failed to read file: %w", err)
	}

	// Apply the change
	if change.LineNum > 0 && change.LineNum <= len(lines) {
		if change.Delete != "" {
			lines[change.LineNum-1] = strings.Replace(lines[change.LineNum-1], change.Delete, "", 1)
		}
		if change.Add != "" {
			lines[change.LineNum-1] = lines[change.LineNum-1] + change.Add
		}
	} else {
		return originalContent, modifiedContent, fmt.Errorf("line number %d out of range", change.LineNum)
	}

	// Join the lines back into a single string
	output := strings.Join(lines, "\n")

	// Write the changes back to the target file
	if err := os.WriteFile(targetPath, []byte(output), 0644); err != nil {
		return originalContent, modifiedContent, fmt.Errorf("failed to write file: %w", err)
	}
	fmt.Printf("Successfully updated existing file: %s\n", targetPath)

	return originalContent, []byte(output), nil
}

// GetFileContent retrieves the content of the specified file.
func GetFileContent(path string) ([]byte, error) {
	return os.ReadFile(path)
}
