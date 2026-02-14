package app

import (
	"testing"

	"github.com/ryan-rushton/rig/internal/home"
	"github.com/ryan-rushton/rig/internal/messages"
	"github.com/ryan-rushton/rig/internal/tools/gitbranch"
)

func TestNew_StartsWithHomeScreen(t *testing.T) {
	m := New()
	if _, ok := m.current.(home.Model); !ok {
		t.Errorf("expected home.Model as initial screen, got %T", m.current)
	}
}

func TestToolSelected_SwitchesToGitBranch(t *testing.T) {
	m := New()
	result, cmd := m.Update(messages.ToolSelectedMsg{ID: "git-branch"})
	got := result.(Model)

	if _, ok := got.current.(gitbranch.Model); !ok {
		t.Errorf("expected gitbranch.Model after selection, got %T", got.current)
	}
	if cmd == nil {
		t.Error("expected Init cmd from git-branch tool")
	}
}

func TestToolSelected_UnknownID_NoTransition(t *testing.T) {
	m := New()
	result, _ := m.Update(messages.ToolSelectedMsg{ID: "nonexistent"})
	got := result.(Model)

	if _, ok := got.current.(home.Model); !ok {
		t.Errorf("expected to stay on home screen for unknown tool, got %T", got.current)
	}
}

func TestBackMsg_ReturnsToHome(t *testing.T) {
	m := New()
	// First navigate to a tool.
	r, _ := m.Update(messages.ToolSelectedMsg{ID: "git-branch"})
	m = r.(Model)

	// Then go back.
	r, _ = m.Update(messages.BackMsg{})
	got := r.(Model)

	if _, ok := got.current.(home.Model); !ok {
		t.Errorf("expected home.Model after BackMsg, got %T", got.current)
	}
}
