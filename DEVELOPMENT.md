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
go run main.go setup
```

Ref: https://github.com/pgvector/pgvector-go/blob/master/ent/schema/item.go

