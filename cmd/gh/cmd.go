package gh

import (
	"github.com/spf13/cobra"
)

// NewGhCmd creates the 'gh' parent command.
func NewGhCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gh",
		Short: "Manage GitHub Actions workflows",
	}

	// Add subcommands to 'gh' here
	cmd.AddCommand(NewRunCmd())

	return cmd
}
