package applier

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nakamasato/aicoder/internal/planner"
)

func applyChanges(changesPlan planner.ChangesPlan) error {
	for _, change := range changesPlan.Changes {
		if err := applyChange(change); err != nil {
			return fmt.Errorf("failed to apply change to %s: %w", change.Path, err)
		}
	}
	return nil
}

func applyChange(change planner.Change) error {
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
	if change.Line > 0 && change.Line <= len(lines) {
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
	if err := os.WriteFile(change.Path, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
