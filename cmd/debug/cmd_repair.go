package debug

import (
	"log"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/locator"
	"github.com/spf13/cobra"
)

var inputFile string

func repairCommand() *cobra.Command {
	repairCmd := &cobra.Command{
		Use:   "repair",
		Short: "Repair the files in location.json",
		Run:   runRepair,
	}

	repairCmd.Flags().StringVarP(&inputFile, "input", "i", "location.json", "Input JSON file for the location")
	repairCmd.Flags().StringVarP(&outputFile, "output", "o", "repair.json", "Output JSON file for the location")

	return repairCmd
}

func runRepair(cmd *cobra.Command, args []string) {
	// ctx := cmd.Context()
	config := config.GetConfig()
	if openaiAPIKey != "" {
		config.OpenAIAPIKey = openaiAPIKey
	}
	if config.OpenAIAPIKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}
	// llmClient := llm.NewOpenAIClient(config.OpenAIAPIKey)

	var location locator.LocationOutput
	if err := file.ReadObject(inputFile, &location); err != nil {
		log.Fatalf("failed to read location: %v", err)
	}
}
