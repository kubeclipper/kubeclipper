package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

// StepEntry holds enriched step data for display in the step panel.
type StepEntry struct {
	Name     string
	ID       string
	Status   string
	Duration string
	Nodes    []NodeEntry
}

// NodeEntry holds enriched node data for display.
type NodeEntry struct {
	Hostname string
	ID       string
	IP       string
	Status   string
	Duration string
}

// tickMsg is sent periodically when follow mode is active.
type tickMsg time.Time

// logFetchedMsg carries newly fetched log content.
type logFetchedMsg struct {
	content string
	offset  int64
}

// operationStatusMsg carries the refreshed operation from a periodic poll.
type operationStatusMsg struct {
	operation *v1.Operation
	err       error
}

// LogModel renders the split-panel log view.
type LogModel struct {
	client     *kc.Client
	operation  *v1.Operation
	steps      []StepEntry
	cursor     int
	followMode bool
	lastOffset map[string]int64
	viewport   viewport.Model
	rawContent string // tracks full log content for correct appending
	width      int
	height     int
}

// followTickInterval controls how often follow mode polls for new logs.
const followTickInterval = 2 * time.Second

// NewLogModel creates a log model for the given operation.
func NewLogModel(client *kc.Client, op *v1.Operation, width, height int) LogModel {
	steps := buildStepEntries(op)

	stepPanelWidth := width * 35 / 100
	logPanelWidth := width - stepPanelWidth - 2
	if logPanelWidth < 10 {
		logPanelWidth = 10
	}

	vp := viewport.New(logPanelWidth, height-3)

	return LogModel{
		client:     client,
		operation:  op,
		steps:      steps,
		cursor:     0,
		followMode: false,
		lastOffset: make(map[string]int64),
		viewport:   vp,
		rawContent: "",
		width:      width,
		height:     height,
	}
}

// buildStepEntries constructs the enriched step list from an operation.
func buildStepEntries(op *v1.Operation) []StepEntry {
	var entries []StepEntry
	for _, step := range op.Steps {
		stepStatus := getStepStatus(op, step.ID)
		startTime := getStepStartTime(op, step.ID)

		var durationStr string
		if !startTime.IsZero() {
			endAt := getStepEndTime(op, step.ID)
			if !endAt.IsZero() {
				d := endAt.Time.Sub(startTime.Time).Round(time.Second)
				if d < time.Second {
					d = time.Second
				}
				durationStr = d.String()
			}
		}

		var nodes []NodeEntry
		for _, node := range step.Nodes {
			nodeStatus, startAt, endAt := getNodeStatusWithTime(op, step.ID, node.ID)
			var nodeDuration string
			if !startAt.IsZero() && !endAt.IsZero() {
				d := endAt.Time.Sub(startAt.Time).Round(time.Second)
				if d < time.Second {
					d = time.Second
				}
				nodeDuration = d.String()
			}
			nodes = append(nodes, NodeEntry{
				Hostname: node.Hostname,
				ID:       node.ID,
				IP:       node.IPv4,
				Status:   nodeStatus,
				Duration: nodeDuration,
			})
		}

		entries = append(entries, StepEntry{
			Name:     step.Name,
			ID:       step.ID,
			Status:   stepStatus,
			Duration: durationStr,
			Nodes:    nodes,
		})
	}
	return entries
}

// getStepEndTime returns the latest end time from all nodes in a step.
func getStepEndTime(op *v1.Operation, stepID string) metav1.Time {
	var endTime metav1.Time
	for _, cond := range op.Status.Conditions {
		if cond.StepID == stepID {
			for _, st := range cond.Status {
				if endTime.IsZero() || st.EndAt.After(endTime.Time) {
					endTime = st.EndAt
				}
			}
		}
	}
	return endTime
}

// getStepStatus returns the overall status for a step.
func getStepStatus(op *v1.Operation, stepID string) string {
	for _, cond := range op.Status.Conditions {
		if cond.StepID == stepID {
			for _, st := range cond.Status {
				if st.Status == v1.StepStatusFailed {
					return string(v1.StepStatusFailed)
				}
			}
			for _, st := range cond.Status {
				if st.Status == "" {
					return "pending"
				}
			}
			return string(v1.StepStatusSuccessful)
		}
	}
	return "pending"
}

