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
	Groups []TaskGroup
}

// TaskGroup keeps all execution attempts for nodes sharing one IP together.
// An operation creates one task per node and attempt, so rendering the raw
// task list directly makes retries look like unrelated nodes.
type TaskGroup struct {
	IP     string
	Status string
	Tasks  []TaskEntry
}

type TaskEntry struct {
	Name            string
	NodeUID         string
	RetryGeneration int64
	Attempt         int32
	Status          string
	Duration        string
	CreatedAt       time.Time
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
	client       *kc.Client
	operation    *operationsv1alpha1.Operation
	tasks        []operationsv1alpha1.OperationTask
	steps        []StepEntry
	cursor       int
	followMode   bool
	lastOffset   map[string]int64
	logContent   map[string]string
	displayed    string
	stepViewport viewport.Model
	stepLines    []int
	viewport     viewport.Model
	rawContent   string
	width        int
	height       int
}

const (
	followTickInterval = 2 * time.Second
	minLogPanelWidth   = 10
)

func NewLogModel(client *kc.Client, op *operationsv1alpha1.Operation, width, height int) LogModel {
	stepPanelWidth := width * 35 / 100
	logPanelWidth := width - stepPanelWidth - 2
	logPanelWidth = max(minLogPanelWidth, logPanelWidth)
	stepViewportWidth := maxInt(1, stepPanelWidth-4)
	viewHeight := maxInt(1, height-3)
	m := LogModel{
		client: client, operation: op, steps: buildStepEntries(op, nil),
		lastOffset: make(map[string]int64), logContent: make(map[string]string),
		stepViewport: viewport.New(stepViewportWidth, viewHeight),
		viewport:     viewport.New(logPanelWidth, viewHeight), width: width, height: height,
	}
	m.rebuildStepViewport()
	return m
}

func buildStepEntries(op *operationsv1alpha1.Operation, tasks []operationsv1alpha1.OperationTask) []StepEntry {
	if op == nil {
		return nil
	}
	byStep := make(map[string][]operationsv1alpha1.OperationTask)
	for i := range tasks {
		byStep[tasks[i].Spec.StepID] = append(byStep[tasks[i].Spec.StepID], tasks[i])
	}
	entries := make([]StepEntry, 0, len(op.Spec.Steps))
	for stepIndex := range op.Spec.Steps {
		step := &op.Spec.Steps[stepIndex]
		stepTasks := byStep[step.ID]
		sort.SliceStable(stepTasks, func(i, j int) bool {
			if stepTasks[i].Spec.RetryGeneration != stepTasks[j].Spec.RetryGeneration {
				return stepTasks[i].Spec.RetryGeneration < stepTasks[j].Spec.RetryGeneration
			}
			if stepTasks[i].Spec.Attempt != stepTasks[j].Spec.Attempt {
				return stepTasks[i].Spec.Attempt < stepTasks[j].Spec.Attempt
			}
			if stepTasks[i].Spec.NodeRef.IP != stepTasks[j].Spec.NodeRef.IP {
				return stepTasks[i].Spec.NodeRef.IP < stepTasks[j].Spec.NodeRef.IP
			}
			return stepTasks[i].Name < stepTasks[j].Name
		})
		entry := StepEntry{ID: step.ID, Status: string(operationsv1alpha1.TaskPending)}
		groups := make([]TaskGroup, 0, len(step.Targets))
		groupByKey := make(map[string]int, len(step.Targets))
		groupByUID := make(map[string]int, len(step.Targets))
		for _, target := range step.Targets {
			key := nodeGroupKey(target)
			groupIndex, exists := groupByKey[key]
			if !exists {
				groupIndex = len(groups)
				groups = append(groups, TaskGroup{IP: target.IP})
				groupByKey[key] = groupIndex
			}
			if target.UID != "" {
				groupByUID[string(target.UID)] = groupIndex
			}
		}
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
			entryTask := TaskEntry{
				Name:            task.Name,
				NodeUID:         string(task.Spec.NodeRef.UID),
				RetryGeneration: task.Spec.RetryGeneration,
				Attempt:         task.Spec.Attempt,
				Status:          string(task.Status.Phase),
				Duration:        duration,
				CreatedAt:       task.CreationTimestamp.Time,
			}
			groupIndex, exists := groupByUID[string(task.Spec.NodeRef.UID)]
			if !exists {
				key := nodeGroupKey(task.Spec.NodeRef)
				groupIndex, exists = groupByKey[key]
				if !exists {
					groupIndex = len(groups)
					groups = append(groups, TaskGroup{IP: task.Spec.NodeRef.IP})
					groupByKey[key] = groupIndex
				}
			}
			groups[groupIndex].Tasks = append(groups[groupIndex].Tasks, entryTask)
		}
		for groupIndex := range groups {
			groups[groupIndex].Status = aggregateTaskGroupPhase(groups[groupIndex].Tasks)
		}
		entry.Groups = groups
		entry.Status = aggregateStepPhase(step, stepTasks, op.Status.Phase)
		entries = append(entries, entry)
	}
	return entries
}

