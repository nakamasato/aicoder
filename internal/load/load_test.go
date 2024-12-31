package load

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

// Mock data and helper functions can be added here

// func TestLoadRepoStructure(t *testing.T) {
// 	// Setup a mock repository or use a temporary directory with a real git repo
// 	ctx := context.Background()
// 	gitRootPath := "/path/to/mock/repo"
// 	branch := "main"
// 	commitHash := ""
// 	targetPath := ""
// 	include := []string{}
// 	exclude := []string{}

// 	// Call the function
// 	repoStructure, err := LoadRepoStructure(ctx, gitRootPath, branch, commitHash, targetPath, include, exclude)

// 	// Assertions
// 	assert.NoError(t, err)
// 	assert.NotNil(t, repoStructure)
// 	assert.Equal(t, gitRootPath, repoStructure.Root.Name)
// }

// func TestTraverseTree(t *testing.T) {
// 	// Setup a mock tree object
// 	ctx := context.Background()
// 	tree := &object.Tree{} // This should be a valid tree object
// 	parentPath := ""
// 	include := []string{}
// 	exclude := []string{}

// 	// Call the function
// 	files, err := traverseTree(ctx, tree, parentPath, exclude, include)

// 	// Assertions
// 	assert.NoError(t, err)
// 	assert.NotNil(t, files)
// 	// Add more assertions based on expected output
// }

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
