package gh

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-github/v60/github"
	"github.com/nakamasato/aicoder/internal/git"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type ghOptions struct {
	token  string
	client *github.Client
	ctx    context.Context
}

// NewGhCmd creates the 'gh' parent command.
func NewGhCmd() *cobra.Command {
	opts := &ghOptions{
		ctx: context.Background(),
	}

	cmd := &cobra.Command{
		Use:   "gh",
		Short: "GitHub の操作を行うサブコマンド",
		// PersistentPreRun で GitHub クライアントを初期化します
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// コマンドライン引数でトークンが指定されていなければ環境変数から取得
			if ghTokenFlag == "" {
				ghTokenFlag = os.Getenv("GH_ACCESS_TOKEN")
			}
			cfg.Token = ghTokenFlag

			var httpClient *http.Client
			if cfg.Token != "" {
				// トークンがある場合は OAuth2 認証付きクライアントを作成
				ctx := context.Background()
				ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.Token})
				httpClient = oauth2.NewClient(ctx, ts)
			} else {
				// トークンがない場合は認証なしのクライアント
				httpClient = http.DefaultClient
			}
			cfg.GitHubClient = github.NewClient(httpClient)
		},
	}

	cmd.PersistentFlags().StringVar(&opts.token, "token", "", "GitHub token (optional)")

	// Add subcommands to 'gh' here
	cmd.AddCommand(
		NewRunCmd(),
		NewPRCmd(),
		NewRepoCmd(),
	)

	return cmd
}

// Config は共通の設定やクライアントを保持する構造体です
type Config struct {
	GitHubClient *github.Client
	Token        string
}

var cfg Config

// ghTokenFlag はコマンドラインから渡す GitHub アクセストークンを一時的に保持する変数です
var ghTokenFlag string

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
			repo, _, err := cfg.GitHubClient.Repositories.Get(ctx, owner, repoName)
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
