# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.14.5] - 2026-04-05

### Added

- **`stoat version` subcommand.** `stoat version` now prints the version, complementing the existing `--version` / `-v` flag.
- **Mode indicator in status bar.** The status bar now shows the current input mode (`NORMAL`, `INSERT`, `DELETE`) at all times, styled with per-theme colors. Lays the groundwork for visual mode selection.

### Fixed

- **Stale data when switching connections.** Switching to a new connection while data was still loading could cause tables and rows from the previous connection to appear in the new session. A connection sequence counter now ensures all in-flight results from prior connections are silently discarded. The table view is also cleared immediately when a new connection is established.

## [0.14.4] - 2026-04-02

### Changed

- **Shortcuts bar trimmed to 3–4 hints per pane.** Each pane now shows only its most-reached-for bindings; everything else is accessible via `?`.

### Fixed

- **PostgreSQL now selects the correct default schema.** `DefaultDatabase` was querying `current_database()`, returning the database name instead of the active schema. Switched to `current_schema()` so the sidebar pre-selects the right schema for non-default `search_path` configurations.
- **MySQL indexes now correctly report uniqueness.** `information_schema.statistics.NON_UNIQUE` was being assigned directly to the `Unique` field, inverting the flag for every index. Unique indexes were shown as non-unique and vice versa.
- **Debug log file permissions tightened to `0600`.** Previously created as `0644`, making it readable by other users on the same machine. The log can contain connection strings.
- **Read-only mode now correctly blocks comment-prefixed write queries.** Queries like `-- comment\nINSERT ...` were not detected as writes due to `HasPrefix` matching. Switched to `FirstSQLKeyword` which skips comments. `WITH` is also blocked since CTEs can wrap write queries. `EXPLAIN`, `SHOW`, `DESCRIBE`, `DESC`, `PRAGMA`, and `TABLE` are explicitly allowed.
- **Connection failures no longer expose passwords in the status bar.** The DSN password is redacted before the error is displayed.
- **Config directory and file permissions are repaired at startup.** If `~/.stoat/` or `config.yaml` were created with loose permissions by an older version, stoat now corrects them to `0700` and `0600` respectively.

## [0.14.3] - 2026-03-30

### Changed

- **Query box grows with content.** On terminals taller than 30 rows, the query box expands as you type, up to 10 rows, so longer queries are visible without scrolling. On smaller terminals it stays at the fixed 3-row height.
- **Removed `sql>` prompt from query box.** The prompt prefix was visual noise. The placeholder text already makes the intent clear.

## [0.14.2] - 2026-03-28

### Added

- **Show `NULL` on NULL fields.** Now NULL fields show `NULL` to have more clarity on the content of a cell. If `y` is pressed on a NULL field, it returns an empty string.
- **Sidebar: vim-style count + `j` / `k`.** Support a numeric count prefix before `j`/`k` so e.g. `5j` / `3k` move the databases/schemas and tables selection by that many rows, consistent with the table view’s digit buffer + motion pattern.

### Chore

- **Refactored `internal/ui/model/` package.** Split monolithic update handler files into focused feature files. No behaviour changes.

## [0.14.1] - 2026-03-25

### Added

- **`$` / `0` column navigation.** Press `$` to jump to the last column and `0` to jump to the first column, vim-style.

### Fixed

- **Query result column offset.** Running a query while scrolled horizontally no longer renders results starting from the wrong column, the table resets to column 0.

## [0.14.0] - 2026-03-24

### Added

- **MySQL and MariaDB support.** Connect to MySQL 8+ and MariaDB using `type: mysql` in your config. Includes full schema inspection: columns, indexes, constraints, and foreign keys.

## [0.13.1] - 2026-03-23

### Added

- **Foreign Keys tab is now scrollable.** Previously, tables with more foreign keys than could fit on screen had them silently clipped. The tab now uses a viewport. You can scroll with `j`/`k` or arrow keys.

### Fixed

