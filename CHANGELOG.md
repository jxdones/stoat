# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Open in `$EDITOR`** (`Ctrl+E`). Press `Ctrl+E` from anywhere to open your `$EDITOR` with a SQL comment template. Save and close to run the query immediately — useful for writing multi-line DDL or complex statements that don't fit the query box.
- **Postgres integration tests** using `testcontainers-go`. A real Postgres instance is spun up in CI to test the full `Connection` interface: databases, tables, rows, indexes, constraints, and foreign keys.

## [0.5.0] - 2026-03-13

### Added

- **PostgreSQL support.** Connect with a `postgres://` DSN via `stoat --dsn "..."`.
  Works with Supabase, Neon, Railway, Render, and other hosted Postgres providers.
- **GitHub Actions CI** for main branch: build, tests, and golangci-lint on push and pull requests.
- **Async database connection.** The TUI now opens immediately on startup. The
  connection to the database is established in the background, so there is no
  blank wait before the interface appears — especially noticeable with hosted
  Postgres providers like Railway, Supabase, and Neon.
- **Loading status messages.** The status bar now shows progress through every
  async operation: `Connecting to <db>…`, `Loading databases…`,
  `Loading tables…`, `Loading <table>…`, and `Loading page…`. Each message is
  replaced by ` Ready` (or an error) when the operation completes.

### Changed

- **Query clear shortcut** is now `Ctrl+L` (was `Ctrl+K`) to avoid conflicting with line-delete in terminals.
- **Schema tabs** (Columns, Constraints, Indexes) refresh when their data loads so the active tab shows up-to-date content.

### Fixed

- Schema tab content not updating when switching tables or when schema data was loaded while that tab was active.
- Inline cell edit producing **0 rows affected** on the first attempt after opening a table. The fallback `WHERE` clause (used when primary key metadata has not yet loaded) was incorrectly comparing the column against the *new* value instead of the *old* one.
- Status bar stuck on `Loading databases…` indefinitely when the database list returned empty.