// getNodeStatusWithTime returns status and time info for a node in a step.
func getNodeStatusWithTime(op *v1.Operation, stepID, nodeID string) (status string, startAt metav1.Time, endAt metav1.Time) {
	for _, cond := range op.Status.Conditions {
		if cond.StepID == stepID {
			for _, st := range cond.Status {
				if st.Node == nodeID {
					return string(st.Status), st.StartAt, st.EndAt
				}
			}
		}
	}
	return "unknown", metav1.Time{}, metav1.Time{}
}

// getStepStartTime returns the earliest start time from all nodes in a step.
func getStepStartTime(op *v1.Operation, stepID string) metav1.Time {
	var startTime metav1.Time
	for _, cond := range op.Status.Conditions {
		if cond.StepID == stepID {
			for _, st := range cond.Status {
				if startTime.IsZero() || st.StartAt.Before(&startTime) {
					startTime = st.StartAt
				}
			}
		}
	}
	return startTime
}

// fetchInitialLogCmd returns a command that fetches the initial log for the current step's first node.
func (m LogModel) fetchInitialLogCmd() tea.Cmd {
	if len(m.steps) == 0 || len(m.steps[0].Nodes) == 0 {
		return nil
	}
	step := m.steps[m.cursor]
	node := step.Nodes[0]
	key := step.ID + "|" + node.ID
	offset := m.lastOffset[key]

	return m.fetchLogCmd(step.ID, node.ID, offset, key)
}

// fetchLogCmd returns a command that fetches logs for a given step/node.
func (m LogModel) fetchLogCmd(stepID, nodeID string, offset int64, key string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		stepLog, err := m.client.GetStepNodeLog(ctx, m.operation.Name, stepID, nodeID, offset)
		if err != nil {
			return logFetchedMsg{content: fmt.Sprintf("Error fetching log: %v\n", err), offset: offset}
		}
		return logFetchedMsg{content: stepLog.Content, offset: offset + int64(len(stepLog.Content))}
	}
}

// followTickCmd returns a command that sends a tick after the follow interval.
func followTickCmd() tea.Cmd {
	return tea.Tick(followTickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// fetchOperationStatusCmd returns a command that re-fetches the operation to update
// its status (needed for follow mode to detect completion).
func (m LogModel) fetchOperationStatusCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		op, err := m.client.DescribeOperation(ctx, m.operation.Name)
		return operationStatusMsg{operation: op, err: err}
	}
}

// Init initializes the log model by fetching the first log.
func (m LogModel) Init() tea.Cmd {
	return m.fetchInitialLogCmd()
}

