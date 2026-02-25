TARGET := stoat
VERSION := 0.1.0
GO := go
GOFMT := gofmt
LINTER := golangci-lint

.PHONY: test fmt lint

test:
	$(GO) test ./...

fmt:
	$(GOFMT) -s -w .

lint:
	$(LINTER) run ./...
