package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/denimcouch/kancli-demo/db"
	"github.com/denimcouch/kancli-demo/model"
)

type Mode int

const (
	ModeBrowse Mode = iota
	ModeAddTask
	ModeEditTask
	ModeAddColumn
	ModeEditColumn
	ModeViewTask
	ModeConfirmDeleteTask
	ModeConfirmDeleteColumn
)

type Model struct {
	columns       []model.Column
	tasks         map[int][]model.Task
	focusedCol    int
	focusedTask   int
	mode          Mode
	form          FormModel
	db            *db.DB
	width         int
	height        int
	showHelp      bool
	err           error
	scrollOffsets map[int]int // keyed by column ID; first visible task index
}

func NewModel(database *db.DB) (Model, error) {
	m := Model{
		db:            database,
		tasks:         make(map[int][]model.Task),
		scrollOffsets: make(map[int]int),
	}

	if err := m.loadData(); err != nil {
		return m, fmt.Errorf("load data: %w", err)
	}

	return m, nil
}

func (m *Model) loadData() error {
	cols, err := m.db.GetColumns()
	if err != nil {
		return err
	}
	m.columns = cols

	for _, col := range cols {
		tasks, err := m.db.GetTasksByColumn(col.ID)
		if err != nil {
			return err
		}
		if tasks == nil {
			tasks = []model.Task{}
		}
		m.tasks[col.ID] = tasks
	}

	return nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case ModeViewTask:
			return m.updateViewTask(msg)
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

func (m Model) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	// Column navigation
	case "left", "h":
		if m.focusedCol > 0 {
			m.focusedCol--
			m.clampTask()
		}
	case "right", "l":
		if m.focusedCol < len(m.columns)-1 {
			m.focusedCol++
			m.clampTask()
		}

	// Task navigation
	case "up", "k":
		if m.focusedTask > 0 {
			m.focusedTask--
		}
	case "down", "j":
		tasks := m.currentTasks()
		if m.focusedTask < len(tasks)-1 {
			m.focusedTask++
		}

	// New task
	case "n":
		if len(m.columns) == 0 {
			return m, nil
		}
		col := m.columns[m.focusedCol]
		task, err := m.db.CreateTask(col.ID, model.PriorityLow)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.tasks[col.ID] = append(m.tasks[col.ID], task)
		m.focusedTask = len(m.tasks[col.ID]) - 1
		m.form = newTaskForm(task, m.columns, m.focusedCol)
		m.mode = ModeEditTask
		return m, nil

	// View task (read-only)
	case "enter", "v":
		if _, ok := m.focusedTaskObj(); ok {
			m.mode = ModeViewTask
		}
		return m, nil

	// Edit task
	case "e":
		task, ok := m.focusedTaskObj()
		if !ok {
			return m, nil
		}
		m.form = newTaskForm(task, m.columns, m.focusedCol)
		m.mode = ModeEditTask
		return m, nil

	// Delete task
	case "d":
		if _, ok := m.focusedTaskObj(); !ok {
			return m, nil
		}
		m.mode = ModeConfirmDeleteTask
		return m, nil

	// Move task between columns
	case "H":
		return m.moveTaskToColumn(m.focusedCol - 1)
	case "L":
		return m.moveTaskToColumn(m.focusedCol + 1)

	// Reorder task within column
	case "K":
		return m.reorderTask(-1)
	case "J":
		return m.reorderTask(1)

	// Reorder column
	case "<":
		return m.reorderColumn(-1)
	case ">":
		return m.reorderColumn(1)

	// Edit column (rename / recolor)
	case "C":
		if len(m.columns) == 0 {
			return m, nil
		}
		m.form = newEditColumnForm(m.columns[m.focusedCol])
		m.mode = ModeEditColumn
		return m, nil

	// New column
	case "N":
		m.form = newColumnForm()
		m.mode = ModeAddColumn
		return m, nil

	// Delete column
	case "X":
		if len(m.columns) == 0 {
			return m, nil
		}
		m.mode = ModeConfirmDeleteColumn
		return m, nil
	}

	m.ensureTaskVisible()
	return m, nil
}

func (m Model) updateViewTask(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.mode = ModeBrowse
	case "e":
		if task, ok := m.focusedTaskObj(); ok {
			m.form = newTaskForm(task, m.columns, m.focusedCol)
			m.mode = ModeEditTask
		}
	}
	return m, nil
}

func (m Model) updateConfirmDeleteTask(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if task, ok := m.focusedTaskObj(); ok {
			if err := m.db.DeleteTask(task.ID); err != nil {
				m.err = err
			} else {
				col := m.columns[m.focusedCol]
				tasks := m.tasks[col.ID]
				updated := make([]model.Task, 0, len(tasks)-1)
				updated = append(updated, tasks[:m.focusedTask]...)
				updated = append(updated, tasks[m.focusedTask+1:]...)
				m.tasks[col.ID] = updated
				m.clampTask()
				m.ensureTaskVisible()
			}
		}
		m.mode = ModeBrowse
	case "n", "N", "esc":
		m.mode = ModeBrowse
	}
	return m, nil
}

