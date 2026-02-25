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

Stoat is **early in development**. Right now the repo contains:

* **Table component:** a Bubbletea component for data grids (scrollable viewport, vim-style navigation, column sizing)
* **UI building blocks:** theme, key bindings, and shortcut helpers

There is no runnable binary yet. To see how the table component is used, look at
`internal/ui/components/table/table_test.go` (usage examples and behavior are covered there).

## License

MIT — see [LICENSE](LICENSE).
