package gh

import (
	"github.com/nakamasato/aicoder/internal/gh"
	"github.com/spf13/cobra"
)

var ghCli *gh.Client
var ghToken string

// NewGhCmd creates the 'gh' parent command.
func NewGhCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "gh",
		Short: "GitHub commands",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			ghCli = gh.NewClient(ctx, ghToken)
		},
	}

	cmd.PersistentFlags().StringVar(&ghToken, "token", "", "GitHub token (default is GH_ACCESS_TOKEN environment variable)")

	cmd.AddCommand(
		NewRunCmd(),
		NewPRCmd(),
		NewRepoCmd(),
	)

	return cmd
}
