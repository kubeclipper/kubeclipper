package option

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/apis/audit"
)

type AuditOptions struct {
	RetentionPeriod time.Duration `json:"retentionPeriod" yaml:"retentionPeriod"`
	MaximumEntries  int           `json:"maximumEntries" yaml:"maximumEntries"`
	AuditLevel      audit.Level   `json:"auditLevel" yaml:"auditLevel"`
}

func NewAuditOptions() *AuditOptions {
	return &AuditOptions{
		RetentionPeriod: 7 * 24 * time.Hour,
		MaximumEntries:  200,
		AuditLevel:      audit.LevelRequest,
	}
}

func (o *AuditOptions) Validate() []error {
	var errs []error
	if o.RetentionPeriod < 10*time.Minute {
		errs = append(errs, errors.New("audit log retention should not less than 10 minutes"))
	}

	if o.MaximumEntries <= 0 {
		errs = append(errs, errors.New("audit log entries must be greater than 0"))
	}

	return errs
}

func (o *AuditOptions) AddFlags(fs *pflag.FlagSet) {
	fs.IntVarP(&o.MaximumEntries, "audit-number", "n", o.MaximumEntries, "Number of log retention")
	fs.DurationVarP(&o.RetentionPeriod, "audit-period", "p", o.RetentionPeriod, "log retention time, minimal value is 10 minutes")
}
