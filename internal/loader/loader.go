package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/ent/document"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/openai/openai-go"
)

type service struct {
	config      *config.AICoderConfig
	structure   *RepoStructure
	llmClient   llm.Client
	entClient   *ent.Client
	vectorstore vectorstore.VectorStore
}

func NewService(cfg *config.AICoderConfig, structure *RepoStructure, entClient *ent.Client, llmClient llm.Client, store vectorstore.VectorStore) *service {
	return &service{
		config:      cfg,
		structure:   structure,
		llmClient:   llmClient,
		entClient:   entClient,
		vectorstore: store,
	}
}

func (s *service) ReadRepoStructure(ctx context.Context, filename string) (*RepoStructure, error) {
	var repo RepoStructure
	if _, err := os.Stat(filename); err == nil {
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("Failed to read existing repo structure: %v", err)
		}
		if err := json.Unmarshal(data, &repo); err != nil {
			return nil, fmt.Errorf("Failed to parse existing repo structure: %v", err)
		}
	}
	return &repo, nil
}

// Load loads the repository structure and documents into the vector store.
func (s *service) UpdateDocuments(ctx context.Context) error {

	// clean up non-existing files
	var filePaths []string
	for fileinfo := range s.structure.Root.FileInfoGenerator() {
		filePaths = append(filePaths, fileinfo.Path)
	}
	s.entClient.Document.Delete().Where(
		document.RepositoryEQ(s.config.Repository),
		document.ContextEQ(s.config.CurrentContext),
		document.FilepathNotIn(filePaths...),
	).ExecX(ctx)

	var wg sync.WaitGroup
	var errChan = make(chan error, s.structure.Root.Size)
	loadCfg := s.config.GetCurrentLoadConfig()
	fmt.Printf("found %d files", s.structure.Root.Size)
	for fileinfo := range s.structure.Root.FileInfoGenerator() {
		wg.Add(1)
		go func(fileInfo FileInfo) {
			defer wg.Done()
			if fileinfo.IsDir {
				return
			}

			if loadCfg.IsExcluded(fileinfo.Path) && !loadCfg.IsIncluded(fileinfo.Path) {
				return
			}

			buf, err := os.ReadFile(fileinfo.Path)
			if err != nil {
				errChan <- fmt.Errorf("failed to open file %s: %v", fileinfo.Path, err)
				return
			}

			doc, err := s.entClient.Document.Query().Where(document.RepositoryEQ(s.config.Repository), document.FilepathEQ(fileinfo.Path), document.ContextEQ(s.config.CurrentContext)).First(ctx)
			if err == nil && doc.UpdatedAt.After(fileinfo.ModifiedAt) {
				fmt.Printf("Document %s is up-to-date\n", fileinfo.Path)
				return
			}

			if err != nil && !ent.IsNotFound(err) {
				errChan <- fmt.Errorf("failed to query document %s: %v", fileinfo.Path, err)
				return
			}

			if string(buf) == "" {
				fmt.Printf("File is empty: %s\n", fileinfo.Path)
				return
			}

			summary, err := s.llmClient.GenerateCompletionSimple(ctx, []openai.ChatCompletionMessageParamUnion{openai.UserMessage(fmt.Sprintf(llm.SUMMARIZE_FILE_CONTENT_PROMPT, string(buf)))})
			if err != nil {
				errChan <- fmt.Errorf("failed to summarize content: %v", err)
				return
			}
			if len(summary) == 0 {
				fmt.Printf("Summary is empty: %s\ncontent:%s", fileinfo.Path, string(buf))
				return
			}
			vsDoc := &vectorstore.Document{
				Repository:  s.config.Repository,
				Context:     s.config.CurrentContext,
				Filepath:    fileinfo.Path,
				Description: summary,
			}

			err = s.vectorstore.AddDocument(ctx, vsDoc)
			if err != nil {
				errChan <- fmt.Errorf("Failed to add vectorstore document %s: %v", fileinfo.Path, err)
				return
			}
			fmt.Printf("upserted document %s\n", fileinfo.Path)
		}(fileinfo)
	}
	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
	return nil
}

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

// FileInfoProvider is an interface for getting file information.
// This is useful for testing os.Stat.
type FileInfoProvider interface {
	Stat(name string) (os.FileInfo, error)
}

type OSFileInfoProvider struct{}

func (p *OSFileInfoProvider) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

