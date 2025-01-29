package retriever

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/loader"
	"github.com/nakamasato/aicoder/internal/vectorstore"
)

type Retriever interface {
	Retrieve(ctx context.Context, query string) ([]file.File, error)
}

type VectorestoreRetriever struct {
	store  vectorstore.VectorStore
	reader file.FileReader
	config *config.AICoderConfig
}

func NewVectorstoreRetriever(store vectorstore.VectorStore, reader file.FileReader, config *config.AICoderConfig) *VectorestoreRetriever {
	return &VectorestoreRetriever{
		store:  store,
		reader: reader,
		config: config,
	}
}

func (v VectorestoreRetriever) Retrieve(ctx context.Context, query string) ([]file.File, error) {

	// Get relevant files based on the query
	res, err := v.store.Search(ctx, v.config.Repository, v.config.CurrentContext, query, 10)
	if err != nil {
		log.Fatalf("failed to search: %v", err)
	}

	// Load file content
	var files []file.File
	fileMap := make(map[string]bool)

	fmt.Printf("Found %d files using embedding\n", len(*res.Documents))
	for i, doc := range *res.Documents {
		if fileMap[doc.Document.Filepath] {
			continue
		}
		fmt.Printf("%d: %s (score: %.2f)\n", i, doc.Document.Filepath, doc.Score)
		content, err := v.reader.ReadContent(doc.Document.Filepath)
		if err != nil {
			log.Fatalf("failed to load file content. you might need to refresh loader by `aicoder load -r`: %v", err)
		}
		files = append(files, file.File{Path: doc.Document.Filepath, Content: content})
		fileMap[doc.Document.Filepath] = true
	}
	return files, nil
}

type LLMRetriever struct {
	llmClient llm.Client
	reader    file.FileReader
	structure *loader.RepoStructure
	config    *config.AICoderConfig
}

func NewLLMRetriever(llmClient llm.Client, reader file.FileReader, config *config.AICoderConfig, structure *loader.RepoStructure) *LLMRetriever {
	return &LLMRetriever{
		llmClient: llmClient,
		reader:    reader,
		config:    config,
		structure: structure,
	}
}

//go:embed templates/repo_summary.tmpl
var RepoSummaryTemplate string

func (l LLMRetriever) Retrieve(ctx context.Context, query string) ([]file.File, error) {
	prompt := "Please extract files that are relevant to the given query.\nRepoStructure:\n```\n%s\n```\n"
	content, err := l.llmClient.GenerateCompletion(ctx,
		[]llm.Message{
			{Role: llm.RoleSystem, Content: fmt.Sprintf(prompt, l.structure.ToTreeString())},
			{Role: llm.RoleUser, Content: fmt.Sprintf("query: %s", query)},
		},
		llm.FileListSchemaParam,
	)
	if err != nil {
		log.Fatalf("failed to generate completion: %v", err)
	}

	var filelist llm.FileList
	err = json.Unmarshal([]byte(content), &filelist)
	if err != nil {
		log.Fatalf("failed to unmarshal relevant files: %v", err)
	}

	var files []file.File
	fmt.Printf("Found %d files using LLM\n", len(filelist.Paths))
	for i, path := range filelist.Paths {
		fmt.Printf("%d: %s\n", i, path)
		content, err := l.reader.ReadContent(path)
		if err != nil {
			fmt.Printf("failed to load file content. skip: %v", err)
			continue
		}
		files = append(files, file.File{Path: path, Content: content})
	}
	return files, nil
}

type EnsembleRetriever struct {
	retrievers []Retriever
}

func NewEnsembleRetriever(retrievers ...Retriever) *EnsembleRetriever {
	return &EnsembleRetriever{retrievers: retrievers}
}

func (e EnsembleRetriever) Retrieve(ctx context.Context, query string) ([]file.File, error) {
	var files []file.File
	fileMap := make(map[string]bool)

	for _, r := range e.retrievers {
		fs, err := r.Retrieve(ctx, query)
		if err != nil {
			log.Fatalf("failed to retrieve files: %v", err)
		}
		for _, f := range fs {
			if !fileMap[f.Path] { // Check if the file is already added
				files = append(files, f)
				fileMap[f.Path] = true // Mark the file as added
			}
		}
	}

	return files, nil
}
