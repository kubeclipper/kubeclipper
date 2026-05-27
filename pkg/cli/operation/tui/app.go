package tui

import (
	"context"
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

// Model is the top-level Bubble Tea model for the TUI log viewer.
type Model struct {
	client      *kc.Client
	clusterName string
	operations  []v1.Operation
	currentView currentView
	listModel   ListModel
	logModel    LogModel
	width       int
	height      int
	ready       bool
}

// NewModel creates the main TUI model with the given operations.
func NewModel(client *kc.Client, clusterName string, operations []v1.Operation) Model {
	return Model{
		client:      client,
		clusterName: clusterName,
		operations:  operations,
		currentView: viewList,
		ready:       false,
	}
}

// NewModelWithSingleOp creates a model that skips the list and goes directly
// to the log view for a single operation.
func NewModelWithSingleOp(client *kc.Client, clusterName string, op v1.Operation) Model {
	return Model{
		client:      client,
		clusterName: clusterName,
		operations:  []v1.Operation{op},
		currentView: viewLog,
		logModel:    NewLogModel(client, &op, 80, 24), // will be resized by WindowSizeMsg
		ready:       false,
	}
}

// Init initializes the TUI model. When starting directly in log view,
// it returns the log model's init command to fetch the initial log.
func (m Model) Init() tea.Cmd {
	if m.currentView == viewLog {
		return m.logModel.Init()
	}
	return nil
}

// Update delegates to the active sub-view and handles view switching.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.listModel = NewListModel(m.operations, m.width, m.height)
		if m.currentView == viewLog {
			m.logModel = NewLogModel(m.client, m.logModel.operation, m.width, m.height)
		}

	case selectOpMsg:
		m.currentView = viewLog
		m.logModel = NewLogModel(m.client, &msg.op, m.width, m.height)
		return m, m.logModel.Init()

	case backMsg:
		m.currentView = viewList
		m.listModel = NewListModel(m.operations, m.width, m.height)
		return m, nil
	}

	switch m.currentView {
	case viewList:
		var cmd tea.Cmd
		m.listModel, cmd = m.listModel.Update(msg)
		return m, cmd
	case viewLog:
		var cmd tea.Cmd
		m.logModel, cmd = m.logModel.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

// View delegates rendering to the active sub-view.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}
	switch m.currentView {
	case viewList:
		return m.listModel.View()
	case viewLog:
		return m.logModel.View()
	default:
		return ""
	}
}

// RunTUI launches the TUI program. It handles the logic of whether to show
// the list view or skip directly to the log view.
// The in and out parameters allow the caller to control the TUI input/output
// (e.g., os.Stdin/os.Stdout), enabling testability.
func RunTUI(client *kc.Client, clusterName string, in io.Reader, out io.Writer) error {
	ctx := context.Background()

	labelSelector := fmt.Sprintf("kubeclipper.io/cluster=%s", clusterName)
	list, err := client.ListOperation(ctx, kc.Queries{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list operations for cluster %s: %w", clusterName, err)
	}

	operations := list.Items
	if len(operations) == 0 {
		return fmt.Errorf("no operations found for cluster %s", clusterName)
	}

	var model Model
	if len(operations) == 1 {
		model = NewModelWithSingleOp(client, clusterName, operations[0])
	} else {
		model = NewModel(client, clusterName, operations)
	}

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithInput(in), tea.WithOutput(out))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
