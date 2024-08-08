/*
 *
 *  * Copyright 2024 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

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
