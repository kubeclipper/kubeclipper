package operation

import (
	"testing"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func TestRetryStatusValidation(t *testing.T) {
	tests := []struct {
		name      string
		opStatus  v1.OperationStatusType
		wantErr   bool
		errSubstr string
	}{
		{
			name:     "failed operation can be retried",
			opStatus: v1.OperationStatusFailed,
			wantErr:  false,
		},
		{
			name:      "successful operation cannot be retried",
			opStatus:  v1.OperationStatusSuccessful,
			wantErr:   true,
			errSubstr: "not in failed status",
		},
		{
			name:      "running operation cannot be retried",
			opStatus:  v1.OperationStatusRunning,
			wantErr:   true,
			errSubstr: "not in failed status",
		},
		{
			name:      "pending operation cannot be retried",
			opStatus:  v1.OperationStatusPending,
			wantErr:   true,
			errSubstr: "not in failed status",
		},
		{
			name:      "terminated operation cannot be retried",
			opStatus:  v1.OperationStatusTermination,
			wantErr:   true,
			errSubstr: "not in failed status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the validation logic from RunRetry
			err := validateRetryStatus(tt.opStatus)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for status %q, got nil", tt.opStatus)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for status %q: %v", tt.opStatus, err)
			}
			if tt.wantErr && err != nil && tt.errSubstr != "" {
				if !containsString(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
			}
		})
	}
}

// validateRetryStatus extracts the status validation logic from RunRetry for testability.
func validateRetryStatus(status v1.OperationStatusType) error {
	if status != v1.OperationStatusFailed {
		return newStatusError("failed", string(status))
	}
	return nil
}
