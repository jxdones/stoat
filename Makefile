TARGET := stoat
VERSION := 0.1.0
GO := go
GOFMT := gofmt
LINTER := golangci-lint

.PHONY: build test fmt lint clean release

build:
	$(GO) build -o bin/$(TARGET) cmd/$(TARGET)/main.go

test:
	$(GO) test ./...

fmt:
	$(GOFMT) -s -w .

lint:
	$(LINTER) run ./...

clean:
	rm -rf bin

release: clean
	$(GO) build -o $(TARGET)
	tar -czvf $(TARGET)-$(VERSION).tar.gz $(TARGET)