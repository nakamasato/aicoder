package applier

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/nakamasato/aicoder/internal/planner"
	"github.com/sergi/go-diff/diffmatchpatch"
	"golang.org/x/sync/errgroup"
)

// ApplyChanges applies changes based on the provided changesPlan.
// If dryrun is true, it displays the diff without modifying the actual files.
func ApplyChanges(changesPlan planner.ChangesPlan, dryrun bool) error {
	// Group changes by file
	fileChanges := groupChangesByFile(changesPlan.Changes)

	var mu sync.Mutex
	var diffs []string

	var g errgroup.Group

	for filePath, changes := range fileChanges {
		filePath := filePath // Correctly reference within closure
		changes := changes   // Correctly reference within closure
		g.Go(func() error {
			var diff string
			var err error

			if dryrun {
				// Generate temporary file path
				tempFilePath := filepath.Join(os.TempDir(), generateTempFileName(filePath))

				// Retrieve original file content
				originalContent, err := GetFileContent(filePath)
				if err != nil {
					return fmt.Errorf("failed to read original file (%s): %w", filePath, err)
				}

				// Apply changes to get modified content
				modifiedContent, err := applyAllChanges(originalContent, changes)
				if err != nil {
					return fmt.Errorf("failed to apply changes (%s): %w", filePath, err)
				}

				// Save modified content to temporary file
				if err := os.WriteFile(tempFilePath, modifiedContent, 0644); err != nil {
					return fmt.Errorf("failed to write to temporary file (%s): %w", tempFilePath, err)
				}
				defer os.Remove(tempFilePath) // Cleanup

				// Generate diff
				diff = GenerateGitDiff(originalContent, modifiedContent, filePath)

				// Collect diffs
				mu.Lock()
				diffs = append(diffs, diff)
				mu.Unlock()
			} else {
				// Apply changes to the actual file
				err = applyAllChangesToFile(filePath, changes)
				if err != nil {
					return fmt.Errorf("failed to apply changes (%s): %w", filePath, err)
				}
			}

			return nil
		})
	}

	// Wait for all goroutines to finish
	if err := g.Wait(); err != nil {
		return err
	}

	if dryrun {
		// Display collected diffs
		for _, diff := range diffs {
			fmt.Println(diff)
		}
	}

	return nil
}

// groupChangesByFile groups changes by their file paths.
func groupChangesByFile(changes []planner.Change) map[string][]planner.Change {
	fileChanges := make(map[string][]planner.Change)
	for _, change := range changes {
		fileChanges[change.Path] = append(fileChanges[change.Path], change)
	}
	return fileChanges
}

// generateTempFileName generates a temporary file name based on the original path.
func generateTempFileName(originalPath string) string {
	base := filepath.Base(originalPath)
	return fmt.Sprintf("%s.tmp", base)
}

// applyAllChanges applies a series of changes to the original content and returns the modified content.
func applyAllChanges(original []byte, changes []planner.Change) ([]byte, error) {
	lines := strings.Split(string(original), "\n")

	for _, change := range changes {
		if change.LineNum == 0 { // Create a new file
			if change.Add == "" {
				return nil, fmt.Errorf("add content is required to create a new file (%s)", change.Path)
			}
			lines = append(lines, change.Add)
		} else if change.LineNum > 0 && change.LineNum <= len(lines) {
			if change.Delete != "" {
				lines[change.LineNum-1] = strings.Replace(lines[change.LineNum-1], change.Delete, "", 1)
			}
			if change.Add != "" {
				lines[change.LineNum-1] = lines[change.LineNum-1] + change.Add
			}
		} else {
			return nil, fmt.Errorf("line number %d out of range (%s)", change.LineNum, change.Path)
		}
	}

	modifiedContent := strings.Join(lines, "\n")
	return []byte(modifiedContent), nil
}

// applyAllChangesToFile applies a series of changes directly to the specified file.
func applyAllChangesToFile(filePath string, changes []planner.Change) error {
	// Retrieve original file content
	originalContent, err := GetFileContent(filePath)
	if err != nil {
		return fmt.Errorf("failed to read original file: %w", err)
	}

	// Apply changes
	modifiedContent, err := applyAllChanges(originalContent, changes)
	if err != nil {
		return err
	}

	// Write modified content back to the file
	if err := os.WriteFile(filePath, modifiedContent, 0644); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// GenerateGitDiff generates a git diff style string between the original and modified content.
func GenerateGitDiff(original, modified []byte, filePath string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(original), string(modified), false)
	dmp.DiffCleanupSemantic(diffs)

	var buffer bytes.Buffer

	// Add headers
	buffer.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))
	buffer.WriteString(fmt.Sprintf("--- a/%s\n", filePath))
	buffer.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))

	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			color.New(color.FgGreen).Fprintf(&buffer, "+ %s\n", diff.Text)
		case diffmatchpatch.DiffDelete:
			color.New(color.FgRed).Fprintf(&buffer, "- %s\n", diff.Text)
		case diffmatchpatch.DiffEqual:
			// In git diff, unchanged lines are prefixed with two spaces
			buffer.WriteString(fmt.Sprintf("  %s\n", diff.Text))
		}
	}

	return buffer.String()
}

// GetFileContent retrieves the content of the specified file.
func GetFileContent(path string) ([]byte, error) {
	return os.ReadFile(path)
}
