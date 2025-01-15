package llm

const (
	SUMMARIZE_FILE_CONTENT_PROMPT = `Please provide a concise summary of the following content.
The summary will be used to retrieve relevant files to answer a user's question.
Please write the summary in the following manner:

- What is the code for? (e.g., "This function calculates the sum of two numbers.", "The document about package management in Go.", etc)
- Type of content: (e.g., "Code", "Documentation", "Article", etc)
- Function names: (e.g., "calculateSum", "main", etc)
- References: where this code is used or referenced.
- Any other relevant information

\n\n%s`

	SUMMARIZE_REPO_CONTENT_PROMPT = `Please provide a concise summary of the repository structure.

This summary is used for new users to understand the repository structure.
Please include the following information:

- What is the repository about?
- What are the main directories and their purposes?
- Any important files or directories that users should know about?
- Important functions or classes that are used throughout the repository
- Concepts or technologies used in the repository

## Repository

Name: %s

Target Directory: %s

Files:

%s

`
)
