// cmd/loader.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// FileInfo represents information about a file or directory.
type FileInfo struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	IsDir       bool       `json:"is_dir"`
	Children    []FileInfo `json:"children,omitempty"`
}

// RepoStructure represents the entire repository structure.
type RepoStructure struct {
	Root FileInfo `json:"root"`
}

var outputFile string

// loaderCmd represents the loader command
var loaderCmd = &cobra.Command{
	Use:   "loader [path]",
	Short: "Load the repository structure and export it to a JSON file.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		repo, err := loadRepoStructure(path)
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
}

// loadRepoStructure traverses the directory and builds the RepoStructure.
func loadRepoStructure(path string) (RepoStructure, error) {
	info, err := os.Stat(path)
	if err != nil {
		return RepoStructure{}, err
	}

	rootFileInfo := FileInfo{
		Name:  info.Name(),
		IsDir: info.IsDir(),
	}

	if info.IsDir() {
		children, err := traverseDirectory(path)
		if err != nil {
			return RepoStructure{}, err
		}
		rootFileInfo.Children = children
	} else {
		rootFileInfo.Description = getFileDescription(path)
	}

	return RepoStructure{Root: rootFileInfo}, nil
}

// traverseDirectory recursively traverses the directory and collects FileInfo.
func traverseDirectory(path string) ([]FileInfo, error) {
	var files []FileInfo

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		filePath := filepath.Join(path, entry.Name())
		_, err := entry.Info()
		if err != nil {
			return nil, err
		}

		fileInfo := FileInfo{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
		}

		if entry.IsDir() {
			children, err := traverseDirectory(filePath)
			if err != nil {
				return nil, err
			}
			fileInfo.Children = children
		} else {
			fileInfo.Description = getFileDescription(filePath)
		}

		files = append(files, fileInfo)
	}

	return files, nil
}

// getFileDescription provides a simple description of the file.
// ここではファイルサイズを使用していますが、必要に応じて変更してください。
func getFileDescription(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "Unable to get description"
	}
	return fmt.Sprintf("File size: %d bytes", info.Size())
}
