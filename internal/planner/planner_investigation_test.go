package planner

import (
	"testing"

	"github.com/nakamasato/aicoder/internal/file"
)

func TestGenerateInvestigationPrompt(t *testing.T) {
	// Define test cases
	tests := []struct {
		name           string
		query          string
		files          []file.File
		examples       []InvestigationResultExample
		expectedString string
		wantErr        bool
	}{
		{
			name:  "Basic test",
			query: "What files need modification?",
			files: []file.File{
				{Path: "file1.go", Content: "package main\nfunc main() {}"},
				{Path: "file2.go", Content: "package main\nfunc main() {}"},
			},
			examples: []InvestigationResultExample{
				{
					Goal: "Update README.md file with the latest implementation.",
					Files: []file.File{
						{Path: "README.md", Content: "This is the README file."},
					},
					Result: InvestigationResult{
						TargetFiles:    []string{"README.md"},
						ReferenceFiles: []string{"README.md"},
						Result:         "Investigation successful",
					},
				},
			},
			expectedString: `You are a helpful assistant to generate the investigation result based on the collected information.
Your investigation result will be used to plan the actual file changes in the next steps.
So you need to collect information that is relevant to the original query and the goal.

Original query: What files need modification?

--- Relevant files ---

--- file1.go start ---
package main
func main() {}
--- file1.go end ---

--- file2.go start ---
package main
func main() {}
--- file2.go end ---

--- Relevant files ---

================= Examples start =================
--- Example 1 ---
Goal: Update README.md file with the latest implementation.
Files:
---
file: README.md
` + "```" + `
This is the README file.
` + "```" + `
Result: Investigation successful
--- Example end 1 ---
================= Examples end ===================

Please generate the investigation result based on the collected information.
This investigation is to extract the necessary information to plan the actual file changes in the next steps.
The output is the information that is necessary to determine the actual file changes in the next step.
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateInvestigationPrompt(tt.query, tt.files, tt.examples)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateInvestigationPrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.expectedString {
				t.Errorf("got:\n```\n%s\n```, want:\n```\n%s\n```", got, tt.expectedString)
			}
		})
	}
}
