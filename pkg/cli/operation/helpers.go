package operation

import (
	"time"
	"unicode/utf8"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

const (
	// timeFormat is the standard time format used in log output.
	timeFormat = "2006-01-02 15:04:05"

	// minDuration is the minimum duration to show (1s).
	minDuration = time.Second

	// pollInterval is how often to poll for updates when following logs.
	pollInterval = 2 * time.Second

	// defaultMaxLength is the default max log length to display.
	defaultMaxLength = 200

	// truncatedSuffix is added to truncated logs.
	truncatedSuffix = "... (truncated)"
)

// calculateDuration calculates the duration between two times with a minimum threshold.
func calculateDuration(startAt, endAt metav1.Time) time.Duration {
	var duration time.Duration
	if !startAt.IsZero() && !endAt.IsZero() {
		duration = endAt.Time.Sub(startAt.Time)
		if duration < minDuration {
			duration = minDuration
		}
	}
	return duration
}

// getNodeStatusWithTime returns status and time information for a node in a step.
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

// truncateLog truncates log content if it exceeds the maximum length.
// Uses rune-based truncation for Unicode safety.
func truncateLog(log string, maxLen int) string {
	if maxLen <= 0 || utf8.RuneCountInString(log) <= maxLen {
		return log
	}
	runes := []rune(log)
	return string(runes[:maxLen]) + truncatedSuffix
}

// splitLines splits a string into lines, handling both \n and \r\n.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			end := i
			if end > start && s[end-1] == '\r' {
				end--
			}
			lines = append(lines, s[start:end])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
