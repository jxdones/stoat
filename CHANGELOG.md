# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.2] - 2026-03-14

### Added

- **Spinner in status bar.** Async operations (connecting, loading tables/rows, running queries) now show an animated spinner in the status bar, making it clear when work is in progress rather than displaying static text.
- **Read-only feedback for query results.** Pressing `Enter` on a cell while viewing query results now flashes a `Query results are read-only` warning in the status bar instead of silently doing nothing.

### Fixed

- Suppressed unchecked error return from `fmt.Fprintf` in timing debug log to satisfy `errcheck` linter.

## [0.5.1] - 2026-03-14

### Added

- **Open in `$EDITOR`** (`Ctrl+E`). Press `Ctrl+E` from anywhere to open your `$EDITOR` with a SQL comment template. Save and close to run the query immediately — useful for writing multi-line DDL or complex statements that don't fit the query box.
- **Postgres integration tests** using `testcontainers-go`. A real Postgres instance is spun up in CI to test the full `Connection` interface: databases, tables, rows, indexes, constraints, and foreign keys.
- **Debug timing log** (`--debug`). Pass `--debug` to write per-call timings for every database operation to `~/.stoat/debug.log`. Useful for diagnosing performance on hosted Postgres providers.
- **Version flag** (`--version`). Print the current version and exit.

### Changed

- **Postgres startup is now faster.** After connecting, the schema list and table list load in parallel instead of sequentially, saving one full round-trip on every startup.
- **Sidebar shows the default schema instantly on connect.** `public` appears in the sidebar as soon as the connection is established — before any network calls complete — so the user always has context while data is loading.
- **Postgres `Indexes` query rewritten** to a single JOIN against `pg_catalog`. Eliminated an N+1 pattern (one query per index) that was causing 2–5s load times on hosted providers with many indexes.
- **Postgres `ForeignKeys` query rewritten** from `information_schema` to `pg_catalog.pg_constraint`. Dropped from 1–5s to ~200ms on hosted providers.
- **Postgres `Constraints` query rewritten** from two sequential `information_schema` queries to a single `pg_catalog` UNION ALL query. One round-trip instead of two.

### Fixed

- Sidebar jumping to the first schema alphabetically (e.g. `auth`) after the full schema list loaded, discarding the active `public` selection.
- Table cursor not resetting to the top when switching to a different table or loading a new page.
- Stale table data remaining visible when switching to a database that has no tables. The table view now clears and shows the "Select a table" placeholder.

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
