# AICoder

AICoder is a AI-powered CLI that helps you code quickly.

1. load
1. search
1. plan (validate)
1. apply (dryrun)
1. check

## Why is AICoder necessary?

- **Fast and secure**: AICoder works in your local, LLM (e.g. OpenAI) is the only external endpoint that AICoder interacts with.
- **CI support**: you can use the same CLI in you CI. (PR review with domain knowledge of the repository.)
- **Compliance**: Not like [devlo.ai](https://devlo.ai/) or [devin.ai](https://devin.ai/) (which I love using), no need of installation and setup at organization level, which is not easy and quick for an enterprise company.
- **Personal**: the concept is to help you improve your productivity by accumulating the domain knowledge in your local and speed up your development speed.

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

search:
  top_n: 5
```
