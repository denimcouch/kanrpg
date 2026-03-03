package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/denimcouch/kanrpg/model"
)

type formField int

// Field indices — used as values for FormModel.focused.
const (
	fieldTitle    formField = iota // 0: shared across all form types
	fieldDesc                      // 1: task form
	fieldPriority                  // 2: task form
	fieldColumn                    // 3: task form
	fieldColor    formField = 1    // 1: column-edit form (replaces fieldDesc slot)
)

// Field counts — used by numFields() to bound tab-cycling.
const (
	fieldCountTask       formField = 4 // fieldTitle..fieldColumn
	fieldCountAddColumn  formField = 1 // fieldTitle only
	fieldCountEditColumn formField = 2 // fieldTitle + fieldColor
)

// palette is the set of colors users can cycle through when editing a column.
var palette = []string{
	"#FF5555", // red
	"#FFB86C", // orange
	"#F1FA8C", // yellow
	"#50FA7B", // green
	"#8BE9FD", // cyan
	"#BD93F9", // purple
	"#FF79C6", // pink
	"#F8F8F2", // white
	"#6272A4", // muted blue
}

type FormModel struct {
	title       textinput.Model
	desc        textarea.Model
	priority    model.Priority
	columns     []model.Column // non-nil only in task form
	columnIdx   int            // index into columns
	focused     formField
	isColForm   bool // true when used for adding a column (title only)
	isEditCol   bool // true when editing an existing column (title + color)
	colorIdx    int  // index into palette
}

func newTaskForm(task model.Task, columns []model.Column, currentColIdx int) FormModel {
	ti := textinput.New()
	ti.Placeholder = "Task title"
	ti.SetValue(task.Title)
	ti.Focus()
	ti.CharLimit = 120
	ti.Width = 50

	ta := textarea.New()
	ta.Placeholder = "Description (optional)"
	ta.SetValue(task.Description)
	ta.CharLimit = 2000
	ta.SetWidth(52)
	ta.SetHeight(4)
	ta.ShowLineNumbers = false

	return FormModel{
		title:     ti,
		desc:      ta,
		priority:  task.Priority,
		columns:   columns,
		columnIdx: currentColIdx,
		focused:   fieldTitle,
		isColForm: false,
	}
}

func newEditColumnForm(col model.Column) FormModel {
	ti := textinput.New()
	ti.Placeholder = "Column name"
	ti.SetValue(col.Name)
	ti.Focus()
	ti.CharLimit = 60
	ti.Width = 50

	// Find the closest palette entry for the current color, defaulting to 0.
	colorIdx := 0
	for i, c := range palette {
		if c == col.Color {
			colorIdx = i
			break
		}
	}

	return FormModel{
		title:     ti,
		focused:   fieldTitle,
		isEditCol: true,
		colorIdx:  colorIdx,
	}
}

func newColumnForm() FormModel {
	ti := textinput.New()
	ti.Placeholder = "Column name"
	ti.Focus()
	ti.CharLimit = 60
	ti.Width = 50

	return FormModel{
		title:     ti,
		focused:   fieldTitle,
		isColForm: true,
	}
}

func (f FormModel) numFields() formField {
	switch {
	case f.isEditCol:
		return fieldCountEditColumn
	case f.isColForm:
		return fieldCountAddColumn
	default:
		return fieldCountTask
	}
}

func (f FormModel) Update(msg tea.Msg) (FormModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			f.focused = (f.focused + 1) % f.numFields()
			f.syncFocus()
			return f, nil
		case "shift+tab":
			n := int(f.numFields())
			f.focused = formField((int(f.focused) - 1 + n) % n)
			f.syncFocus()
			return f, nil
		case "left":
			switch f.focused {
			case fieldPriority:
				if f.priority > model.PriorityLow {
					f.priority--
				}
				return f, nil
			case fieldColumn:
				if f.columnIdx > 0 {
					f.columnIdx--
				}
				return f, nil
			case fieldColor:
				if f.colorIdx > 0 {
					f.colorIdx--
				}
				return f, nil
			}
		case "right":
			switch f.focused {
			case fieldPriority:
				if f.priority < model.PriorityHigh {
					f.priority++
				}
				return f, nil
			case fieldColumn:
				if f.columnIdx < len(f.columns)-1 {
					f.columnIdx++
				}
				return f, nil
			case fieldColor:
				if f.colorIdx < len(palette)-1 {
					f.colorIdx++
				}
				return f, nil
			}
		}
	}

	switch f.focused {
	case fieldTitle:
		var cmd tea.Cmd
		f.title, cmd = f.title.Update(msg)
		cmds = append(cmds, cmd)
	case fieldDesc:
		var cmd tea.Cmd
		f.desc, cmd = f.desc.Update(msg)
		cmds = append(cmds, cmd)
	}

	return f, tea.Batch(cmds...)
}

