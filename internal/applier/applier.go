package applier

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nakamasato/aicoder/internal/planner"
	"golang.org/x/sync/errgroup"
)

func ApplyChanges(changesPlan planner.ChangesPlan, dryrun bool) error {
	var g errgroup.Group
	for _, change := range changesPlan.Changes {
		g.Go(func() error {
			return applyChange(change, dryrun)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to apply changes: %w", err)
	}
	return nil
}

// when dryrun is true, the changes are applied to a temporary file and the diff is shown
func applyChange(change planner.Change, dryrun bool) error {
	filepath := change.Path
	if dryrun {
		filepath = change.Path + ".tmp"
	}
	file, err := os.Open(change.Path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Apply the change
	if change.Line == 0 { // Create a new file
		if change.Add == "" {
			return fmt.Errorf("add content is required to create a new file")
		}
		lines = append(lines, change.Add)
	} else if change.Line > 0 && change.Line <= len(lines) {
		if change.Delete != "" {
			lines[change.Line-1] = strings.Replace(lines[change.Line-1], change.Delete, "", 1)
		}
		if change.Add != "" {
			lines[change.Line-1] = lines[change.Line-1] + change.Add
		}
	} else {
		return fmt.Errorf("line number %d out of range", change.Line)
	}

	// Write the changes back to the file
	output := strings.Join(lines, "\n")
	if err := os.WriteFile(filepath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
