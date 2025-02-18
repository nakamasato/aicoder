package gh

import (
	"fmt"

	"github.com/nakamasato/aicoder/internal/git"
	"github.com/spf13/cobra"
)

// NewPRCmd creates the 'pr' parent command.
func NewPRCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Manage Pull Requests",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Propagate context to subcommands
			if parentCtx := cmd.Parent().Context(); parentCtx != nil {
				cmd.SetContext(parentCtx)
				for _, subcmd := range cmd.Commands() {
					subcmd.SetContext(parentCtx)
				}
			}
			return nil
		},
	}

	cmd.AddCommand(NewPRListCmd())

	return cmd
}

// NewPRListCmd creates the 'pr list' command.
func NewPRListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Pull Requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get repository information from git remote
			repoOwner, repoName, err := git.GetRepoInfo()
			if err != nil {
				return fmt.Errorf("failed to get repository information: %v", err)
			}

			// Get parent command options
			opts := cmd.Parent().Parent().Context().Value("ghOptions").(*ghOptions)

			// List pull requests
			prs, _, err := opts.client.PullRequests.List(opts.ctx, repoOwner, repoName, nil)
			if err != nil {
				return fmt.Errorf("failed to list pull requests: %v", err)
			}

			// Display pull requests
			for _, pr := range prs {
				fmt.Printf("#%d %s [%s]\n", pr.GetNumber(), pr.GetTitle(), pr.GetState())
			}

			return nil
		},
	}

	return cmd
}
