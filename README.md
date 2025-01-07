# AICoder
## Prerequisites

- Go version 1.23.2 or later
- PostgreSQL 15 or later with the `pgvector` extension installed.
- `OPENAI_API_KEY` environment variable set for using OpenAI API.

## Available Commands
1. **load**: Load the repository structure from a Git repository and export it to a JSON file with summaries.
2. **search**: Search for files related to a given query.
3. **plan**: Generate a project plan based on a specified goal.
4. **apply**: Apply changes based on the configuration provided in a plan file.
5. **check**: Validate the configuration and ensure all parameters are correctly set up.

## Example Usages
- To load the repository structure and summarize files:
  ```bash
  aicoder load --output=repo_structure.json
  ```
- To search for a specific file related to a query:
  ```bash
  aicoder search --query="function example"
  ```
- To generate a plan based on a goal:
  ```bash
  aicoder plan --goal="improve CLI documentation" --output=plan.json
  ```
- To apply changes defined in a plan:
  ```bash
  aicoder apply --planfile=plan.json
  ```

AICoder is a AI-powered CLI that helps you code quickly.

1. load
1. search
1. plan (validate)
1. apply (dryrun)
1. check

## Why is AICoder necessary?

- **Fast and secure**: AICoder works in your local, LLM (e.g. OpenAI) is the only external endpoint that AICoder interacts with.
- **CI support** (TODO): you can use the same CLI in you CI. (PR review with domain knowledge of the repository.)
- **Compliance**: Not like [devlo.ai](https://devlo.ai/) or [devin.ai](https://devin.ai/) (which I love using), no need of organization-level installation and setup, which is not easy and quick for an enterprise company.
- **Personal**: the concept is to help you improve your productivity by accumulating the domain knowledge in your local and accelarate up your development speed.
- **Multi files change** (TODO): AICoder aims at complicated changes which requires changes in multiple files.

## Environment Variables

- `OPENAI_API_KEY`

## PGVector

```
brew install postgresql@15
```

```sql
CREATE DATABASE aicoder;
CREATE EXTENSION IF NOT EXISTS vector;
CREATE USER aicoder WITH PASSWORD 'aicoder';
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO aicoder;
GRANT ALL ON SCHEMA public TO aicoder;
```

https://github.com/pgvector/pgvector-go

## Configuration

```yaml
repository: aicoder
load:
  exclude:
    - ent
  include:
    - ent/schema
  target_path: cmd

search:
  top_n: 5
```