// Update handles messages for the log view.
func (m LogModel) Update(msg tea.Msg) (LogModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		stepPanelWidth := m.width * 35 / 100
		logPanelWidth := m.width - stepPanelWidth - 2
		if logPanelWidth < 10 {
			logPanelWidth = 10
		}
		m.viewport.Width = logPanelWidth
		m.viewport.Height = m.height - 3

	case logFetchedMsg:
		if msg.content != "" {
			m.rawContent += msg.content
			m.viewport.SetContent(m.rawContent)
			if len(m.steps) > 0 && m.cursor < len(m.steps) && len(m.steps[m.cursor].Nodes) > 0 {
				step := m.steps[m.cursor]
				node := step.Nodes[0]
				key := step.ID + "|" + node.ID
				m.lastOffset[key] = msg.offset
			}
			if m.followMode {
				m.viewport.GotoBottom()
			}
		}

	case tickMsg:
		if m.followMode {
			if len(m.steps) > 0 && m.cursor < len(m.steps) && len(m.steps[m.cursor].Nodes) > 0 {
				step := m.steps[m.cursor]
				node := step.Nodes[0]
				key := step.ID + "|" + node.ID
				offset := m.lastOffset[key]
				cmds = append(cmds, m.fetchLogCmd(step.ID, node.ID, offset, key))
			}
			cmds = append(cmds, followTickCmd())
			cmds = append(cmds, m.fetchOperationStatusCmd())
		}

	case operationStatusMsg:
		if msg.err == nil && msg.operation != nil {
			m.operation = msg.operation
			m.steps = buildStepEntries(msg.operation)
			if msg.operation.Status.Status == v1.OperationStatusSuccessful ||
				msg.operation.Status.Status == v1.OperationStatusFailed ||
				msg.operation.Status.Status == v1.OperationStatusTermination {
				m.followMode = false
				m.rawContent += "\n--- Operation completed, follow mode disabled ---\n"
				m.viewport.SetContent(m.rawContent)
				m.viewport.GotoBottom()
				return m, nil
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case DefaultKeyMap.Up, "k":
			if m.cursor > 0 {
				m.cursor--
				m.rawContent = ""
				m.viewport.SetContent("")
				m.lastOffset = make(map[string]int64)
				cmds = append(cmds, m.fetchCurrentStepLogCmd())
			}
		case DefaultKeyMap.Down, "j":
			if m.cursor < len(m.steps)-1 {
				m.cursor++
				m.rawContent = ""
				m.viewport.SetContent("")
				m.lastOffset = make(map[string]int64)
				cmds = append(cmds, m.fetchCurrentStepLogCmd())
			}
		case DefaultKeyMap.PageUp:
			m.viewport.HalfPageUp()
		case DefaultKeyMap.PageDown:
			m.viewport.HalfPageDown()
		case DefaultKeyMap.Follow:
			m.followMode = !m.followMode
			if m.followMode {
				m.viewport.GotoBottom()
				cmds = append(cmds, followTickCmd())
			}
		case DefaultKeyMap.Back, "esc":
			return m, func() tea.Msg { return backMsg{} }
		case DefaultKeyMap.Quit, "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// fetchCurrentStepLogCmd fetches log for the currently selected step.
func (m LogModel) fetchCurrentStepLogCmd() tea.Cmd {
	if m.cursor >= len(m.steps) {
		return nil
	}
	step := m.steps[m.cursor]
	if len(step.Nodes) == 0 {
		return func() tea.Msg { return logFetchedMsg{content: "(no nodes for this step)\n", offset: 0} }
	}
	node := step.Nodes[0]
	key := step.ID + "|" + node.ID
	offset := m.lastOffset[key]
	return m.fetchLogCmd(step.ID, node.ID, offset, key)
}

// backMsg signals the user wants to go back to the list view.
type backMsg struct{}

// View renders the split-panel log view.
func (m LogModel) View() string {
	if len(m.steps) == 0 {
		return "No steps in this operation."
	}

	stepPanelWidth := m.width * 35 / 100
	logPanelWidth := m.width - stepPanelWidth - 2
	if logPanelWidth < 10 {
		logPanelWidth = 10
	}

	var leftBuilder strings.Builder
	leftBuilder.WriteString(HeaderStyle.Render("Steps"))
	leftBuilder.WriteString("\n")

	for i, step := range m.steps {
		mark := stepStatusMark(step.Status)
		label := step.Name
		if label == "" {
			label = step.ID
		}
		durText := ""
		if step.Duration != "" {
			durText = " " + step.Duration
		}
		line := fmt.Sprintf(" %s %s%s", mark, label, durText)

		var nodeLines []string
		for _, node := range step.Nodes {
			nodeMark := stepStatusMark(node.Status)
			nodeLine := fmt.Sprintf("   %s %s (%s)", nodeMark, node.Hostname, node.IP)
			if node.Duration != "" {
				nodeLine += fmt.Sprintf(" [%s]", node.Duration)
			}
			nodeLines = append(nodeLines, nodeLine)
		}

		if i == m.cursor {
			line = SelectedStyle.Render(line)
			for idx, nl := range nodeLines {
				nodeLines[idx] = SelectedStyle.Render(nl)
			}
		}

		leftBuilder.WriteString(line)
		leftBuilder.WriteString("\n")
		for _, nl := range nodeLines {
			leftBuilder.WriteString(nl)
			leftBuilder.WriteString("\n")
		}
	}

	leftPanel := StepPanelWidth(leftBuilder.String(), stepPanelWidth)
	rightPanel := LogPanelStyle.Width(logPanelWidth).Render(m.viewport.View())
	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	followIndicator := "off"
	if m.followMode {
		followIndicator = "on"
	}
	helpText := fmt.Sprintf("up/k: up  down/j: down  pgup/pgdn: scroll  f: follow[%s]  b: back  q: quit", followIndicator)
	helpBar := HelpStyle.Render(helpText)

	return combined + "\n" + helpBar
}

// stepStatusMark returns the indicator character for a step status.
func stepStatusMark(status string) string {
	switch status {
	case string(v1.OperationStatusSuccessful):
		return StepSuccessMark
	case string(v1.OperationStatusFailed):
		return StepFailedMark
	case string(v1.OperationStatusRunning):
		return StepRunningMark
	default:
		return StepPendingMark
	}
}

// StepPanelWidth applies width constraint to the step panel content.
func StepPanelWidth(content string, width int) string {
	return StepPanelStyle.Width(width).Render(content)
}
