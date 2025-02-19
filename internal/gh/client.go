package gh

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

type Client struct {
	RawCli *github.Client
}

// NewClient creates a new GitHub client
func NewClient(ctx context.Context, token string) *Client {
	if token == "" {
		token = os.Getenv("GH_ACCESS_TOKEN")
	}

	var httpClient *http.Client
	if token != "" {
		// if token exists, create a client with OAuth2 authentication
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		httpClient = oauth2.NewClient(ctx, ts)
	} else {
		// if token does not exist, create a client without authentication
		httpClient = http.DefaultClient
	}
	return &Client{
		RawCli: github.NewClient(httpClient),
	}
}


// GetWorkflowRuns gets the workflow runs for a PR
func (c Client) GetWorkflowRuns(ctx context.Context, owner, repo string, prID int) ([]*github.WorkflowRun, error) {
	// Get PR details to get the head SHA
	pr, _, err := c.RawCli.PullRequests.Get(ctx, owner, repo, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR details: %v", err)
	}

	// List workflow runs for the PR's head SHA
	runs, _, err := c.RawCli.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, &github.ListWorkflowRunsOptions{
		Branch: pr.Head.GetRef(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow runs: %v", err)
	}

	return runs.WorkflowRuns, nil
}

// GetWorkflowRunLogs gets the logs URL for a workflow run
func (c Client) GetWorkflowRunLogs(ctx context.Context, owner, repo string, runID int64) (string, error) {
	logsURL, _, err := c.RawCli.Actions.GetWorkflowRunLogs(ctx, owner, repo, runID, 1)
	if err != nil {
		return "", fmt.Errorf("failed to get logs URL: %v", err)
	}
	return logsURL.String(), nil
}
