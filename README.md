# KanRPG - Kanban & Sorcery!

A terminal-based Kanban board with fun RPG-like progression for developers who live in the CLI. Tasks are persisted in a local SQLite database. No accounts, no internet, no setup. Create a character who gains experience as you "defeat" tasks that you create.

## Requirements

- Go 1.25+

## Run

```sh
go run .
```

## Build

```sh
go build -o kancli .
./kancli
```

The database is stored at `$XDG_CONFIG_HOME/kancli/kancli.db` (falls back to `$HOME/kancli/kancli.db`).

## Test

```sh
go test ./...
```

## Project Structure

```
├── main.go        # entry point
├── model/         # Task, Column, Priority types
├── db/            # SQLite migrations and CRUD
└── ui/            # Bubbletea TUI (app, board, form)
```

## Keybindings

| Key | Action |
|-----|--------|
| `←` `→` / `h` `l` | Navigate columns |
| `↑` `↓` / `k` `j` | Navigate tasks |
| `enter` / `v` | View focused task |
| `n` | New task in focused column |
| `e` | Edit focused task |
| `d` | Delete focused task |
| `H` `L` | Move task to previous/next column |
| `K` `J` | Reorder task up/down within column |
| `<` `>` | Reorder column left/right |
| `N` | New column |
| `C` | Edit column name/color |
| `X` | Delete focused column |
| `?` | Toggle help |
| `q` | Quit |
