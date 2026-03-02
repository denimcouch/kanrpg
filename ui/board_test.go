package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/denimcouch/kancli-demo/model"
)

func newTestRenderer(t *testing.T, width int) *glamour.TermRenderer {
	t.Helper()
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		t.Fatalf("failed to create glamour renderer: %v", err)
	}
	return r
}

func TestRenderMarkdown_Basic(t *testing.T) {
	r := newTestRenderer(t, 80)
	out := renderMarkdown("**bold** and _italic_", r)
	if out == "" {
		t.Fatal("expected non-empty output")
	}
}

func TestRenderMarkdown_WidthRespected(t *testing.T) {
	// A single long word won't wrap, but a sentence of words should.
	// Verify rendered output is non-empty and contains expected content.
	r := newTestRenderer(t, 40)
	longLine := strings.Repeat("word ", 30)
	out := renderMarkdown(longLine, r)
	if out == "" {
		t.Fatal("expected non-empty output for long line")
	}
	// At width=40, a 150-char line should produce at least one newline in the output.
	if !strings.Contains(out, "\n") {
		t.Errorf("expected wrapped output to contain newlines, got: %q", out)
	}
}

func TestRenderMarkdown_FallbackOnNilRenderer(t *testing.T) {
	content := "plain text"
	out := renderMarkdown(content, nil)
	if out == "" {
		t.Fatal("expected non-empty fallback output")
	}
	// Should contain the original content text.
	if !strings.Contains(out, content) {
		t.Errorf("expected fallback output to contain %q, got: %q", content, out)
	}
}

func TestRenderTaskViewContent_EmptyDescription(t *testing.T) {
	task := model.Task{
		ID:        1,
		Title:     "Test Task",
		Priority:  model.PriorityLow,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	col := model.Column{ID: 1, Name: "Todo", Color: "#FFFFFF"}

	out := renderTaskViewContent(task, col, nil)
	if !strings.Contains(out, "(no description)") {
		t.Errorf("expected '(no description)' in output, got: %q", out)
	}
}
