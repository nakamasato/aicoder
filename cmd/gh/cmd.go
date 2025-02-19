package gh

import (
	"fmt"

	"github.com/nakamasato/aicoder/internal/gh"
	"github.com/nakamasato/aicoder/internal/git"
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

	// Add subcommands to 'gh' here
	cmd.AddCommand(
		NewRunCmd(),
		NewPRCmd(),
		NewRepoCmd(),
	)

	return cmd
}

func NewRepoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repo",
		Short: "GitHub リポジトリに対する操作",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			owner, repoName, err := git.GetRepoInfo()
			if err != nil {
				fmt.Println("Error fetching repository information:", err)
				return
			}
			repo, _, err := ghCli.RawCli.Repositories.Get(ctx, owner, repoName)
			if err != nil {
				fmt.Println("Error fetching repositories:", err)
				return
			}
			if repo.FullName != nil {
				fmt.Println(*repo.FullName)
			}
		},
	}
}
