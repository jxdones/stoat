TARGET := stoat
VERSION := 0.4.0
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
	rm -rf bin
	rm -f $(TARGET)-$(VERSION).tar.gz

release: clean
	$(GO) build -o bin/$(TARGET) cmd/$(TARGET)/main.go
	tar -czvf $(TARGET)-$(VERSION).tar.gz bin/$(TARGET)

# Install to $GOBIN. Ensure $GOBIN is in your PATH.
install:
	$(GO) install ./cmd/$(TARGET)

install-prefix: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 bin/$(TARGET) $(DESTDIR)$(PREFIX)/bin/$(TARGET)