# Kancli - Kanban on the Command Line

A terminal-based Kanban board for developers who live in the CLI. Tasks are persisted in a local SQLite database. No accounts, no internet, no setup.

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
| `←` `→` / `h` `l` | Move between columns |
| `↑` `↓` / `k` `j` | Move between tasks |
| `H` `L` | Move task to previous/next column |
| `K` `J` | Move task up/down within column |
| `n` | New task |
| `e` | Edit task |
| `d` | Delete task |
| `N` | New column |
| `X` | Delete column |
| `?` | Toggle help |
| `q` | Quit |