func (f *FormModel) syncFocus() {
	switch f.focused {
	case fieldTitle:
		f.title.Focus()
		f.desc.Blur()
	case fieldDesc:
		f.title.Blur()
		f.desc.Focus()
	default:
		f.title.Blur()
		f.desc.Blur()
	}
}

func (f FormModel) Title() string            { return f.title.Value() }
func (f FormModel) Description() string      { return f.desc.Value() }
func (f FormModel) Priority() model.Priority { return f.priority }
func (f FormModel) ColumnIdx() int           { return f.columnIdx }
func (f FormModel) Color() string            { return palette[f.colorIdx] }

// InterceptsEnter reports whether the form should consume an enter key press
// itself (e.g. to insert a newline in the description) rather than treating it
// as a form submission signal.
func (f FormModel) InterceptsEnter() bool {
	return f.focused == fieldDesc
}

var (
	formBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			BorderForeground(lipgloss.Color("#BD93F9"))

	formLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F8F8F2"))

	formFocusedLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#BD93F9"))

	priorityLowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	priorityMedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C"))
	priorityHighStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))

	formHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
)

func (f FormModel) View(title string, h int) string {
	heading := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8F8F2")).Render(title)

	titleLabel := labelStyle(f.focused == fieldTitle, "Title")
	rows := []string{heading, "", titleLabel, f.title.View()}

	if f.isEditCol {
		colorLabel := labelStyle(f.focused == fieldColor, "Color")
		rows = append(rows, "", colorLabel, renderColorPicker(f.colorIdx, f.focused == fieldColor))
	} else if !f.isColForm {
		descLabel := labelStyle(f.focused == fieldDesc, "Description")
		rows = append(rows, "", descLabel, f.desc.View())

		priorityLabel := labelStyle(f.focused == fieldPriority, "Priority")
		rows = append(rows, "", priorityLabel, renderPriority(f.priority, f.focused == fieldPriority))

		if len(f.columns) > 0 {
			colLabel := labelStyle(f.focused == fieldColumn, "Column")
			rows = append(rows, "", colLabel, renderColumnSelector(f.columns, f.columnIdx, f.focused == fieldColumn))
		}
	}

	var help string
	switch f.focused {
	case fieldPriority, fieldColumn:
		help = formHelpStyle.Render("←/→: change  •  tab: next field  •  enter: confirm  •  esc: cancel")
	default:
		help = formHelpStyle.Render("tab/shift+tab: next/prev field  •  enter: confirm  •  esc: cancel")
	}
	rows = append(rows, "", help)

	// Expand vertically: target ~80% of terminal height, minus box borders+padding (4 lines).
	targetHeight := (h * 4 / 5) - 4
	padding := targetHeight - len(rows)
	// Insert blank lines before the help row so content stays at the top.
	if padding > 0 {
		helpRow := rows[len(rows)-1]
		rows = rows[:len(rows)-1]
		for range padding {
			rows = append(rows, "")
		}
		rows = append(rows, helpRow)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return formBoxStyle.Render(body)
}

func labelStyle(focused bool, text string) string {
	if focused {
		return formFocusedLabelStyle.Render(text)
	}
	return formLabelStyle.Render(text)
}

func renderPriority(p model.Priority, focused bool) string {
	opts := []model.Priority{model.PriorityLow, model.PriorityMed, model.PriorityHigh}
	var parts []string
	for _, opt := range opts {
		var s string
		var style lipgloss.Style
		switch opt {
		case model.PriorityLow:
			style = priorityLowStyle
		case model.PriorityMed:
			style = priorityMedStyle
		case model.PriorityHigh:
			style = priorityHighStyle
		}
		if opt == p {
			s = style.Bold(true).Underline(focused).Render("[" + opt.String() + "]")
		} else {
			s = style.Render(opt.String())
		}
		parts = append(parts, s)
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, parts[0], "  ", parts[1], "  ", parts[2])
}

func renderColorPicker(idx int, focused bool) string {
	var parts []string
	for i, hex := range palette {
		swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
		if i == idx {
			parts = append(parts, swatch.Bold(true).Underline(focused).Render("[●]"))
		} else {
			parts = append(parts, swatch.Render("●"))
		}
	}
	return strings.Join(parts, "  ")
}

func renderColumnSelector(columns []model.Column, idx int, focused bool) string {
	var parts []string
	for i, col := range columns {
		var name string
		if i == idx {
			name = lipgloss.NewStyle().Bold(true).Underline(focused).Foreground(lipgloss.Color(col.Color)).Render("[" + col.Name + "]")
		} else {
			name = lipgloss.NewStyle().Foreground(lipgloss.Color(col.Color)).Render(col.Name)
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, "  ")
}
