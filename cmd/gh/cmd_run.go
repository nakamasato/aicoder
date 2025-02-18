package gh

import (
	"fmt"
	"strconv"

	"github.com/nakamasato/aicoder/internal/gh"
	"github.com/nakamasato/aicoder/internal/git"
	"github.com/spf13/cobra"
)

// NewRunCmd creates the 'run' parent command.
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Manage GitHub Actions workflow runs",
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

	// Add subcommands to 'run' here
	cmd.AddCommand(NewRunListCmd())

	return cmd
}

func NewRunListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List GitHub Actions workflow runs for a PR",
		RunE: func(cmd *cobra.Command, args []string) error {
			prNumber, _ := cmd.Flags().GetString("pr")
			failed, _ := cmd.Flags().GetBool("failed")

			if prNumber == "" {
				return fmt.Errorf("please specify a pull request number using --pr")
			}

			prID, err := strconv.Atoi(prNumber)
			if err != nil {
				return fmt.Errorf("invalid PR number: %v", err)
			}

			// Get repository information from git remote
			repoOwner, repoName, err := git.GetRepoInfo()
			if err != nil {
				return fmt.Errorf("failed to get repository information: %v", err)
			}

			// Get parent command options
			opts := cmd.Parent().Parent().Context().Value("ghOptions").(*ghOptions)

			runs, err := gh.GetWorkflowRuns(opts.ctx, opts.token, repoOwner, repoName, prID)
			if err != nil {
				return fmt.Errorf("failed to get workflow runs: %v", err)
			}

			// Display workflow status
			for _, run := range runs {
				status := run.GetStatus()
				conclusion := run.GetConclusion()
				if status == "completed" {
					status = conclusion
				}

				if failed && conclusion != "failure" {
					continue
				}

				fmt.Printf("Workflow %s: %s\n", run.GetName(), status)

				// If workflow failed, get and display the logs
				if status == "completed" && conclusion == "failure" {
					fmt.Printf("Failed workflow URL: %s\n", run.GetHTMLURL())

					// Get workflow logs
					logsURL, err := gh.GetWorkflowRunLogs(opts.ctx, opts.token, repoOwner, repoName, *run.ID)
					if err != nil {
						fmt.Printf("Failed to get logs URL: %v\n", err)
						continue
					}
					fmt.Printf("Logs URL: %s\n", logsURL)
				}
			}

			return nil
		},
	}

	cmd.Flags().String("pr", "", "Pull request number")
	cmd.Flags().String("token", "", "GitHub token (optional, can also use GH_ACCESS_TOKEN env var)")
	cmd.Flags().Bool("failed", true, "Filter for failed workflows")

	return cmd
}
