package operation

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func TestCalculateDuration(t *testing.T) {
	tests := []struct {
		name    string
		startAt metav1.Time
		endAt   metav1.Time
		want    time.Duration
	}{
		{
			name:    "normal case",
			startAt: metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, 0, time.UTC)},
			endAt:   metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 57, 0, time.UTC)},
			want:    5 * time.Second,
		},
		{
			name:    "zero start time returns zero duration",
			startAt: metav1.Time{},
			endAt:   metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 57, 0, time.UTC)},
			want:    0,
		},
		{
			name:    "zero end time returns zero duration",
			startAt: metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, 0, time.UTC)},
			endAt:   metav1.Time{},
			want:    0,
		},
		{
			name:    "both zero returns zero duration",
			startAt: metav1.Time{},
			endAt:   metav1.Time{},
			want:    0,
		},
		{
			name:    "minimum threshold enforced when sub-second",
			startAt: metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, 0, time.UTC)},
			endAt:   metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, int(time.Millisecond*500), time.UTC)},
			want:    minDuration,
		},
		{
			name:    "same start and end returns minimum duration",
			startAt: metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, 0, time.UTC)},
			endAt:   metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, 0, time.UTC)},
			want:    minDuration,
		},
		{
			name:    "large duration",
			startAt: metav1.Time{Time: time.Date(2025, 6, 24, 8, 0, 0, 0, time.UTC)},
			endAt:   metav1.Time{Time: time.Date(2025, 6, 24, 9, 30, 0, 0, time.UTC)},
			want:    90 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateDuration(tt.startAt, tt.endAt)
			if got != tt.want {
				t.Errorf("calculateDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNodeStatusWithTime(t *testing.T) {
	op := &v1.Operation{
		Status: v1.OperationStatus{
			Conditions: []v1.OperationCondition{
				{
					StepID: "step-1",
					Status: []v1.StepStatus{
						{
							Node:    "node-1",
							Status:  v1.StepStatusSuccessful,
							StartAt: metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, 0, time.UTC)},
							EndAt:   metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 57, 0, time.UTC)},
						},
						{
							Node:    "node-2",
							Status:  v1.StepStatusFailed,
							StartAt: metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, 0, time.UTC)},
							EndAt:   metav1.Time{Time: time.Date(2025, 6, 24, 8, 10, 0, 0, time.UTC)},
							Reason:  "timeout",
							Message: "step timed out",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name         string
		stepID       string
		nodeID       string
		wantStatus   string
		wantStartSet bool
		wantEndSet   bool
	}{
		{
			name:         "found node-1 in step-1",
			stepID:       "step-1",
			nodeID:       "node-1",
			wantStatus:   string(v1.StepStatusSuccessful),
			wantStartSet: true,
			wantEndSet:   true,
		},
		{
			name:         "found node-2 in step-1",
			stepID:       "step-1",
			nodeID:       "node-2",
			wantStatus:   string(v1.StepStatusFailed),
			wantStartSet: true,
			wantEndSet:   true,
		},
		{
			name:         "node not found returns unknown",
			stepID:       "step-1",
			nodeID:       "node-999",
			wantStatus:   "unknown",
			wantStartSet: false,
			wantEndSet:   false,
		},
		{
			name:         "step not found returns unknown",
			stepID:       "step-999",
			nodeID:       "node-1",
			wantStatus:   "unknown",
			wantStartSet: false,
			wantEndSet:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, startAt, endAt := getNodeStatusWithTime(op, tt.stepID, tt.nodeID)
			if status != tt.wantStatus {
				t.Errorf("status = %q, want %q", status, tt.wantStatus)
			}
			if startAt.IsZero() == tt.wantStartSet {
				t.Errorf("startAt.IsZero() = %v, want %v", startAt.IsZero(), !tt.wantStartSet)
			}
			if endAt.IsZero() == tt.wantEndSet {
				t.Errorf("endAt.IsZero() = %v, want %v", endAt.IsZero(), !tt.wantEndSet)
			}
		})
	}
}

func TestGetStepStartTime(t *testing.T) {
	earliest := time.Date(2025, 6, 24, 8, 9, 50, 0, time.UTC)
	later := time.Date(2025, 6, 24, 8, 9, 55, 0, time.UTC)

	op := &v1.Operation{
		Status: v1.OperationStatus{
			Conditions: []v1.OperationCondition{
				{
					StepID: "step-1",
					Status: []v1.StepStatus{
						{Node: "node-1", StartAt: metav1.Time{Time: later}},
						{Node: "node-2", StartAt: metav1.Time{Time: earliest}},
					},
				},
				{
					StepID: "step-empty",
					Status: []v1.StepStatus{},
				},
			},
		},
	}

	tests := []struct {
		name       string
		stepID     string
		wantZero   bool
		wantTime   time.Time
	}{
		{
			name:     "returns earliest from multiple nodes",
			stepID:   "step-1",
			wantZero: false,
			wantTime: earliest,
		},
		{
			name:     "non-existent step returns zero time",
			stepID:   "step-999",
			wantZero: true,
		},
		{
			name:     "step with empty status returns zero time",
			stepID:   "step-empty",
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStepStartTime(op, tt.stepID)
			if got.IsZero() != tt.wantZero {
				t.Errorf("getStepStartTime().IsZero() = %v, want %v", got.IsZero(), tt.wantZero)
			}
			if !tt.wantZero && !got.Time.Equal(tt.wantTime) {
				t.Errorf("getStepStartTime() = %v, want %v", got.Time, tt.wantTime)
			}
		})
	}
}

func TestTruncateLog(t *testing.T) {
	tests := []struct {
		name   string
		log    string
		maxLen int
		want   string
	}{
		{
			name:   "short string not truncated",
			log:    "hello",
			maxLen: 100,
			want:   "hello",
		},
		{
			name:   "exact length not truncated",
			log:    "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "long string truncated",
			log:    "This is a very long log line that should be truncated",
			maxLen: 10,
			want:   "This is a ... (truncated)",
		},
		{
			name:   "maxLen zero means unlimited",
			log:    "This is a very long log line that should not be truncated",
			maxLen: 0,
			want:   "This is a very long log line that should not be truncated",
		},
		{
			name:   "negative maxLen means unlimited",
			log:    "Another long line",
			maxLen: -1,
			want:   "Another long line",
		},
		{
			name:   "unicode Chinese characters truncated correctly",
			log:    "这是一段中文日志内容用于测试",
			maxLen: 5,
			want:   "这是一段中... (truncated)",
		},
		{
			name:   "unicode emoji truncated correctly",
			log:    "Status: ✅ OK 🎉 done 🚀",
			maxLen: 9,
			want:   "Status: ✅... (truncated)",
		},
		{
			name:   "mixed ascii and unicode at boundary",
			log:    "abc你好def",
			maxLen: 5,
			want:   "abc你好... (truncated)",
		},
		{
			name:   "empty string",
			log:    "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateLog(tt.log, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateLog() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single line",
			input: "hello",
			want:  []string{"hello"},
		},
		{
			name:  "two lines",
			input: "hello\nworld",
			want:  []string{"hello", "world"},
		},
		{
			name:  "trailing newline",
			input: "hello\nworld\n",
			want:  []string{"hello", "world"},
		},
		{
			name:  "CRLF line endings",
			input: "hello\r\nworld",
			want:  []string{"hello", "world"},
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "only newline",
			input: "\n",
			want:  []string{""},
		},
		{
			name:  "multiple blank lines",
			input: "a\n\nb",
			want:  []string{"a", "", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("splitLines() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitLines()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
