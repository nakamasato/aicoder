package gh

import (
	"fmt"

	"github.com/nakamasato/aicoder/internal/git"
	"github.com/spf13/cobra"
)

func NewRepoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repo",
		Short: "GitHub repository commands",
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
