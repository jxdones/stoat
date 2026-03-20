# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- **`EXPLAIN` queries now return results on Postgres.** Running `EXPLAIN` or `EXPLAIN ANALYZE` in the query box was showing "0 row(s) affected" instead of the query plan. The Postgres query executor was routing any non-`SELECT` statement through `ExecContext`, discarding the result set. `EXPLAIN` is now correctly routed through `QueryContext`.

## [0.10.0] - 2026-03-19

### Added

- **Help shows tab-switch shortcuts.** The help panel now documents `Ctrl+1` through `Ctrl+5` for quickly switching between tabs.
- **Column-specific filter syntax.** The filter now accepts two new forms alongside the existing plain text search:
  - `column = value` — case-insensitive exact match (e.g. `first_name = penelope` matches `PENELOPE`)
  - `column = "value"` — case-sensitive substring match (e.g. `title = "ACAD"` matches `ACADEMY DINOSAUR` but `title = "acad"` does not)
  - Re-filtering always works against the full loaded page, not the previous filtered result
  - Filter works on query results; clearing the filter while viewing a query result restores the full result instead of reloading a table

### Changed

- **Help copy is shorter and clearer.** The `?` help binding label now reads `help` instead of `toggle help` for consistency with the current behavior.
- **Debug log format improved.** The `--debug` log now uses structured logfmt output via Go's `log/slog` where each entry includes `time`, `level`, `msg`, and contextual fields like `target` and `elapsed`. Easier to read and grep than the previous tab-separated format.

### Fixed

- Updated debug log file mode literal to Go's modern octal format (`0o644`) to align with current Go style and avoid legacy-octal lint noise.

## [0.9.1] - 2026-03-18

### Fixed

- **SQL textarea wrapping.** Long lines in the query box now soft-wrap correctly, matching the underlying textarea's word-wrap algorithm. Fixes blank renders when a line scrolled past the viewport, double-subtraction of prefix width, and cursor misalignment on wrapped lines.

## [0.9.0] - 2026-03-18

### Added

- **Delete row.** Press `dd` on any row in the Records tab to delete it. When the confirmation prompt appears press `y` to confirm or `n`/`Esc` to cancel. Blocked in read-only mode and when viewing query results. Uses primary key columns for the WHERE clause when available.
- **SQL syntax highlighting.** The query box now highlights SQL as you type — keywords, strings, numbers, comments, and operators are each colored using the active theme's syntax palette. Falls back to the plain textarea renderer when the input is empty (to preserve placeholder display).
- **Per-theme syntax colors.** Each built-in theme defines its own `SyntaxKeyword`, `SyntaxString`, `SyntaxNumber`, `SyntaxComment`, and `SyntaxOperator` colors so highlighting feels at home in every colorscheme.
- **Sidebar selection colors.** Each theme now defines explicit `SidebarSelectedBg` and `SidebarSelectedFg` fields for the active item highlight in both the Databases and Tables sections, replacing the previous reuse of unrelated accent colors.

## [0.8.0] - 2026-03-17

### Added

- **More themes.** Eight new built-in themes: `catppuccin`, `everforest`, `gruvbox`, `one-dark`, `rose-pine`, `princess`, `one-shell`, and `blueish`. Set `theme: <name>` in `~/.stoat/config.yaml`.
- **True color support.** All themes now use 24-bit hex colors on true color terminals, with ANSI256 and 16-color fallbacks for older terminals. No configuration required. Stoat detects the terminal's color profile automatically.

### Changed

- **Saved queries scoped to connections.** Saved queries are now defined per connection in `~/.stoat/config.yaml` instead of at the top level. Move your `saved_queries` entries inside the relevant `connections` entry.

## [0.7.0] - 2026-03-16

### Added

- **Read-only mode.** Pass `--read-only` at startup to open any connection in read-only mode, or set `read_only: true` per connection in `~/.stoat/config.yaml`.
Enforced at the DB level (SQLite opens with `mode=ro`; Postgres sets `SET SESSION CHARACTERISTICS AS TRANSACTION READ ONLY`). 
Write queries typed in the query box or editor are blocked before reaching the DB with a clear status bar warning. Inline cell editing is also blocked. A `[RO]` badge appears on the right of the status bar when active.
- **Connection name in status bar.** The active connection name is now shown on the right side of the status bar, so it's always clear which environment you're currently connected.

## [0.6.0] - 2026-03-16

### Added

- **Connection picker.** Open stoat without arguments to choose a saved connection from a centered modal overlay. Press `c` when focus is clear to switch connections at any time.
- **Saved connections in config.** Define connections in `~/.stoat/config.yaml` with structured fields (`host`, `port`, `user`, `password`, `database` for Postgres; `path` for SQLite). No more passing DSN strings on the command line.
- **Modal overlay system.** Modals render as a compositor layer over the main UI. Background content is dimmed to reinforce depth — groundwork for the upcoming settings modal.

## [0.5.3] - 2026-03-15

### Added

- **Homebrew tap.** Install stoat via `brew tap jxdones/stoat && brew install stoat`.

### Fixed

- Release binaries for darwin were broken due to CGO cross-compilation being attempted from Linux. The release workflow now builds darwin targets natively on macOS and linux targets on Ubuntu. `make release` also now only builds for the current OS to avoid the same issue locally.

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
