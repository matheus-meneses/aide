package prompt

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestModel(n int) selectModel {
	choices := make([]Choice, n)
	for i := range choices {
		choices[i] = Choice{Title: string(rune('a' + i))}
	}
	return selectModel{header: "pick", choices: choices, chosen: -1}
}

func send(m selectModel, key tea.Key) (selectModel, tea.Cmd) {
	next, cmd := m.Update(tea.KeyMsg(key))
	return next.(selectModel), cmd
}

func TestSelectNavigation(t *testing.T) {
	m := newTestModel(3)

	m, _ = send(m, tea.Key{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Fatalf("up at top: cursor = %d, want 0", m.cursor)
	}

	m, _ = send(m, tea.Key{Type: tea.KeyDown})
	m, _ = send(m, tea.Key{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Fatalf("after two downs: cursor = %d, want 2", m.cursor)
	}

	m, _ = send(m, tea.Key{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Fatalf("down at bottom: cursor = %d, want 2", m.cursor)
	}

	m, _ = send(m, tea.Key{Type: tea.KeyHome})
	if m.cursor != 0 {
		t.Fatalf("home: cursor = %d, want 0", m.cursor)
	}

	m, _ = send(m, tea.Key{Type: tea.KeyEnd})
	if m.cursor != 2 {
		t.Fatalf("end: cursor = %d, want 2", m.cursor)
	}
}

func TestSelectEnterChooses(t *testing.T) {
	m := newTestModel(3)
	m, _ = send(m, tea.Key{Type: tea.KeyDown})
	m, cmd := send(m, tea.Key{Type: tea.KeyEnter})
	if m.chosen != 1 {
		t.Fatalf("chosen = %d, want 1", m.chosen)
	}
	if cmd == nil {
		t.Fatal("enter should return a quit command")
	}
	if v := m.View(); v != "" {
		t.Fatalf("view after choosing should be empty, got %q", v)
	}
}

func TestSelectCancelLeavesNoChoice(t *testing.T) {
	m := newTestModel(3)
	m, cmd := send(m, tea.Key{Type: tea.KeyEsc})
	if m.chosen != -1 {
		t.Fatalf("chosen after esc = %d, want -1", m.chosen)
	}
	if cmd == nil {
		t.Fatal("esc should return a quit command")
	}
}

func TestSelectWindowing(t *testing.T) {
	m := newTestModel(40)
	m, _ = send(m, tea.Key{Type: tea.KeyEnd})
	view := m.View()
	if !strings.Contains(view, "more above") {
		t.Fatalf("expected scroll indicator for long list, got:\n%s", view)
	}
}
