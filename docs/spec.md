# Kanban CLI — Specification Document

**Version:** 0.1  
**Date:** 2026-03-01  
**Status:** Draft

---

## Table of Contents

1. [Architecture](#architecture)
2. [Data Model](#data-model)
3. [Database Schema](#database-schema)
4. [Database Operations](#database-operations)
5. [UI Structure](#ui-structure)
6. [Keybindings](#keybindings)
7. [Package Structure](#package-structure)
8. [Dependencies](#dependencies)

---

## Architecture

The app follows the Bubbletea Elm architecture: a single `Model` drives all state, `Update` handles events and messages, and `View` renders to the terminal.

```
SQLite (db/) ←→ Model (ui/app.go) ←→ Bubbletea runtime
                      ↕
              View (ui/board.go, ui/form.go)
```

On startup the app loads all columns and tasks from SQLite into memory. All mutations write through to SQLite before updating in-memory state.

---

## Data Model

```go
// model/task.go
package model

import "time"

type Priority int

const (
    PriorityLow  Priority = 1
    PriorityMed  Priority = 2
    PriorityHigh Priority = 3
)

type Column struct {
    ID       int
    Name     string
    Position int    // left-to-right ordering
    Color    string // hex color string e.g. "#FF0000"
}

type Task struct {
    ID          int
    Title       string
    Description string
    ColumnID    int
    Position    int // ordering within column
    Priority    Priority
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

**Default task title:** If `Title` is empty on creation, the db layer sets it to `"Task {id}"` after insert using the auto-incremented ID.

**Column color:** Stored as a hex string. Lipgloss accepts hex natively. Default is `"#FFFFFF"`.

---

## Database Schema

```sql
CREATE TABLE IF NOT EXISTS columns (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    name     TEXT    NOT NULL,
    position INTEGER NOT NULL,
    color    TEXT    NOT NULL DEFAULT '#FFFFFF'
);

CREATE TABLE IF NOT EXISTS tasks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT    NOT NULL DEFAULT '',
    description TEXT,
    column_id   INTEGER NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
    position    INTEGER NOT NULL,
    priority    INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Notes:**
- `ON DELETE CASCADE` — deleting a column deletes its tasks
- `updated_at` is maintained by the Go layer, not a SQLite trigger
- Default columns (Todo, In Progress, Done) are seeded on first run if the columns table is empty

---

## Database Operations

### DB Type

```go
// db/db.go
type DB struct {
    conn *sql.DB
}

func New(path string) (*DB, error)  // opens connection, runs migrations, seeds defaults
func (db *DB) Close() error
```

### Column Operations

```go
func (db *DB) CreateColumn(name, color string, position int) (Column, error)
func (db *DB) GetColumns() ([]Column, error)                         // ordered by position
func (db *DB) UpdateColumn(col Column) error
func (db *DB) DeleteColumn(id int) error                             // cascades to tasks
func (db *DB) ReorderColumns(ids []int) error                        // resets positions by slice order
```

### Task Operations

```go
func (db *DB) CreateTask(columnID int, priority Priority) (Task, error)  // title set to "Task {id}" after insert
func (db *DB) GetTasksByColumn(columnID int) ([]Task, error)             // ordered by position
func (db *DB) UpdateTask(task Task) error                                // updates title, description, priority, updated_at
func (db *DB) DeleteTask(id int) error
func (db *DB) MoveTask(taskID, targetColumnID, targetPosition int) error // transactional
```

### MoveTask Detail

`MoveTask` runs in a single transaction:

1. Decrement `position` of all tasks in the source column with `position > task.position`
2. Increment `position` of all tasks in the target column with `position >= targetPosition`
3. Update the task's `column_id` and `position`

---

## UI Structure

### Modes

The app operates in one of four modes, stored on the root model:

| Mode | Description |
|---|---|
| `ModeBrowse` | Default. Navigate columns and tasks |
| `ModeAddTask` | Text input form for new task |
| `ModeEditTask` | Text input form to edit selected task |
| `ModeAddColumn` | Text input for new column name and color |

### Components

**`ui/app.go`** — root Bubbletea model

```
Model {
    columns       []Column
    tasks         map[int][]Task  // keyed by column ID
    focusedCol    int             // index into columns slice
    focusedTask   int             // index into tasks[focusedCol]
    mode          Mode
    form          FormModel
    db            *db.DB
    width, height int             // terminal dimensions
}
```

**`ui/board.go`** — renders the board in `ModeBrowse`

- Columns rendered side-by-side using lipgloss
- Focused column and focused task are highlighted
- Column header displays name and color accent
- Task card displays title, priority indicator, and ID

**`ui/form.go`** — renders input form in add/edit modes

- Title field (single line)
- Description field (multi-line)
- Priority selector (Low / Med / High)
- Confirm and cancel keybindings

---

## Keybindings

### Browse Mode

| Key | Action |
|---|---|
| `←` / `→` or `h` / `l` | Move focus between columns |
| `↑` / `↓` or `k` / `j` | Move focus between tasks in column |
| `n` | New task in focused column |
| `e` | Edit focused task |
| `d` | Delete focused task (with confirmation prompt) |
| `H` | Move task to previous column |
| `L` | Move task to next column |
| `K` | Move task up within column |
| `J` | Move task down within column |
| `N` | New column |
| `X` | Delete focused column (with confirmation prompt) |
| `?` | Toggle help overlay |
| `q` | Quit |

### Form Mode

| Key | Action |
|---|---|
| `Tab` | Next field |
| `Shift+Tab` | Previous field |
| `Enter` | Confirm / submit |
| `Esc` | Cancel, return to browse |

---

## Package Structure

```
kanban/
├── main.go           # entry point, wires db and starts bubbletea
├── go.mod
├── go.sum
├── model/
│   └── task.go       # Task, Column, Priority types and constants
├── db/
│   └── db.go         # DB type, schema migration, all CRUD operations
└── ui/
    ├── app.go        # root bubbletea Model, Update, View
    ├── board.go      # board rendering (browse mode)
    └── form.go       # add/edit form rendering and input handling
```

---

## Dependencies

| Package | Purpose |
|---|---|
| `github.com/charmbracelet/bubbletea` | TUI framework (Elm architecture) |
| `github.com/charmbracelet/lipgloss` | Terminal styling and layout |
| `modernc.org/sqlite` | Pure-Go SQLite driver (no CGo required) |

`modernc.org/sqlite` is preferred over `github.com/mattn/go-sqlite3` because it requires no C compiler, simplifying builds and cross-compilation.