func (m Model) updateConfirmDeleteColumn(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.focusedCol < len(m.columns) {
			col := m.columns[m.focusedCol]
			if err := m.db.DeleteColumn(col.ID); err != nil {
				m.err = err
			} else {
				delete(m.tasks, col.ID)
				updated := make([]model.Column, 0, len(m.columns)-1)
				updated = append(updated, m.columns[:m.focusedCol]...)
				updated = append(updated, m.columns[m.focusedCol+1:]...)
				m.columns = updated
				if m.focusedCol >= len(m.columns) && m.focusedCol > 0 {
					m.focusedCol--
				}
				m.focusedTask = 0
			}
		}
		m.mode = ModeBrowse
	case "n", "N", "esc":
		m.mode = ModeBrowse
	}
	return m, nil
}

func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = ModeBrowse
		return m, nil

	case "enter":
		if !m.form.InterceptsEnter() {
			return m.submitForm()
		}
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m Model) submitForm() (tea.Model, tea.Cmd) {
	switch m.mode {
	case ModeEditTask, ModeAddTask:
		task, ok := m.focusedTaskObj()
		if !ok {
			m.mode = ModeBrowse
			return m, nil
		}

		newTitle := m.form.Title()
		if newTitle == "" {
			newTitle = fmt.Sprintf("Task %d", task.ID)
		}
		task.Title = newTitle
		task.Description = m.form.Description()
		task.Priority = m.form.Priority()

		// Persist field edits (title, description, priority) first, while the
		// task is still in its current column. This keeps the two concerns
		// separate: UpdateTask owns field data, MoveTask owns position/column.
		if err := m.db.UpdateTask(task); err != nil {
			m.err = err
			m.mode = ModeBrowse
			return m, nil
		}

		// Handle column change via the column selector
		newColIdx := m.form.ColumnIdx()
		if newColIdx != m.focusedCol && newColIdx >= 0 && newColIdx < len(m.columns) {
			dstCol := m.columns[newColIdx]
			targetPos := len(m.tasks[dstCol.ID])
			if err := m.db.MoveTask(task.ID, dstCol.ID, targetPos); err != nil {
				m.err = err
				m.mode = ModeBrowse
				return m, nil
			}

			// Update in-memory: remove from src
			srcCol := m.columns[m.focusedCol]
			srcTasks := m.tasks[srcCol.ID]
			newSrc := make([]model.Task, 0, len(srcTasks)-1)
			newSrc = append(newSrc, srcTasks[:m.focusedTask]...)
			newSrc = append(newSrc, srcTasks[m.focusedTask+1:]...)
			m.tasks[srcCol.ID] = newSrc

			task.ColumnID = dstCol.ID
			task.Position = targetPos
			m.tasks[dstCol.ID] = append(m.tasks[dstCol.ID], task)

			m.focusedCol = newColIdx
			m.focusedTask = len(m.tasks[dstCol.ID]) - 1
		} else {
			// Same column: reflect updated fields in-memory.
			col := m.columns[m.focusedCol]
			tasks := m.tasks[col.ID]
			tasks[m.focusedTask] = task
			m.tasks[col.ID] = tasks
		}
		m.mode = ModeBrowse

	case ModeAddColumn:
		name := m.form.Title()
		if name == "" {
			m.mode = ModeBrowse
			return m, nil
		}
		position := len(m.columns)
		col, err := m.db.CreateColumn(name, "#FFFFFF", position)
		if err != nil {
			m.err = err
		} else {
			m.columns = append(m.columns, col)
			m.tasks[col.ID] = []model.Task{}
			m.focusedCol = len(m.columns) - 1
			m.focusedTask = 0
		}
		m.mode = ModeBrowse

	case ModeEditColumn:
		if m.focusedCol >= len(m.columns) {
			m.mode = ModeBrowse
			return m, nil
		}
		col := m.columns[m.focusedCol]
		name := m.form.Title()
		if name == "" {
			name = col.Name
		}
		col.Name = name
		col.Color = m.form.Color()
		if err := m.db.UpdateColumn(col); err != nil {
			m.err = err
		} else {
			m.columns[m.focusedCol] = col
		}
		m.mode = ModeBrowse
	}

	return m, nil
}

