package ui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/denimcouch/kancli-demo/model"
)

type formField int

const (
	fieldTitle formField = iota
	fieldDesc
	fieldPriority
	fieldColumn
	fieldCount // total number of task form fields

	fieldCountColumn formField = 1 // column form only has title
)

type FormModel struct {
	title       textinput.Model
	desc        textarea.Model
	priority    model.Priority
	columns     []model.Column // non-nil only in task form
	columnIdx   int            // index into columns
	focused     formField
	isColForm   bool // true when used for adding a column (title only)
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
	ta.CharLimit = 500
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

func newColumnForm() FormModel {
	ti := textinput.New()
	ti.Placeholder = "Column name"
	ti.Focus()
	ti.CharLimit = 60
	ti.Width = 50

	ta := textarea.New()
	ta.SetWidth(52)
	ta.SetHeight(4)
	ta.ShowLineNumbers = false

	return FormModel{
		title:     ti,
		desc:      ta,
		priority:  model.PriorityLow,
		focused:   fieldTitle,
		isColForm: true,
	}
}

func (f FormModel) numFields() formField {
	if f.isColForm {
		return fieldCountColumn
	}
	return fieldCount
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
			}
		}
	}

	if f.focused == fieldTitle {
		var cmd tea.Cmd
		f.title, cmd = f.title.Update(msg)
		cmds = append(cmds, cmd)
	} else if f.focused == fieldDesc {
		var cmd tea.Cmd
		f.desc, cmd = f.desc.Update(msg)
		cmds = append(cmds, cmd)
	}

	return f, tea.Batch(cmds...)
}

func (f *FormModel) syncFocus() {
	if f.focused == fieldTitle {
		f.title.Focus()
		f.desc.Blur()
	} else if f.focused == fieldDesc {
		f.title.Blur()
		f.desc.Focus()
	} else {
		f.title.Blur()
		f.desc.Blur()
	}
}

func (f FormModel) Title() string            { return f.title.Value() }
func (f FormModel) Description() string      { return f.desc.Value() }
func (f FormModel) Priority() model.Priority { return f.priority }
func (f FormModel) ColumnIdx() int           { return f.columnIdx }

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

func (f FormModel) View(title string) string {
	heading := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8F8F2")).Render(title)

	titleLabel := labelStyle(f.focused == fieldTitle, "Title")
	rows := []string{heading, "", titleLabel, f.title.View()}

	if !f.isColForm {
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
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "  "
		}
		result += p
	}
	return result
}
