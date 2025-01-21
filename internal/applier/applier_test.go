package applier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nakamasato/aicoder/internal/planner"
)

func TestApplyChangesGo(t *testing.T) {
	testCases := []struct {
		name              string
		query             string
		changes           []planner.BlockChange
		originalContent   string
		expectedNewContent string
	}{
		{
			name:  "Update Func1",
			query: "Please update the function Func1 in testfile.go",
			changes: []planner.BlockChange{
				{
					Block: planner.Block{
						Path:       "testfile.go",
						TargetName: "Func1",
						TargetType: "function",
					},
					NewContent: " fmt.Println(\"This is the new content.\")",
					NewComment: "This is a new comment.",
				},
			},
			originalContent: `package main
func Func1() {
	fmt.Println("This is the original file.")
}
`,
			expectedNewContent: `package main

// This is a new comment.
func Func1() {
	fmt.
		Println("This is the new content.")
}
`,
		},
		// {
		// 	name: "summarizer",
		// 	query: "Please update the function summarizer in testfile.go",
		// 	changes: []planner.BlockChange{
		// 		{
		// 			Block: planner.Block{
		// 				Path: "summarizer.go",
		// 				TargetType: "function",
		// 				TargetName: "UpdateRepoSummary",
		// 				Content: "// UpdateRepoSummary retrieves the repository summary and saves it to the output file.\nfunc (s *service) UpdateRepoSummary(ctx context.Context, language Language, outputfile string) (string, error) {\n    // Call existing logic\n    summary, err := s.UpdateRepoSummaryLogic(ctx, language)\n    if err != nil {\n        return \"\", err\n    }\n    \n    // Save summary to repo_summary.json\n    if err := os.WriteFile(\"repo_summary.json\", []byte(summary), 0644); err != nil {\n        return \"\", fmt.Errorf(\"failed to write summary to repo_summary.json: %v\", err)\n    }\n    \n    return summary, nil\n}",
		// 			},
		// 			NewContent: "\n\t// Query the database for documents related to the specified repository and context.\n\tdocs, err := s.entClient.Document.Query().Where(document.RepositoryEQ(s.config.Repository), document.ContextEQ(s.config.CurrentContext)).All(ctx)\n\tif err != nil {\n\t\treturn \"\", fmt.Errorf(\"failed to query documents: %v\", err)\n\t}\n\n\tvar documents []*vectorstore.Document\n\tfor _, doc := range docs {\n\t\tdocuments = append(documents, \u0026vectorstore.Document{\n\t\t\tRepository:  doc.Repository,\n\t\t\tContext:     doc.Context,\n\t\t\tFilepath:    doc.Filepath,\n\t\t\tDescription: doc.Description,\n\t\t})\n\t}\n\n\tvar builder strings.Builder\n\tfor _, doc := range documents {\n\t\tbuilder.WriteString(fmt.Sprintf(\"File: %s\\n\", doc.Filepath))\n\t\tbuilder.WriteString(fmt.Sprintf(\"Summary: %s\\n\", doc.Description))\n\t}\n\n\t// Prepare a prompt for the LLM using the gathered documents' summaries\n\tprompt := fmt.Sprintf(llm.SUMMARIZE_REPO_CONTENT_PROMPT, s.config.Repository, s.config.CurrentContext, builder.String(), language)\n\tmessages := []openai.ChatCompletionMessageParamUnion{\n\t\topenai.SystemMessage(prompt),\n\t}\n\n\t// Generate a summary using the LLM based on the prompt\n\tsummary, err := s.llmClient.GenerateCompletionSimple(ctx, messages)\n\tif err != nil {\n\t\treturn \"\", fmt.Errorf(\"failed to generate completion: %v\", err)\n\t}\n\n\t// Marshal the summary data into JSON format\n\tsummaryJSON, err := json.MarshalIndent(\n\t\tmap[string]string{\n\t\t\t\"summary\": summary,\n\t\t}, \"\", \"  \")\n\tif err != nil {\n\t\tlog.Fatalf(\"failed to marshal summary to JSON: %v\", err)\n\t}\n\n\t// Write the JSON output to the specified file\n\tif err := os.WriteFile(outputfile, summaryJSON, 0644); err != nil {\n\t\tlog.Fatalf(\"failed to write summary to file: %v\", err)\n\t}\n\n\treturn summary, nil\n",
		// 			NewComment: "Updated logic for generating and writing repo summary.",
		// 		},
		// 	},
		// 	originalContent: "// UpdateRepoSummary retrieves the repository summary and saves it to the output file.\nfunc (s *service) UpdateRepoSummary(ctx context.Context, language Language, outputfile string) (string, error) {\n    // Call existing logic\n    summary, err := s.UpdateRepoSummaryLogic(ctx, language)\n    if err != nil {\n        return \"\", err\n    }\n    \n    // Save summary to repo_summary.json\n    if err := os.WriteFile(\"repo_summary.json\", []byte(summary), 0644); err != nil {\n        return \"\", fmt.Errorf(\"failed to write summary to repo_summary.json: %v\", err)\n    }\n    \n    return summary, nil\n}",
		// 	expectedNewContent: "// Updated logic for generating and writing repo summary.\nfunc (s *service) UpdateRepoSummary(ctx context.Context, language Language, outputfile string) (string, error) {\n    // Call existing logic\n    summary, err := s.UpdateRepoSummaryLogic(ctx, language)\n    if err != nil {\n        return \"\", err\n    }\n    \n    // Save summary to repo_summary.json\n    if err := os.WriteFile(\"repo_summary.json\", []byte(summary), 0644); err != nil {\n        return \"\", fmt.Errorf(\"failed to write summary to repo_summary.json: %v\", err)\n    }\n    \n    return summary, nil\n}",
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			changesPlan := &planner.ChangesPlan{
				Query:   tc.query,
				Changes: tc.changes,
			}

			tempDir := filepath.Join(os.TempDir(), "applier_test")
			err := os.Mkdir(tempDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create temporary directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			for _, ch := range changesPlan.Changes {
				originalFilePath := filepath.Join(tempDir, ch.Block.Path)
				if err := os.WriteFile(originalFilePath, []byte(tc.originalContent), 0644); err != nil {
					t.Fatalf("Failed to create original file: %v", err)
				}
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
			if err := os.WriteFile(testFile, []byte(tc.originalContent), 0644); err != nil {
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

			if string(newContent) != tc.expectedNewContent {
				t.Errorf("Expected new file content:\n%s\nActual new file content:\n%s", tc.expectedNewContent, string(newContent))
			}
		})
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
		t.Errorf("Expected new file content:\n```\n%s\n```\nActual new file content:\n```\n%s\n```\n", expectedNewContent, string(newContent))
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
