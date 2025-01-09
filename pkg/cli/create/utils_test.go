package create

import "testing"

func Test_hasConflict(t *testing.T) {
	tests := []struct {
		name     string
		masters  []string
		workers  []string
		expected bool
	}{
		{
			name:     "no conflict, both masters and workers are empty",
			masters:  []string{},
			workers:  []string{},
			expected: false, // 无冲突
		},
		{
			name:     "no conflict, masters and workers are different IPs",
			masters:  []string{"192.168.1.1", "192.168.1.2"},
			workers:  []string{"192.168.2.1", "192.168.2.2"},
			expected: false, // 无冲突
		},
		{
			name:     "conflict, one IP in both masters and workers",
			masters:  []string{"192.168.1.1"},
			workers:  []string{"192.168.1.1"}, // 有重复 IP
			expected: true,                    // 有冲突
		},
		{
			name:     "no conflict, masters and workers have distinct IPs",
			masters:  []string{"192.168.1.1", "192.168.1.2"},
			workers:  []string{"192.168.2.1", "192.168.2.2"},
			expected: false, // 无冲突
		},
		{
			name:     "conflict, one element each, same IP",
			masters:  []string{"192.168.1.1"},
			workers:  []string{"192.168.1.1"}, // 冲突
			expected: true,                    // 有冲突
		},
		{
			name:     "conflict, multiple IPs with overlap",
			masters:  []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
			workers:  []string{"192.168.2.1", "192.168.1.2", "192.168.2.2"}, // `192.168.1.2` 与 workers 有重复
			expected: true,                                                  // 有冲突
		},
		{
			name:     "no conflict, workers are distinct IPs",
			masters:  []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
			workers:  []string{"192.168.2.1", "192.168.2.2", "192.168.2.3"}, // 没有重复
			expected: false,                                                 // 无冲突
		},
		{
			name:     "no conflict, masters only",
			masters:  []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
			workers:  []string{}, // 只有 masters，workers 为空
			expected: false,      // 无冲突
		},
		{
			name:     "no conflict, workers only",
			masters:  []string{},
			workers:  []string{"192.168.2.1", "192.168.2.2", "192.168.2.3"}, // 只有 workers，masters 为空
			expected: false,                                                 // 无冲突
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasConflict(tt.masters, tt.workers)
			if result != tt.expected {
				t.Errorf("hasConflict(%v, %v) = %v; want %v", tt.masters, tt.workers, result, tt.expected)
			}
		})
	}
}
