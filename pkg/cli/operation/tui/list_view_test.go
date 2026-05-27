package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func makeTestOperations() []v1.Operation {
	return []v1.Operation{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "op-001",
				CreationTimestamp: metav1.Now(),
				Labels: map[string]string{
					"kubeclipper.io/operation": "CreateCluster",
				},
			},
			Steps: []v1.Step{
				{ID: "s1", Name: "step1"},
			},
			Status: v1.OperationStatus{
				Status: v1.OperationStatusSuccessful,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "op-002",
				CreationTimestamp: metav1.Now(),
				Labels: map[string]string{
					"kubeclipper.io/operation": "DeleteCluster",
				},
			},
			Steps: []v1.Step{
				{ID: "s1", Name: "step1"},
				{ID: "s2", Name: "step2"},
			},
			Status: v1.OperationStatus{
				Status: v1.OperationStatusFailed,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "op-003",
				CreationTimestamp: metav1.Now(),
				Labels: map[string]string{
					"kubeclipper.io/operation": "UpgradeCluster",
				},
			},
			Steps: []v1.Step{
				{ID: "s1", Name: "step1"},
			},
			Status: v1.OperationStatus{
				Status: v1.OperationStatusRunning,
			},
		},
	}
}

// keyMsg creates a tea.KeyMsg with the given string representation.
// For special keys like "up", "down", "enter", it uses the appropriate KeyType.
// For rune keys like "j", "k", "q", it uses KeyRunes.
func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "pgup":
		return tea.KeyMsg{Type: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestListModelCursorMovement(t *testing.T) {
	ops := makeTestOperations()
	m := NewListModel(ops, 80, 24)

	// Initial cursor at 0
	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}

	// Move down
	m, _ = m.Update(keyMsg(DefaultKeyMap.Down))
	if m.cursor != 1 {
		t.Errorf("after down: cursor = %d, want 1", m.cursor)
	}

	// Move down again
	m, _ = m.Update(keyMsg(DefaultKeyMap.Down))
	if m.cursor != 2 {
		t.Errorf("after second down: cursor = %d, want 2", m.cursor)
	}

	// Try to move down past last item -- should stay
	m, _ = m.Update(keyMsg(DefaultKeyMap.Down))
	if m.cursor != 2 {
		t.Errorf("after down at bottom: cursor = %d, want 2 (stayed)", m.cursor)
	}

	// Move up
	m, _ = m.Update(keyMsg(DefaultKeyMap.Up))
	if m.cursor != 1 {
		t.Errorf("after up: cursor = %d, want 1", m.cursor)
	}

	// Move up again
	m, _ = m.Update(keyMsg(DefaultKeyMap.Up))
	if m.cursor != 0 {
		t.Errorf("after second up: cursor = %d, want 0", m.cursor)
	}

	// Try to move up past first item -- should stay
	m, _ = m.Update(keyMsg(DefaultKeyMap.Up))
	if m.cursor != 0 {
		t.Errorf("after up at top: cursor = %d, want 0 (stayed)", m.cursor)
	}
}

func TestListModelCursorMovementJAndK(t *testing.T) {
	ops := makeTestOperations()
	m := NewListModel(ops, 80, 24)

	// j = down
	m, _ = m.Update(keyMsg("j"))
	if m.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", m.cursor)
	}

	// k = up
	m, _ = m.Update(keyMsg("k"))
	if m.cursor != 0 {
		t.Errorf("after k: cursor = %d, want 0", m.cursor)
	}
}

func TestListModelSelectOperation(t *testing.T) {
	ops := makeTestOperations()
	m := NewListModel(ops, 80, 24)

	// Move to second item
	m, _ = m.Update(keyMsg(DefaultKeyMap.Down))
	if m.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", m.cursor)
	}

	// Press enter
	_, cmd := m.Update(keyMsg(DefaultKeyMap.Enter))
	if cmd == nil {
		t.Fatal("expected a command from enter key")
	}

	// Execute the command and verify the message type
	msg := cmd()
	sel, ok := msg.(selectOpMsg)
	if !ok {
		t.Fatalf("expected selectOpMsg, got %T", msg)
	}
	if sel.op.Name != "op-002" {
		t.Errorf("selected operation name = %q, want %q", sel.op.Name, "op-002")
	}
}

func TestListModelEmptyOperations(t *testing.T) {
	m := NewListModel(nil, 80, 24)
	view := m.View()
	if view != "No operations found." {
		t.Errorf("empty list view = %q, want %q", view, "No operations found.")
	}
}

func TestListModelQuit(t *testing.T) {
	ops := makeTestOperations()
	m := NewListModel(ops, 80, 24)

	_, cmd := m.Update(keyMsg(DefaultKeyMap.Quit))
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}
