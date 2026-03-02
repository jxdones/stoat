# Stoat

**A super light terminal-native database client.**

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea), Stoat is designed
for developers who want to inspect schemas, run queries, and navigate data without
their hands ever leaving the keyboard.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.25.5+-00ADD8.svg)

## Why Stoat?

Most database clients fall into two camps: bloated Electron apps or raw CLI clients (`psql`, `mysql`) that are painful for browsing complex tables.

Stoat hits the sweet spot:

* **Instant Startup:** A single static binary. No JVM, no Electron, no lag.
* **Vim-Native Navigation:** Traverse tables, rows, and cells naturally with `h`, `j`, `k`, `l`.
* **Clean UI:** A TUI that organizes your data into a readable, navigable grid.

## Current state

Stoat is **early in development**. The TUI shell is in place with:

* **Layout:** Responsive two-pane layout (sidebar + main). The main area has a header, tab bar, data table, detail row, and query box. Layout adapts to narrow or short terminals.
* **Sidebar:** Databases and Tables sections with vi-style navigation (`j`/`k`, `h`/`l`, `g`/`G`). Enter selects a database or table and moves focus to the table view.
* **Tabs:** Metadata views for the selected table.
* **Table:** Data grid with the same vi-style keys for moving around rows and cells.
* **Filter box:** Text input to filter the current table view.
* **Query box:** SQL input area for running queries.
* **Status bar:** Message line (e.g. "Ready") with severity (info, success, warning, error).
* **Focus:** Tab / Shift+Tab cycles focus: Sidebar → Filterbox → Table → Querybox → Sidebar.

No database driver or connection logic is implemented yet.

## Building

```bash
make build    # builds bin/stoat
make test     # run tests
make fmt      # format code
make lint     # run golangci-lint
```

Run the app:

```bash
./bin/stoat
```

## License

MIT — see [LICENSE](LICENSE).
