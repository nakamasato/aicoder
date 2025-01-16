# Development

## [cobra-cli](https://github.com/spf13/cobra-cli)

```
go install github.com/spf13/cobra-cli@latest
```

```
cobra-cli init
```

```
cobra-cli add loader
```

> [!NOTE]
> We don't use the default package structure as it's hard to manage each subcommand if they are in the same package.
> Instead we use a separate package for each subcommand.
> ```
> cmd
> ├── loader
> │   └── cmd.go
> ├── root.go
> └── setup
>     └── cmd.go
> ```

## [pgvector-go](https://github.com/pgvector/pgvector-go)



## [entgo](https://github.com/ent/ent)

```
go run -mod=mod entgo.io/ent/cmd/ent new Document
```

```
make generate
```

or

```
go run main.go db migrate
```

Ref: https://github.com/pgvector/pgvector-go/blob/master/ent/schema/item.go

## Test

```
TEST_DATABASE_URL=postgres://aicoder:aicoder@localhost:5432/aicoder_test?sslmode=disable
```

```sql
CREATE DATABASE aicoder_test;
\c aicoder_test
CREATE EXTENSION IF NOT EXISTS vector;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO aicoder;
GRANT ALL ON SCHEMA public TO aicoder;
```

```
go run main.go db migrate --db-conn $TEST_DATABASE_URL
```

If you're using VSCode, set `Go: Test Env File` to `${workspaceFolder}/.env` to load env var in `run test`

```
make test
```
