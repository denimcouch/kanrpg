# Plan: Markdown Rendering for Task Descriptions

**Version:** v2 (2026-03-02)

---

## Context

Users write task descriptions as plain text. They want to use markdown formatting (headers, bold, lists, code blocks, etc.) and see it rendered when viewing tasks. Currently, `renderTaskView()` in `ui/board.go` displays the raw description string with only lipgloss color styling.

## Approach

Use **glamour** (Charmbracelet's markdown renderer for terminals) to render descriptions in the view modal. Add a **viewport** (from bubbles) for scrolling when rendered content exceeds the visible area. The edit form continues showing raw markdown text — no changes there.

---

## Changes

### 1. Add glamour dependency

```
go get github.com/charmbracelet/glamour@latest
go mod tidy
```

### 2. `ui/board.go` — Add `renderMarkdown` helper + update `renderTaskView`

Add a `renderMarkdown(content string, width int, r *glamour.TermRenderer) string` helper that:
- Accepts the pre-initialized `TermRenderer` from `Model` (see §4 below)
- Calls `r.Render(content)`, trims trailing newlines from the output
- Falls back to `viewDescStyle.Render(content)` on any error

**Content width calculation:** The view modal uses `formBoxStyle` which has `Padding(1, 2)` and a rounded border — that's 2 (padding left+right) + 2 (border left+right) = 4 chars of overhead. The outer `renderTaskView` adds its own `Padding(1, 3)` = 6 more horizontal chars, totaling 10. Use `clamp(m.width - 10, 20, m.width - 10)` — i.e., no arbitrary upper cap; let the content fill the box. Document this arithmetic in a comment.

**Empty description guard:** The caller (`renderTaskView`) retains responsibility for guarding the empty-description case (current behavior: `viewDescEmptyStyle.Render("(no description)")`). `renderMarkdown` is only called with non-empty content.

**Refactor `renderTaskView`** into two functions:
- `renderTaskViewContent(task model.Task, col model.Column, contentWidth int) string` — builds the inner content string (title, pills, rendered description, timestamps, help text). No box styling, no vertical padding.
- `renderTaskView(task model.Task, col model.Column, w, h int) string` — the outer wrapper: computes viewport dimensions, calls `renderTaskViewContent`, sets viewport content (via `initViewport`), and returns `formBoxStyle.Padding(1, 3).Render(m.viewVP.View())`.

**Remove** the existing vertical-padding logic from `renderTaskView` — that responsibility moves entirely to the viewport (the viewport fills the box height; no manual blank-line padding needed).

Update help text from `"e: edit   esc: back"` to `"↑/↓: scroll  e: edit  esc: back"`.

### 3. `ui/app.go` — TermRenderer caching + viewport

**Add fields to `Model` struct:**
```go
viewVP      viewport.Model
glamourRenderer *glamour.TermRenderer
glamourWidth    int  // terminal width when renderer was last created
```

**Add `initGlamourRenderer(width int)` method** on `*Model`:
- Creates a `glamour.TermRenderer` with `WithStandardStyle("dark")` and `WithWordWrap(width)`
- Stores it in `m.glamourRenderer` and records `m.glamourWidth = width`
- Called lazily in `initViewport` when `m.glamourRenderer == nil || m.glamourWidth != contentWidth`
- This is the single place a renderer is ever constructed

> **Note on glamour style:** `WithStandardStyle("dark")` may produce colors that conflict with the app's Dracula-inspired palette. A custom stylesheet matching the existing palette is deferred to a follow-up. The fallback behavior (plain lipgloss render on error) means visual glitches never break functionality.

**Add `initViewport(task model.Task, col model.Column)` method** on `*Model`:
- Computes `contentWidth = clamp(m.width - 10, 20, m.width - 10)`
- Calls `initGlamourRenderer(contentWidth)` if renderer needs creation/recreation
- Computes viewport height: `(m.height * 4 / 5) - 4` (same formula used by `formBoxStyle` height expansion)
- Calls `renderTaskViewContent(task, col, contentWidth)` to produce the content string
- Initializes `m.viewVP` with the computed width and height, sets content
- This is the **single source of truth** for all viewport setup — called both when entering view mode and on resize

**Restructure `Update()` dispatch:**

The viewport needs to receive `tea.MouseMsg` and `tea.WindowSizeMsg` in addition to key events. Currently, all non-key messages fall through without reaching mode handlers. The new structure:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // 1. ModeViewTask gets first look at ALL message types
    if m.mode == ModeViewTask {
        return m.updateViewTask(msg)
    }

    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.clampAllScrollOffsets()
        return m, nil
    case tea.KeyMsg:
        switch m.mode {
        case ModeBrowse:
            return m.updateBrowse(msg)
        case ModeConfirmDeleteTask:
            return m.updateConfirmDeleteTask(msg)
        case ModeConfirmDeleteColumn:
            return m.updateConfirmDeleteColumn(msg)
        case ModeAddTask, ModeEditTask, ModeAddColumn, ModeEditColumn:
            return m.updateForm(msg)
        }
    }
    return m, nil
}
```

All non-`ModeViewTask` modes continue to work identically. The global `WindowSizeMsg` handler is no longer reached when in view mode — `updateViewTask` handles resize instead by calling `initViewport`.

**Update `updateViewTask`** signature to `(msg tea.Msg) (tea.Model, tea.Cmd)`:
```go
func (m Model) updateViewTask(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        if task, ok := m.focusedTaskObj(); ok {
            col := m.columns[m.focusedCol]
            m.initViewport(task, col)
        }
        return m, nil
    case tea.KeyMsg:
        switch msg.String() {
        case "esc", "q":
            m.mode = ModeBrowse
            return m, nil
        case "e":
            if task, ok := m.focusedTaskObj(); ok {
                m.form = newTaskForm(task, m.columns, m.focusedCol)
                m.mode = ModeEditTask
            }
            return m, nil
        }
    }
    // All other messages (mouse scroll, etc.) pass through to the viewport.
    var cmd tea.Cmd
    m.viewVP, cmd = m.viewVP.Update(msg)
    return m, cmd
}
```

**When entering `ModeViewTask`** (in `updateBrowse`, `case "enter", "v":` — currently lines 154-158), call `m.initViewport(task, col)` before setting `m.mode = ModeViewTask`.

**Update `View()` case for `ModeViewTask`**: Replace the direct `renderTaskView` call with:
```go
case ModeViewTask:
    return centerView(formBoxStyle.Padding(1, 3).Render(m.viewVP.View()), m.width, m.height)
