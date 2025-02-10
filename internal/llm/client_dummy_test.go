package llm_test

import (
	"context"
	"testing"

	"github.com/nakamasato/aicoder/internal/llm"
)

func TestGenerateCompletion(t *testing.T) {
	client := llm.DummyClient{ReturnValue: "test completion"}
	ctx := context.Background()
	messages := []llm.Message{}
	schema := llm.Schema{}

	result, err := client.GenerateCompletion(ctx, messages, schema)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "test completion" {
		t.Errorf("expected 'test completion', got %v", result)
	}
}

func TestGenerateCompletions(t *testing.T) {
	client := llm.DummyClient{}
	ctx := context.Background()
	messages := []llm.Message{}
	schema := llm.Schema{}

	results, err := client.GenerateCompletions(ctx, messages, schema, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 || results[0] != "dummy result" {
		t.Errorf("expected ['dummy result'], got %v", results)
	}
}

func TestGenerateCompletionSimple(t *testing.T) {
	client := llm.DummyClient{ReturnValue: "simple completion"}
	ctx := context.Background()
	messages := []llm.Message{}

	result, err := client.GenerateCompletionSimple(ctx, messages)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "simple completion" {
		t.Errorf("expected 'simple completion', got %v", result)
	}
}

func TestGenerateFunctionCalling(t *testing.T) {
	client := llm.DummyClient{}
	ctx := context.Background()
	messages := []llm.Message{}
	tools := []llm.Tool{}

	toolCalls, err := client.GenerateFunctionCalling(ctx, messages, tools)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if toolCalls != nil {
		t.Errorf("expected nil, got %v", toolCalls)
	}
}

func TestGetEmbedding(t *testing.T) {
	client := llm.DummyClient{}
	ctx := context.Background()
	content := "test content"

	embedding, err := client.GetEmbedding(ctx, content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(embedding) != 1536 {
		t.Errorf("expected embedding of length 1536, got %d", len(embedding))
	}
	for _, value := range embedding {
		if value != 0.1 {
			t.Errorf("expected all values to be 0.1, got %v", value)
		}
	}
}