func nodeGroupKey(node operationsv1alpha1.NodeReference) string {
	if node.IP != "" {
		return "ip:" + node.IP
	}
	if node.UID != "" {
		return "uid:" + string(node.UID)
	}
	return "name:" + node.Name
}

func aggregateTaskGroupPhase(tasks []TaskEntry) string {
	if len(tasks) == 0 {
		return string(operationsv1alpha1.TaskPending)
	}
	latestByNode := make(map[string]TaskEntry, len(tasks))
	for _, task := range tasks {
		current, exists := latestByNode[task.NodeUID]
		if !exists || newerTask(task, current) {
			latestByNode[task.NodeUID] = task
		}
	}
	allSucceeded := true
	for _, task := range latestByNode {
		switch task.Status {
		case string(operationsv1alpha1.TaskFailed), string(operationsv1alpha1.TaskTimedOut), string(operationsv1alpha1.TaskCancelled):
			return task.Status
		case string(operationsv1alpha1.TaskRunning):
			allSucceeded = false
		case string(operationsv1alpha1.TaskSucceeded):
		default:
			allSucceeded = false
		}
	}
	if allSucceeded {
		return string(operationsv1alpha1.TaskSucceeded)
	}
	for _, task := range latestByNode {
		if task.Status == string(operationsv1alpha1.TaskRunning) {
			return string(operationsv1alpha1.TaskRunning)
		}
	}
	return string(operationsv1alpha1.TaskPending)
}

func newerTask(left, right TaskEntry) bool {
	if left.RetryGeneration != right.RetryGeneration {
		return left.RetryGeneration > right.RetryGeneration
	}
	if left.Attempt != right.Attempt {
		return left.Attempt > right.Attempt
	}
	if !left.CreatedAt.Equal(right.CreatedAt) {
		return left.CreatedAt.After(right.CreatedAt)
	}
	return left.Name > right.Name
}

func aggregateStepPhase(
	step *operationsv1alpha1.OperationStep,
	tasks []operationsv1alpha1.OperationTask,
	operationPhase operationsv1alpha1.OperationPhase,
) string {
	effective := effectiveTasksByNode(tasks)
	if len(step.Targets) == 0 {
		return missingStepPhase(operationPhase)
	}
	return aggregateTargetPhases(step, effective, operationPhase)
}

func effectiveTasksByNode(tasks []operationsv1alpha1.OperationTask) map[string]*operationsv1alpha1.OperationTask {
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
	return effective
}

