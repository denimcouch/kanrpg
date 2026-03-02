package model

import "testing"

func TestPriority_String(t *testing.T) {
	tests := []struct {
		priority Priority
		want     string
	}{
		{PriorityLow, "Low"},
		{PriorityMed, "Med"},
		{PriorityHigh, "High"},
		{Priority(0), "Low"},  // out-of-range defaults to Low
		{Priority(99), "Low"}, // out-of-range defaults to Low
	}

	for _, tc := range tests {
		got := tc.priority.String()
		if got != tc.want {
			t.Errorf("Priority(%d).String() = %q, want %q", int(tc.priority), got, tc.want)
		}
	}
}

func TestPriority_Label(t *testing.T) {
	tests := []struct {
		priority Priority
		want     string
	}{
		{PriorityLow, "! Low"},
		{PriorityMed, "!! Med"},
		{PriorityHigh, "!!! High"},
		{Priority(0), "! Low"},  // out-of-range defaults to Low
		{Priority(99), "! Low"}, // out-of-range defaults to Low
	}

	for _, tc := range tests {
		got := tc.priority.Label()
		if got != tc.want {
			t.Errorf("Priority(%d).Label() = %q, want %q", int(tc.priority), got, tc.want)
		}
	}
}
