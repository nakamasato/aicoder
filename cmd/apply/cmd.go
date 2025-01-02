package apply

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/nakamasato/aicoder/internal/applier"
	"github.com/nakamasato/aicoder/internal/planner"
	"github.com/spf13/cobra"
)

var planFile string
var dryrun bool

// NewApplyCmd creates a new apply command
func Command() *cobra.Command {
	cmdApply := &cobra.Command{
		Use:   "apply",
		Short: "Apply changes",
		Long:  `Apply changes to the system based on the provided configuration.`,
		Run:   runApply,
	}

	cmdApply.Flags().StringVarP(&planFile, "planfile", "p", "plan.json", "Path to the plan file to apply")
	cmdApply.Flags().BoolVarP(&dryrun, "dryrun", "d", false, "Dry run the changes")

	return cmdApply
}

func runApply(cmd *cobra.Command, args []string) {
	if planFile == "" {
		log.Fatalln("plan file is required")
	}
	fmt.Printf("apply %s", planFile)

	// Read the plan file
	data, err := os.ReadFile(planFile)
	if err != nil {
		log.Fatalf("failed to read plan file: %v", err)
	}

	// Unmarshal the JSON data into changesPlan
	var changesPlan planner.ChangesPlan
	if err := json.Unmarshal(data, &changesPlan); err != nil {
		log.Fatalf("failed to unmarshal plan file: %v", err)
	}

	// Apply the changes
	if err := applier.ApplyChanges(changesPlan, dryrun); err != nil {
		log.Fatalf("failed to apply changes: %v", err)
	}

	fmt.Printf("Successfully applied changes from %s", planFile)
}