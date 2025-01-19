package retriever

import (
	"context"
	"testing"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/stretchr/testify/assert"
)

// Mock implementations for testing
type MockVectorStore struct{}

func (m *MockVectorStore) Search(ctx context.Context, repo, context, query string, limit int) (*vectorstore.SearchResult, error) {
	// Return a mock search result
	return &vectorstore.SearchResult{
		Documents: &[]vectorstore.DocumentWithScore{
			{Document: &vectorstore.Document{Filepath: "mock/file1.go"}, Score: 0.9},
			{Document: &vectorstore.Document{Filepath: "mock/file2.go"}, Score: 0.8},
		},
	}, nil
}

func (m *MockVectorStore) AddDocument(ctx context.Context, doc *vectorstore.Document) error {
	// Mock implementation of AddDocument
	return nil
}

func TestVectorestoreRetriever_Retrieve(t *testing.T) {
	mockStore := &MockVectorStore{}
	config := &config.AICoderConfig{Repository: "mockRepo", CurrentContext: "mockContext"}
	mockReader := file.MockFileReader{
		Content: "test content",
	}
	retriever := NewVectorstoreRetriever(mockStore, mockReader, config)

	files, err := retriever.Retrieve(context.Background(), "test query")
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Equal(t, "mock/file1.go", files[0].Path)
	assert.Equal(t, "mock/file2.go", files[1].Path)
}

func TestLLMRetriever_Retrieve(t *testing.T) {
	mockClient := llm.DummyClient{
		ReturnValue: `{"paths": ["mock/file1.go", "mock/file3.go"]}`,
	}
	mockReader := file.MockFileReader{
		Content: "test content",
	}
	config := &config.AICoderConfig{}
	retriever := NewLLMRetriever(mockClient, mockReader, config, "mock summary")

	files, err := retriever.Retrieve(context.Background(), "test query")
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Equal(t, "mock/file1.go", files[0].Path)
	assert.Equal(t, "mock/file3.go", files[1].Path)
}

// Mock implementations for testing
type MockRetriever struct {
	ReturnFiles []file.File
}

func (m *MockRetriever) Retrieve(ctx context.Context, query string) ([]file.File, error) {
	return m.ReturnFiles, nil
}

// Test for EnsembleRetriever
// Duplicated file should be removed
func TestEnsembleRetriever_Retrieve(t *testing.T) {
	mockRetriever1 := &MockRetriever{
		ReturnFiles: []file.File{
			{Path: "mock/file1.go"}, // common file
			{Path: "mock/file2.go"},
		},
	}
	mockRetriever2 := &MockRetriever{
		ReturnFiles: []file.File{
			{Path: "mock/file1.go"}, // common file
			{Path: "mock/file4.go"},
		},
	}
	ensembleRetriever := NewEnsembleRetriever(mockRetriever1, mockRetriever2)

	files, err := ensembleRetriever.Retrieve(context.Background(), "test query")
	assert.NoError(t, err)
	assert.Len(t, files, 3)
	assert.Equal(t, "mock/file1.go", files[0].Path)
	assert.Equal(t, "mock/file2.go", files[1].Path)
	assert.Equal(t, "mock/file4.go", files[2].Path)
}
