package debug

import (
	"fmt"
	"log"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/spf13/cobra"
)

func agentCommand() *cobra.Command {
	agentCmd := &cobra.Command{
		Use:   "agent",
		Short: "Run agent",
		Run:   runAgent,
	}
	return agentCmd
}

func runAgent(cmd *cobra.Command, args []string) {
	// Run agent
	ctx := cmd.Context()
	config := config.GetConfig()
	if openaiAPIKey != "" {
		config.OpenAIAPIKey = openaiAPIKey
	}
	if config.OpenAIAPIKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}
	llmClient := llm.NewOpenAIClient(config.OpenAIAPIKey)
	toolCalls, err := llmClient.GenerateFunctionCalling(
		ctx,
		[]llm.Message{{Role: llm.RoleUser, Content: "What's the weather in Tokyo?"}},
		[]llm.Tool{{Name: "weather", Description: "Get the weather", Properties: map[string]interface{}{"location": map[string]string{"type": "string"}}, RequiredProperties: []string{"location"}}},
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Generated tool calls: %d\n", len(toolCalls))
	for _, tc := range toolCalls {
		location := tc.Arguments["location"].(string)
		log.Println(tc.FunctionName, location)
	}
}
