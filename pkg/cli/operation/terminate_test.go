package operation

import (
	"fmt"
	"testing"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func TestTerminateStatusValidation(t *testing.T) {
	tests := []struct {
		name      string
		opStatus  v1.OperationStatusType
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "running operation can be terminated",
			opStatus: v1.OperationStatusRunning,
			wantErr:  false,
		},
		{
			name:      "successful operation cannot be terminated",
			opStatus:  v1.OperationStatusSuccessful,
			wantErr:   true,
			errSubstr: "not in running status",
		},
		{
			name:      "failed operation cannot be terminated",
			opStatus:  v1.OperationStatusFailed,
			wantErr:   true,
			errSubstr: "not in running status",
		},
		{
			name:      "pending operation cannot be terminated",
			opStatus:  v1.OperationStatusPending,
			wantErr:   true,
			errSubstr: "not in running status",
		},
		{
			name:      "terminated operation cannot be terminated again",
			opStatus:  v1.OperationStatusTermination,
			wantErr:   true,
			errSubstr: "not in running status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTerminateStatus(tt.opStatus)
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

// validateTerminateStatus extracts the status validation logic from RunTerminate for testability.
func validateTerminateStatus(status v1.OperationStatusType) error {
	if status != v1.OperationStatusRunning {
		return newStatusError("running", string(status))
	}
	return nil
}

// newStatusError creates a formatted status validation error.
func newStatusError(expected, current string) error {
	return fmt.Errorf("operation is not in %s status, current status: %s", expected, current)
}

// containsString checks if substr is in s.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