```

### 4. `ui/form.go` — Increase char limit

Change `ta.CharLimit` in `newTaskForm` from `500` to `2000`.

> **Rationale:** The DB column is `TEXT` (unbounded); the char limit is a UX guard. 2000 chars supports substantial task descriptions with markdown overhead while remaining reasonable to display in the viewport. This is not derived from markdown overhead but from a judgment that 2000 characters is a practical upper bound for a task description.

### 5. Tests — `ui/board_test.go` (new file)

- `TestRenderMarkdown_Basic` — markdown input produces styled/non-empty output
- `TestRenderMarkdown_Empty` — empty string guard is respected by caller; helper itself doesn't need to handle it (document this explicitly in test comment)
- `TestRenderMarkdown_WidthRespected` — long single-line text wraps within the given width
- `TestRenderMarkdown_FallbackOnNilRenderer` — passing `nil` renderer falls back gracefully without panic

Tests operate on the `renderMarkdown` helper in isolation. No TUI startup required.

### 6. Run tests and lint

```
go test ./...
go vet ./...
```

---

## Files Modified

- `go.mod` / `go.sum` — new glamour dependency
- `ui/board.go` — add `renderMarkdown`, refactor `renderTaskView` into content + wrapper, remove vertical-padding logic
- `ui/app.go` — add `viewVP`, `glamourRenderer`, `glamourWidth` fields; add `initViewport` and `initGlamourRenderer` methods; restructure `Update()` dispatch; update `updateViewTask` signature; update `View()` for `ModeViewTask`
- `ui/form.go` — increase `ta.CharLimit` from 500 to 2000
- `ui/board_test.go` — new test file

---

## Verification

1. `go build ./...` compiles
2. `go test ./...` passes
3. `go vet ./...` clean
4. Manual: view a task with markdown (headers, bold, lists, code blocks) — renders formatted
5. Manual: view a task with plain text — still looks good
6. Manual: empty description shows `(no description)` unchanged
7. Manual: scroll long content with `↑`/`↓` arrow keys
8. Manual: resize terminal while in view mode — viewport reflows correctly
9. Manual: `esc`/`q` closes modal; `e` opens edit form showing raw markdown
10. Manual: open, close, and reopen the view modal — no renderer reconstruction on second open (same width)
