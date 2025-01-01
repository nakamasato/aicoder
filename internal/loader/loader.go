package loader

import (
	"context"
	"fmt"
	"log"
	"os"
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
	ModifiedAt  time.Time  `json:"modified_at,omitempty"`
	Size        int64      `json:"size,omitempty"`
}

// RepoStructure represents the entire repository structure.
type RepoStructure struct {
	GeneratedAt time.Time `json:"generated_at"`
	Root        FileInfo  `json:"root"`
}

// FilePathGenerator generates file paths from the FileInfo structure.
func (f *FileInfo) FileInfoGenerator() <-chan FileInfo {
	ch := make(chan FileInfo)
	go func() {
		defer close(ch) // Ensure the channel is closed when done
		for _, child := range f.Children {
			ch <- child
			if child.IsDir {

				for fileinfo := range child.FileInfoGenerator() {
					ch <- fileinfo
				}
			}
		}
	}()
	return ch
}

func getTreeFromHead(gitRootPath string) (*object.Tree, error) {
	repo, err := git.PlainOpen(gitRootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit for HEAD: %w", err)
	}
	return commit.Tree()
}

func getTreeFromCommitHash(gitRootPath, commitHash string) (*object.Tree, error) {
	repo, err := git.PlainOpen(gitRootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}
	commit, err := repo.CommitObject(plumbing.NewHash(commitHash))
	if err != nil {
		return nil, fmt.Errorf("failed to get commit %s: %w", commitHash, err)
	}
	return commit.Tree()
}

func getTreeFromBranch(gitRootPath, branch string) (*object.Tree, error) {
	repo, err := git.PlainOpen(gitRootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}
	ref, err := repo.Reference(plumbing.NewBranchReferenceName(branch), true)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch %s: %w", branch, err)
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit for branch %s: %w", branch, err)
	}
	return commit.Tree()
}

// LoadRepoStructure loads the repository structure from the specified Git repository.
// Using git is to exclude files that are not git tracked.
func LoadRepoStructureFromHead(ctx context.Context, gitRootPath, targetPath string, include, exclude []string) (RepoStructure, error) {
	tree, err := getTreeFromHead(gitRootPath)
	if err != nil {
		return RepoStructure{}, err
	}
	return LoadRepoStructure(ctx, gitRootPath, tree, targetPath, include, exclude)
}

func LoadRepoStructureFromCommitHash(ctx context.Context, gitRootPath, commitHash, targetPath string, include, exclude []string) (RepoStructure, error) {
	tree, err := getTreeFromCommitHash(gitRootPath, commitHash)
	if err != nil {
		return RepoStructure{}, err
	}
	return LoadRepoStructure(ctx, gitRootPath, tree, targetPath, include, exclude)
}

func LoadRepoStructureFromBranch(ctx context.Context, gitRootPath, branch, targetPath string, include, exclude []string) (RepoStructure, error) {
	tree, err := getTreeFromBranch(gitRootPath, branch)
	if err != nil {
		return RepoStructure{}, err
	}
	return LoadRepoStructure(ctx, gitRootPath, tree, targetPath, include, exclude)
}

func LoadRepoStructure(ctx context.Context, gitRootPath string, tree *object.Tree, targetPath string, include, exclude []string) (RepoStructure, error) {
	rootFileInfo := FileInfo{
		Name:  gitRootPath,
		Path:  targetPath,
		IsDir: true,
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
			Size:  0,
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
			for _, child := range children {
				fileInfo.Size += child.Size
			}
		} else {
			fileInfo.BlobHash = entry.Hash.String()
			info, err := os.Stat(fileInfo.Path)
			if err != nil {
				log.Printf("Failed to stat file %s: %v", fileInfo.Path, err)
				continue
			} else {
				fileInfo.ModifiedAt = info.ModTime()
				fileInfo.Size += 1
			}
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

func LoadFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return string(data), nil
}
