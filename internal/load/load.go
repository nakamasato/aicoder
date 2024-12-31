package load

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

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

// FilePathGenerator generates file paths from the FileInfo structure.
func (f *FileInfo) FilePathGenerator() <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch) // Ensure the channel is closed when done
		for _, child := range f.Children {
			if child.IsDir {
				for path := range child.FilePathGenerator() {
					ch <- path
				}
			} else {
				ch <- child.Path
			}
		}
	}()
	return ch
}

// LoadRepoStructure loads the repository structure from the specified Git repository.
// Using git is to exclude files that are not git tracked.
func LoadRepoStructure(ctx context.Context, gitRootPath, branch, commitHash, targetPath string, include, exclude []string) (RepoStructure, error) {
	repo, err := git.PlainOpen(gitRootPath)
	if err != nil {
		return RepoStructure{}, fmt.Errorf("failed to open git repository: %w", err)
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
		Name:  gitRootPath,
		Path:  targetPath,
		IsDir: true,
	}

	if targetPath != "" {
		log.Printf("targetPath: %s", targetPath)
		tree, err = tree.Tree(targetPath)
		if err != nil {
			return RepoStructure{}, fmt.Errorf("failed to get tree for target path: %w", err)
		}
	}
	children, err := traverseTree(ctx, tree, targetPath, exclude, include)
	if err != nil {
		return RepoStructure{}, err
	}
	rootFileInfo.Children = children

	return RepoStructure{
		GeneratedAt: time.Now(),
		Root:        rootFileInfo,
	}, nil
}

// traverseTree recursively traverses the Git tree and collects FileInfo.
// It updates the Description using OpenAI and stores embeddings in PostgreSQL.
func traverseTree(ctx context.Context, tree *object.Tree, parentPath string, exclude, include []string) ([]FileInfo, error) {
	var files []FileInfo

	for _, entry := range tree.Entries {
		filePath := filepath.Join(parentPath, entry.Name)
		fileInfo := FileInfo{
			Name:  entry.Name,
			Path:  filePath,
			IsDir: entry.Mode == filemode.Dir,
		}

		if skip(filePath, exclude, include) {
			log.Printf("Skipping %s\n", filePath)
			continue
		}

		if entry.Mode == filemode.Dir {
			subtree, err := tree.Tree(entry.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get subtree for %s: %w", entry.Name, err)
			}
			children, err := traverseTree(ctx, subtree, filePath, exclude, include)
			if err != nil {
				return nil, err
			}
			fileInfo.Children = children
		} else {
			fileInfo.BlobHash = entry.Hash.String()
			// file, err := tree.File(entry.Name)
			// if err != nil {
			// 	return nil, fmt.Errorf("failed to get file %s: %w", entry.Name, err)
			// }
		}
		files = append(files, fileInfo)
	}
	return files, nil
}

func skip(path string, exclude, include []string) bool {
	return isExcluded(path, exclude) && !isIncluded(path, include)
}

func isExcluded(path string, exclude []string) bool {
	for _, excl := range exclude {
		if matchesPath(path, excl) {
			return true
		}
	}
	return false
}

func isIncluded(path string, include []string) bool {
	for _, incl := range include {
		if matchesPath(path, incl) {
			return true
		}
	}
	return false
}

func matchesPath(target, pattern string) bool {
	return strings.HasPrefix(target, pattern)
}
