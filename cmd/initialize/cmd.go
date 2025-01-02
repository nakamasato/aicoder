package initialize

import (
	"fmt"
	"os"
	"path/filepath"

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
	if _, err := os.Stat(outputFile); err == nil {
		// ファイルが既に存在する場合の処理
		cmd.Println(".aicoder.yaml already exists")
		return
	}

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		os.Exit(1) // Exit if unable to get the current directory
	}

	// Extract the directory name
	dirName := filepath.Base(cwd)

		content := fmt.Sprintf(`repository: %s
load:
  exclude:
    - ent
    - go.sum
    - repo_structure.json
  include:
    - ent/schema

search:
  top_n: 5
`, dirName) // Default content for the .aicoder.yaml file

	if err := os.WriteFile(outputFile, []byte(content), 0644); err != nil {
		os.Exit(1) // Exit if unable to write the file
	}
	// Print success message
	cmd.Println("Default configuration file created at .aicoder.yaml")
}
