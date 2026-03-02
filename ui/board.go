package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/denimcouch/kancli-demo/model"
)

var (
	columnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#44475A"))

	focusedColumnStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#BD93F9"))

	columnHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Padding(0, 1)

	taskStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#44475A"))

	focusedTaskStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#F1FA8C"))

	taskTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F8F8F2"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Padding(0, 1)

	helpBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6272A4")).
			Padding(1, 2)

	confirmStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5555")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF5555")).
			Padding(0, 2)

	taskMetaStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))

	taskTitleFocusedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F1FA8C"))

	viewTitleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8F8F2"))
	viewDescStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#BFBFBF"))
	viewDescEmptyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#44475A"))
	viewMetaSepStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#44475A"))
	viewTimeLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Width(9)
	viewTimeValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2"))

	colPillBaseStyle = lipgloss.NewStyle()
)

const taskCardHeight = 5 // 3 content lines + 2 border lines

func tasksVisible(colHeight int) int {
	usable := colHeight - 3 // subtract col border (2) + header (1)
	if usable < taskCardHeight {
		return 1
	}
	return usable / taskCardHeight
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

func priorityStyle(p model.Priority) lipgloss.Style {
	switch p {
	case model.PriorityMed:
		return priorityMedStyle
	case model.PriorityHigh:
		return priorityHighStyle
	default:
		return priorityLowStyle
	}
}

func renderBoard(m Model) string {
	if len(m.columns) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Render("No columns. Press N to create one.")
	}

	colWidth := columnWidth(m.width, len(m.columns))
	colHeight := m.height - 4 // leave room for status bar

	var cols []string
	for i, col := range m.columns {
		focused := i == m.focusedCol
		cols = append(cols, renderColumn(m, col, focused, colWidth, colHeight, m.scrollOffsets[col.ID]))
	}

	board := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	status := renderStatusBar(m)

	return lipgloss.JoinVertical(lipgloss.Left, board, status)
}

func columnWidth(totalWidth, numCols int) int {
	if numCols == 0 {
		return totalWidth
	}
	// Each column border costs 2 chars (left+right). Subtract that from available width.
	w := (totalWidth / numCols) - 2
	if w < 22 {
		return 22
	}
	return w
}

// innerWidth is the usable content width inside a column (inside border, no padding on column itself).
func innerWidth(colWidth int) int {
	return colWidth - 2 // subtract column border (left + right)
}

func renderColumn(m Model, col model.Column, focused bool, width, height, scrollOffset int) string {
	inner := innerWidth(width)
	tasks := m.tasks[col.ID]

	nameStr := columnHeaderStyle.Foreground(lipgloss.Color(col.Color)).Render(col.Name)
	badgeStr := taskMetaStyle.Render(fmt.Sprintf("[%d]", len(tasks)))
	header := lipgloss.JoinHorizontal(lipgloss.Top, nameStr, "  ", badgeStr)

	visible := tasksVisible(height)
	start := scrollOffset
	if start > len(tasks) {
		start = len(tasks)
	}
	end := start + visible
	if end > len(tasks) {
		end = len(tasks)
	}
	visibleTasks := tasks[start:end]

	var taskViews []string
	for i, task := range visibleTasks {
		isFocused := focused && (start+i) == m.focusedTask
		taskViews = append(taskViews, renderTask(task, isFocused, inner))
	}

	if len(tasks) == 0 {
		taskViews = append(taskViews,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#44475A")).
				Width(inner).
				Render("  No tasks"),
		)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, taskViews...)
	content := lipgloss.JoinVertical(lipgloss.Left, header, body)

	// Pad to fill column height so borders are consistent.
	// Use lipgloss.Height which is ANSI-escape-aware, unlike strings.Count.
	if h := lipgloss.Height(content); h < height {
		content += strings.Repeat("\n", height-h)
	}

	style := columnStyle.Width(width)
	if focused {
		style = focusedColumnStyle.Width(width)
	}

	return style.Render(content)
}

