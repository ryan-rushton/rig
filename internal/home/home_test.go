package home

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ryan-rushton/rig/internal/messages"
	_ "github.com/ryan-rushton/rig/internal/tools/gitbranch" // registers tool via init()
)

func keyRune(r rune) tea.KeyMsg        { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }
func keyType(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

func TestNavigation_BoundsChecking(t *testing.T) {
	m := New("dev")

	// With only one tool, cursor should stay at 0.
	r, _ := m.Update(keyRune('j'))
	got := r.(Model)
	if got.cursor != 0 {
		t.Errorf("expected cursor=0, got %d", got.cursor)
	}

	r, _ = got.Update(keyRune('k'))
	got = r.(Model)
	if got.cursor != 0 {
		t.Errorf("expected cursor=0, got %d", got.cursor)
	}
}

func TestEnter_SelectsTool(t *testing.T) {
	m := New("dev")

	_, cmd := m.Update(keyType(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("expected non-nil cmd on enter")
	}

	msg := cmd()
	sel, ok := msg.(messages.ToolSelectedMsg)
	if !ok {
		t.Fatalf("expected ToolSelectedMsg, got %T", msg)
	}
	if sel.ID != "git-branch" {
		t.Errorf("expected tool ID 'git-branch', got %q", sel.ID)
	}
}

func TestSpace_SelectsTool(t *testing.T) {
	m := New("dev")

	_, cmd := m.Update(keyRune(' '))
	if cmd == nil {
		t.Fatal("expected non-nil cmd on space")
	}

	msg := cmd()
	if _, ok := msg.(messages.ToolSelectedMsg); !ok {
		t.Errorf("expected ToolSelectedMsg, got %T", msg)
	}
}

func TestQuit_CtrlC(t *testing.T) {
	m := New("dev")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("expected quit cmd on ctrl+c")
	}
}

func TestQuit_Q(t *testing.T) {
	m := New("dev")

	_, cmd := m.Update(keyRune('q'))
	if cmd == nil {
		t.Error("expected quit cmd on q")
	}
}
