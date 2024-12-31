package apply

import (
	"encoding/json"
	"fmt"

	"github.com/nakamasato/aicoder/internal/applier"
	"github.com/nakamasato/aicoder/internal/planner"
	"github.com/spf13/cobra"
)

var planFile string

// NewApplyCmd creates a new apply command
func Command() *cobra.Command {
	cmdApply := &cobra.Command{
		Use:   "apply",
		Short: "Apply changes",
		Long:  `Apply changes to the system based on the provided configuration.`,
		RunE:  runApply,
	}

	cmdApply.Flags().StringVarP(&planFile, "planfile", "p", "", "Path to the plan file to apply")

	return cmdApply
}

func runApply(cmd *cobra.Command, args []string) error {
	if planFile == "" {
		return fmt.Errorf("plan file is required")
	}
	fmt.Printf("apply %s", planFile)

	var changesPlan planner.ChangesPlan
	if err := json.Unmarshal([]byte(planFile), &changesPlan); err != nil {
		return fmt.Errorf("failed to unmarshal plan file: %v", err)
	}

	if err := applier.ApplyChanges(changesPlan); err != nil {
		return err
	}

	fmt.Printf("Successfully applied changes from %s", planFile)
	return nil
}
