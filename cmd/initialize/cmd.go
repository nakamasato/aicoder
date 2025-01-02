package initialize

import (
	"os"
	"github.com/spf13/cobra"
)

var (
	outputFile = ".aicoder.yaml"
)

// Command creates the init command.
func Command() *cobra.Command {
	cmdInit := &cobra.Command{
		Use:   "init",
		Short: "Create a default .aicoder.yaml configuration file",
		Run:   runInit,
	}
	return cmdInit
}

func runInit(cmd *cobra.Command, args []string) {
	content := "repository: default-repo\nload:\n  exclude:\n    - ent\n    - go.sum\n    - repo_structure.json\n  include:\n    - ent/schema\n\nsearch:\n  top_n: 5\n" // Default content for the .aicoder.yaml file

	if err := os.WriteFile(outputFile, []byte(content), 0644); err != nil {
		os.Exit(1) // Exit if unable to write the file
	}
	// Print success message
	cmd.Println("Default configuration file created at .aicoder.yaml")
}