func (m Model) moveTaskToColumn(targetColIdx int) (tea.Model, tea.Cmd) {
	if targetColIdx < 0 || targetColIdx >= len(m.columns) {
		return m, nil
	}

	task, ok := m.focusedTaskObj()
	if !ok {
		return m, nil
	}

	srcCol := m.columns[m.focusedCol]
	dstCol := m.columns[targetColIdx]
	targetPos := len(m.tasks[dstCol.ID])

	if err := m.db.MoveTask(task.ID, dstCol.ID, targetPos); err != nil {
		m.err = err
		return m, nil
	}

	task.ColumnID = dstCol.ID
	task.Position = targetPos

	// Build new src slice without the moved task (avoid slice aliasing)
	srcTasks := m.tasks[srcCol.ID]
	newSrc := make([]model.Task, 0, len(srcTasks)-1)
	newSrc = append(newSrc, srcTasks[:m.focusedTask]...)
	newSrc = append(newSrc, srcTasks[m.focusedTask+1:]...)
	m.tasks[srcCol.ID] = newSrc

	m.tasks[dstCol.ID] = append(m.tasks[dstCol.ID], task)

	m.focusedCol = targetColIdx
	m.focusedTask = len(m.tasks[dstCol.ID]) - 1
	m.ensureTaskVisible()

	return m, nil
}

func (m Model) reorderColumn(delta int) (tea.Model, tea.Cmd) {
	newIdx := m.focusedCol + delta
	if newIdx < 0 || newIdx >= len(m.columns) {
		return m, nil
	}

	m.columns[m.focusedCol], m.columns[newIdx] = m.columns[newIdx], m.columns[m.focusedCol]

	ids := make([]int, len(m.columns))
	for i, col := range m.columns {
		ids[i] = col.ID
	}
	if err := m.db.ReorderColumns(ids); err != nil {
		m.err = err
		return m, nil
	}

	m.focusedCol = newIdx
	return m, nil
}

func (m Model) reorderTask(delta int) (tea.Model, tea.Cmd) {
	if len(m.columns) == 0 {
		return m, nil
	}

	col := m.columns[m.focusedCol]
	tasks := m.tasks[col.ID]
	newIdx := m.focusedTask + delta

	if newIdx < 0 || newIdx >= len(tasks) {
		return m, nil
	}

	targetPos := tasks[newIdx].Position

	if err := m.db.MoveTask(tasks[m.focusedTask].ID, col.ID, targetPos); err != nil {
		m.err = err
		return m, nil
	}

	// Reload from DB to get consistent positions after the swap
	updated, err := m.db.GetTasksByColumn(col.ID)
	if err != nil {
		m.err = err
		return m, nil
	}
	m.tasks[col.ID] = updated
	m.focusedTask = newIdx
	m.ensureTaskVisible()

	return m, nil
}

func (m Model) currentTasks() []model.Task {
	if len(m.columns) == 0 {
		return nil
	}
	return m.tasks[m.columns[m.focusedCol].ID]
}

func (m Model) focusedTaskObj() (model.Task, bool) {
	tasks := m.currentTasks()
	if len(tasks) == 0 || m.focusedTask >= len(tasks) {
		return model.Task{}, false
	}
	return tasks[m.focusedTask], true
}

func (m *Model) clampTask() {
	tasks := m.currentTasks()
	if m.focusedTask >= len(tasks) {
		if len(tasks) == 0 {
			m.focusedTask = 0
		} else {
			m.focusedTask = len(tasks) - 1
		}
	}
}

func (m *Model) clampAllScrollOffsets() {
	visible := tasksVisible(m.height - 4)
	for _, col := range m.columns {
		max := len(m.tasks[col.ID]) - visible
		if max < 0 {
			max = 0
		}
		if m.scrollOffsets[col.ID] > max {
			m.scrollOffsets[col.ID] = max
		}
	}
}

func (m *Model) ensureTaskVisible() {
	if len(m.columns) == 0 {
		return
	}
	col := m.columns[m.focusedCol]
	visible := tasksVisible(m.height - 4)
	offset := m.scrollOffsets[col.ID]
	if m.focusedTask >= offset+visible {
		offset = m.focusedTask - visible + 1
	}
	if m.focusedTask < offset {
		offset = m.focusedTask
	}
	if offset < 0 {
		offset = 0
	}
	m.scrollOffsets[col.ID] = offset
}

func (m Model) View() string {
	if m.err != nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Render(fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err))
	}

	switch m.mode {
	case ModeAddTask, ModeEditTask:
		title := "New Task"
		if m.mode == ModeEditTask {
			title = "Edit Task"
		}
		return centerView(m.form.View(title), m.width, m.height)

	case ModeAddColumn:
		return centerView(m.form.View("New Column"), m.width, m.height)

	case ModeEditColumn:
		return centerView(m.form.View("Edit Column"), m.width, m.height)

	case ModeViewTask:
		if task, ok := m.focusedTaskObj(); ok {
			col := m.columns[m.focusedCol]
			return centerView(renderTaskView(task, col, m.width, m.height), m.width, m.height)
		}
	}

	return renderBoard(m)
}

func centerView(content string, w, h int) string {
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}
