package planner

import (
	"bytes"
	"fmt"
	_ "embed"
	"text/template"
	"strings"

	"github.com/nakamasato/aicoder/internal/file"
)

type ActionPlanExample struct {
	Goal string
	Plan ActionPlan
}

type InvestigationResultExample struct {
	Goal   string
	Files  []file.File
	Result InvestigationResult
}


//go:embed templates/investigation_result.tmpl
var investigationResultTemplate string

var DefaultActionPlanExamples = []ActionPlanExample{
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
	{
		Goal: "Grant the same permissions to the backend service account in Prod as those assigned to the backend service account in Dev",
		Plan: ActionPlan{
			InvestigateSteps: []string{
				"Check the current backend service account in the Dev environment.",
				"Check the current permissions assigned to the backend service account in the Dev environment.",
				"Check the current backend service account in the Prod environment.",
				"Check the current permissions assigned to the backend service account in the Prod environment.",
			},
			ChangeSteps: []string{
				"Assign the same permissions to the backend service account in the Prod environment as in the Dev environment.",
			},
		},
	},
}

func convertActionPlanExaplesToStr(examples []ActionPlanExample) string {
	var sb strings.Builder

	for i, example := range examples {
		sb.WriteString(fmt.Sprintf("--- Example %d ---\n", i+1))
		sb.WriteString(fmt.Sprintf("Goal: %s\n", example.Goal))
		sb.WriteString("Steps:\n")
		sb.WriteString("- Investigation:\n")
		for _, step := range example.Plan.InvestigateSteps {
			sb.WriteString(fmt.Sprintf("\t- %s\n", step))
		}
		sb.WriteString("- Changes Plan:\n")
		for _, step := range example.Plan.ChangeSteps {
			sb.WriteString(fmt.Sprintf("\t- %s\n", step))
		}
		sb.WriteString(fmt.Sprintf("--- Example end %d ---\n", i+1))
	}

	return sb.String()
}

var DefaultInvestigationExamples = []InvestigationResultExample{
	{
		Goal: "Check the current backend service account in the Dev environment.",
		Files: []file.File{
			{
				Path: "terraform/development/google_service_account_iam_member.tf",
				Content: `resource "google_service_account" "sa" {
  account_id   = "my-service-account"
  display_name = "A service account that Jane can use"
}

resource "google_storage_bucket" "example" {
  name          = "example"
  location      = "US"
}

resource "google_storage_bucket_iam_member" "member" {
  bucket = google_storage_bucket.example.name
  role = "roles/storage.admin"
  member  = "serviceAccount:${google_service_account.sa.email}"
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
  member  = "serviceAccount:${google_service_account.sa.email}"
}
` + "```\n",
		},
	},
}

func convertInvestigationResultExamplesToStr(examples []InvestigationResultExample) (string, error) {
	// Load the template file
	tmpl, err := template.New("examples").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(investigationResultTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, examples); err != nil {
		return "", err
	}

	return buf.String(), nil
}
