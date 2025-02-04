package repairer

import (
	"testing"

	"github.com/nakamasato/aicoder/internal/llm"
)

func TestMakePrompt(t *testing.T) {
	templateContent := `Query: {{.Query}}
Instruction: {{.RepairRelevantFileInstruction}}
Content: {{.Content}}`

	query := "What is the issue?"
	content := "This is the content of the block."
	expectedPrompt := `Query: What is the issue?
Instruction: Below are some code segments, each from a relevant file. One or more of these files may contain bugs.
Content: This is the content of the block.`

	prompt, err := makePrompt(templateContent, query, content)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if prompt != expectedPrompt {
		t.Errorf("Expected prompt to be:\n%s\nGot:\n%s", expectedPrompt, prompt)
	}
}

func TestMakeGetBlockContentPrompt(t *testing.T) {
	// Define a simple template for testing
	const testTemplate = `Block: {{.Block.Name}}, Content: {{.Content}}`

	// Create a sample block and content
	block := llm.Block{
		Name: "TestBlock",
	}
	content := "This is a test content."

	// Call the function with the test template
	result, err := makeGetBlockContentPrompt(testTemplate, block, content)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Define the expected result
	expected := "Block: TestBlock, Content: This is a test content."

	// Check if the result matches the expected output
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}
