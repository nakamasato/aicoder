package loader

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
)

type MockFileInfoProvider struct{}

func (m *MockFileInfoProvider) Stat(name string) (os.FileInfo, error) {
	// Return a mock file info or an error based on the file name
	if name == "file1.txt" {
		return &mockFileInfo{name: "file1.txt", modTime: time.Now()}, nil
	}
	return nil, os.ErrNotExist
}

type mockFileInfo struct {
	name    string
	modTime time.Time
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0 }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }

func TestFileInfoGenerator(t *testing.T) {
	// Create a sample FileInfo structure
	root := FileInfo{
		Name:  "root",
		Path:  "/root",
		IsDir: true,
		Children: []FileInfo{
			{
				Name:  "file1.txt",
				Path:  "/root/file1.txt",
				IsDir: false,
			},
			{
				Name:  "subdir",
				Path:  "/root/subdir",
				IsDir: true,
				Children: []FileInfo{
					{
						Name:  "file2.txt",
						Path:  "/root/subdir/file2.txt",
						IsDir: false,
					},
				},
			},
		},
	}

	// Expected paths
	expectedPaths := []string{
		"/root/file1.txt",
		"/root/subdir",
		"/root/subdir/file2.txt",
	}

	// Collect paths from FileInfoGenerator
	var generatedPaths []string
	for fileinfo := range root.FileInfoGenerator() {
		generatedPaths = append(generatedPaths, fileinfo.Path)
	}

	// Check if the generated paths match the expected paths
	if len(generatedPaths) != len(expectedPaths) {
		t.Fatalf("expected %d paths, got %d", len(expectedPaths), len(generatedPaths))
	}

	for i, path := range generatedPaths {
		if path != expectedPaths[i] {
			t.Errorf("expected path %s, got %s", expectedPaths[i], path)
		}
	}
}

// countFiles returns the number of files or directories in the specified directory.
func countFiles(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	return len(entries), err
}

func TestLoadRepoStructure(t *testing.T) {
	ctx := context.Background()
	gitRootPath := "../../"
	targetPath := "cmd"
	include := []string{"ent/schema"}
	exclude := []string{"ent"}

	repoStructure, err := LoadRepoStructureFromHead(ctx, gitRootPath, targetPath, include, exclude)
	assert.NoError(t, err)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, gitRootPath, repoStructure.Root.Name)
	assert.True(t, repoStructure.Root.IsDir)

	// expected
	count, err := countFiles(filepath.Join(gitRootPath, targetPath))
	assert.NoError(t, err)
	if len(repoStructure.Root.Children) != count {
		for i, child := range repoStructure.Root.Children {
			t.Logf("child: %d %v", i, child.Path)
		}
		t.Fatalf("expected %d children, got %d", count, len(repoStructure.Root.Children))
	}
}

func TestTraverseTree(t *testing.T) {
	// Setup a mock tree object
	ctx := context.Background()
	gitRootPath := "../../"
	targetPath := "cmd"
	repo, err := git.PlainOpen(gitRootPath)
	assert.NoError(t, err)
	ref, err := repo.Head()
	assert.NoError(t, err)
	commit, err := repo.CommitObject(ref.Hash())
	assert.NoError(t, err)

	tree, err := commit.Tree()
	assert.NoError(t, err)
	tree, err = tree.Tree(targetPath)
	assert.NoError(t, err)
	include := []string{"ent/schema"}
	exclude := []string{"ent"}

	// Call the function
	files, err := traverseTree(ctx, tree, gitRootPath, targetPath, exclude, include, osFileInfoProvider)
	assert.NoError(t, err)

	// expected
	count, err := countFiles(filepath.Join(gitRootPath, targetPath))
	assert.NoError(t, err)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, files)
	if len(files) != count {
		t.Fatalf("expected %d files, got %d", count, len(files))
	}
}

func TestSkip(t *testing.T) {
	path := "some/path/to/file"
	exclude := []string{"some/path"}
	include := []string{"some/path/to"}

	// Call the function
	result := skip(path, exclude, include)

	// Assertions
	assert.False(t, result)
}

func TestMatchesPath(t *testing.T) {
	target := "some/path/to/file"
	pattern := "some/path"

	// Call the function
	result := matchesPath(target, pattern)

	// Assertions
	assert.True(t, result)
}
