// cmd/loader.go
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/cobra"
)

// FileInfo represents information about a file or directory.
type FileInfo struct {
	Name        string     `json:"name"`
	Path        string     `json:"path"`
	Description string     `json:"description,omitempty"`
	IsDir       bool       `json:"is_dir"`
	Children    []FileInfo `json:"children,omitempty"`
	BlobHash    string     `json:"blob_hash,omitempty"`
}

// RepoStructure represents the entire repository structure.
type RepoStructure struct {
	GeneratedAt string   `json:"generated_at"`
	Root        FileInfo `json:"root"`
}

var (
	outputFile    string
	branch        string
	commitHash    string
	openaiAPIKey  string
	openaiModel   string
	maxTokens     int
	summaryEngine string
)

// loaderCmd represents the loader command
var loaderCmd = &cobra.Command{
	Use:   "loader [path]",
	Short: "Load the repository structure from a Git repository and export it to a JSON file with summaries.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoPath := args[0]

		// Initialize OpenAI client
		if openaiAPIKey == "" {
			openaiAPIKey = os.Getenv("OPENAI_API_KEY")
		}
		if openaiAPIKey == "" {
			log.Fatal("OPENAI_API_KEY environment variable is not set")
		}
		client := openai.NewClient(option.WithAPIKey(openaiAPIKey))

		// Load existing RepoStructure if exists
		var previousRepo RepoStructure
		if _, err := os.Stat(outputFile); err == nil {
			data, err := os.ReadFile(outputFile)
			if err != nil {
				log.Fatalf("Failed to read existing repo structure: %v", err)
			}
			if err := json.Unmarshal(data, &previousRepo); err != nil {
				log.Fatalf("Failed to parse existing repo structure: %v", err)
			}
		}

		// Load current RepoStructure
		currentRepo, err := loadRepoStructure(repoPath, branch, commitHash, client, previousRepo)
		if err != nil {
			fmt.Printf("Error loading repo structure: %v\n", err)
			os.Exit(1)
		}

		// Marshal to JSON
		output, err := json.MarshalIndent(currentRepo, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}

		// Write to file
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
	loaderCmd.Flags().StringVarP(&openaiAPIKey, "api-key", "k", "", "OpenAI API key (can also set via OPENAI_API_KEY environment variable)")
	loaderCmd.Flags().StringVar(&openaiModel, "model", "gpt-4o-mini", "OpenAI model to use for summarization") // デフォルトを gpt-4o-mini に変更
	loaderCmd.Flags().IntVar(&maxTokens, "max-tokens", 150, "Maximum number of tokens for the summary")
}

// loadRepoStructure loads the repository structure using go-git and generates summaries.
func loadRepoStructure(path, branch, commitHash string, client *openai.Client, previousRepo RepoStructure) (RepoStructure, error) {
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

	children, err := traverseTree(tree, "", client, previousRepo)
	if err != nil {
		return RepoStructure{}, fmt.Errorf("failed to traverse tree: %w", err)
	}

	rootFileInfo.Children = children

	return RepoStructure{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Root:        rootFileInfo,
	}, nil
}

// traverseTree recursively traverses the Git tree and collects FileInfo.
// It updates the Description using OpenAI only if the BlobHash has changed.
func traverseTree(tree *object.Tree, parentPath string, client *openai.Client, previousRepo RepoStructure) ([]FileInfo, error) {
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
			children, err := traverseTree(subtree, filePath, client, previousRepo)
			if err != nil {
				return nil, err
			}
			fileInfo.Children = children
		} else {
			file, err := tree.File(entry.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get file %s: %w", entry.Name, err)
			}

			blob := file
			if blob == nil {
				return nil, fmt.Errorf("failed to get blob for %s: %w", entry.Name, err)
			}

			fileInfo.BlobHash = blob.Hash.String()

			// Check if the file was previously summarized
			previousDescription := ""
			previousBlobHash := ""
			if previousRepo.Root.IsDir {
				previousFileInfo := findFileInRepo(previousRepo.Root, filePath)
				if previousFileInfo != nil {
					previousDescription = previousFileInfo.Description
					previousBlobHash = previousFileInfo.BlobHash
				}
			}
			if previousBlobHash == fileInfo.BlobHash {
				fileInfo.Description = previousDescription
			} else {

				// ファイルの内容を読み出す
				reader, err := file.Reader()
				if err != nil {
					return nil, fmt.Errorf("failed to get file reader: %w", err)
				}
				defer reader.Close()

				content, err := io.ReadAll(reader)
				if err != nil {
					return nil, fmt.Errorf("failed to read file content: %w", err)
				}

				log.Printf("summarizing %s\n", filePath)
				summary, err := summarizeContent(client, string(content))
				if err != nil {
					return nil, fmt.Errorf("failed to summarize content: %w", err)
				}
				log.Printf("%s:%s\n", filePath, summary)
				fileInfo.Description = summary
			}

		}

		files = append(files, fileInfo)
	}

	return files, nil
}

// findFileInRepo searches for a file in the previous RepoStructure by its path.
func findFileInRepo(current FileInfo, targetPath string) *FileInfo {
	if current.Path == targetPath {
		return &current
	}
	for _, child := range current.Children {
		if child.IsDir {
			if found := findFileInRepo(child, targetPath); found != nil {
				return found
			}
		} else {
			if child.Path == targetPath {
				return &child
			}
		}
	}
	return nil
}

// readBlobContent reads the content of a blob as a string.
func readBlobContent(blob *object.Blob) (string, error) {
	reader, err := blob.Reader()
	if err != nil {
		return "", err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// summarizeContent uses OpenAI to summarize the given text content.
func summarizeContent(client *openai.Client, content string) (string, error) {
	if len(content) == 0 {
		return "", nil
	}
	ctx := context.Background()

	// Prepare the prompt for summarization
	prompt := fmt.Sprintf("Please provide a concise summary of the following content:\n\n%s", content)

	// Create a chat completion request
	resp, err := client.Chat.Completions.New(ctx,
		openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(prompt),
			}),
			Model: openai.F(openai.ChatModelGPT4oMini),
		})
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}
