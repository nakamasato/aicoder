package applier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nakamasato/aicoder/internal/planner"
)

func TestApplyChangesGo(t *testing.T) {
	changesPlan := &planner.ChangesPlan{
		Changes: []planner.BlockChange{
			{Block: planner.Block{Path: "testfile.go", TargetName: "Func1", TargetType: "function"}, NewContent: " fmt.Println(\"This is the new content.\")", NewComment: "This is a new comment."},
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

// This is a new comment.
func Func1() {
	fmt.
		Println("This is the new content.")
}
`
	if string(newContent) != expectedNewContent {
		t.Errorf("Expected new file content:\n%s\nActual new file content:\n%s", expectedNewContent, string(newContent))
	}
}

func TestApplyChangesHCL(t *testing.T) {
	// 1. HCLの内容を一時ファイルに書き出す
	hclContent := `
resource "google_secret_manager_secret_iam_member" "example_sa_is_slack_token_secret_accessor" {
  project   = var.gcp_project_id
  member    = google_service_account.example_sa.member
  secret_id = google_secret_manager_secret.slack_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
}

resource "google_storage_bucket" "example_bucket" {
  name     = "example-bucket"
  location = "US"
}
`
	tempDir := filepath.Join(os.TempDir(), "applier_test_hcl")
	err := os.Mkdir(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	hclFilePath := filepath.Join(tempDir, "example.hcl")
	if err := os.WriteFile(hclFilePath, []byte(hclContent), 0644); err != nil {
		t.Fatalf("Failed to create HCL file: %v", err)
	}

	// 2. 変更内容をChangesPlanに準備する
	changesPlan := &planner.ChangesPlan{
		Changes: []planner.BlockChange{
			{Block: planner.Block{Path: hclFilePath, TargetName: "google_secret_manager_secret_iam_member,example_sa_is_slack_token_secret_accessor", TargetType: "resource"}, NewContent: `
project   = "new_project_id"
member    = "new_member"
secret_id = google_secret_manager_secret.new_token.secret_id
role      = "roles/secretmanager.admin"
`},
		},
	}

	// 3. ApplyChangesを呼ぶ
	err = ApplyChanges(changesPlan, false)
	if err != nil {
		t.Fatalf("ApplyChanges failed: %v", err)
	}

	// 4. 意図したものになっているか確認する
	newContent, err := os.ReadFile(hclFilePath)
	if err != nil {
		t.Fatalf("Failed to read new file: %v", err)
	}
	expectedNewContent := `
resource "google_secret_manager_secret_iam_member" "example_sa_is_slack_token_secret_accessor" {
  project   = "new_project_id"
  member    = "new_member"
  secret_id = google_secret_manager_secret.new_token.secret_id
  role      = "roles/secretmanager.admin"
}

resource "google_storage_bucket" "example_bucket" {
  name     = "example-bucket"
  location = "US"
}
`
	if string(newContent) != expectedNewContent {
		t.Errorf("Expected new file content:\n%s\nActual new file content:\n%s", expectedNewContent, string(newContent))
	}
}

func TestApplyChangeFilePlan(t *testing.T) {
	// Setup: Create a temporary file
	tempFile, err := os.CreateTemp("", "testfile-*.tmp")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up

	// Define a test change
	change := &planner.BlockChange{
		NewContent: "new content for the file",
	}

	// Execute the function
	err = ApplyChangeFilePlan(change, tempFile.Name())
	if err != nil {
		t.Fatalf("ApplyChangeFilePlan failed: %v", err)
	}

	// Verify the file content
	content, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	expectedContent := "new content for the file"
	if string(content) != expectedContent {
		t.Errorf("Expected file content to be %q, but got %q", expectedContent, string(content))
	}
}

func TestApplyChanges_UnsupportedFileType(t *testing.T) {
	// Create a mock ChangesPlan with an function target type
	changesPlan := &planner.ChangesPlan{
		Changes: []planner.BlockChange{
			{
				Block: planner.Block{
					Path:       "unsupported_file_type.py",
					TargetType: "function",
				},
			},
		},
	}

	// Call ApplyChanges with dryrun set to false
	err := ApplyChanges(changesPlan, false)

	// Check if the error message is as expected
	expectedError := "unsupported file type: unsupported_file_type.py"
	if err == nil || err.Error() != expectedError {
		t.Fatalf("expected error: %s, got: %v", expectedError, err)
	}
}
