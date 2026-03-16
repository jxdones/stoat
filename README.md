<div align="center">
  <p>
    <img src="assets/stoat.png" alt="Stoat logo" width="150"/>
    <h2>Stoat</h2>
  </p>

  <p>The database client for people who don't leave the terminal.</p>

  <p>
    <img src="assets/stoat.gif" alt="Stoat demo" width="720"/>
  </p>

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.26+-00ADD8.svg)
</div>

## Why Stoat?

You're already in a shell. You need to check a table, inspect a schema, or run a quick query. Opening a GUI for that is friction you don't need.

Raw CLI clients like `psql` or `sqlite3` are great for scripts but rough for browsing data. You have no visual navigation, no schema overview, no easy paging.

Stoat sits between those two extremes: a keyboard-driven TUI that gives you real database inspection without leaving your workflow.

Built for anyone who lives in the terminal and wants database access that doesn't interrupt their flow.

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea) by Charmbracelet.

## Features

- Schema exploration — browse columns, indexes, constraints, and foreign keys in dedicated tabs without writing `PRAGMA` or `\d`
- Inline SQL — run ad-hoc queries from a built-in query box; save snippets you reuse often
- Open in `$EDITOR` — press `Ctrl+E` to write multi-line SQL in your editor; saves and runs on close
- Vim-style navigation — `hjkl`, `gg`/`G`, count prefixes (`10j`), all the muscle memory you already have
- Edit in place — press `Enter` on any cell to edit its value inline; confirm with `Enter`, cancel with `Esc`
- Filter without SQL — narrow down loaded rows without rewriting your query
- Themes — `default`, `dracula`, or `solarized`

## Database support

SQLite and PostgreSQL are supported. MariaDB is planned.

## Works with hosted databases

If you use Supabase, Neon, Railway, or Render, paste your connection string and you're in. No browser, no dashboard, no context switch.

```bash
# Supabase
stoat --dsn "postgres://postgres:[password]@db.[project].supabase.co:[port]/postgres?sslmode=require"

# Neon
stoat --dsn "postgres://[user]:[password]@[host].neon.tech/[dbname]?sslmode=require"

# Railway
stoat --dsn "postgres://[user]:[password]@[host].railway.app:[port]/[dbname]?sslmode=require"

# Render
stoat --dsn "postgres://[user]:[password]@[host].render.com:[port]/[dbname]?sslmode=require"
```

Any provider that gives you a `postgres://` connection string works. Including AWS RDS, GCP Cloud SQL, and Azure Database for PostgreSQL.

## Installation

**One-liner:**

```bash
curl -fsSL https://raw.githubusercontent.com/jxdones/stoat/main/install.sh | sh
```

To install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/jxdones/stoat/main/install.sh | sh -s -- v0.5.3
```

**Homebrew** (macOS):

```bash
brew tap jxdones/stoat
brew install stoat
```

**From the repo root** (developers):

```bash
make install
```

To install to a specific prefix (e.g. user-local without sudo):

```bash
PREFIX=$HOME/.local
make install-prefix
```

Then add `$HOME/.local/bin` to your `PATH` if needed.

## Quick start

```bash
# Open the connection picker (reads from ~/.stoat/config.yaml)
stoat

# SQLite (one-off, bypasses picker)
stoat --db path/to/database.sqlite

# PostgreSQL (one-off, bypasses picker)
stoat --dsn "postgres://user:password@host:5432/dbname?sslmode=disable"

# Print version
stoat --version

# Write per-call timings to ~/.stoat/debug.log
stoat --db path/to/database.sqlite --debug
```

Run `stoat` with no arguments to open the connection picker and choose from your saved connections. Pass `--db` or `--dsn` to connect directly, bypassing the picker.

### Development commands

```bash
make test     # run tests
make fmt      # format code
make lint     # run golangci-lint
```

## Key controls

| Key | Action | Scope |
| --- | --- | --- |
| `Ctrl+C` | Quit (always) | Global |
| `q` | Quit (only when focus is clear) | Global |
| `Esc` | Clear focus (then use `q` to quit) | Global |
| `Tab` / `Shift+Tab` | Cycle focus forward/backward | Global |
| `/` | Focus filter box | Global |
| `Ctrl+R` | Reload current table (first page) | Global (when a table is selected) |
| `h` `j` `k` `l` | Move cursor | Sidebar / Table |
| `g` / `G` | Jump to top / bottom | Sidebar / Table |
| `Enter` | Open selected table | Sidebar |
| `Enter` | Apply filter to currently loaded rows (empty filter resets table) | Filter box |
| `Ctrl+1` – `Ctrl+5` | Switch tabs (Records, Columns, Constraints, Foreign Keys, Indexes) | Table |
| `Ctrl+N` / `Ctrl+B` | Next / previous page | Table |
| `N` + motion (e.g. `4h`, `4l`, `10j`) | Repeat motion N times (vim count prefix) | Table |
| `Enter` | Enter inline edit mode for the selected cell | Table |
| `Enter` | Confirm edit and run UPDATE | Edit mode |
| `Esc` | Cancel edit | Edit mode |
| `y` | Copy value from active cell to clipboard | Table |
| `Ctrl+E` | Open `$EDITOR` with a SQL template; save and close to run | Query box |
| `Ctrl+S` | Run query | Query box |
| `Ctrl+N` | Expand saved query (type `@Name` then Ctrl+N to insert) | Query box |
| `Ctrl+L` | Clear query | Query box |

The options bar at the bottom shows shortcuts for the currently focused pane (sidebar, filter, table, or query). When focus is clear, it shows `q` to quit.

## Configuration

Stoat reads configuration from **`~/.stoat/config.yaml`**. This file is created automatically on first run.

| Option | Description |
|--------|-------------|
| `theme` | UI theme: `default`, `dracula`, or `solarized`. |
| `saved_queries` | Named SQL snippets. In the query box, type `@Name` and press **Ctrl+N** to expand. |

Example:

```yaml
# ~/.stoat/config.yaml
theme: default

saved_queries:
  - name: recent_users
    query: SELECT * FROM users ORDER BY updated_at DESC LIMIT 10
  - name: schema
    query: SELECT name, sql FROM sqlite_master WHERE type = 'table'
```

## License

MIT — see [LICENSE](LICENSE).
