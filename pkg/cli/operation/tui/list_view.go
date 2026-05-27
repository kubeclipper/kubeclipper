package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

// currentView tracks which sub-view is active.
type currentView int

const (
	viewList currentView = iota
	viewLog
)

// ListModel renders the operation selection list as a table.
type ListModel struct {
	operations []v1.Operation
	cursor     int
	width      int
	height     int
}

// NewListModel creates a list model with the given operations.
func NewListModel(operations []v1.Operation, width, height int) ListModel {
	return ListModel{
		operations: operations,
		cursor:     0,
		width:      width,
		height:     height,
	}
}

// selectOpMsg is emitted when the user selects an operation.
type selectOpMsg struct {
	op v1.Operation
}

// Update handles key events for the list view.
func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case DefaultKeyMap.Up, "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case DefaultKeyMap.Down, "j":
			if m.cursor < len(m.operations)-1 {
				m.cursor++
			}
		case DefaultKeyMap.Enter:
			return m, func() tea.Msg {
				return selectOpMsg{op: m.operations[m.cursor]}
			}
		case DefaultKeyMap.Quit, "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the operation list table.
func (m ListModel) View() string {
	if len(m.operations) == 0 {
		return "No operations found."
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf("%-30s %-14s %-22s %-6s", "NAME", "STATUS", "CREATED", "STEPS")
	b.WriteString(HeaderStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	// Rows
	for i, op := range m.operations {
		status := string(op.Status.Status)
		name := op.Name
		if len(name) > 28 {
			name = name[:28] + ".."
		}

		action := op.Labels["kubeclipper.io/operation"]
		if action != "" {
			name = action
		}

		created := op.CreationTimestamp.Format("2006-01-02 15:04:05")
		steps := fmt.Sprintf("%d", len(op.Steps))

		statusStyled := statusStyle(status).Render(status)
		row := fmt.Sprintf("%-30s %-14s %-22s %-6s", name, statusStyled, created, steps)

		if i == m.cursor {
			row = SelectedStyle.Render(row)
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	// Help bar
	b.WriteString("\n")
	helpText := "↑/k: up  ↓/j: down  enter: select  q: quit"
	b.WriteString(HelpStyle.Render(helpText))

	return b.String()
}

// statusStyle returns the lipgloss style for a given status string.
func statusStyle(status string) lipgloss.Style {
	switch status {
	case string(v1.OperationStatusSuccessful):
		return StatusSuccessful
	case string(v1.OperationStatusFailed):
		return StatusFailed
	case string(v1.OperationStatusRunning):
		return StatusRunning
	default:
		return StatusPending
	}
}
