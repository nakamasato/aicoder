// cmd/loader.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
)

// FileInfo represents information about a file or directory.
type FileInfo struct {
	Name        string     `json:"name"`
	Path        string     `json:"path"`
	Description string     `json:"description,omitempty"`
	IsDir       bool       `json:"is_dir"`
	Children    []FileInfo `json:"children,omitempty"`
}

// RepoStructure represents the entire repository structure.
type RepoStructure struct {
	Root FileInfo `json:"root"`
}

var (
	outputFile string
	branch     string
	commitHash string
)

// loaderCmd represents the loader command
var loaderCmd = &cobra.Command{
	Use:   "loader [path]",
	Short: "Load the repository structure from a Git repository and export it to a JSON file.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		repo, err := loadRepoStructure(path, branch, commitHash)
		if err != nil {
			fmt.Printf("Error loading repo structure: %v\n", err)
			os.Exit(1)
		}

		output, err := json.MarshalIndent(repo, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}

		err = os.WriteFile(outputFile, output, 0644)
		if err != nil {
			fmt.Printf("Error writing JSON to file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Repository structure has been written to %s\n", outputFile)
	},
}

func init() {
	rootCmd.AddCommand(loaderCmd)

	// Define flags and configuration settings for loaderCmd
	loaderCmd.Flags().StringVarP(&outputFile, "output", "o", "repo_structure.json", "Output JSON file")
	loaderCmd.Flags().StringVarP(&branch, "branch", "b", "main", "Branch to load the structure from")
	loaderCmd.Flags().StringVarP(&commitHash, "commit", "c", "", "Specific commit hash to load the structure from")
}

// loadRepoStructure loads the repository structure using go-git.
func loadRepoStructure(path, branch, commitHash string) (RepoStructure, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return RepoStructure{}, fmt.Errorf("failed to open repository: %w", err)
	}

	// Determine the reference to use (branch or specific commit)
	var commit *object.Commit
	if commitHash != "" {
		commit, err = repo.CommitObject(plumbing.NewHash(commitHash))
		if err != nil {
			return RepoStructure{}, fmt.Errorf("failed to get commit %s: %w", commitHash, err)
		}
	} else {
		ref, err := repo.Reference(plumbing.NewBranchReferenceName(branch), true)
		if err != nil {
			return RepoStructure{}, fmt.Errorf("failed to get branch %s: %w", branch, err)
		}
		commit, err = repo.CommitObject(ref.Hash())
		if err != nil {
			return RepoStructure{}, fmt.Errorf("failed to get commit for branch %s: %w", branch, err)
		}
	}

	tree, err := commit.Tree()
	if err != nil {
		return RepoStructure{}, fmt.Errorf("failed to get tree from commit: %w", err)
	}

	rootFileInfo := FileInfo{
		Name:  filepath.Base(path),
		Path:  "",
		IsDir: true,
	}

	children, err := traverseTree(tree, "")
	if err != nil {
		return RepoStructure{}, fmt.Errorf("failed to traverse tree: %w", err)
	}

	rootFileInfo.Children = children

	return RepoStructure{Root: rootFileInfo}, nil
}

// traverseTree recursively traverses the Git tree and collects FileInfo.
func traverseTree(tree *object.Tree, parentPath string) ([]FileInfo, error) {
	var files []FileInfo

	for _, entry := range tree.Entries {
		filePath := filepath.Join(parentPath, entry.Name)
		fileInfo := FileInfo{
			Name:  entry.Name,
			Path:  filePath,
			IsDir: entry.Mode == filemode.Dir,
		}

		if entry.Mode == filemode.Dir {
			subtree, err := tree.Tree(entry.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get subtree for %s: %w", entry.Name, err)
			}
			children, err := traverseTree(subtree, filePath)
			if err != nil {
				return nil, err
			}
			fileInfo.Children = children
		} else {
			blob, err := tree.TreeEntryFile(&entry)
			if err != nil {
				return nil, fmt.Errorf("failed to get blob for %s: %w", entry.Name, err)
			}
			fileInfo.Description = fmt.Sprintf("Blob size: %d bytes", blob.Size)
		}

		files = append(files, fileInfo)
	}

	return files, nil
}
