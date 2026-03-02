# Plan: Markdown Rendering for Task Descriptions

## Context

Users write task descriptions as plain text. They want to use markdown formatting (headers, bold, lists, code blocks, etc.) and see it rendered when viewing tasks. Currently, `renderTaskView()` displays the raw description string with only lipgloss color styling.

## Approach

Use **glamour** (Charmbracelet's markdown renderer for terminals) to render descriptions in the view modal. Add a **viewport** (from bubbles) for scrolling when rendered content exceeds the visible area. The edit form continues showing raw markdown text — no changes there.

## Changes

### 1. Add glamour dependency
- `go get github.com/charmbracelet/glamour@latest`

### 2. `ui/board.go` — Add `renderMarkdown` helper + update `renderTaskView`

Add a `renderMarkdown(content string, width int) string` helper that:
- Creates a glamour `TermRenderer` with `WithStandardStyle("dark")` and `WithWordWrap(width)`
- Falls back to plain `viewDescStyle.Render(content)` on error
- Trims trailing newlines from glamour output

In `renderTaskView`, replace `viewDescStyle.Render(task.Description)` with a call to `renderMarkdown()`. Calculate the available content width as `clamp(terminalWidth - 12, 20, 76)`.

**Refactor `renderTaskView`** into two functions:
- `renderTaskViewContent(task, col, w, h) string` — builds the inner content (title, pills, markdown description, timestamps, help text). No box styling, no padding calculation.
- Keep `renderTaskView` or rename for the outer wrapper that uses the viewport.

Update help text from `"e: edit   esc: back"` to `"↑/↓: scroll  e: edit  esc: back"`.

### 3. `ui/app.go` — Add viewport for scrollable view modal

Add `viewVP viewport.Model` field to `Model` struct.

**When entering ModeViewTask** (line 154-158), initialize the viewport:
- Compute viewport dimensions from terminal size
- Pre-render the full task view content via `renderTaskViewContent()`
- Set viewport content

**Update `updateViewTask`** signature to accept `tea.Msg` (not `tea.KeyMsg`) so the viewport can handle mouse scroll too. Intercept `esc`/`q`/`e` before passing remaining messages to the viewport.

**Update `Update()` dispatch**: Route `ModeViewTask` at the `tea.Msg` level (before the `tea.KeyMsg` type switch) so all message types reach the viewport.

**Update `View()` case for ModeViewTask**: Render `formBoxStyle.Padding(1, 3).Render(m.viewVP.View())` wrapped in `centerView`.

**Handle `WindowSizeMsg` in ModeViewTask**: Re-render viewport content at new width so terminal resize works.

### 4. `ui/form.go` — Increase char limit
- Change `ta.CharLimit` from `500` to `1000` (line 65) — markdown syntax has overhead

### 5. Tests — `ui/board_test.go` (new file)
- `TestRenderMarkdown` — basic markdown produces styled output
- `TestRenderMarkdown_Empty` — empty input produces empty/minimal output
- `TestRenderMarkdown_WidthRespected` — long text wraps within width
- `TestRenderMarkdown_Fallback` — zero/negative width doesn't panic

### 6. Run tests and lint
- `go test ./...`
- `go vet ./...`

## Files Modified
- `go.mod` / `go.sum` — new dependency
- `ui/board.go` — add `renderMarkdown`, refactor `renderTaskView`
- `ui/app.go` — add viewport field, update dispatch + view mode logic
- `ui/form.go` — increase char limit
- `ui/board_test.go` — new test file

## Verification
1. `go build ./...` compiles
2. `go test ./...` passes
3. `go vet ./...` clean
4. Manual: view a task with markdown (headers, bold, lists, code) — renders formatted
5. Manual: view a task with plain text — still looks good
6. Manual: empty description shows "(no description)"
7. Manual: scroll long content with arrow keys
8. Manual: esc/q closes modal, e opens edit, edit shows raw markdown