- **Sidebar "Databases" label now reflects the correct terminology per driver.** Postgres connections show "Schemas" instead of "Databases", since stoat browses schemas within a single connected database. The label updates correctly when switching connections.
- **Cell detail modal now wraps long text content.** Long text values no longer render as a single horizontal line requiring sideways scrolling — content wraps to the modal width. The modal height adjusts to fit the wrapped content. JSON columns are unaffected.

## [0.13.0] - 2026-03-22

### Changed

- **Cell detail viewer (`v`) now shows JSON/JSONB columns formatted and syntax-highlighted by default.** Formatted display is always on for JSON columns.
- **Cell detail modal now sizes to fit the content.** The modal no longer always opens at half the terminal height; it shrinks to match the actual content, expanding only when the content needs the space.

### Fixed

- **Cell detail modal growing when viewing long JSON on small terminals.** A `Width(N)` call in `modal.Render` was setting the *outer* width of the lipgloss box, leaving the content area 4 chars narrower than the viewport. Long ANSI-colored JSON lines overflowed and word-wrapped, adding a phantom row to the modal.
- **Cell detail modal not centered correctly.** The overlay was being placed using `height - 3` instead of the full terminal height, shifting the modal toward the top of the screen.

## [0.12.0] - 2026-03-21

### Added

- **Edit any cell in an external editor.** Press `e` on a focused table cell to open it in `$EDITOR` (falling back to `vim`). The editor is pre-populated with the current cell value. Saving and closing fires an UPDATE; quitting without changes is a no-op. JSON and JSONB values are automatically pretty-printed before opening and minified back on save.

### Fixed

- **SELECT queries starting with SQL comments not returning results on Postgres.** Queries opened via the external editor (or any query beginning with `--` or `/* */` comments) were incorrectly routed to the DML execution path, showing "rows affected" instead of displaying the result set.

## [0.11.0] - 2026-03-20

### Added

- **Paste now works in inline cell edit mode.** Pressing the system paste shortcut while editing a cell inline now inserts the clipboard content at the cursor position.
- **Table columns now fill the available width.** Column widths are now computed in two passes: first, each column's minimum width is derived from the widest cell value in the current page (capped at 50 chars) rather than the header name alone. Second, any horizontal space left after placing the visible columns is distributed proportionally among them, wider columns receive a larger share. Tables with only a few columns no longer leave a large blank area on the right.

### Fixed
- **UPDATE and DELETE queries failing** with `"relation does not exist` on Postgres when the active schema is not in the connection's search_path.

## [0.10.2] - 2026-03-20

### Added

- **Pagination indicator now shows `+` when more pages exist.** The header line (e.g. `page 2 | 100 rows`) now reads `page 2+ | 100 rows` when there is a next page, making it clear without navigating forward.
- **Schema tab detail bar now reflects the focused schema cell.** When on the Columns, Constraints, Indexes, or Foreign Keys tabs, the detail bar at the bottom now shows the cursor position and value of the focused cell in the schema table instead of the main data table. The `type` field is omitted on schema tabs since it is not meaningful there.

### Fixed

- **Sidebar overflow marker (`…`) is now always pinned to the edge of the list.** When the selected item was at the last visible row with more items below (or the first visible row with items above), the `…` was shifting inward, leaving table names visible past it. The marker now always appears at the absolute first or last visible row, and scrolling is adjusted so the selected item is always the row immediately adjacent to it.

## [0.10.1] - 2026-03-19

### Fixed

- **`EXPLAIN` queries now return results on Postgres.** Running `EXPLAIN` or `EXPLAIN ANALYZE` in the query box was showing "0 row(s) affected" instead of the query plan. The Postgres query executor was routing any non-`SELECT` statement through `ExecContext`, discarding the result set. `EXPLAIN` is now correctly routed through `QueryContext`.
- **Duplicate columns no longer appear for tables with multi-constraint columns.** Columns that participate in more than one constraint (e.g. a FK and a UNIQUE index) were showing up multiple times in the Records tab. The schema query was joining `key_column_usage` before filtering to PK constraints, causing one row per constraint per column. The join order is now fixed so only the PK constraint row is matched, giving exactly one entry per column.

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
