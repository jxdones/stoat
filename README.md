<div align="center">
  <p>
    <img src="assets/stoat.png" alt="Stoat logo" width="150"/>
    <h2>Stoat</h2>
  </p>

  <p>The database client for people who don't leave the terminal.</p>

  <p>
    <img src="assets/stoat.gif" alt="Stoat demo" width="830"/>
  </p>

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.26+-00ADD8.svg)

[Quick Start](#quick-start) • [Features](#features) • [Installation](#installation) • [Database Support](#database-support) • [Key Controls](#key-controls) • [Themes](#themes) • [Configuration](#configuration) • [Contributing](#contributing)
</div>

## Why Stoat?

You're already in a shell. You need to check a table, inspect a schema, or run a quick query. Opening a GUI for that is friction you don't need.

Raw CLI clients like `psql` or `sqlite3` are great for scripts but rough for browsing data. You have no visual navigation, no schema overview, no easy paging.

Stoat sits between those two extremes: a keyboard-driven TUI that gives you real database inspection without leaving your workflow.

Built for anyone who lives in the terminal and wants database access that doesn't interrupt their flow.

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea) by Charmbracelet.

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

# Open in read-only mode (works with --db, --dsn, or the connection picker)
stoat --read-only
```

Run `stoat` with no arguments to open the connection picker and choose from your saved connections. Pass `--db` or `--dsn` to connect directly, bypassing the picker.

## Features

- Schema exploration — browse columns, indexes, constraints, and foreign keys in dedicated tabs without writing `PRAGMA` or `\d`
- Inline SQL — run ad-hoc queries from a built-in query box; save snippets you reuse often
- Open in `$EDITOR` — press `Ctrl+E` to write multi-line SQL in your editor; saves and runs on close
- Vim-style navigation — `hjkl`, `gg`/`G`, count prefixes (`10j`), all the muscle memory you already have
- Edit in place — press `Enter` on any cell to edit its value inline; confirm with `Enter`, cancel with `Esc`
- Filter without SQL — narrow down loaded rows without rewriting your query
- Read-only mode — connect safely to production with `--read-only` or `read_only: true` per connection in config; enforced at the DB level and in the UI
- SQL syntax highlighting — keywords, strings, numbers, comments, and operators are colored as you type; palette adapts to the active theme
- Themes — 11 built-in themes including dracula, catppuccin, gruvbox, rose-pine, and more

## Installation

**One-liner:**

```bash
curl -fsSL https://raw.githubusercontent.com/jxdones/stoat/main/install.sh | sh
```

To install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/jxdones/stoat/main/install.sh | sh -s -- v0.14.5
```

**Homebrew** (macOS):

```bash
brew tap jxdones/stoat
brew install --cask stoat
```

**Go install:**

```bash
go install github.com/jxdones/stoat@latest
```

**Docker:**

```bash
docker build -t stoat .
docker run --rm -it -v "$(pwd)/data:/data" stoat --db /data/your.db
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

## Database support

SQLite, PostgreSQL, MySQL, and MariaDB are supported.

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

## Key controls

The options bar at the bottom shows shortcuts for the currently focused pane. When focus is clear, it shows `q` to quit.

### Global

| Key | Action |
| --- | --- |
| `Ctrl+C` | Quit (always) |
| `q` | Quit (only when focus is clear) |
| `Esc` | Clear focus |
| `Tab` / `Shift+Tab` | Cycle focus forward / backward |
| `/` | Focus filter box |
| `c` | Show connection picker (only when focus is clear) |
| `Ctrl+R` | Reload current table (first page) |
| `?` | Show help |

### Sidebar

| Key | Action |
| --- | --- |
| `h` `j` `k` `l` | Move cursor |
| `g` / `G` | Jump to top / bottom |
| `Enter` | Open selected table |

### Table

| Key | Action |
| --- | --- |
| `h` `j` `k` `l` | Move cursor |
| `g` / `G` | Jump to top / bottom |
| `Ctrl+1` – `Ctrl+5` | Switch tabs (Records, Columns, Constraints, Foreign Keys, Indexes) |
| `Ctrl+N` / `Ctrl+B` | Next / previous page |
| `0` / `$` | Go to first / last column |
| `N` + motion (e.g. `4h`, `10j`) | Repeat motion N times (vim count prefix) |
| `Enter` | Enter inline edit mode for the selected cell |
| `y` | Copy value from active cell to clipboard |
| `dd` | Delete selected row |

### Table — Edit mode

| Key | Action |
| --- | --- |
| `Enter` | Confirm edit and run UPDATE |
| `Esc` | Cancel edit |

### Table — Delete mode

| Key | Action |
| --- | --- |
| `y` | Confirm delete |
| `n` / `Esc` | Cancel delete |

### Filter box

| Key | Action |
| --- | --- |
| `Enter` | Apply filter to currently loaded rows (empty filter resets table) |

### Query box

| Key | Action |
| --- | --- |
| `Ctrl+S` | Run query |
| `Ctrl+E` | Open `$EDITOR` with a SQL template; save and close to run |
| `Ctrl+N` | Expand saved query (type `@Name` then Ctrl+N to insert) |
| `Ctrl+L` | Clear query |

## Themes

| | |
|---|---|
| ![Dracula](assets/stoat_dracula.png) | ![Everforest](assets/stoat_everforest.png) |
| ![Gruvbox](assets/stoat_gruvbox.png) | ![Princess](assets/stoat_princess.png) |

Available themes: `default`, `dracula`, `solarized`, `catppuccin`, `everforest`, `gruvbox`, `one-dark`, `rose-pine`, `princess`, `one-shell`, `blueish`. Set via `theme` in `~/.stoat/config.yaml`.

## Configuration

Stoat reads configuration from **`~/.stoat/config.yaml`**. This file is created automatically on first run.

| Option | Description |
|--------|-------------|
| `theme` | UI theme. Available: `default`, `dracula`, `solarized`, `catppuccin`, `everforest`, `gruvbox`, `one-dark`, `rose-pine`, `princess`, `one-shell`, `blueish`. |
| `connections` | Saved database connections. Each connection supports `read_only: true` to enforce read-only mode at the DB and UI level, and `saved_queries` to define named SQL snippets scoped to that connection. |

Example:

```yaml
# ~/.stoat/config.yaml
theme: default

connections:
  - name: local
    type: sqlite
    path: /path/to/database.sqlite
    saved_queries:
      - name: recent_users
        query: SELECT * FROM users ORDER BY updated_at DESC LIMIT 10
      - name: schema
        query: SELECT name, sql FROM sqlite_master WHERE type = 'table'

  - name: my-postgres
    type: postgres
    host: localhost
    port: 5432        # optional, defaults to 5432
    user: postgres
    password: secret
    database: mydb

  - name: supabase-prod
    type: postgres
    host: db.[project].supabase.co
    port: 5432
    user: postgres
    password: secret
    database: postgres
    read_only: true   # blocks writes in the UI and at the DB level

  - name: local-mysql
    type: mysql
    host: 127.0.0.1
    port: 3306        # optional, defaults to 3306
    user: root
    password: secret
    database: mydb
    # tls_mode: true  # set to true for hosted/remote MySQL, skip-verify to skip cert validation

```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT — see [LICENSE](LICENSE).
