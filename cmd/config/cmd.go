package config

import (
	"github.com/spf13/cobra"
)

var (
	outputFile = ".aicoder.yaml"
)

// Command creates the init command.
func Command() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "config",
	}
	configCmd.AddCommand(
		initCommand(),
		setCommand(),
	)
	return configCmd
}
