package load

import (
	"testing"
)

func TestPathGenerator(t *testing.T) {
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
		"/root/subdir/file2.txt",
	}

	// Collect paths from PathGenerator
	var generatedPaths []string
	for path := range root.FilePathGenerator() {
		generatedPaths = append(generatedPaths, path)
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
