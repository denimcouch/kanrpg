package ui

import (
	"fmt"
	"strings"

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

	taskIDStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4"))

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
)

func priorityStyle(p model.Priority) lipgloss.Style {
	switch p {
	case model.PriorityMed:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C"))
	case model.PriorityHigh:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
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
		cols = append(cols, renderColumn(m, col, focused, colWidth, colHeight))
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

func renderColumn(m Model, col model.Column, focused bool, width, height int) string {
	inner := innerWidth(width)
	tasks := m.tasks[col.ID]

	header := columnHeaderStyle.
		Foreground(lipgloss.Color(col.Color)).
		Width(inner).
		Render(fmt.Sprintf("%s (%d)", col.Name, len(tasks)))

	var taskViews []string
	for i, task := range tasks {
		isFocused := focused && i == m.focusedTask
		taskViews = append(taskViews, renderTask(task, isFocused, inner))
	}

	if len(taskViews) == 0 {
		taskViews = append(taskViews,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#44475A")).
				Width(inner).
				Render("  No tasks"),
		)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, taskViews...)
	content := lipgloss.JoinVertical(lipgloss.Left, header, body)

	// Pad to fill column height so borders are consistent
	contentHeight := strings.Count(content, "\n") + 1
	if contentHeight < height {
		content += strings.Repeat("\n", height-contentHeight)
	}

	style := columnStyle.Width(width)
	if focused {
		style = focusedColumnStyle.Width(width)
	}

	return style.Render(content)
}

func renderTask(task model.Task, focused bool, colInnerWidth int) string {
	// Task card sits inside the column's inner area.
	// Task border costs 2 chars (left+right), so card inner width = colInnerWidth - 2.
	cardInner := colInnerWidth - 2

	pStyle := priorityStyle(task.Priority)
	priorityIndicator := pStyle.Render(task.Priority.Label())
	idStr := taskIDStyle.Render(fmt.Sprintf("#%d", task.ID))

	// Reserve space for id prefix and a space separator.
	idLen := len(fmt.Sprintf("#%d ", task.ID))
	titleWidth := cardInner - idLen
	if titleWidth < 1 {
		titleWidth = 1
	}
	title := taskTitleStyle.Width(titleWidth).Render(task.Title)

	top := lipgloss.JoinHorizontal(lipgloss.Top, idStr, " ", title)
	card := lipgloss.JoinVertical(lipgloss.Left, top, priorityIndicator)

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
		task := m.focusedTaskObj()
		if task != nil {
			return confirmStyle.Render(fmt.Sprintf("Delete task %q? (y/n)", task.Title))
		}
	case ModeConfirmDeleteColumn:
		if m.focusedCol < len(m.columns) {
			col := m.columns[m.focusedCol]
			return confirmStyle.Render(fmt.Sprintf("Delete column %q and all its tasks? (y/n)", col.Name))
		}
	}

	return helpStyle.Render("n: new task  e: edit  d: delete  H/L: move task  K/J: reorder  N: new col  X: delete col  ?: help  q: quit")
}

func renderTaskView(task model.Task, col model.Column, w, h int) string {
	const timeFormat = "2006-01-02 15:04"

	heading := taskIDStyle.Render(fmt.Sprintf("Task #%d", task.ID))

	titleLabel := formLabelStyle.Render("Title")
	titleVal := task.Title

	descLabel := formLabelStyle.Render("Description")
	descVal := task.Description
	if descVal == "" {
		descVal = formHelpStyle.Render("(no description)")
	}

	labelWidth := lipgloss.NewStyle().Width(10)
	pStyle := priorityStyle(task.Priority)
	metaPriority := lipgloss.JoinHorizontal(lipgloss.Top, labelWidth.Render(formLabelStyle.Render("Priority")), pStyle.Render(task.Priority.Label()))
	metaColumn := lipgloss.JoinHorizontal(lipgloss.Top, labelWidth.Render(formLabelStyle.Render("Column")), col.Name)
	metaCreated := lipgloss.JoinHorizontal(lipgloss.Top, labelWidth.Render(formLabelStyle.Render("Created")), task.CreatedAt.Format(timeFormat))
	metaUpdated := lipgloss.JoinHorizontal(lipgloss.Top, labelWidth.Render(formLabelStyle.Render("Updated")), task.UpdatedAt.Format(timeFormat))

	help := formHelpStyle.Render("e: edit  esc: back")

	rows := []string{
		heading,
		"",
		titleLabel,
		titleVal,
		"",
		descLabel,
		descVal,
		"",
		metaPriority,
		metaColumn,
		metaCreated,
		metaUpdated,
		"",
		help,
	}

	body := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return formBoxStyle.Render(body)
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
		"N                New column",
		"X                Delete focused column",
		"?                Toggle this help",
		"q                Quit",
	}
	return helpBoxStyle.Render(strings.Join(lines, "\n"))
}
