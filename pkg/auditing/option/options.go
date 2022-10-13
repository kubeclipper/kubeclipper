package option

import (
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/apis/audit"
)

var (
	allowedlevel = sets.NewString(string(audit.LevelNone), string(audit.LevelMetadata), string(audit.LevelRequest), string(audit.LevelRequestResponse))
)

type AuditOptions struct {
	EventLogHistoryRetentionPeriod time.Duration `json:"eventLogHistoryRetentionPeriod" yaml:"eventLogHistoryRetentionPeriod"`
	EventLogHistoryMaximumEntries  int           `json:"eventLogHistoryMaximumEntries" yaml:"eventLogHistoryMaximumEntries"`
	AuditLevel                     audit.Level   `json:"auditLevel" yaml:"auditLevel"`
	period                         string
	level                          string
}

func NewAuditOptions() *AuditOptions {
	return &AuditOptions{
		EventLogHistoryRetentionPeriod: 7 * 24 * time.Hour,
		EventLogHistoryMaximumEntries:  200,
		AuditLevel:                     audit.LevelRequest,
	}
}

func (o *AuditOptions) Validate() error {
	role := "^[1-9]+[hHdDwW]$"
	m, err := regexp.Compile(role)
	if err != nil {
		return err
	}

	if ok := m.MatchString(o.period); ok {
		re, err := regexp.Compile("[1-9]+")
		if err != nil {
			return err
		}

		loc := re.FindStringIndex(o.period)
		if loc == nil || len(loc) != 2 {
			return errors.New("parse period error, [--period] format error")
		}
		n, err := strconv.Atoi(o.period[:loc[1]])
		if err != nil {
			return err
		}

		switch o.period[loc[1]:] {
		case "h":
			o.EventLogHistoryRetentionPeriod = time.Duration(n) * time.Hour
		case "d":
			o.EventLogHistoryRetentionPeriod = time.Duration(n) * 24 * time.Hour
		case "w":
			o.EventLogHistoryRetentionPeriod = time.Duration(n) * 7 * 24 * time.Hour
		default:
			return errors.New("[--period] format error, only support [h, d, w]")
		}
	} else {
		return errors.New("only support [h, d, w]")
	}

	if allowedlevel.HasAny(o.level) {
		o.AuditLevel = audit.Level(o.level)
	} else {
		return errors.New("only support [ None ｜ Metadata | Request | RequestResponse ]")
	}

	return err
}

func (o *AuditOptions) AddFlags(fs *pflag.FlagSet) {
	fs.IntVarP(&o.EventLogHistoryMaximumEntries, "audit-number", "n", o.EventLogHistoryMaximumEntries, "Number of log retention")
	fs.StringVarP(&o.level, "audit-level", "l", o.level, "can only be specified as [ None ｜ Metadata | Request | RequestResponse ]")
	fs.StringVarP(&o.period, "audit-period", "p", o.period, "log retention time, could specify h, d, w, such as 12h, 3d")
}
