package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	operationsv1alpha1 "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

type StepEntry struct {
	ID     string
	Status string
	Tasks  []TaskEntry
}

type TaskEntry struct {
	Name     string
	Node     string
	Attempt  int32
	Status   string
	Duration string
}

type tickMsg time.Time

type logFetchedMsg struct {
	content string
	offset  int64
	key     string
}

type operationStatusMsg struct {
	operation *operationsv1alpha1.Operation
	tasks     []operationsv1alpha1.OperationTask
	err       error
}

type LogModel struct {
	client     *kc.Client
	operation  *operationsv1alpha1.Operation
	tasks      []operationsv1alpha1.OperationTask
	steps      []StepEntry
	cursor     int
	followMode bool
	lastOffset map[string]int64
	viewport   viewport.Model
	rawContent string
	width      int
	height     int
}

const followTickInterval = 2 * time.Second

func NewLogModel(client *kc.Client, op *operationsv1alpha1.Operation, width, height int) LogModel {
	stepPanelWidth := width * 35 / 100
	logPanelWidth := width - stepPanelWidth - 2
	if logPanelWidth < 10 {
		logPanelWidth = 10
	}
	return LogModel{
		client: client, operation: op, steps: buildStepEntries(op, nil),
		lastOffset: make(map[string]int64), viewport: viewport.New(logPanelWidth, height-3),
		width: width, height: height,
	}
}

func buildStepEntries(op *operationsv1alpha1.Operation, tasks []operationsv1alpha1.OperationTask) []StepEntry {
	byStep := make(map[string][]operationsv1alpha1.OperationTask)
	for i := range tasks {
		byStep[tasks[i].Spec.StepID] = append(byStep[tasks[i].Spec.StepID], tasks[i])
	}
	entries := make([]StepEntry, 0, len(op.Spec.Steps))
	for _, step := range op.Spec.Steps {
		stepTasks := byStep[step.ID]
		sort.SliceStable(stepTasks, func(i, j int) bool {
			if stepTasks[i].Spec.RetryGeneration != stepTasks[j].Spec.RetryGeneration {
				return stepTasks[i].Spec.RetryGeneration < stepTasks[j].Spec.RetryGeneration
			}
			if stepTasks[i].Spec.Attempt != stepTasks[j].Spec.Attempt {
				return stepTasks[i].Spec.Attempt < stepTasks[j].Spec.Attempt
			}
			return stepTasks[i].Spec.NodeRef.Name < stepTasks[j].Spec.NodeRef.Name
		})
		entry := StepEntry{ID: step.ID, Status: string(operationsv1alpha1.TaskPending)}
		for i := range stepTasks {
			task := &stepTasks[i]
			duration := ""
			if task.Status.StartedAt != nil && task.Status.FinishedAt != nil {
				d := task.Status.FinishedAt.Sub(task.Status.StartedAt.Time).Round(time.Second)
				if d < time.Second {
					d = time.Second
				}
				duration = d.String()
			}
			entry.Tasks = append(entry.Tasks, TaskEntry{Name: task.Name, Node: task.Spec.NodeRef.Name, Attempt: task.Spec.Attempt, Status: string(task.Status.Phase), Duration: duration})
		}
		entry.Status = aggregateStepPhase(step, stepTasks, op.Status.Phase)
		entries = append(entries, entry)
	}
	return entries
}

func aggregateStepPhase(step operationsv1alpha1.OperationStep, tasks []operationsv1alpha1.OperationTask, operationPhase operationsv1alpha1.OperationPhase) string {
	effective := make(map[string]*operationsv1alpha1.OperationTask)
	for i := range tasks {
		task := &tasks[i]
		key := string(task.Spec.NodeRef.UID)
		current := effective[key]
		if current == nil || task.Status.Phase == operationsv1alpha1.TaskSucceeded ||
			(current.Status.Phase != operationsv1alpha1.TaskSucceeded && (task.Spec.RetryGeneration > current.Spec.RetryGeneration ||
				task.Spec.RetryGeneration == current.Spec.RetryGeneration && task.Spec.Attempt > current.Spec.Attempt)) {
			effective[key] = task
		}
	}
	if len(step.Targets) == 0 {
		return missingStepPhase(operationPhase)
	}
	allSucceeded := true
	phase := operationsv1alpha1.TaskPending
	for _, target := range step.Targets {
		task := effective[string(target.UID)]
		if task == nil {
			if operationPhase == operationsv1alpha1.OperationCancelled {
				return string(operationsv1alpha1.TaskCancelled)
			}
			allSucceeded = false
			continue
		}
		switch task.Status.Phase {
		case operationsv1alpha1.TaskFailed, operationsv1alpha1.TaskTimedOut, operationsv1alpha1.TaskCancelled:
			return string(task.Status.Phase)
		case operationsv1alpha1.TaskRunning:
			allSucceeded = false
			phase = operationsv1alpha1.TaskRunning
		case operationsv1alpha1.TaskSucceeded:
		default:
			allSucceeded = false
		}
	}
	if allSucceeded {
		return string(operationsv1alpha1.TaskSucceeded)
	}
	return string(phase)
}

func missingStepPhase(operationPhase operationsv1alpha1.OperationPhase) string {
	if operationPhase == operationsv1alpha1.OperationCancelled {
		return string(operationsv1alpha1.TaskCancelled)
	}
	return string(operationsv1alpha1.TaskPending)
}

