package load

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/pgvector/pgvector-go"
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
	GeneratedAt time.Time `json:"generated_at"`
	Root        FileInfo  `json:"root"`
}

var (
	outputFile   string
	branch       string
	commitHash   string
	openaiAPIKey string
	openaiModel  string
	maxTokens    int
	dbConnString string
)

func Command() *cobra.Command {
	loadCmd := &cobra.Command{
		Use:   "load",
		Short: "Load the repository structure from a Git repository and export it to a JSON file with summaries.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			config := config.GetConfig()

			gitRootPath := "."

			// Initialize OpenAI client
			if openaiAPIKey == "" {
				openaiAPIKey = os.Getenv("OPENAI_API_KEY")
			}
			if openaiAPIKey == "" {
				log.Fatal("OPENAI_API_KEY environment variable is not set")
			}
			client := openai.NewClient(option.WithAPIKey(openaiAPIKey))

			// Initialize PostgreSQL connection
			if dbConnString == "" {
				log.Fatal("Database connection string must be provided via --db-conn")
			}

			entClient, err := ent.Open("postgres", dbConnString)
			if err != nil {
				log.Fatalf("failed opening connection to postgres: %v", err)
			}
			defer entClient.Close()

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
			currentRepo, err := loadRepoStructure(ctx, gitRootPath, branch, commitHash, client, entClient, previousRepo, config)
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

	// Define flags and configuration settings for loaderCmd
	loadCmd.Flags().StringVarP(&outputFile, "output", "o", "repo_structure.json", "Output JSON file")
	loadCmd.Flags().StringVarP(&branch, "branch", "b", "main", "Branch to load the structure from")
	loadCmd.Flags().StringVarP(&commitHash, "commit", "c", "", "Specific commit hash to load the structure from")
	loadCmd.Flags().StringVarP(&openaiAPIKey, "api-key", "k", "", "OpenAI API key (can also set via OPENAI_API_KEY environment variable)")
	loadCmd.Flags().StringVar(&openaiModel, "model", "gpt-4o-mini", "OpenAI model to use for summarization")
	loadCmd.Flags().IntVar(&maxTokens, "max-tokens", 150, "Maximum number of tokens for the summary")
	loadCmd.Flags().StringVar(&dbConnString, "db-conn", "postgres://aicoder:aicoder@localhost:5432/aicoder?sslmode=disable", "PostgreSQL connection string (e.g., postgres://aicoder:aicoder@localhost:5432/aicoder)")

	return loadCmd
}

// loadRepoStructure loads the repository structure using go-git and generates summaries.
func loadRepoStructure(ctx context.Context, gitRootPath, branch, commitHash string, client *openai.Client, entClient *ent.Client, previousRepo RepoStructure, config config.AICoderConfig) (RepoStructure, error) {
	repo, err := git.PlainOpen(gitRootPath)
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

	// // Convert the relative path to an absolute path
	// absPath, err := filepath.Abs(path)
	// if err != nil {
	// 	log.Fatalf("failed to get absolute path: %v", err)
	// }

	// // Get the base name of the absolute path
	// repoName := filepath.Base(absPath)
	// fmt.Printf("Repository name: %s\n", repoName)

	rootFileInfo := FileInfo{
		Name:  gitRootPath,
		Path:  config.Load.TargetPath,
		IsDir: true,
	}

	if config.Load.TargetPath != "" {
		log.Printf("targetPath: %s", config.Load.TargetPath)
		tree, err = tree.Tree(config.Load.TargetPath)
		if err != nil {
			return RepoStructure{}, fmt.Errorf("failed to get tree for target path: %w", err)
		}
	}

	children, err := traverseTree(ctx, tree, config.Load.TargetPath, client, entClient, previousRepo, config)
	if err != nil {
		return RepoStructure{}, fmt.Errorf("failed to traverse tree: %w", err)
	}

	rootFileInfo.Children = children

	return RepoStructure{
		GeneratedAt: time.Now(),
		Root:        rootFileInfo,
	}, nil
}

// traverseTree recursively traverses the Git tree and collects FileInfo.
// It updates the Description using OpenAI and stores embeddings in PostgreSQL.
func traverseTree(ctx context.Context, tree *object.Tree, parentPath string, client *openai.Client, entClient *ent.Client, previousRepo RepoStructure, config config.AICoderConfig) ([]FileInfo, error) {
	var files []FileInfo
	var mu sync.Mutex
	var wg sync.WaitGroup
	var errChan = make(chan error, len(tree.Entries))

	for _, entry := range tree.Entries {
		filePath := filepath.Join(parentPath, entry.Name)
		fileInfo := FileInfo{
			Name:  entry.Name,
			Path:  filePath,
			IsDir: entry.Mode == filemode.Dir,
		}

		if !fileInfo.IsDir && config.Load.IsExcluded(filePath) && !config.Load.IsIncluded(filePath) {
			log.Printf("Skipping %s\n", filePath)
			continue
		}

		if entry.Mode == filemode.Dir {
			subtree, err := tree.Tree(entry.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get subtree for %s: %w", entry.Name, err)
			}
			children, err := traverseTree(ctx, subtree, filePath, client, entClient, previousRepo, config)
			if err != nil {
				return nil, err
			}
			fileInfo.Children = children
		} else {
			// Retrieve the file outside the goroutine
			file, err := tree.File(entry.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get file %s: %w", entry.Name, err)
			}

			blob := file
			if blob == nil {
				return nil, fmt.Errorf("failed to get blob for %s: %w", entry.Name, err)
			}

			fileInfo.BlobHash = blob.Hash.String()

			wg.Add(1)
			go func(fileInfo FileInfo, file *object.File) {
				defer wg.Done()

				// Check if the file was previously summarized
				previousDescription := ""
				previousBlobHash := ""
				if previousRepo.Root.IsDir {
					previousFileInfo := findFileInRepo(previousRepo.Root, fileInfo.Path)
					if previousFileInfo != nil {
						previousDescription = previousFileInfo.Description
						previousBlobHash = previousFileInfo.BlobHash
					}
				}

				info, err := os.Stat(fileInfo.Path)
				if err != nil {
					log.Printf("Failed to stat file %s: %v", fileInfo.Path, err)
					return
				}

				modTime := info.ModTime()

				// Determine if the file needs to be summarized
				needsSummary := true
				if !modTime.IsZero() && !previousRepo.GeneratedAt.IsZero() {
					if modTime.Before(previousRepo.GeneratedAt) || modTime.Equal(previousRepo.GeneratedAt) {
						// File has not been modified since the last summary
						if fileInfo.BlobHash == previousBlobHash && previousDescription != "" {
							log.Printf("[description] %s: use previous description (last modified: %s, generated_at: %s)\n", fileInfo.Path, modTime, previousRepo.GeneratedAt)
							fileInfo.Description = previousDescription
							needsSummary = false
						}
					}
				} else if previousBlobHash == fileInfo.BlobHash {
					log.Printf("[description] %s: use previous description (blob hash is same): (last modified: %s, generated_at: %s)\n", fileInfo.Path, modTime, previousRepo.GeneratedAt)
					fileInfo.Description = previousDescription
					needsSummary = false
				}

				if needsSummary {
					log.Printf("[description] %s: generating\n", fileInfo.Path)
					reader, err := file.Reader()
					if err != nil {
						errChan <- fmt.Errorf("failed to get file reader: %w", err)
						return
					}
					defer reader.Close()

					content, err := io.ReadAll(reader)
					if err != nil {
						errChan <- fmt.Errorf("failed to read file content: %w", err)
						return
					}

					log.Printf("summarizing %s\n", fileInfo.Path)
					summary, err := summarizeContent(client, string(content))
					if err != nil {
						errChan <- fmt.Errorf("failed to summarize content: %w", err)
						return
					}
					log.Printf("[description] %s: generated\n", fileInfo.Path)
					fileInfo.Description = summary

					// Get embedding for the description
					embedding, err := getEmbeddingFromDescription(client, summary)
					if err != nil {
						log.Printf("Failed to get embedding for %s: %v", fileInfo.Path, err)
					} else {
						// Insert or update the document in PostgreSQL
						err = upsertDocument(ctx, entClient, fileInfo.Path, summary, embedding, config.Repository)
						if err != nil {
							log.Printf("Failed to upsert document %s: %v", fileInfo.Path, err)
							errChan <- err
							return
						}
					}
				}

				mu.Lock()
				files = append(files, fileInfo)
				mu.Unlock()
			}(fileInfo, file)
		}
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

// upsertDocument inserts or updates a document in the PostgreSQL database.
func upsertDocument(ctx context.Context, entClient *ent.Client, path, description string, embedding []float32, repository string) error {
	vector := pgvector.NewVector(embedding)

	_, err := entClient.Document.Create().
		SetFilepath(path).
		SetRepository(repository).
		SetDescription(description).
		SetEmbedding(vector).
		Save(ctx)
	return err
}

// getEmbeddingFromDescription fetches the embedding for a given description using OpenAI.
func getEmbeddingFromDescription(client *openai.Client, description string) ([]float32, error) {
	if len(description) == 0 {
		return nil, fmt.Errorf("description is empty")
	}
	ctx := context.Background()

	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.F(openai.EmbeddingModelTextEmbedding3Small),
		Input: openai.F(openai.EmbeddingNewParamsInputUnion(openai.EmbeddingNewParamsInputArrayOfStrings{description})),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	var embedding []float32
	for _, v := range resp.Data[0].Embedding {
		embedding = append(embedding, float32(v))
	}

	return embedding, nil
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

// summarizeContent uses OpenAI to summarize the given text content.
func summarizeContent(client *openai.Client, content string) (string, error) {
	if len(content) == 0 {
		return "", nil
	}
	ctx := context.Background()

	// Prepare the prompt for summarization
	prompt := fmt.Sprintf(`Please provide a concise summary of the following content.
The summary will be used to retrieve relevant files to answer a user's question.
Please write the summary in the following manner:

- What is the code for? (e.g., "This function calculates the sum of two numbers.", "The document about package management in Go.", etc)
- Type of content: (e.g., "Code", "Documentation", "Article", etc)
- Function names: (e.g., "calculateSum", "main", etc)
- References: where this code is used or referenced.
- Any other relevant information

\n\n%s`, content)

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
