package applier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nakamasato/aicoder/internal/planner"
)

func TestApplyChanges(t *testing.T) {
	changesPlan := &planner.ChangesPlan{
		Changes: []planner.BlockChange{
			{Block: planner.Block{Path: "testfile.go", TargetName: "Func1"}, NewContent: " fmt.Println(\"This is the new content.\")"},
		},
	}
	tempDir := filepath.Join(os.TempDir(), "applier_test")
	err := os.Mkdir(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalFilePath := filepath.Join(tempDir, "testfile.go")
	originalContent := `package main
func Func1() {
	fmt.Println("This is the original file.")
}
`
	if err := os.WriteFile(originalFilePath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	for i := range changesPlan.Changes {
		changesPlan.Changes[i].Block.Path = filepath.Join(tempDir, changesPlan.Changes[i].Block.Path)
	}

	// Dry run
	err = ApplyChanges(changesPlan, true)
	if err != nil {
		t.Fatalf("ApplyChanges (dryrun) failed: %v", err)
	}

	testFile := filepath.Join(tempDir, "testfile.go")
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}

	// Actual application
	err = ApplyChanges(changesPlan, false)
	if err != nil {
		t.Fatalf("ApplyChanges (actual) failed: %v", err)
	}

	newContent, err := GetFileContent(testFile)
	if err != nil {
		t.Fatalf("Failed to read new file: %v", err)
	}
	expectedNewContent := `package main

func Func1() {
	fmt.
		Println("This is the new content.")
}
`
	if string(newContent) != expectedNewContent {
		t.Errorf("Expected new file content:\n%s\nActual new file content:\n%s", expectedNewContent, string(newContent))
	}
}
