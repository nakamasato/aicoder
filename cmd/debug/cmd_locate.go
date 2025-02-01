package debug

import (
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

var irrelevant bool

func locateCommand() *cobra.Command {
	locateCmd := &cobra.Command{
		Use:   "locate [query]",
		Short: "Locate the file in the repository",
		Run:   runLocate,
		Args:  cobra.MinimumNArgs(1),
	}
	locateCmd.Flags().BoolVarP(&irrelevant, "irrelevant", "r", false, "Use irrelevant search")


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
	res, err := lctr.Locate(ctx, irrelevant, query, repoStructure)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
}
