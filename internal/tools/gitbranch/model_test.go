package gitbranch

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ryan-rushton/rig/internal/messages"
)

// Key helpers to keep tests readable.
func keyRune(r rune) tea.KeyMsg        { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }
func keyType(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

// modelWithBranches returns a Model in stateBrowse with the supplied branches.
func modelWithBranches(branches []Branch) Model {
	m := New()
	m.state = stateBrowse
	m.branches = branches
	return m
}

var testBranches = []Branch{
	{Name: "main", Upstream: "origin/main", IsCurrent: true, HasRemote: true},
	{Name: "feature/foo", Upstream: "origin/feature/foo", IsCurrent: false, HasRemote: true},
	{Name: "local-only", Upstream: "", IsCurrent: false, HasRemote: false},
}

// ---------------------------------------------------------------------------
// Pure functions
// ---------------------------------------------------------------------------

func TestSplitUpstream(t *testing.T) {
	tests := []struct {
		input      string
		wantRemote string
		wantBranch string
	}{
		{"origin/main", "origin", "main"},
		{"origin/feature/foo", "origin", "feature/foo"},
		{"upstream/deep/nested/branch", "upstream", "deep/nested/branch"},
		{"noSlash", "noSlash", "noSlash"},
		{"", "", ""},
	}
	for _, tt := range tests {
		remote, branch := splitUpstream(tt.input)
		if remote != tt.wantRemote || branch != tt.wantBranch {
			t.Errorf("splitUpstream(%q) = (%q, %q), want (%q, %q)",
				tt.input, remote, branch, tt.wantRemote, tt.wantBranch)
		}
	}
}

func TestBranchExists(t *testing.T) {
	m := modelWithBranches(testBranches)

	if !m.branchExists("main") {
		t.Error("expected main to exist")
	}
	if !m.branchExists("feature/foo") {
		t.Error("expected feature/foo to exist")
	}
	if m.branchExists("nonexistent") {
		t.Error("expected nonexistent to not exist")
	}

	empty := modelWithBranches(nil)
	if empty.branchExists("anything") {
		t.Error("expected empty model to have no branches")
	}
}

// ---------------------------------------------------------------------------
// Error splash
// ---------------------------------------------------------------------------

func TestErrorSplash_DismissedOnAnyKey(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.errSplash = "something went wrong"

	result, _ := m.Update(keyRune('x'))
	got := result.(Model)

	if got.errSplash != "" {
		t.Errorf("expected errSplash to be cleared, got %q", got.errSplash)
	}
	if got.state != stateBrowse {
		t.Errorf("expected stateBrowse, got %d", got.state)
	}
}

// ---------------------------------------------------------------------------
// Browse navigation
// ---------------------------------------------------------------------------

func TestBrowse_CursorNavigation(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.cursor = 0

	// Move down twice.
	r, _ := m.Update(keyRune('j'))
	m = r.(Model)
	if m.cursor != 1 {
		t.Fatalf("expected cursor=1 after j, got %d", m.cursor)
	}

	r, _ = m.Update(keyRune('j'))
	m = r.(Model)
	if m.cursor != 2 {
		t.Fatalf("expected cursor=2 after j, got %d", m.cursor)
	}

	// Can't go below last branch.
	r, _ = m.Update(keyRune('j'))
	m = r.(Model)
	if m.cursor != 2 {
		t.Fatalf("expected cursor to stay at 2, got %d", m.cursor)
	}

	// Move up.
	r, _ = m.Update(keyRune('k'))
	m = r.(Model)
	if m.cursor != 1 {
		t.Fatalf("expected cursor=1 after k, got %d", m.cursor)
	}

	// Arrow keys work too.
	r, _ = m.Update(keyType(tea.KeyUp))
	m = r.(Model)
	if m.cursor != 0 {
		t.Fatalf("expected cursor=0 after up, got %d", m.cursor)
	}

	// Can't go above first branch.
	r, _ = m.Update(keyType(tea.KeyUp))
	m = r.(Model)
	if m.cursor != 0 {
		t.Fatalf("expected cursor to stay at 0, got %d", m.cursor)
	}
}

func TestBrowse_EnterOnCurrentBranch_NoOp(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.cursor = 0 // main, IsCurrent=true

	r, cmd := m.Update(keyType(tea.KeyEnter))
	got := r.(Model)

	if got.state != stateBrowse {
		t.Errorf("expected stateBrowse when pressing enter on current branch, got %d", got.state)
	}
	if cmd != nil {
		t.Errorf("expected nil cmd when pressing enter on current branch")
	}
}

func TestBrowse_EnterOnNonCurrentBranch_StartsCheckout(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.cursor = 1 // feature/foo

	r, cmd := m.Update(keyType(tea.KeyEnter))
	got := r.(Model)

	if got.state != stateProcessing {
		t.Errorf("expected stateProcessing, got %d", got.state)
	}
	if got.processingMsg != "Switching branch..." {
		t.Errorf("expected 'Switching branch...', got %q", got.processingMsg)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for checkout")
	}
}

// ---------------------------------------------------------------------------
// Edit mode
// ---------------------------------------------------------------------------

func TestBrowse_EditModeEntry(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.cursor = 1

	r, _ := m.Update(keyRune('e'))
	got := r.(Model)

	if got.state != stateEdit {
		t.Fatalf("expected stateEdit, got %d", got.state)
	}
	if got.editing.Name != "feature/foo" {
		t.Errorf("expected editing feature/foo, got %q", got.editing.Name)
	}
	if got.input.Value() != "feature/foo" {
		t.Errorf("expected input pre-filled with branch name, got %q", got.input.Value())
	}
}

func TestEdit_EscCancels(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateEdit
	m.editing = testBranches[1]

	r, _ := m.Update(keyType(tea.KeyEsc))
	got := r.(Model)

	if got.state != stateBrowse {
		t.Errorf("expected stateBrowse after esc, got %d", got.state)
	}
}

func TestEdit_EnterWithSameName_Cancels(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateEdit
	m.editing = testBranches[1]
	m.input.SetValue("feature/foo")

	r, _ := m.Update(keyType(tea.KeyEnter))
	got := r.(Model)

	if got.state != stateBrowse {
		t.Errorf("expected cancel when name unchanged, got state %d", got.state)
	}
}

func TestEdit_EnterWithRemote_GoesToConfirm(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateEdit
	m.editing = testBranches[1] // HasRemote=true
	m.input.SetValue("feature/bar")

	r, _ := m.Update(keyType(tea.KeyEnter))
	got := r.(Model)

	if got.state != stateConfirmRemote {
		t.Errorf("expected stateConfirmRemote for branch with remote, got %d", got.state)
	}
}

func TestEdit_EnterWithoutRemote_StartsRename(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateEdit
	m.editing = testBranches[2] // local-only, HasRemote=false
	m.input.SetValue("new-local")

	r, cmd := m.Update(keyType(tea.KeyEnter))
	got := r.(Model)

	if got.state != stateProcessing {
		t.Errorf("expected stateProcessing, got %d", got.state)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for rename")
	}
}

// ---------------------------------------------------------------------------
// Create mode
// ---------------------------------------------------------------------------

func TestBrowse_CreateModeEntry(t *testing.T) {
	m := modelWithBranches(testBranches)

	r, _ := m.Update(keyRune('c'))
	got := r.(Model)

	if got.state != stateCreate {
		t.Fatalf("expected stateCreate, got %d", got.state)
	}
	if got.input.Value() != "" {
		t.Errorf("expected empty input, got %q", got.input.Value())
	}
}

func TestCreate_EnterWithExistingName_Blocked(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateCreate
	m.input.SetValue("main") // already exists

	r, cmd := m.Update(keyType(tea.KeyEnter))
	got := r.(Model)

	if got.state != stateCreate {
		t.Errorf("expected to stay in stateCreate when name exists, got %d", got.state)
	}
	if cmd != nil {
		t.Error("expected nil cmd when branch name exists")
	}
}

func TestCreate_EnterWithEmptyName_Blocked(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateCreate
	m.input.SetValue("")

	r, cmd := m.Update(keyType(tea.KeyEnter))
	got := r.(Model)

	if got.state != stateCreate {
		t.Errorf("expected to stay in stateCreate with empty name, got %d", got.state)
	}
	if cmd != nil {
		t.Error("expected nil cmd with empty name")
	}
}

func TestCreate_EnterWithNewName_StartsCreate(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateCreate
	m.input.SetValue("feature/new")

	r, cmd := m.Update(keyType(tea.KeyEnter))
	got := r.(Model)

	if got.state != stateProcessing {
		t.Errorf("expected stateProcessing, got %d", got.state)
	}
	if got.processingMsg != "Creating branch..." {
		t.Errorf("expected 'Creating branch...', got %q", got.processingMsg)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for create")
	}
}

// ---------------------------------------------------------------------------
// Delete staging (dd)
// ---------------------------------------------------------------------------

func TestDelete_FirstD_StagesBranch(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.cursor = 1 // feature/foo

	r, _ := m.Update(keyRune('d'))
	got := r.(Model)

	if !got.deleteStaged {
		t.Error("expected deleteStaged=true after first d")
	}
	if got.deleteStagedIdx != 1 {
		t.Errorf("expected deleteStagedIdx=1, got %d", got.deleteStagedIdx)
	}
	if got.state != stateBrowse {
		t.Errorf("expected to stay in stateBrowse, got %d", got.state)
	}
}

func TestDelete_SecondD_ExecutesDelete(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.cursor = 1
	m.deleteStaged = true
	m.deleteStagedIdx = 1

	r, cmd := m.Update(keyRune('d'))
	got := r.(Model)

	if got.deleteStaged {
		t.Error("expected deleteStaged=false after second d")
	}
	if got.state != stateProcessing {
		t.Errorf("expected stateProcessing, got %d", got.state)
	}
	if got.processingMsg != "Deleting branch..." {
		t.Errorf("expected 'Deleting branch...', got %q", got.processingMsg)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for delete")
	}
}

func TestDelete_OtherKeyClearsStagng(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.cursor = 1
	m.deleteStaged = true
	m.deleteStagedIdx = 1

	r, _ := m.Update(keyRune('j'))
	got := r.(Model)

	if got.deleteStaged {
		t.Error("expected deleteStaged cleared after non-d key")
	}
}

func TestDelete_OnCurrentBranch_Ignored(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.cursor = 0 // main, IsCurrent=true

	r, _ := m.Update(keyRune('d'))
	got := r.(Model)

	if got.deleteStaged {
		t.Error("expected d to be ignored on current branch")
	}
}

func TestDelete_DifferentCursorClearsAndRestages(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.cursor = 1
	m.deleteStaged = true
	m.deleteStagedIdx = 2 // staged on a different branch

	r, _ := m.Update(keyRune('d'))
	got := r.(Model)

	// Should stage the new cursor position, not execute a delete.
	if !got.deleteStaged {
		t.Error("expected deleteStaged=true")
	}
	if got.deleteStagedIdx != 1 {
		t.Errorf("expected deleteStagedIdx=1, got %d", got.deleteStagedIdx)
	}
	if got.state != stateBrowse {
		t.Errorf("expected stateBrowse, got %d", got.state)
	}
}

// ---------------------------------------------------------------------------
// Confirm remote
// ---------------------------------------------------------------------------

func TestConfirmRemote_Navigation(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateConfirmRemote
	m.confirmIdx = 0

	r, _ := m.Update(keyRune('l'))
	got := r.(Model)
	if got.confirmIdx != 1 {
		t.Errorf("expected confirmIdx=1 after l, got %d", got.confirmIdx)
	}

	r, _ = got.Update(keyRune('h'))
	got = r.(Model)
	if got.confirmIdx != 0 {
		t.Errorf("expected confirmIdx=0 after h, got %d", got.confirmIdx)
	}
}

func TestConfirmRemote_YShortcut(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateConfirmRemote
	m.editing = testBranches[1]
	m.input.SetValue("feature/bar")

	r, cmd := m.Update(keyRune('y'))
	got := r.(Model)

	if got.state != stateProcessing {
		t.Errorf("expected stateProcessing, got %d", got.state)
	}
	if !got.didRemote {
		t.Error("expected didRemote=true")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

func TestConfirmRemote_NShortcut(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateConfirmRemote
	m.editing = testBranches[1]
	m.input.SetValue("feature/bar")

	r, cmd := m.Update(keyRune('n'))
	got := r.(Model)

	if got.state != stateProcessing {
		t.Errorf("expected stateProcessing, got %d", got.state)
	}
	if got.didRemote {
		t.Error("expected didRemote=false")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

// ---------------------------------------------------------------------------
// Async result messages
// ---------------------------------------------------------------------------

func TestBranchesLoadedMsg_Success(t *testing.T) {
	m := New()
	m.state = stateLoading

	r, _ := m.Update(branchesLoadedMsg{branches: testBranches})
	got := r.(Model)

	if got.state != stateBrowse {
		t.Errorf("expected stateBrowse, got %d", got.state)
	}
	if len(got.branches) != 3 {
		t.Errorf("expected 3 branches, got %d", len(got.branches))
	}
}

func TestBranchesLoadedMsg_Error(t *testing.T) {
	m := New()
	m.state = stateLoading

	r, _ := m.Update(branchesLoadedMsg{err: errForTest("git failed")})
	got := r.(Model)

	if got.state != stateBrowse {
		t.Errorf("expected stateBrowse (splash), got %d", got.state)
	}
	if got.errSplash == "" {
		t.Error("expected errSplash to be set")
	}
}

func TestDeleteResultMsg_Success(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateProcessing
	m.editing = testBranches[1]

	r, _ := m.Update(deleteResultMsg{err: nil})
	got := r.(Model)

	if got.state != stateResult {
		t.Errorf("expected stateResult, got %d", got.state)
	}
}

func TestDeleteResultMsg_Error(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateProcessing

	r, _ := m.Update(deleteResultMsg{err: errForTest("cannot delete")})
	got := r.(Model)

	if got.state != stateBrowse {
		t.Errorf("expected stateBrowse (splash), got %d", got.state)
	}
	if got.errSplash != "cannot delete" {
		t.Errorf("expected errSplash 'cannot delete', got %q", got.errSplash)
	}
}

func TestCheckoutResultMsg_Success_ReloadsBranches(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateProcessing

	r, cmd := m.Update(checkoutResultMsg{err: nil})
	got := r.(Model)

	if got.state != stateLoading {
		t.Errorf("expected stateLoading for branch reload, got %d", got.state)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (fetch + tick)")
	}
}

func TestCheckoutResultMsg_Error(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateProcessing

	r, _ := m.Update(checkoutResultMsg{err: errForTest("dirty worktree")})
	got := r.(Model)

	if got.state != stateBrowse {
		t.Errorf("expected stateBrowse (splash), got %d", got.state)
	}
	if got.errSplash != "dirty worktree" {
		t.Errorf("expected errSplash 'dirty worktree', got %q", got.errSplash)
	}
}

func TestRenameResultMsg_PartialSuccess(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateProcessing
	m.editing = testBranches[1] // feature/foo with origin/feature/foo upstream
	m.input.SetValue("feature/bar")
	m.didRemote = true

	r, _ := m.Update(renameResultMsg{localOk: true, err: errForTest("push failed")})
	got := r.(Model)

	if got.state != stateResult {
		t.Errorf("expected stateResult for partial success, got %d", got.state)
	}
	if got.errSplash != "" {
		t.Error("expected no errSplash for partial success")
	}
	if got.result == "" {
		t.Error("expected result message to be set")
	}
}

func TestRenameResultMsg_FullFailure(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateProcessing
	m.editing = testBranches[1]

	r, _ := m.Update(renameResultMsg{localOk: false, err: errForTest("rename failed")})
	got := r.(Model)

	if got.state != stateBrowse {
		t.Errorf("expected stateBrowse (splash), got %d", got.state)
	}
	if got.errSplash != "rename failed" {
		t.Errorf("expected errSplash 'rename failed', got %q", got.errSplash)
	}
}

func TestRenameResultMsg_FullSuccess(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateProcessing
	m.editing = testBranches[1]
	m.input.SetValue("feature/bar")
	m.didRemote = true

	r, _ := m.Update(renameResultMsg{localOk: true, remoteOk: true, err: nil})
	got := r.(Model)

	if got.state != stateResult {
		t.Errorf("expected stateResult, got %d", got.state)
	}
	if got.result == "" {
		t.Error("expected result message to be set")
	}
}

func TestCreateResultMsg_Success(t *testing.T) {
	m := modelWithBranches(testBranches)
	m.state = stateProcessing
	m.input.SetValue("feature/new")

	r, _ := m.Update(createResultMsg{err: nil})
	got := r.(Model)

	if got.state != stateResult {
		t.Errorf("expected stateResult, got %d", got.state)
	}
}

// ---------------------------------------------------------------------------
// startAsync
// ---------------------------------------------------------------------------

func TestStartAsync_SetsState(t *testing.T) {
	m := modelWithBranches(testBranches)

	got, cmd := startAsync(m, stateProcessing, "Doing things...", nil)

	if got.state != stateProcessing {
		t.Errorf("expected stateProcessing, got %d", got.state)
	}
	if got.processingMsg != "Doing things..." {
		t.Errorf("expected processingMsg 'Doing things...', got %q", got.processingMsg)
	}
	if cmd == nil {
		t.Error("expected non-nil batched cmd")
	}
}

// ---------------------------------------------------------------------------
// showError
// ---------------------------------------------------------------------------

func TestShowError_SetsSplash(t *testing.T) {
	m := New()
	m.state = stateProcessing

	got := showError(m, errForTest("oh no"))

	if got.state != stateBrowse {
		t.Errorf("expected stateBrowse, got %d", got.state)
	}
	if got.errSplash != "oh no" {
		t.Errorf("expected errSplash 'oh no', got %q", got.errSplash)
	}
}

// ---------------------------------------------------------------------------
// Quit behaviour
// ---------------------------------------------------------------------------

func TestBrowse_CtrlC_Quits(t *testing.T) {
	m := modelWithBranches(testBranches)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	// tea.Quit is a function; the only way to check is that cmd is non-nil.
	if cmd == nil {
		t.Error("expected quit cmd on ctrl+c")
	}
}

func TestBrowse_Q_SendsBackMsg(t *testing.T) {
	m := modelWithBranches(testBranches)

	_, cmd := m.Update(keyRune('q'))

	if cmd == nil {
		t.Fatal("expected non-nil cmd on q")
	}
	msg := cmd()
	if _, ok := msg.(messages.BackMsg); !ok {
		t.Errorf("expected BackMsg, got %T", msg)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type errForTest string

func (e errForTest) Error() string { return string(e) }
