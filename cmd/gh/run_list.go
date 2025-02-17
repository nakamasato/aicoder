package gh

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v60/github"
	"github.com/spf13/cobra"
)

// getGitHubToken gets the token from --token flag, GH_ACCESS_TOKEN env var, or gh CLI config
func getGitHubToken(flagToken string) (string, error) {
	// Check --token flag first
	if flagToken != "" {
		return flagToken, nil
	}

	// Check GH_ACCESS_TOKEN environment variable
	if envToken := os.Getenv("GH_ACCESS_TOKEN"); envToken != "" {
		return envToken, nil
	}

	// Fallback to gh CLI config
	homeDir, err := getHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %v", err)
	}

	configPath := filepath.Join(homeDir, ".config", "gh", "hosts.yml")
	content, err := readFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read gh config file: %v", err)
	}

	// Simple parsing of the YAML file to get the oauth_token
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, "oauth_token:") {
			token := strings.TrimSpace(strings.TrimPrefix(line, "oauth_token:"))
			return token, nil
		}
	}

	return "", fmt.Errorf("no GitHub token found. Please provide one via --token flag, GH_ACCESS_TOKEN environment variable, or authenticate with gh CLI")
}

// getRepoInfo gets the owner and repo name from the git remote URL
func getRepoInfo() (owner, repo string, err error) {
	gitRepo, err := git.PlainOpen(".")
	if err != nil {
		return "", "", fmt.Errorf("failed to open git repository: %v", err)
	}

	remote, err := gitRepo.Remote("origin")
	if err != nil {
		return "", "", fmt.Errorf("failed to get origin remote: %v", err)
	}

	urls := remote.Config().URLs
	if len(urls) == 0 {
		return "", "", fmt.Errorf("no remote URLs found")
	}

	// Parse the repository URL
	url := urls[0]
	parts := strings.Split(strings.TrimSuffix(url, ".git"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository URL format")
	}

	owner = parts[len(parts)-2]
	repo = parts[len(parts)-1]
	return owner, repo, nil
}

// readFile reads the content of a file.
func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// getHomeDir returns the home directory of the user
func getHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homeDir, nil
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

			// Get GitHub token
			token, err := getGitHubToken(cmd.Flag("token").Value.String())
			if err != nil {
				return fmt.Errorf("failed to get GitHub token: %v (make sure gh CLI is installed and authenticated)", err)
			}

			ctx := context.Background()
			client := github.NewClient(nil).WithAuthToken(token)

			// Get repository information from git remote
			repoOwner, repoName, err := getRepoInfo()
			if err != nil {
				return fmt.Errorf("failed to get repository information: %v", err)
			}

			// Get PR details to get the head SHA
			pr, _, err := client.PullRequests.Get(ctx, repoOwner, repoName, prID)
			if err != nil {
				return fmt.Errorf("failed to get PR details: %v", err)
			}

			// List workflow runs for the PR's head SHA
			runs, _, err := client.Actions.ListRepositoryWorkflowRuns(ctx, repoOwner, repoName, &github.ListWorkflowRunsOptions{
				Branch: pr.Head.GetRef(),
			})
			if err != nil {
				return fmt.Errorf("failed to list workflow runs: %v", err)
			}

			// Display workflow status
			for _, run := range runs.WorkflowRuns {
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
					logsURL, _, err := client.Actions.GetWorkflowRunLogs(ctx, repoOwner, repoName, *run.ID, 1)
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