func aggregateTargetPhases(
	step *operationsv1alpha1.OperationStep,
	effective map[string]*operationsv1alpha1.OperationTask,
	operationPhase operationsv1alpha1.OperationPhase,
) string {
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

func (m *LogModel) currentTask() *TaskEntry {
	if m.cursor >= len(m.steps) || len(m.steps[m.cursor].Groups) == 0 {
		return nil
	}
	var latest *TaskEntry
	for groupIndex := range m.steps[m.cursor].Groups {
		for taskIndex := range m.steps[m.cursor].Groups[groupIndex].Tasks {
			task := &m.steps[m.cursor].Groups[groupIndex].Tasks[taskIndex]
			if latest == nil || newerTask(*task, *latest) {
				latest = task
			}
		}
	}
	return latest
}

func (m *LogModel) rebuildStepViewport() {
	lines := []string{HeaderStyle.Render("Steps and Tasks")}
	m.stepLines = make([]int, len(m.steps))
	for stepIndex, step := range m.steps {
		m.stepLines[stepIndex] = len(lines)
		stepLine := fmt.Sprintf(" %s %s", stepStatusMark(step.Status), step.ID)
		if stepIndex == m.cursor {
			stepLine = SelectedStyle.Render(stepLine)
		}
		lines = append(lines, stepLine)
		for _, group := range step.Groups {
			groupLine := fmt.Sprintf("   %s %s", stepStatusMark(group.Status), displayIP(group.IP))
			if stepIndex == m.cursor {
				groupLine = SelectedStyle.Render(groupLine)
			}
			lines = append(lines, groupLine)
			for _, task := range group.Tasks {
				taskLine := fmt.Sprintf("      %s attempt=%d", stepStatusMark(task.Status), task.Attempt)
				if task.Duration != "" {
					taskLine += " [" + task.Duration + "]"
				}
				if stepIndex == m.cursor {
					taskLine = SelectedStyle.Render(taskLine)
				}
				lines = append(lines, taskLine)
			}
		}
	}
	m.stepViewport.SetContent(strings.Join(lines, "\n"))
	m.ensureStepCursorVisible()
}

func (m *LogModel) ensureStepCursorVisible() {
	if m.cursor < 0 || m.cursor >= len(m.stepLines) {
		return
	}
	line := m.stepLines[m.cursor]
	height := maxInt(1, m.stepViewport.Height)
	if line < m.stepViewport.YOffset || line >= m.stepViewport.YOffset+height {
		offset := line
		maxOffset := maxInt(0, m.stepViewport.TotalLineCount()-height)
		if offset > maxOffset {
			offset = maxOffset
		}
		m.stepViewport.SetYOffset(offset)
	}
}

func displayIP(ip string) string {
	if ip == "" {
		return "-"
	}
	return ip
}

func (m *LogModel) showCurrentLog() bool {
	task := m.currentTask()
	if task == nil {
		m.displayed = ""
		m.rawContent = "(no Task has been created for this step)\n"
		m.viewport.SetContent(m.rawContent)
		return false
	}
	m.displayed = task.Name
	m.rawContent = m.logContent[task.Name]
	m.viewport.SetContent(m.rawContent)
	m.viewport.GotoTop()
	return true
}

func (m *LogModel) fetchCurrentLogCmd() tea.Cmd {
	task := m.currentTask()
	if task == nil {
		return nil
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

func (m *LogModel) Init() tea.Cmd { return m.fetchOperationStatusCmd() }

func (m LogModel) Update(msg tea.Msg) (LogModel, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.viewport.Width = maxInt(minLogPanelWidth, m.width-m.width*35/100-2)
		m.viewport.Height = maxInt(1, m.height-3)
		m.stepViewport.Width = maxInt(1, m.width*35/100-4)
		m.stepViewport.Height = maxInt(1, m.height-3)
		m.rebuildStepViewport()
	case logFetchedMsg:
		if msg.content != "" && msg.key != "" {
			m.logContent[msg.key] += msg.content
			m.lastOffset[msg.key] = msg.offset
			if msg.key == m.displayed {
				m.rawContent = m.logContent[msg.key]
				m.viewport.SetContent(m.rawContent)
				if m.followMode {
					m.viewport.GotoBottom()
				}
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
			m.rebuildStepViewport()
			if m.currentTask() == nil || m.currentTask().Name != m.displayed {
				if m.showCurrentLog() {
					cmds = append(cmds, m.fetchCurrentLogCmd())
				}
			} else if m.rawContent == "" {
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
				m.rebuildStepViewport()
				if m.showCurrentLog() {
					cmds = append(cmds, m.fetchCurrentLogCmd())
				}
			}
		case DefaultKeyMap.Down, "j":
			if m.cursor < len(m.steps)-1 {
				m.cursor++
				m.rebuildStepViewport()
				if m.showCurrentLog() {
					cmds = append(cmds, m.fetchCurrentLogCmd())
				}
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
	m.stepViewport, cmd = m.stepViewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.viewport, cmd = m.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if key, ok := msg.(tea.KeyMsg); ok && (key.String() == DefaultKeyMap.Up || key.String() == DefaultKeyMap.Down || key.String() == "k" || key.String() == "j") {
		m.ensureStepCursorVisible()
	}
	return m, tea.Batch(cmds...)
}

type backMsg struct{}

func (m LogModel) View() string {
	if len(m.steps) == 0 {
		return "No steps in this operation."
	}
	stepPanelWidth := m.width * 35 / 100
	logPanelWidth := maxInt(minLogPanelWidth, m.width-stepPanelWidth-2)
	combined := lipgloss.JoinHorizontal(
		lipgloss.Top,
		StepPanelStyle.Width(stepPanelWidth).Render(m.stepViewport.View()),
		LogPanelStyle.Width(logPanelWidth).Render(m.viewport.View()),
	)
	follow := "off"
	if m.followMode {
		follow = "on"
	}
	return combined + "\n" + HelpStyle.Render(
		fmt.Sprintf("up/k: up  down/j: down  pgup/pgdn: scroll  f: follow[%s]  b: back  q: quit", follow),
	)
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
