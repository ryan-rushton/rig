package home

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ryan-rushton/rig/internal/messages"
	_ "github.com/ryan-rushton/rig/internal/tools/gitbranch"
)

func keyRune(r rune) tea.KeyPressMsg    { return tea.KeyPressMsg{Code: r, Text: string(r)} }
func keyCode(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

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

	_, cmd := m.Update(keyCode(tea.KeyEnter))
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

func TestCtrlC_NoOp(t *testing.T) {
	m := New("dev")

	// ctrl+c is handled at the app level, not individual screens.
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd != nil {
		t.Error("expected nil cmd — ctrl+c should be handled by the app, not the home screen")
	}
}

func TestQuit_Q(t *testing.T) {
	m := New("dev")

	_, cmd := m.Update(keyRune('q'))
	if cmd == nil {
		t.Error("expected quit cmd on q")
	}
}
