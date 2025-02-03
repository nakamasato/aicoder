package debug

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/loader"
	"github.com/nakamasato/aicoder/internal/locator"
	"github.com/spf13/cobra"
)

var locatorType locator.LocatorType

func locateCommand() *cobra.Command {
	locateCmd := &cobra.Command{
		Use:   "locate [query]",
		Short: "Locate the file in the repository",
		Run:   runLocate,
		Args:  cobra.MinimumNArgs(1),
	}
	locateCmd.Flags().StringVarP(&outputFile, "output", "o", "location.json", "Output JSON file for the location")
	locateCmd.Flags().VarP(&locatorType, "irrelevant", "r", "locator type")

	return locateCmd
}

func runLocate(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	config := config.GetConfig()
	if openaiAPIKey != "" {
		config.OpenAIAPIKey = openaiAPIKey
	}
	if config.OpenAIAPIKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}
	llmClient := llm.NewOpenAIClient(config.OpenAIAPIKey)

	lctr := locator.NewLocator(llmClient, &config)

	var repoStructure loader.RepoStructure
	if err := file.ReadObject("repo_structure.json", &repoStructure); err != nil {
		log.Fatalf("failed to read repo structure: %v", err)
	}

	query := strings.Join(args, " ")
	location, err := lctr.Locate(ctx, locatorType, query, repoStructure, 2)

	if err != nil {
		log.Fatal(err)
	}

	data, err := json.Marshal(location)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))

	// Save location to file
	if err := file.SaveObject(location, outputFile); err != nil {
		log.Fatalf("failed to save location to file: %v", err)
	}
}
