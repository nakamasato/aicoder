package git

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
)

// GetRepoInfo gets the owner and repo name from the git remote URL
func GetRepoInfo() (owner, repo string, err error) {
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
