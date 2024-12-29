# AICoder

## Environment Variables

- `OPENAI_API_KEY`

## PGVector

```
brew install postgresql@15
```

```
CREATE DATABASE aicoder;
CREATE EXTENSION IF NOT EXISTS vector;
CREATE USER aicoder WITH PASSWORD 'aicoder';
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO aicoder;
GRANT ALL ON SCHEMA public TO aicoder;
```

https://github.com/pgvector/pgvector-go
