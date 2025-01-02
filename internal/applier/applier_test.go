package applier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nakamasato/aicoder/internal/planner"
)

func TestApplyChanges(t *testing.T) {
	changesPlan := planner.ChangesPlan{
		Changes: []planner.Change{
			{
				Path:    "testfile.txt",
				LineNum: 2,
				Add:     " Added text.",
				Delete:  "Text to be deleted.",
			},
			{
				Path:    "newfile.txt",
				LineNum: 0,
				Add:     "Content of the newly created file.",
			},
		},
	}

	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "applier_test")
	err := os.Mkdir(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Cleanup

	// Create the original file within the temporary directory
	originalFilePath := filepath.Join(tempDir, "testfile.txt")
	originalContent := "This is the original file.\nText to be deleted.\n"
	if err := os.WriteFile(originalFilePath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	// Update the paths in changesPlan to point to the temporary directory
	for i := range changesPlan.Changes {
		changesPlan.Changes[i].Path = filepath.Join(tempDir, changesPlan.Changes[i].Path)
	}

	// Perform a dry run
	err = ApplyChanges(changesPlan, true)
	if err != nil {
		t.Fatalf("ApplyChanges (dryrun) failed: %v", err)
	}

	// Verify the temporary modified file exists and has the expected content
	modifiedFilePath := originalFilePath + ".tmp"
	modifiedContent, err := GetFileContent(modifiedFilePath)
	if err != nil {
		t.Fatalf("Failed to read modified temp file: %v", err)
	}

	expectedModifiedContent := "This is the original file.\n Added text."
	if string(modifiedContent) != expectedModifiedContent {
		t.Errorf("Expected modified content:\n%s\nActual modified content:\n%s", expectedModifiedContent, string(modifiedContent))
	}

	// Verify the new file was not created during dry run
	newFilePath := filepath.Join(tempDir, "newfile.txt")
	if _, err := os.Stat(newFilePath); !os.IsNotExist(err) {
		t.Errorf("New file should not exist during dry run")
	}

	// Perform the actual apply
	err = ApplyChanges(changesPlan, false)
	if err != nil {
		t.Fatalf("ApplyChanges (actual) failed: %v", err)
	}

	// Verify the original file was updated correctly
	updatedContent, err := GetFileContent(originalFilePath)
	if err != nil {
		t.Fatalf("Failed to read updated original file: %v", err)
	}

	expectedUpdatedContent := "This is the original file.\n Added text."
	if string(updatedContent) != expectedUpdatedContent {
		t.Errorf("Expected updated content:\n%s\nActual updated content:\n%s", expectedUpdatedContent, string(updatedContent))
	}

	// Verify the new file was created correctly
	newContent, err := GetFileContent(newFilePath)
	if err != nil {
		t.Fatalf("Failed to read new file: %v", err)
	}

	expectedNewContent := "Content of the newly created file."
	if string(newContent) != expectedNewContent {
		t.Errorf("Expected new file content:\n%s\nActual new file content:\n%s", expectedNewContent, string(newContent))
	}
}