func renderTask(task model.Task, focused bool, colInnerWidth int) string {
	cardInner := colInnerWidth - 2

	pStyle := priorityStyle(task.Priority)
	rawID := fmt.Sprintf("#%d", task.ID)
	idStr := taskMetaStyle.Render(rawID)
	idLen := len(rawID)

	titleWidth := max(cardInner-idLen, 1)
	var titleRendered string
	if focused {
		titleRendered = taskTitleFocusedStyle.Width(titleWidth).Render(task.Title)
	} else {
		titleRendered = taskTitleStyle.Width(titleWidth).Render(task.Title)
	}
	titleRow := lipgloss.JoinHorizontal(lipgloss.Top, titleRendered, idStr)

	icon := task.Priority.Icon()
	iconStr := pStyle.Render(icon)
	ageStr := relativeTime(task.CreatedAt)
	padLen := max(cardInner-lipgloss.Width(icon)-len(ageStr), 1)
	midRow := lipgloss.JoinHorizontal(lipgloss.Top, iconStr, strings.Repeat(" ", padLen), taskMetaStyle.Render(ageStr))

	filled := int(task.Priority)
	bar := pStyle.Render(strings.Repeat("▓", filled) + strings.Repeat("░", 3-filled))

	card := lipgloss.JoinVertical(lipgloss.Left, titleRow, midRow, bar)

	style := taskStyle.Width(colInnerWidth)
	if focused {
		style = focusedTaskStyle.Width(colInnerWidth)
	}
	return style.Render(card)
}

func renderStatusBar(m Model) string {
	if m.showHelp {
		return renderHelpOverlay()
	}

	switch m.mode {
	case ModeConfirmDeleteTask:
		if task, ok := m.focusedTaskObj(); ok {
			return confirmStyle.Render(fmt.Sprintf("Delete task %q? (y/n)", task.Title))
		}
	case ModeConfirmDeleteColumn:
		if m.focusedCol < len(m.columns) {
			col := m.columns[m.focusedCol]
			return confirmStyle.Render(fmt.Sprintf("Delete column %q and all its tasks? (y/n)", col.Name))
		}
	}

	return helpStyle.Render("n: new task  e: edit  d: delete  H/L: move task  K/J: reorder  </>/: reorder col  N: new col  C: edit col  X: delete col  ?: help  q: quit")
}

// renderMarkdown renders markdown content using the provided glamour TermRenderer.
// Falls back to plain lipgloss rendering on any error or nil renderer.
func renderMarkdown(content string, r *glamour.TermRenderer) string {
	if r == nil {
		return viewDescStyle.Render(content)
	}
	out, err := r.Render(content)
	if err != nil {
		return viewDescStyle.Render(content)
	}
	return strings.TrimRight(out, "\n")
}

// renderTaskViewContent builds the inner content string for the task view modal.
// It does not apply box styling; the caller wraps it in a viewport.
// r is the pre-initialised glamour renderer (sized to the usable content width).
//
// Width overhead: formBoxStyle has Padding(1,2) and rounded border = 4 horizontal chars;
// the outer Padding(1,3) adds 6 more = 10 total. Use max(m.width-10, 20).
func renderTaskViewContent(task model.Task, col model.Column, r *glamour.TermRenderer) string {
	title := viewTitleStyle.Render(task.Title)

	sep := viewMetaSepStyle.Render("  ·  ")
	priorityPill := priorityStyle(task.Priority).Render(task.Priority.Icon())
	colPill := colPillBaseStyle.Foreground(lipgloss.Color(col.Color)).Render("◉ " + col.Name)
	idPill := taskMetaStyle.Render(fmt.Sprintf("#%d", task.ID))
	pillRow := lipgloss.JoinHorizontal(lipgloss.Top, priorityPill, sep, colPill, sep, idPill)

	var descRendered string
	if task.Description == "" {
		descRendered = viewDescEmptyStyle.Render("(no description)")
	} else {
		descRendered = renderMarkdown(task.Description, r)
	}

	createdRow := lipgloss.JoinHorizontal(lipgloss.Top,
		viewTimeLabelStyle.Render("Created"),
		viewTimeValueStyle.Render(task.CreatedAt.Format("Jan 2 2006  15:04")),
	)
	updatedRow := lipgloss.JoinHorizontal(lipgloss.Top,
		viewTimeLabelStyle.Render("Updated"),
		viewTimeValueStyle.Render(relativeTime(task.UpdatedAt)),
	)

	help := formHelpStyle.Render("↑/↓: scroll  e: edit  esc: back")

	rows := []string{
		"",
		title,
		"",
		pillRow,
		"",
		descRendered,
		"",
		createdRow,
		updatedRow,
		"",
		help,
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderHelpOverlay() string {
	lines := []string{
		"Keybindings",
		"",
		"← / → / h / l   Navigate columns",
		"↑ / ↓ / k / j   Navigate tasks",
		"enter / v        View focused task (read-only)",
		"n                New task in focused column",
		"e                Edit focused task",
		"d                Delete focused task",
		"H                Move task to previous column",
		"L                Move task to next column",
		"K                Move task up within column",
		"J                Move task down within column",
		"< / >            Reorder column left/right",
		"N                New column",
		"C                Edit column name / color",
		"X                Delete focused column",
		"?                Toggle this help",
		"q                Quit",
	}
	return helpBoxStyle.Render(strings.Join(lines, "\n"))
}