func (m LogModel) currentTask() *TaskEntry {
	if m.cursor >= len(m.steps) || len(m.steps[m.cursor].Tasks) == 0 {
		return nil
	}
	return &m.steps[m.cursor].Tasks[len(m.steps[m.cursor].Tasks)-1]
}

func (m LogModel) fetchCurrentLogCmd() tea.Cmd {
	task := m.currentTask()
	if task == nil {
		return func() tea.Msg { return logFetchedMsg{content: "(no Task has been created for this step)\n"} }
	}
	offset := m.lastOffset[task.Name]
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		log, err := m.client.GetOperationTaskLog(ctx, task.Name, offset)
		if err != nil {
			return logFetchedMsg{content: fmt.Sprintf("Error fetching log: %v\n", err), offset: offset, key: task.Name}
		}
		return logFetchedMsg{content: log.Content, offset: offset + log.DeliverySize, key: task.Name}
	}
}

func followTickCmd() tea.Cmd {
	return tea.Tick(followTickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m LogModel) fetchOperationStatusCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		op, err := m.client.DescribeOperation(ctx, m.operation.Name)
		if err != nil {
			return operationStatusMsg{err: err}
		}
		tasks, err := m.client.ListOperationTasks(ctx, string(op.UID))
		if err != nil {
			return operationStatusMsg{err: err}
		}
		return operationStatusMsg{operation: op, tasks: tasks.Items}
	}
}

func (m LogModel) Init() tea.Cmd { return m.fetchOperationStatusCmd() }

func (m LogModel) Update(msg tea.Msg) (LogModel, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.viewport.Width = maxInt(10, m.width-m.width*35/100-2)
		m.viewport.Height = m.height - 3
	case logFetchedMsg:
		if msg.content != "" {
			m.rawContent += msg.content
			m.viewport.SetContent(m.rawContent)
			if msg.key != "" {
				m.lastOffset[msg.key] = msg.offset
			}
			if m.followMode {
				m.viewport.GotoBottom()
			}
		}
	case tickMsg:
		if m.followMode {
			cmds = append(cmds, m.fetchCurrentLogCmd(), m.fetchOperationStatusCmd(), followTickCmd())
		}
	case operationStatusMsg:
		if msg.err == nil && msg.operation != nil {
			m.operation, m.tasks = msg.operation, msg.tasks
			m.steps = buildStepEntries(msg.operation, msg.tasks)
			if m.cursor >= len(m.steps) {
				m.cursor = maxInt(0, len(m.steps)-1)
			}
			if len(m.rawContent) == 0 {
				cmds = append(cmds, m.fetchCurrentLogCmd())
			}
			if msg.operation.Status.Phase.IsTerminal() {
				m.followMode = false
			}
		}
	case tea.KeyMsg:
		switch msg.String() {
		case DefaultKeyMap.Up, "k":
			if m.cursor > 0 {
				m.cursor--
				m.resetLog()
				cmds = append(cmds, m.fetchCurrentLogCmd())
			}
		case DefaultKeyMap.Down, "j":
			if m.cursor < len(m.steps)-1 {
				m.cursor++
				m.resetLog()
				cmds = append(cmds, m.fetchCurrentLogCmd())
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

func (m *LogModel) resetLog() { m.rawContent = ""; m.viewport.SetContent("") }

type backMsg struct{}

func (m LogModel) View() string {
	if len(m.steps) == 0 {
		return "No steps in this operation."
	}
	stepPanelWidth := m.width * 35 / 100
	logPanelWidth := maxInt(10, m.width-stepPanelWidth-2)
	var left strings.Builder
	left.WriteString(HeaderStyle.Render("Steps and Tasks"))
	left.WriteString("\n")
	for i, step := range m.steps {
		line := fmt.Sprintf(" %s %s", stepStatusMark(step.Status), step.ID)
		if i == m.cursor {
			line = SelectedStyle.Render(line)
		}
		left.WriteString(line)
		left.WriteString("\n")
		for _, task := range step.Tasks {
			taskLine := fmt.Sprintf("   %s %s attempt=%d", stepStatusMark(task.Status), task.Node, task.Attempt)
			if task.Duration != "" {
				taskLine += " [" + task.Duration + "]"
			}
			if i == m.cursor {
				taskLine = SelectedStyle.Render(taskLine)
			}
			left.WriteString(taskLine)
			left.WriteString("\n")
		}
	}
	combined := lipgloss.JoinHorizontal(lipgloss.Top, StepPanelStyle.Width(stepPanelWidth).Render(left.String()), LogPanelStyle.Width(logPanelWidth).Render(m.viewport.View()))
	follow := "off"
	if m.followMode {
		follow = "on"
	}
	return combined + "\n" + HelpStyle.Render(fmt.Sprintf("up/k: up  down/j: down  pgup/pgdn: scroll  f: follow[%s]  b: back  q: quit", follow))
}

func stepStatusMark(status string) string {
	switch status {
	case string(operationsv1alpha1.TaskSucceeded):
		return StepSuccessMark
	case string(operationsv1alpha1.TaskFailed), string(operationsv1alpha1.TaskTimedOut), string(operationsv1alpha1.TaskCancelled):
		return StepFailedMark
	case string(operationsv1alpha1.TaskRunning):
		return StepRunningMark
	default:
		return StepPendingMark
	}
}

func StepPanelWidth(content string, width int) string {
	return StepPanelStyle.Width(width).Render(content)
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
