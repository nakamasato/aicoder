package gh

import (
	"github.com/spf13/cobra"
)

// NewGhCmd creates the 'gh' parent command.
func NewPRCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Manage Pull Requests",
	}

	cmd.AddCommand(NewPRListCmd())

	return cmd
}

// NewPRListCmd creates the 'pr list' command.
func NewPRListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Pull Requests",
		Run: func(cmd *cobra.Command, args []string) {
			// Do something
		},
	}

	return cmd
}
