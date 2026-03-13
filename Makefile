TARGET := stoat
VERSION := 0.5.0
GO := go
GOFMT := gofmt
LINTER := golangci-lint
PREFIX ?= /usr/local

.PHONY: build test fmt lint clean release install install-prefix

build:
	$(GO) build -o bin/$(TARGET) cmd/$(TARGET)/main.go

test:
	$(GO) test ./...

fmt:
	$(GOFMT) -s -w .

lint:
	$(LINTER) run ./...

clean:
	rm -rf bin dist

release: clean
	mkdir -p dist
	GOOS=darwin  GOARCH=amd64 $(GO) build -ldflags="-s -w" -o dist/$(TARGET)-darwin-amd64   ./cmd/$(TARGET)
	GOOS=darwin  GOARCH=arm64 $(GO) build -ldflags="-s -w" -o dist/$(TARGET)-darwin-arm64   ./cmd/$(TARGET)
	GOOS=linux   GOARCH=amd64 $(GO) build -ldflags="-s -w" -o dist/$(TARGET)-linux-amd64    ./cmd/$(TARGET)
	GOOS=linux   GOARCH=arm64 $(GO) build -ldflags="-s -w" -o dist/$(TARGET)-linux-arm64    ./cmd/$(TARGET)

# Install to $GOBIN. Ensure $GOBIN is in your PATH.
install:
	$(GO) install ./cmd/$(TARGET)

install-prefix: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 bin/$(TARGET) $(DESTDIR)$(PREFIX)/bin/$(TARGET)