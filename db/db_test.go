package db

import (
	"fmt"
	"testing"

	"github.com/denimcouch/kancli-demo/model"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := New(":memory:")
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

// ── Column tests ──────────────────────────────────────────────────────────────

func TestCreateColumn(t *testing.T) {
	d := newTestDB(t)

	col, err := d.CreateColumn("Backlog", "#FF0000", 10)
	if err != nil {
		t.Fatalf("CreateColumn: %v", err)
	}

	cols, err := d.GetColumns()
	if err != nil {
		t.Fatalf("GetColumns: %v", err)
	}

	var found *model.Column
	for i := range cols {
		if cols[i].ID == col.ID {
			found = &cols[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("created column id=%d not found in GetColumns", col.ID)
	}
	if found.Name != "Backlog" {
		t.Errorf("name: got %q, want %q", found.Name, "Backlog")
	}
	if found.Color != "#FF0000" {
		t.Errorf("color: got %q, want %q", found.Color, "#FF0000")
	}
	if found.Position != 10 {
		t.Errorf("position: got %d, want %d", found.Position, 10)
	}
}

func TestGetColumns_OrderedByPosition(t *testing.T) {
	d := newTestDB(t)

	// seed() already created 3 columns at positions 0,1,2 — delete them so we
	// control the full set.
	existing, err := d.GetColumns()
	if err != nil {
		t.Fatalf("GetColumns: %v", err)
	}
	for _, c := range existing {
		if err := d.DeleteColumn(c.ID); err != nil {
			t.Fatalf("DeleteColumn: %v", err)
		}
	}

	_, err = d.CreateColumn("C", "#000003", 20)
	if err != nil {
		t.Fatalf("CreateColumn C: %v", err)
	}
	_, err = d.CreateColumn("A", "#000001", 5)
	if err != nil {
		t.Fatalf("CreateColumn A: %v", err)
	}
	_, err = d.CreateColumn("B", "#000002", 12)
	if err != nil {
		t.Fatalf("CreateColumn B: %v", err)
	}

	cols, err := d.GetColumns()
	if err != nil {
		t.Fatalf("GetColumns: %v", err)
	}
	if len(cols) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(cols))
	}

	want := []string{"A", "B", "C"}
	for i, c := range cols {
		if c.Name != want[i] {
			t.Errorf("cols[%d].Name = %q, want %q", i, c.Name, want[i])
		}
	}
}

func TestUpdateColumn(t *testing.T) {
	d := newTestDB(t)

	col, err := d.CreateColumn("Original", "#111111", 0)
	if err != nil {
		t.Fatalf("CreateColumn: %v", err)
	}

	col.Name = "Updated"
	col.Color = "#222222"
	col.Position = 99
	if err := d.UpdateColumn(col); err != nil {
		t.Fatalf("UpdateColumn: %v", err)
	}

	cols, err := d.GetColumns()
	if err != nil {
		t.Fatalf("GetColumns: %v", err)
	}

	var found *model.Column
	for i := range cols {
		if cols[i].ID == col.ID {
			found = &cols[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("column id=%d not found after update", col.ID)
	}
	if found.Name != "Updated" {
		t.Errorf("name: got %q, want %q", found.Name, "Updated")
	}
	if found.Color != "#222222" {
		t.Errorf("color: got %q, want %q", found.Color, "#222222")
	}
	if found.Position != 99 {
		t.Errorf("position: got %d, want %d", found.Position, 99)
	}
}

func TestDeleteColumn(t *testing.T) {
	d := newTestDB(t)

	a, err := d.CreateColumn("Keep", "#AAAAAA", 0)
	if err != nil {
		t.Fatalf("CreateColumn Keep: %v", err)
	}
	b, err := d.CreateColumn("Delete", "#BBBBBB", 1)
	if err != nil {
		t.Fatalf("CreateColumn Delete: %v", err)
	}

	// Remove the seeded columns so we control the full set.
	cols, err := d.GetColumns()
	if err != nil {
		t.Fatalf("GetColumns: %v", err)
	}
	for _, c := range cols {
		if c.ID != a.ID && c.ID != b.ID {
			if err := d.DeleteColumn(c.ID); err != nil {
				t.Fatalf("DeleteColumn seed: %v", err)
			}
		}
	}

	if err := d.DeleteColumn(b.ID); err != nil {
		t.Fatalf("DeleteColumn: %v", err)
	}

	cols, err = d.GetColumns()
	if err != nil {
		t.Fatalf("GetColumns: %v", err)
	}
	if len(cols) != 1 {
		t.Fatalf("expected 1 column, got %d", len(cols))
	}
	if cols[0].ID != a.ID {
		t.Errorf("remaining column id=%d, want %d", cols[0].ID, a.ID)
	}
}

func TestReorderColumns(t *testing.T) {
	d := newTestDB(t)

	// Use the 3 seeded columns and reorder them.
	cols, err := d.GetColumns()
	if err != nil {
		t.Fatalf("GetColumns: %v", err)
	}
	if len(cols) != 3 {
		t.Fatalf("expected 3 seeded columns, got %d", len(cols))
	}

	// Reverse the order.
	reversed := []int{cols[2].ID, cols[1].ID, cols[0].ID}
	if err := d.ReorderColumns(reversed); err != nil {
		t.Fatalf("ReorderColumns: %v", err)
	}

	reordered, err := d.GetColumns()
	if err != nil {
		t.Fatalf("GetColumns after reorder: %v", err)
	}

	for i, c := range reordered {
		if c.ID != reversed[i] {
			t.Errorf("position %d: got id=%d, want id=%d", i, c.ID, reversed[i])
		}
		if c.Position != i {
			t.Errorf("position %d: position field = %d, want %d", i, c.Position, i)
		}
	}
}

func TestSeed_DefaultColumns(t *testing.T) {
	d := newTestDB(t)

	cols, err := d.GetColumns()
	if err != nil {
		t.Fatalf("GetColumns: %v", err)
	}
	if len(cols) != 3 {
		t.Fatalf("expected 3 seeded columns, got %d", len(cols))
	}

	wantNames := []string{"Todo", "In Progress", "Done"}
	for i, c := range cols {
		if c.Name != wantNames[i] {
			t.Errorf("cols[%d].Name = %q, want %q", i, c.Name, wantNames[i])
		}
	}
}

// ── Task tests ────────────────────────────────────────────────────────────────

func TestCreateTask_DefaultTitle(t *testing.T) {
	d := newTestDB(t)

	cols, _ := d.GetColumns()
	colID := cols[0].ID

	task, err := d.CreateTask(colID, model.PriorityLow)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	want := fmt.Sprintf("Task %d", task.ID)
	if task.Title != want {
		t.Errorf("title: got %q, want %q", task.Title, want)
	}
}

func TestCreateTask_PositionAutoIncrement(t *testing.T) {
	d := newTestDB(t)

	cols, _ := d.GetColumns()
	colID := cols[0].ID

	for i := 0; i < 3; i++ {
		task, err := d.CreateTask(colID, model.PriorityLow)
		if err != nil {
			t.Fatalf("CreateTask %d: %v", i, err)
		}
		if task.Position != i {
			t.Errorf("task %d: position = %d, want %d", i, task.Position, i)
		}
	}
}

func TestGetTasksByColumn_Empty(t *testing.T) {
	d := newTestDB(t)

	cols, _ := d.GetColumns()
	tasks, err := d.GetTasksByColumn(cols[0].ID)
	if err != nil {
		t.Fatalf("GetTasksByColumn: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestGetTasksByColumn_RoundTrip(t *testing.T) {
	d := newTestDB(t)

	cols, _ := d.GetColumns()
	colID := cols[0].ID

	created, err := d.CreateTask(colID, model.PriorityHigh)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	tasks, err := d.GetTasksByColumn(colID)
	if err != nil {
		t.Fatalf("GetTasksByColumn: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	got := tasks[0]
	if got.ID != created.ID {
		t.Errorf("id: got %d, want %d", got.ID, created.ID)
	}
	if got.Title != created.Title {
		t.Errorf("title: got %q, want %q", got.Title, created.Title)
	}
	if got.ColumnID != colID {
		t.Errorf("column_id: got %d, want %d", got.ColumnID, colID)
	}
	if got.Priority != model.PriorityHigh {
		t.Errorf("priority: got %v, want %v", got.Priority, model.PriorityHigh)
	}
	if got.Position != 0 {
		t.Errorf("position: got %d, want 0", got.Position)
	}
}

func TestUpdateTask(t *testing.T) {
	d := newTestDB(t)

	cols, _ := d.GetColumns()
	colID := cols[0].ID

	task, err := d.CreateTask(colID, model.PriorityLow)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	task.Title = "Updated Title"
	task.Description = "Some description"
	task.Priority = model.PriorityHigh
	if err := d.UpdateTask(task); err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}

	tasks, err := d.GetTasksByColumn(colID)
	if err != nil {
		t.Fatalf("GetTasksByColumn: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	got := tasks[0]
	if got.Title != "Updated Title" {
		t.Errorf("title: got %q, want %q", got.Title, "Updated Title")
	}
	if got.Description != "Some description" {
		t.Errorf("description: got %q, want %q", got.Description, "Some description")
	}
	if got.Priority != model.PriorityHigh {
		t.Errorf("priority: got %v, want High", got.Priority)
	}
}

func TestDeleteTask_GapClose(t *testing.T) {
	d := newTestDB(t)

	cols, _ := d.GetColumns()
	colID := cols[0].ID

	t0, _ := d.CreateTask(colID, model.PriorityLow)
	t1, _ := d.CreateTask(colID, model.PriorityLow)
	t2, _ := d.CreateTask(colID, model.PriorityLow)

	if err := d.DeleteTask(t1.ID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	tasks, err := d.GetTasksByColumn(colID)
	if err != nil {
		t.Fatalf("GetTasksByColumn: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != t0.ID || tasks[0].Position != 0 {
		t.Errorf("tasks[0]: id=%d pos=%d, want id=%d pos=0", tasks[0].ID, tasks[0].Position, t0.ID)
	}
	if tasks[1].ID != t2.ID || tasks[1].Position != 1 {
		t.Errorf("tasks[1]: id=%d pos=%d, want id=%d pos=1", tasks[1].ID, tasks[1].Position, t2.ID)
	}
}

func TestDeleteTask_OnlyTask(t *testing.T) {
	d := newTestDB(t)

	cols, _ := d.GetColumns()
	colID := cols[0].ID

	task, err := d.CreateTask(colID, model.PriorityLow)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if err := d.DeleteTask(task.ID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	tasks, err := d.GetTasksByColumn(colID)
	if err != nil {
		t.Fatalf("GetTasksByColumn: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

// ── MoveTask tests ────────────────────────────────────────────────────────────

func TestMoveTask_CrossColumn(t *testing.T) {
	d := newTestDB(t)

	cols, _ := d.GetColumns()
	colA := cols[0].ID
	colB := cols[1].ID

	// 3 tasks in col A at positions 0, 1, 2
	tA0, _ := d.CreateTask(colA, model.PriorityLow)
	tA1, _ := d.CreateTask(colA, model.PriorityLow)
	tA2, _ := d.CreateTask(colA, model.PriorityLow)

	// Move tA1 (position 1) to col B at position 0
	if err := d.MoveTask(tA1.ID, colB, 0); err != nil {
		t.Fatalf("MoveTask: %v", err)
	}

	// Col A should compact: tA0 at 0, tA2 at 1
	tasksA, err := d.GetTasksByColumn(colA)
	if err != nil {
		t.Fatalf("GetTasksByColumn A: %v", err)
	}
	if len(tasksA) != 2 {
		t.Fatalf("col A: expected 2 tasks, got %d", len(tasksA))
	}
	if tasksA[0].ID != tA0.ID || tasksA[0].Position != 0 {
		t.Errorf("col A[0]: id=%d pos=%d, want id=%d pos=0", tasksA[0].ID, tasksA[0].Position, tA0.ID)
	}
	if tasksA[1].ID != tA2.ID || tasksA[1].Position != 1 {
		t.Errorf("col A[1]: id=%d pos=%d, want id=%d pos=1", tasksA[1].ID, tasksA[1].Position, tA2.ID)
	}

	// Col B should have the moved task at position 0
	tasksB, err := d.GetTasksByColumn(colB)
	if err != nil {
		t.Fatalf("GetTasksByColumn B: %v", err)
	}
	if len(tasksB) != 1 {
		t.Fatalf("col B: expected 1 task, got %d", len(tasksB))
	}
	if tasksB[0].ID != tA1.ID {
		t.Errorf("col B[0].ID = %d, want %d", tasksB[0].ID, tA1.ID)
	}
	if tasksB[0].ColumnID != colB {
		t.Errorf("col B[0].ColumnID = %d, want %d", tasksB[0].ColumnID, colB)
	}
	if tasksB[0].Position != 0 {
		t.Errorf("col B[0].Position = %d, want 0", tasksB[0].Position)
	}
}

func TestMoveTask_SameColumn_Reorder(t *testing.T) {
	d := newTestDB(t)

	cols, _ := d.GetColumns()
	colID := cols[0].ID

	t0, _ := d.CreateTask(colID, model.PriorityLow) // position 0
	t1, _ := d.CreateTask(colID, model.PriorityLow) // position 1

	// Move t0 (pos 0) to pos 1 — should swap
	if err := d.MoveTask(t0.ID, colID, 1); err != nil {
		t.Fatalf("MoveTask: %v", err)
	}

	tasks, err := d.GetTasksByColumn(colID)
	if err != nil {
		t.Fatalf("GetTasksByColumn: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != t1.ID || tasks[0].Position != 0 {
		t.Errorf("tasks[0]: id=%d pos=%d, want id=%d pos=0", tasks[0].ID, tasks[0].Position, t1.ID)
	}
	if tasks[1].ID != t0.ID || tasks[1].Position != 1 {
		t.Errorf("tasks[1]: id=%d pos=%d, want id=%d pos=1", tasks[1].ID, tasks[1].Position, t0.ID)
	}
}

func TestMoveTask_ToEmptyColumn(t *testing.T) {
	d := newTestDB(t)

	cols, _ := d.GetColumns()
	colA := cols[0].ID
	colB := cols[1].ID

	task, err := d.CreateTask(colA, model.PriorityMed)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if err := d.MoveTask(task.ID, colB, 0); err != nil {
		t.Fatalf("MoveTask: %v", err)
	}

	tasksB, err := d.GetTasksByColumn(colB)
	if err != nil {
		t.Fatalf("GetTasksByColumn B: %v", err)
	}
	if len(tasksB) != 1 {
		t.Fatalf("expected 1 task in col B, got %d", len(tasksB))
	}
	if tasksB[0].Position != 0 {
		t.Errorf("position: got %d, want 0", tasksB[0].Position)
	}
	if tasksB[0].ColumnID != colB {
		t.Errorf("column_id: got %d, want %d", tasksB[0].ColumnID, colB)
	}

	tasksA, err := d.GetTasksByColumn(colA)
	if err != nil {
		t.Fatalf("GetTasksByColumn A: %v", err)
	}
	if len(tasksA) != 0 {
		t.Errorf("expected 0 tasks in col A, got %d", len(tasksA))
	}
}
