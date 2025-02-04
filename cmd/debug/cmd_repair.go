package debug

import (
	"log"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/locator"
	"github.com/nakamasato/aicoder/internal/repairer"
	"github.com/spf13/cobra"
)

var inputFile string
var numOfSamples int64
var repairOutputFile string

func repairCommand() *cobra.Command {
	repairCmd := &cobra.Command{
		Use:   "repair",
		Short: "Repair the files in location.json",
		Run:   runRepair,
	}

	repairCmd.Flags().StringVarP(&inputFile, "input", "i", "location.json", "Input JSON file for the location")
	repairCmd.Flags().StringVarP(&repairOutputFile, "output", "o", "repair.json", "Output JSON file for the location")
	repairCmd.Flags().Int64VarP(&numOfSamples, "samples", "s", 1, "Number of repair samples to generate")

	return repairCmd
}

func runRepair(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	config := config.GetConfig()
	if openaiAPIKey != "" {
		config.OpenAIAPIKey = openaiAPIKey
	}
	if config.OpenAIAPIKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}
	llmClient := llm.NewOpenAIClient(config.OpenAIAPIKey)

	var location locator.LocationOutput
	if err := file.ReadObject(inputFile, &location); err != nil {
		log.Fatalf("failed to read location: %v", err)
	}

	// Repair the files
	rprr := repairer.NewRepairer(llmClient, &config)
	res, err := rprr.Repair(ctx, &location, numOfSamples)
	if err != nil {
		log.Fatalf("failed to repair: %v", err)
	}

	// Write the repair result
	if err := file.SaveObject(res, repairOutputFile); err != nil {
		log.Fatalf("failed to write repair result: %v", err)
	}
}
