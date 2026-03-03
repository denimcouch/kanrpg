# Developer Onboarding

This guide gets a new contributor up and running on kancli.

## Prerequisites

- Go 1.25+ (`go version`)
- A terminal that supports 256 colors (most modern terminals do)
- No C compiler required — the SQLite driver is pure Go

## Setup

```sh
git clone https://github.com/denimcouch/kanrpg
cd kancli-demo
go mod download
go run .
```

The app creates its SQLite database at `$XDG_CONFIG_HOME/kancli/kancli.db` on first run (falls back to `$HOME/kancli/kancli.db`). Three default columns — Todo, In Progress, Done — are seeded automatically if the database is empty.

## Project Layout

```
├── main.go        # Entry point: resolves DB path, wires db.DB into ui.Model
├── model/
│   └── task.go    # Task, Column, Priority types — no dependencies
├── db/
│   └── db.go      # SQLite: schema migration, seed, all CRUD operations
└── ui/
    ├── app.go     # Root Bubbletea model, Update dispatch, mode state machine
    ├── board.go   # Board and task card rendering (browse mode)
    └── form.go    # Task/column input form rendering and field logic
```

**Dependency direction:** `model` ← `db` ← `ui` ← `main`. The `model` package has no internal imports; `db` imports `model`; `ui` imports both.

## Architecture in Brief

The app follows the [Bubbletea Elm architecture](https://github.com/charmbracelet/bubbletea):

1. **Model** (`ui.Model` in `app.go`) holds all runtime state: columns, tasks map, focused positions, current mode.
2. **Update** dispatches key messages to mode-specific handlers (`updateBrowse`, `updateForm`, etc.).
3. **View** calls `renderBoard` or `renderTaskView`/`form.View` depending on mode.

On startup, all columns and tasks are loaded from SQLite into memory. Every mutation writes through to SQLite first, then updates the in-memory state. There is no background sync.

See [`spec.md`](spec.md) for detailed schema, CRUD API, and mode/keybinding documentation.

## Running Tests

```sh
go test ./...
```

Tests live in `db/db_test.go` and `model/task_test.go`. The DB tests use a temporary in-memory SQLite instance — no fixtures or test database needed.

## Making Changes

### Adding a keybinding

1. Add a `case` to `updateBrowse` (or the relevant mode handler) in `ui/app.go`.
2. Add the binding to `renderHelpOverlay` in `ui/board.go`.
3. Update the status bar hint string in `renderStatusBar` if it belongs in the always-visible hint line.
4. Update the keybindings table in `README.md`.

### Adding a new field to Task

1. Add the field to `model.Task` in `model/task.go`.
2. Add the column to the `CREATE TABLE` statement in `db.migrate()`.
3. Update `db.CreateTask`, `db.GetTasksByColumn`, and `db.UpdateTask` to include the new field.
4. Update `ui/form.go` if the field needs user input.

### Adding a new mode

1. Add a `Mode` constant in `ui/app.go`.
2. Add a `case` to the `Update` switch to route to a new handler.
3. Add a `case` to `View` to render the new mode.

## Common Gotchas

- **`updated_at` is managed in Go**, not via a SQLite trigger. `db.UpdateTask` sets it to `time.Now()` before the query.
- **Task default title** is set after insert using the auto-incremented ID (`"Task {id}"`). The initial insert uses an empty string.
- **Column deletion cascades** to tasks via `ON DELETE CASCADE`. No manual cleanup needed.
- **`MoveTask` is transactional** — it closes the gap in the source column, opens a gap in the target, then moves the task atomically. Reordering within the same column goes through the same path.
- **`modernc.org/sqlite`** is a CGo-free driver. Do not swap it for `github.com/mattn/go-sqlite3` without updating the build process.
