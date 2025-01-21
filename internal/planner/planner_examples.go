package planner

import (
	"fmt"
	"strings"
)

type ActionPlanExample struct {
	Goal string
	Plan ActionPlan
}

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
		Goal: "Devのbackendのsaに付与されてる権限と同じ権限をProdのbackendのsaに付与",
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
