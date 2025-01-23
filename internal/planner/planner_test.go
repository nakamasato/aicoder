// planner_test.go
package planner

import (
	"context"
	"strings"
	"testing"

	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/stretchr/testify/assert"
)

func TestNewPlanner(t *testing.T) {
	llmClient := llm.DummyClient{} // Assuming you have a dummy client
	entClient := &ent.Client{}     // Assuming you have a way to initialize this

	planner := NewPlanner(llmClient, entClient)

	assert.NotNil(t, planner)
	assert.Equal(t, llmClient, planner.llmClient)
	assert.Equal(t, entClient, planner.entClient)
}

func TestGeneratePromptWithFiles(t *testing.T) {
	llmClient := llm.DummyClient{} // Assuming you have a dummy client
	entClient := &ent.Client{}     // Assuming you have a way to initialize this
	planner := NewPlanner(llmClient, entClient)

	prompt := "Modify the following files: %s to achieve the goal: %s"
	goal := "Refactor code"
	files := []file.File{
		{Path: "example.go", Content: "package main\nfunc main() {}"},
	}
	fileBlocks := map[string][]Block{
		"example.go": {
			{Path: "example.go", TargetType: "function", TargetName: "main", Content: "func main() {}"},
		},
	}

	result, err := planner.generateBlockPromptWithFiles(prompt, goal, files, fileBlocks)
	assert.NoError(t, err)
	assert.Contains(t, result, "example.go")
	assert.Contains(t, result, "main")
}

func TestMakeActionPlan(t *testing.T) {
	// Mock LLM client
	mockLLMClient := &llm.DummyClient{
		ReturnValue: `{
  "investigate_steps": [
    "Analyze code structure",
    "Check file dependencies"
  ],
  "change_steps": [
    "Update module B"
  ]
}`,
	}
	mockEntClient := &ent.Client{} // Assuming you have a mock or a real client

	// Create a new planner instance
	planner := NewPlanner(mockLLMClient, mockEntClient)

	// Define test cases
	tests := []struct {
		name            string
		candidateBlocks map[string][]Block
		currentPlan     *ChangesPlan
		query           string
		review          string
		expectedError   bool
	}{
		{
			name: "Basic test case",
			candidateBlocks: map[string][]Block{
				"file1.go": {
					{Path: "file1.go", TargetType: "function", TargetName: "Func1", Content: "func Func1() {}"},
				},
			},
			currentPlan:   nil,
			query:         "Refactor Func1",
			review:        "",
			expectedError: false,
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function
			actionPlan, err := planner.makeActionPlan(context.Background(), tt.candidateBlocks, tt.currentPlan, tt.query, tt.review)

			// Check for errors
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, actionPlan)
				assert.Equal(t, 2, len(actionPlan.InvestigateSteps))
				assert.Equal(t, 1, len(actionPlan.ChangeSteps))
			}
		})
	}
}

func TestConvertActionPlanExamplesToStr(t *testing.T) {
	examples := []ActionPlanExample{
		{
			Goal: "Update README.md file with the latest implementation.",
			Plan: ActionPlan{
				InvestigateSteps: []string{
					"Check the current README.md file.",
					"Search for the content written in the README.md file.",
					"Check the current implementation.",
				},
				ChangeSteps: []string{
					"Update the title in the README.md file.",
					"Add feature lists in the README.md file.",
				},
			},
		},
	}

	expectedOutput := `--- Example 1 ---
Goal: Update README.md file with the latest implementation.
Steps:
- Investigation:
	- Check the current README.md file.
	- Search for the content written in the README.md file.
	- Check the current implementation.
- Changes Plan:
	- Update the title in the README.md file.
	- Add feature lists in the README.md file.
--- Example end 1 ---
`

	result := convertActionPlanExaplesToStr(examples)

	if strings.TrimSpace(result) != strings.TrimSpace(expectedOutput) {
		t.Errorf("Expected:\n%s\nGot:\n%s", expectedOutput, result)
	}
}

func TestInvestigationResult_String(t *testing.T) {
	tests := []struct {
		name     string
		input    InvestigationResult
		expected string
	}{
		{
			name: "Basic test",
			input: InvestigationResult{
				TargetFiles:    []string{"file1.go", "file2.go"},
				ReferenceFiles: []string{"ref1.go", "ref2.go"},
				Result:         "Investigation successful",
			},
			expected: "Target Files: [file1.go file2.go]\nReference Files: [ref1.go ref2.go]\nResult: Investigation successful",
		},
		{
			name: "Empty fields",
			input: InvestigationResult{
				TargetFiles:    []string{},
				ReferenceFiles: []string{},
				Result:         "",
			},
			expected: "Target Files: []\nReference Files: []\nResult: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.String()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestConvertInvestigationResultExamplesToStr(t *testing.T) {
	examples := []InvestigationResultExample{
		{
			Goal: "Check the current backend service account in the Dev environment.",
			Files: []file.File{
				{
					Path: "terraform/development/google_service_account_iam_member.tf",
					Content: `resource "google_storage_bucket_iam_member" "member" {
  bucket = google_storage_bucket.example.name
  role = "roles/storage.admin"
  member = "serviceAccount:${google_service_account.sa.email}"
}`,
				},
			},
			Result: InvestigationResult{
				TargetFiles:    []string{},
				ReferenceFiles: []string{"terraform/development/google_service_account_iam_member.tf"},
				Result: `The current backend service account in the Dev environment is defined in the file 'terraform/development/google_service_account_iam_member.tf'.

` + "```hcl" + `
resource "google_storage_bucket_iam_member" "member" {
  bucket = google_storage_bucket.example.name
  role = "roles/storage.admin"
  member = "serviceAccount:${google_service_account.sa.email}"
}
` + "```\n",
			},
		},
	}

	expectedOutput := `
--- Example 1 ---
Goal: Check the current backend service account in the Dev environment.
Files:
---
file: terraform/development/google_service_account_iam_member.tf
` + "```" + `
resource "google_storage_bucket_iam_member" "member" {
  bucket = google_storage_bucket.example.name
  role = "roles/storage.admin"
  member = "serviceAccount:${google_service_account.sa.email}"
}
` + "```\n" + `Result: The current backend service account in the Dev environment is defined in the file 'terraform/development/google_service_account_iam_member.tf'.

` + "```hcl" + `
resource "google_storage_bucket_iam_member" "member" {
  bucket = google_storage_bucket.example.name
  role = "roles/storage.admin"
  member = "serviceAccount:${google_service_account.sa.email}"
}
` + "```\n" + "\n--- Example end 1 ---\n"

	output, err := convertInvestigationResultExamplesToStr(examples)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if output != expectedOutput {
		t.Errorf("Expected output:\n````\n%s\n````\nGot:\n````\n%s````\n", expectedOutput, output)
	}
}
