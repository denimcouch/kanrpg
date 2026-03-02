package model

import "time"

type Priority int

const (
	PriorityLow  Priority = 1
	PriorityMed  Priority = 2
	PriorityHigh Priority = 3
)

func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "Low"
	case PriorityMed:
		return "Med"
	case PriorityHigh:
		return "High"
	default:
		return "Low"
	}
}

func (p Priority) Label() string {
	switch p {
	case PriorityLow:
		return "! Low"
	case PriorityMed:
		return "!! Med"
	case PriorityHigh:
		return "!!! High"
	default:
		return "! Low"
	}
}

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
