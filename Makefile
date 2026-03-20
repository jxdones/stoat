TARGET := stoat
VERSION := 0.10.1
GO := go
GOFMT := gofmt
LINTER := golangci-lint
PREFIX ?= /usr/local

.PHONY: build test test-integration fmt lint clean release install install-prefix

LDFLAGS := -ldflags "-s -w -X main.version=v$(VERSION)"

build:
	$(GO) build $(LDFLAGS) -o bin/$(TARGET) cmd/$(TARGET)/main.go

test:
	$(GO) test ./...

test-integration:
	TESTCONTAINERS_RYUK_DISABLED=true $(GO) test -count=1 -tags integration ./internal/database/integration/...

fmt:
	$(GOFMT) -s -w .

lint:
	$(LINTER) run ./...

clean:
	rm -rf bin dist

GOOS := $(shell go env GOOS)

release: clean
	mkdir -p dist
	GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(TARGET)-$(GOOS)-amd64 ./cmd/$(TARGET)
	GOARCH=arm64 $(GO) build $(LDFLAGS) -o dist/$(TARGET)-$(GOOS)-arm64 ./cmd/$(TARGET)

# Install to $GOBIN. Ensure $GOBIN is in your PATH.
install:
	$(GO) install $(LDFLAGS) ./cmd/$(TARGET)

install-prefix: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 bin/$(TARGET) $(DESTDIR)$(PREFIX)/bin/$(TARGET)
