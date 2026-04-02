# Contributing to Stoat

Thanks for your interest in contributing! Here's everything you need to get started.

## Before you start

For bug fixes, please [open an issue](https://github.com/jxdones/stoat/issues) first to describe the problem.
For new features, open an issue to discuss the idea before submitting a PR. This avoids wasted effort if the feature isn't a good fit.

## Running locally

**Requirements:** Go 1.22+, a C compiler (for the SQLite driver)

```sh
make build       # build binary to bin/stoat
make install     # install to $GOBIN
```

## Running tests

```sh
make test        # unit tests
make test-integration  # integration tests (requires Docker)
```

Integration tests spin up real database containers using `testcontainers-go`.

If you're running Docker via Colima, use:

```sh
DOCKER_HOST=unix://${HOME}/.colima/default/docker.sock \
TESTCONTAINERS_RYUK_DISABLED=true \
TESTCONTAINERS_HOST_OVERRIDE=127.0.0.1 \
make test-integration
```

## Code style

Run before submitting:

```sh
make fmt   # format
make lint  # lint (requires golangci-lint)
```

## Naming conventions

- **Packages**: short, lowercase, no underscores — e.g. `sqlite`, `postgres`, `filterbox`
- **Interfaces**: descriptive nouns, no `I` prefix — e.g. `Connection`, `DataSource`
- **Handler functions**: `handle<Action>` — e.g. `handleKeyPress`, `handleConnected`
- **Test files**: `_test.go` suffix; use `package foo_test` for black-box tests
- **Integration tests**: `//go:build integration` build tag, placed under `internal/database/integration/`
- **Test style**: table-driven tests preferred for anything with more than one case

## Submitting a PR

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Run `make fmt`, `make lint`, and `make test`
4. Open a PR referencing the issue it addresses

**Commit message style:** imperative, capitalized, no prefix — e.g. `Add dark mode support`, `Fix crash on empty table`.