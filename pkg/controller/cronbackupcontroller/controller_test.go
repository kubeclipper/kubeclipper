package cronbackupcontroller

import "testing"

func Test_parseSchedule(t *testing.T) {
	type args struct {
		schedule string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test parse cron schedule without day of month is 'L'",
			args: args{
				schedule: "0 0 1 * *",
			},
			want: "0 0 1 * *",
		},
		{
			name: "test parse cron schedule with day of month is 'L'",
			args: args{
				schedule: "0 0 L * *",
			},
			want: "0 0 31 * *",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseSchedule(tt.args.schedule); got != tt.want {
				t.Errorf("parseSchedule() = %v, want %v", got, tt.want)
			}
		})
	}
}