var osFileInfoProvider FileInfoProvider = &OSFileInfoProvider{}

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
	fmt.Printf("Loading... gitRootPath:%s, targetPath:%s, include:%s, exclude:%s", gitRootPath, targetPath, strings.Join(include, ","), strings.Join(exclude, ","))
	tree, err := getTreeFromHead(gitRootPath)
	if err != nil {
		return RepoStructure{}, err
	}
	return LoadRepoStructure(ctx, gitRootPath, tree, targetPath, include, exclude, osFileInfoProvider)
}

func LoadRepoStructureFromCommitHash(ctx context.Context, gitRootPath, commitHash, targetPath string, include, exclude []string) (RepoStructure, error) {
	tree, err := getTreeFromCommitHash(gitRootPath, commitHash)
	if err != nil {
		return RepoStructure{}, err
	}
	return LoadRepoStructure(ctx, gitRootPath, tree, targetPath, include, exclude, osFileInfoProvider)
}

func LoadRepoStructureFromBranch(ctx context.Context, gitRootPath, branch, targetPath string, include, exclude []string) (RepoStructure, error) {
	tree, err := getTreeFromBranch(gitRootPath, branch)
	if err != nil {
		return RepoStructure{}, err
	}
	return LoadRepoStructure(ctx, gitRootPath, tree, targetPath, include, exclude, osFileInfoProvider)
}

func LoadRepoStructure(ctx context.Context, gitRootPath string, tree *object.Tree, targetPath string, include, exclude []string, fileInfoProvider FileInfoProvider) (RepoStructure, error) {
	rootFileInfo := FileInfo{
		Name:  gitRootPath,
		Path:  targetPath,
		IsDir: true,
	}

	var err error
	fmt.Printf("targetPath: %s", targetPath)
	if targetPath != "" {
		tree, err = tree.Tree(targetPath)
		if err != nil {
			return RepoStructure{}, fmt.Errorf("failed to get tree for target path: %w", err)
		}
	}
	children, err := traverseTree(ctx, tree, gitRootPath, targetPath, exclude, include, fileInfoProvider)
	if err != nil {
		return RepoStructure{}, err
	}
	rootFileInfo.Children = children
	for _, child := range children {
		rootFileInfo.Size += child.Size
	}

	return RepoStructure{
		GeneratedAt: time.Now(),
		Root:        rootFileInfo,
	}, nil
}

// traverseTree recursively traverses the Git tree and collects FileInfo.
// It updates the Description using OpenAI and stores embeddings in PostgreSQL.
func traverseTree(ctx context.Context, tree *object.Tree, gitRootPath, parentPath string, exclude, include []string, fileInfoProvider FileInfoProvider) ([]FileInfo, error) {
	var files []FileInfo

	for _, entry := range tree.Entries {
		filePath := filepath.Join(parentPath, entry.Name)
		// fmt.Printf("parentPath:%s,entry.Name:%s,filePath:%s\n", filePath, entry.Name, filePath)
		fileInfo := FileInfo{
			Name:  entry.Name,
			Path:  filePath,
			IsDir: entry.Mode == filemode.Dir,
			Size:  0,
		}

		if skip(filePath, exclude, include) {
			fmt.Printf("Skipping %s\n", filePath)
			continue
		}

		if entry.Mode == filemode.Dir {
			// fmt.Printf("traversing dir:%s", entry.Name)
			subtree, err := tree.Tree(entry.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get subtree for %s: %w", entry.Name, err)
			}
			children, err := traverseTree(ctx, subtree, gitRootPath, filePath, exclude, include, fileInfoProvider)
			if err != nil {
				return nil, err
			}
			fileInfo.Children = children
			for _, child := range children {
				fileInfo.Size += child.Size
			}
		} else {
			fileInfo.BlobHash = entry.Hash.String()
			info, err := fileInfoProvider.Stat(filepath.Join(gitRootPath, fileInfo.Path))
			if err != nil {
				fmt.Printf("Failed to stat file fileInfo.Path:%s, entry.Name:%s, err:%v", fileInfo.Path, entry.Name, err)
				continue
			} else {
				fileInfo.ModifiedAt = info.ModTime()
				fileInfo.Size += 1
			}
		}
		// fmt.Printf("fileInfo Path:%s, Size: %d", fileInfo.Path, fileInfo.Size)
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
