package operation

import (
	"sort"
	"time"
	"unicode/utf8"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"

	operationsv1alpha1 "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

const (
	timeFormat       = "2006-01-02 15:04:05"
	minDuration      = time.Second
	pollInterval     = 2 * time.Second
	defaultMaxLength = 200
	truncatedSuffix  = "... (truncated)"
)

func calculateDuration(startAt, endAt *metav1.Time) time.Duration {
	if startAt == nil || endAt == nil {
		return 0
	}
	duration := endAt.Sub(startAt.Time)
	if duration < minDuration {
		return minDuration
	}
	return duration
}

func tasksByExecution(tasks []operationsv1alpha1.OperationTask) map[string][]operationsv1alpha1.OperationTask {
	grouped := make(map[string][]operationsv1alpha1.OperationTask)
	for taskIndex := range tasks {
		task := &tasks[taskIndex]
		key := task.Spec.StepID + "|" + string(task.Spec.NodeRef.UID)
		grouped[key] = append(grouped[key], *task)
	}
	for key := range grouped {
		sort.SliceStable(grouped[key], func(i, j int) bool {
			left, right := grouped[key][i], grouped[key][j]
			if left.Spec.RetryGeneration != right.Spec.RetryGeneration {
				return left.Spec.RetryGeneration > right.Spec.RetryGeneration
			}
			return left.Spec.Attempt > right.Spec.Attempt
		})
	}
	return grouped
}

func effectiveTask(
	grouped map[string][]operationsv1alpha1.OperationTask,
	stepID string,
	nodeUID types.UID,
) *operationsv1alpha1.OperationTask {
	tasks := grouped[stepID+"|"+string(nodeUID)]
	for i := range tasks {
		if tasks[i].Status.Phase == operationsv1alpha1.TaskSucceeded {
			return &tasks[i]
		}
	}
	if len(tasks) == 0 {
		return nil
	}
	return &tasks[0]
}

func stepStatus(
	step *operationsv1alpha1.OperationStep,
	grouped map[string][]operationsv1alpha1.OperationTask,
	operationPhase operationsv1alpha1.OperationPhase,
) string {
	if len(step.Targets) == 0 {
		return missingTaskPhase(operationPhase)
	}
	allSucceeded := true
	result := operationsv1alpha1.TaskPending
	for _, target := range step.Targets {
		task := effectiveTask(grouped, step.ID, target.UID)
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
			result = operationsv1alpha1.TaskRunning
		case operationsv1alpha1.TaskSucceeded:
		default:
			allSucceeded = false
		}
	}
	if allSucceeded {
		return string(operationsv1alpha1.TaskSucceeded)
	}
	return string(result)
}

func missingTaskPhase(operationPhase operationsv1alpha1.OperationPhase) string {
	if operationPhase == operationsv1alpha1.OperationCancelled {
		return string(operationsv1alpha1.TaskCancelled)
	}
	return string(operationsv1alpha1.TaskPending)
}

func stepStartTime(stepID string, tasks []operationsv1alpha1.OperationTask) *metav1.Time {
	var earliest *metav1.Time
	for i := range tasks {
		if tasks[i].Spec.StepID != stepID || tasks[i].Status.StartedAt == nil {
			continue
		}
		if earliest == nil || tasks[i].Status.StartedAt.Before(earliest) {
			earliest = tasks[i].Status.StartedAt.DeepCopy()
		}
	}
	return earliest
}

func truncateLog(log string, maxLen int) string {
	if maxLen <= 0 || utf8.RuneCountInString(log) <= maxLen {
		return log
	}
	return string([]rune(log)[:maxLen]) + truncatedSuffix
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' {
			continue
		}
		end := i
		if end > start && s[end-1] == '\r' {
			end--
		}
		lines = append(lines, s[start:end])
		start = i + 1
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
