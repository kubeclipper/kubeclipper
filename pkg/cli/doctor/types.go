/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package doctor

import (
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/platformstatus"
)

const (
	platformTarget          = "platform"
	platformStatus          = "platform-status"
	natsConnectivity        = "nats-connectivity"
	heartbeatCheck          = "heartbeat"
	apiConfigCheck          = "api-config"
	commandNotFoundExitCode = 127
)

type Report struct {
	Status         platformstatus.Status
	CheckedAt      time.Time
	DurationMillis int64
	Components     []Component
}

type Component struct {
	Name    string
	Status  platformstatus.Status
	Message string
	Checks  []Check
}

type Check struct {
	Name     string
	Target   string
	Status   platformstatus.Status
	Message  string
	Evidence []string
	Logs     []string
	Commands []string
}

func newReport(checkedAt time.Time, duration time.Duration, components []Component) *Report {
	status := platformstatus.Healthy
	for i := range components {
		components[i].Status = aggregateChecks(components[i].Checks)
		status = worseStatus(status, components[i].Status)
	}
	return &Report{
		Status:         status,
		CheckedAt:      checkedAt,
		DurationMillis: duration.Milliseconds(),
		Components:     components,
	}
}

func aggregateChecks(checks []Check) platformstatus.Status {
	status := platformstatus.Healthy
	meaningful := false
	for i := range checks {
		if checks[i].Status == platformstatus.Skipped {
			continue
		}
		meaningful = true
		status = worseStatus(status, checks[i].Status)
	}
	if !meaningful {
		return platformstatus.Skipped
	}
	return status
}

func worseStatus(current, candidate platformstatus.Status) platformstatus.Status {
	rank := map[platformstatus.Status]int{
		platformstatus.Skipped:   0,
		platformstatus.Healthy:   1,
		platformstatus.Unknown:   2,
		platformstatus.Degraded:  3,
		platformstatus.Unhealthy: 4,
	}
	if rank[candidate] > rank[current] {
		return candidate
	}
	return current
}
