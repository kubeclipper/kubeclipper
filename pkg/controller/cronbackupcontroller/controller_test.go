package cronbackupcontroller

import (
	"fmt"
	"testing"
	"time"
)

func TestCronBackupReconciler_parseSchedule(t *testing.T) {
	type fields struct {
		Now func() time.Time
	}
	type args struct {
		schedule string
	}
	type spec struct {
		name   string
		fields fields
		args   args
		want   string
	}

	var generateIntercalaryYearSpec = func() []spec {
		specs := make([]spec, 366)
		date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < len(specs); i++ {
			t := date
			specs[i].fields.Now = func() time.Time {
				return t
			}
			specs[i].args = args{
				schedule: "0 0 L * *",
			}
			specs[i].want = generateWant(366, specs[i].fields.Now().Month())
			specs[i].name = fmt.Sprintf("day %s", specs[i].fields.Now().String())
			date = date.AddDate(0, 0, 1)
		}
		return specs
	}

	var generateNormalYearSpec = func() []spec {
		specs := make([]spec, 365)
		date := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < len(specs); i++ {
			t := date
			specs[i].fields.Now = func() time.Time {
				return t
			}
			specs[i].args = args{
				schedule: "0 0 L * *",
			}
			specs[i].want = generateWant(365, specs[i].fields.Now().Month())
			specs[i].name = fmt.Sprintf("day %s", specs[i].fields.Now().String())
			date = date.AddDate(0, 0, 1)
		}
		return specs
	}

	var tests []spec
	tests = append(tests, generateNormalYearSpec()...)
	tests = append(tests, generateIntercalaryYearSpec()...)
	tests = append(tests, []spec{
		{
			name: "without day of month synax 'L'",
			fields: fields{
				Now: func() time.Time {
					return time.Now()
				},
			},
			args: args{
				schedule: "0 0 1 * *",
			},
			want: "0 0 1 * *",
		},
	}...)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &CronBackupReconciler{
				Now: tt.fields.Now,
			}
			if got := ParseSchedule(tt.args.schedule, s.Now()); got != tt.want {
				t.Errorf("CronBackupReconciler.parseSchedule() = %v, want %v, %v", got, tt.want, tt.fields.Now().String())
			}
		})
	}
}

func generateWant(days int, month time.Month) string {
	if days == 365 {
		switch month {
		case time.January, time.March, time.May, time.July, time.August, time.October, time.December:
			return "0 0 31 * *"
		case time.April, time.June, time.September, time.November:
			return "0 0 30 * *"
		case time.February:
			return "0 0 28 * *"
		}
	}
	switch month {
	case time.January, time.March, time.May, time.July, time.August, time.October, time.December:
		return "0 0 31 * *"
	case time.April, time.June, time.September, time.November:
		return "0 0 30 * *"
	case time.February:
		return "0 0 29 * *"
	}
	panic("unexpected")
}
