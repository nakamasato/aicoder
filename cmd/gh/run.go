package gh

import (
	"github.com/spf13/cobra"
)

// NewRunCmd creates the 'run' parent command.
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Manage GitHub Actions workflows",
	}

	// Add subcommands to 'run' here
	cmd.AddCommand(NewRunListCmd())

	return cmd
}
