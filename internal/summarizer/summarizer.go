package summarizer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/ent/document"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/vectorstore"
)

type Language string

const (
	LanguageEnglish  Language = "en"
	LanguageJapanese Language = "ja"
)

type EnvVar struct {
	Name     string `json:"name" jsonschema_description:"The name of the environment variable."`
	Desc     string `json:"desc" jsonschema_description:"The description of the environment variable."`
	Required bool   `json:"required" jsonschema_description:"Whether the environment variable is required or not."`
}

type OverallSummary struct {
	Overview           string   `json:"overview" jsonschema_description:"The overview of the repository."`
	Features           []string `json:"features" jsonschema_description:"The main features of the repository."`
	Configuration      string   `json:"configuration" jsonschema_description:"The configuration of the repository. Configuration files (include simple example if exists)"`
	EnvVars            []EnvVar `json:"environment_variables" jsonschema_description:"The environment variables used in the repository."`
	DirectoryStructure string   `json:"directory_structure" jsonschema_description:"The directory structure of the repository. Not only the root directories but also subdirectories if they contains core implementations. Include simplified directory structure diagram like the result of tree command with short explanation for each directory. You can omit unimportant directories."`
	Entrypoints        []string `json:"entrypoints" jsonschema_description:"All the entry points of the repository. Please write the executable commands. CLIs, web servers start command, etc."`
	ImportantFiles     []string `json:"important_files" jsonschema_description:"Important files or directories that users should know about. Configuration files, main entry points, files that contain important functions or classes."`
	ImportantFunctions []string `json:"important_functions" jsonschema_description:"Important functions or classes that are used throughout the repository."`
	Dependencies       string   `json:"dependencies" jsonschema_description:"Internal dependencies or relationships between files or directories. Simplified diagram in mermaid format would be helpful."`
	Technologies       []string `json:"technologies" jsonschema_description:"Concepts or technologies used in the repository. e.g. frameworks, libraries, etc."`
}

type RepoSummary struct {
	OverallSummary     OverallSummary              `json:"summary" jsonschema_description:"The summary of the repository."`
	DirectorySummaries map[string]DirectorySummary `json:"directory_summaries" jsonschema_description:"Summaries of each directory in the repository."`
}

var RepoSummarySchemaParam = llm.GenerateSchema[OverallSummary]("summary", "The summary of the repository.")

type service struct {
	config      *config.AICoderConfig
	llmClient   llm.Client
	entClient   *ent.Client
	vectorstore vectorstore.VectorStore
}

func NewService(cfg *config.AICoderConfig, entClient *ent.Client, llmClient llm.Client, store vectorstore.VectorStore) *service {
	return &service{
		config:      cfg,
		llmClient:   llmClient,
		entClient:   entClient,
		vectorstore: store,
	}
}

// groupDocumentsByDirectory organizes documents into directory groups
func groupDocumentsByDirectory(documents []*vectorstore.Document) map[string][]*vectorstore.Document {
	docsByDir := make(map[string][]*vectorstore.Document)
	for _, doc := range documents {
		dir := filepath.Dir(doc.Filepath)
		docsByDir[dir] = append(docsByDir[dir], doc)
	}
	return docsByDir
}

type DirectorySummary struct {
	Purpose     string   `json:"purpose" jsonschema_description:"The main purpose of this directory"`
	Components  []string `json:"components" jsonschema_description:"Key components and their roles in this directory"`
	Description string   `json:"description" jsonschema_description:"Detailed description of the directory's contents and organization"`
}

var DirectorySummarySchemaParam = llm.GenerateSchema[DirectorySummary]("directory_summary", "Summary of a directory's contents and purpose")

// generateDirectorySummary creates a summary for a specific directory
func (s *service) generateDirectorySummary(ctx context.Context, dirPath string, docs []*vectorstore.Document) (*DirectorySummary, error) {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Directory: %s\n\n", dirPath))
	for _, doc := range docs {
		builder.WriteString(fmt.Sprintf("File: %s\n", doc.Filepath))
		builder.WriteString(fmt.Sprintf("Summary: %s\n\n", doc.Description))
	}

	prompt := fmt.Sprintf("Please analyze the following directory and provide a structured summary of its purpose and components:\n\n%s", builder.String())
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: prompt},
	}

	jsonSummary, err := s.llmClient.GenerateCompletion(ctx, messages, DirectorySummarySchemaParam)
	if err != nil {
		return nil, err
	}

	var dirSummary DirectorySummary
	if err := json.Unmarshal([]byte(jsonSummary), &dirSummary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal directory summary: %w", err)
	}

	return &dirSummary, nil
}

func (s *service) UpdateRepoSummary(ctx context.Context, language Language, outputfile string) (string, error) {
	docs, err := s.entClient.Document.Query().Where(document.RepositoryEQ(s.config.Repository), document.ContextEQ(s.config.CurrentContext)).All(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to query documents: %v", err)
	}

	var documents []*vectorstore.Document
	for _, doc := range docs {
		documents = append(documents, &vectorstore.Document{
			Repository:  doc.Repository,
			Context:     doc.Context,
			Filepath:    doc.Filepath,
			Description: doc.Description,
		})
	}

	// Group documents by directory
	docsByDir := groupDocumentsByDirectory(documents)

	// Generate summaries for each directory
	var directorySummariesText []string
	directorySummaries := make(map[string]DirectorySummary)
	for dir, dirDocs := range docsByDir {
		dirSummary, err := s.generateDirectorySummary(ctx, dir, dirDocs)
		if err != nil {
			return "", fmt.Errorf("failed to generate summary for directory %s: %v", dir, err)
		}

		// Format the summary for the final repository summary prompt
		summaryText := fmt.Sprintf("Purpose: %s\n\nKey Components:\n%s\n\nDescription: %s",
			dirSummary.Purpose,
			strings.Join(dirSummary.Components, "\n"),
			dirSummary.Description)
		directorySummariesText = append(directorySummariesText, fmt.Sprintf("Directory %s:\n%s", dir, summaryText))

		directorySummaries[dir] = *dirSummary
	}

	// Generate final repository summary
	prompt := fmt.Sprintf(llm.SUMMARIZE_REPO_CONTENT_PROMPT, s.config.Repository, s.config.CurrentContext, strings.Join(directorySummariesText, "\n\n"), language)
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: prompt},
	}
	res, err := s.llmClient.GenerateCompletion(ctx, messages, RepoSummarySchemaParam)
	if err != nil {
		return "", fmt.Errorf("failed to generate completion: %v", err)
	}

	// Marshal the summary data to JSON
	var summary OverallSummary
	if err := json.Unmarshal([]byte(res), &summary); err != nil {
		return "", fmt.Errorf("failed to unmarshal summary: %v", err)
	}

	// Add directory summaries to the final summary
	repoSummary := RepoSummary{
		OverallSummary:     summary,
		DirectorySummaries: directorySummaries,
	}

	summaryJSON, err := json.MarshalIndent(repoSummary, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal summary to JSON: %v", err)
	}

	// Write the JSON to the output file
	if err := os.WriteFile(outputfile, summaryJSON, 0644); err != nil {
		log.Fatalf("failed to write summary to file: %v", err)
	}

	return res, nil
}

// ReadSummary reads the summary from the given file
func ReadSummary(ctx context.Context, filename string) (*OverallSummary, error) {
	// Read the file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Unmarshal the JSON content
	var summary OverallSummary
	if err := json.Unmarshal(content, &summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return &summary, nil
}
