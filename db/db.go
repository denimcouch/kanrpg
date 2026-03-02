package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/denimcouch/kancli-demo/model"
	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db := &DB{conn: conn}

	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	if err := db.seed(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("seed: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
		PRAGMA foreign_keys = ON;

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
	`)
	return err
}

func (db *DB) seed() error {
	var count int
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM columns`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaults := []struct {
		name  string
		color string
	}{
		{"Todo", "#8BE9FD"},
		{"In Progress", "#FFB86C"},
		{"Done", "#50FA7B"},
	}

	for i, col := range defaults {
		if _, err := db.CreateColumn(col.name, col.color, i); err != nil {
			return fmt.Errorf("seed column %q: %w", col.name, err)
		}
	}

	return nil
}

// Column Operations

func (db *DB) CreateColumn(name, color string, position int) (model.Column, error) {
	res, err := db.conn.Exec(
		`INSERT INTO columns (name, color, position) VALUES (?, ?, ?)`,
		name, color, position,
	)
	if err != nil {
		return model.Column{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return model.Column{}, err
	}

	return model.Column{
		ID:       int(id),
		Name:     name,
		Color:    color,
		Position: position,
	}, nil
}

func (db *DB) GetColumns() ([]model.Column, error) {
	rows, err := db.conn.Query(`SELECT id, name, position, color FROM columns ORDER BY position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []model.Column
	for rows.Next() {
		var c model.Column
		if err := rows.Scan(&c.ID, &c.Name, &c.Position, &c.Color); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

func (db *DB) UpdateColumn(col model.Column) error {
	_, err := db.conn.Exec(
		`UPDATE columns SET name = ?, color = ?, position = ? WHERE id = ?`,
		col.Name, col.Color, col.Position, col.ID,
	)
	return err
}

func (db *DB) DeleteColumn(id int) error {
	_, err := db.conn.Exec(`DELETE FROM columns WHERE id = ?`, id)
	return err
}

func (db *DB) ReorderColumns(ids []int) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE columns SET position = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, id := range ids {
		if _, err := stmt.Exec(i, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Task Operations

func (db *DB) CreateTask(columnID int, priority model.Priority) (model.Task, error) {
	var maxPos sql.NullInt64
	if err := db.conn.QueryRow(
		`SELECT MAX(position) FROM tasks WHERE column_id = ?`, columnID,
	).Scan(&maxPos); err != nil {
		return model.Task{}, err
	}

	position := 0
	if maxPos.Valid {
		position = int(maxPos.Int64) + 1
	}

	now := time.Now()

	tx, err := db.conn.Begin()
	if err != nil {
		return model.Task{}, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT INTO tasks (title, description, column_id, position, priority, created_at, updated_at)
		 VALUES ('', '', ?, ?, ?, ?, ?)`,
		columnID, position, int(priority), now, now,
	)
	if err != nil {
		return model.Task{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return model.Task{}, err
	}

	// Set default title to "Task {id}" — done in the same transaction so both
	// writes are atomic: either the task exists with a title, or not at all.
	title := fmt.Sprintf("Task %d", id)
	if _, err := tx.Exec(`UPDATE tasks SET title = ? WHERE id = ?`, title, id); err != nil {
		return model.Task{}, err
	}

	if err := tx.Commit(); err != nil {
		return model.Task{}, err
	}

	return model.Task{
		ID:          int(id),
		Title:       title,
		Description: "",
		ColumnID:    columnID,
		Position:    position,
		Priority:    priority,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (db *DB) GetTasksByColumn(columnID int) ([]model.Task, error) {
	rows, err := db.conn.Query(
		`SELECT id, title, COALESCE(description, ''), column_id, position, priority, created_at, updated_at
		 FROM tasks WHERE column_id = ? ORDER BY position`,
		columnID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		var t model.Task
		var p int
		if err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.ColumnID,
			&t.Position, &p, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		t.Priority = model.Priority(p)
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (db *DB) UpdateTask(task model.Task) error {
	task.UpdatedAt = time.Now()
	_, err := db.conn.Exec(
		`UPDATE tasks SET title = ?, description = ?, priority = ?, updated_at = ? WHERE id = ?`,
		task.Title, task.Description, int(task.Priority), task.UpdatedAt, task.ID,
	)
	return err
}

func (db *DB) DeleteTask(id int) error {
	// Get current task position and column before deleting
	var columnID, position int
	if err := db.conn.QueryRow(
		`SELECT column_id, position FROM tasks WHERE id = ?`, id,
	).Scan(&columnID, &position); err != nil {
		return err
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM tasks WHERE id = ?`, id); err != nil {
		return err
	}

	// Close the gap in positions
	if _, err := tx.Exec(
		`UPDATE tasks SET position = position - 1 WHERE column_id = ? AND position > ?`,
		columnID, position,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) MoveTask(taskID, targetColumnID, targetPosition int) error {
	// Get current task state
	var srcColumnID, srcPosition int
	if err := db.conn.QueryRow(
		`SELECT column_id, position FROM tasks WHERE id = ?`, taskID,
	).Scan(&srcColumnID, &srcPosition); err != nil {
		return err
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Close gap in source column
	if _, err := tx.Exec(
		`UPDATE tasks SET position = position - 1 WHERE column_id = ? AND position > ?`,
		srcColumnID, srcPosition,
	); err != nil {
		return err
	}

	// 2. Open gap in target column
	if _, err := tx.Exec(
		`UPDATE tasks SET position = position + 1 WHERE column_id = ? AND position >= ?`,
		targetColumnID, targetPosition,
	); err != nil {
		return err
	}

	// 3. Move the task
	if _, err := tx.Exec(
		`UPDATE tasks SET column_id = ?, position = ?, updated_at = ? WHERE id = ?`,
		targetColumnID, targetPosition, time.Now(), taskID,
	); err != nil {
		return err
	}

	return tx.Commit()
}
