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
)
