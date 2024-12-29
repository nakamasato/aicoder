.PHONY: generate
generate:
	go generate ./ent
.PHONY: lint
lint:
	golangci-lint run
.PHONY: test
test:
	go test -v ./...
.PHONY: fmt
fmt:
	go fmt ./...
.PHONY: run
run:
	go run main.go
.PHONY: db
db:
	psql aicoder
.PHONY: migrate
migrate:
	go run ent/migrate.go
