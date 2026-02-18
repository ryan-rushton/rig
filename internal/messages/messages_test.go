package messages

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// mockModel is a minimal tea.Model for testing the Standalone wrapper.
type mockModel struct {
	lastMsg    tea.Msg
	viewString string
}

func (m mockModel) Init() tea.Cmd                           { return nil }
func (m mockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { m.lastMsg = msg; return m, nil }
func (m mockModel) View() string                            { return m.viewString }

func TestStandalone_BackMsg_Quits(t *testing.T) {
	inner := mockModel{viewString: "inner"}
	s := Standalone(inner)

	_, cmd := s.Update(BackMsg{})

	if cmd == nil {
		t.Fatal("expected non-nil cmd for quit")
	}
	// tea.Quit returns a special quit message.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestStandalone_CtrlC_Quits(t *testing.T) {
	inner := mockModel{viewString: "inner"}
	s := Standalone(inner)

	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	if cmd == nil {
		t.Fatal("expected non-nil cmd for quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestStandalone_OtherMsg_Delegates(t *testing.T) {
	inner := mockModel{viewString: "inner"}
	s := Standalone(inner)

	type customMsg struct{}
	result, _ := s.Update(customMsg{})

	// The returned model should be a standalone wrapping the updated inner.
	st, ok := result.(standalone)
	if !ok {
		t.Fatalf("expected standalone, got %T", result)
	}
	updated, ok := st.inner.(mockModel)
	if !ok {
		t.Fatalf("expected mockModel inner, got %T", st.inner)
	}
	if updated.lastMsg != (customMsg{}) {
		t.Errorf("expected customMsg passed to inner, got %T", updated.lastMsg)
	}
}

func TestStandalone_View_Delegates(t *testing.T) {
	inner := mockModel{viewString: "hello"}
	s := Standalone(inner)

	if got := s.View(); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestStandalone_Init_Delegates(t *testing.T) {
	inner := mockModel{}
	s := Standalone(inner)

	cmd := s.Init()
	if cmd != nil {
		t.Error("expected nil cmd from mock Init")
	}
}
