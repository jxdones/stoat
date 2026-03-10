<h1>
<p align="center">
  <img src="assets/stoat.png" alt="Stoat logo" width="150"/>
  <br>Stoat
</p>
</h1>

A super light, terminal-native database client.

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea), Stoat is for developers who want to inspect schemas, browse table data, and run SQL without leaving the keyboard.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.25.5+-00ADD8.svg)

## Why Stoat?

Most database tools fall into two camps:
- heavy GUI apps with slow startup and lots of chrome
- raw CLIs (`psql`, `mysql`) that are great for scripts but rough for data browsing

Stoat aims for a different path:
- fast startup
- keyboard-first navigation
- a clean TUI optimized for inspection work

## Current status

Stoat is early in development and currently focused on local SQLite workflows.

## Features 
- Browse databases
- Run ad-hoc SQL from the query box
- Navigate the UI with vim-style keys

## Quick start

```bash
make build    # builds bin/stoat
./bin/stoat --db ./mydb.sqlite
```

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
| `Ctrl+N` / `Ctrl+B` | Next / previous page | Table |
| `N` + motion (e.g. `4h`, `4l`, `10j`) | Repeat motion `N` times (vim count prefix) | Table |
| `Enter` | Open editor with UPDATE query for selected cell (save & quit to run) | Table |
| `Ctrl+S` | Run query | Query box |

The options bar at the bottom shows shortcuts for the currently focused pane (sidebar, filter, table, or query). When focus is clear, it shows `q` quit.

## License

MIT — see [LICENSE](LICENSE).
