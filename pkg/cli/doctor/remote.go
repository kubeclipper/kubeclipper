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
	"context"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/platformstatus"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

const (
	journalSince         = "15 min ago"
	journalLines         = 50
	shownLogs            = 3
	defaultSSHPort       = 22
	sshConnectionTimeout = 5 * time.Second
	remoteExecutionLimit = 15 * time.Second
	remoteCommandTimeout = 10
)

var (
	keyValueSecretPattern = regexp.MustCompile(
		`(?i)((?:"?)(?:password|passwd|token|secret|authorization)(?:"?)[=:][[:space:]]*"?)([^",[:space:]}]+)`,
	)
	authorizationPattern    = regexp.MustCompile(`(?i)(authorization"?[=:][[:space:]]*)(?:bearer[[:space:]]+)?[^",;[:space:]}]+`)
	bearerPattern           = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/-]+=*`)
	jwtPattern              = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
	brokerCredentialPattern = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://)[^@\s]+@`)
	pemPrivateKeyPattern    = regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)
	interestingLogPattern   = regexp.MustCompile(
		`(?i)(error|failed|failure|fatal|panic|refused|timeout|unavailable|denied|stopp|disconnect|reconnect)`,
	)
)

type commandRunner func(context.Context, *sshutils.SSH, string, string) (sshutils.Result, error)

type remoteRunner struct {
	ctx context.Context
	ssh *sshutils.SSH
	run commandRunner
}

type serviceState struct {
	LoadState      string
	UnitFileState  string
	ActiveState    string
	SubState       string
	Result         string
	ExecMainStatus string
	NRestarts      string
}

func newRemoteRunner(ctx context.Context, sshConfig *sshutils.SSH) *remoteRunner {
	if sshConfig == nil {
		return nil
	}
	config := *sshConfig
	timeout := sshConnectionTimeout
	config.ConnectionTimeout = &timeout
	return &remoteRunner{ctx: ctx, ssh: &config, run: sshutils.SSHCmdWithSudoContext}
}

func (r *remoteRunner) runCommand(host, command string) (sshutils.Result, error) {
	ctx := r.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, remoteExecutionLimit)
	defer cancel()
	if r.ssh == nil {
		return r.run(ctx, nil, host, command)
	}
	config := *r.ssh
	return r.run(ctx, &config, host, command)
}

func (r *remoteRunner) service(host, unit string) Check {
	properties := "-p LoadState -p UnitFileState -p ActiveState -p SubState " +
		"-p Result -p ExecMainStatus -p NRestarts"
	command := fmt.Sprintf("timeout %d systemctl show %s %s", remoteCommandTimeout, unit, properties)
	result, err := r.runCommand(host, command)
	check := Check{Name: unit + "-service", Target: host}
	if err != nil {
		check.Status = platformstatus.Unknown
		check.Message = fmt.Sprintf("cannot inspect %s on %s", unit, host)
		check.Evidence = []string{sanitize(err.Error())}
		return check
	}
	if result.ExitCode != 0 {
		check.Status = platformstatus.Unknown
		check.Message = fmt.Sprintf("cannot inspect %s on %s", unit, host)
		check.Evidence = compactOutput(result.Stdout, result.Stderr)
		return check
	}

	state := parseServiceState(result.Stdout)
	check.Evidence = state.evidence()
	switch {
	case state.LoadState == "not-found":
		check.Status = platformstatus.Unhealthy
		check.Message = fmt.Sprintf("%s is not installed on %s", unit, host)
	case state.ActiveState == "failed" || state.ActiveState == "inactive":
		check.Status = platformstatus.Unhealthy
		check.Message = fmt.Sprintf("%s on %s is not running", unit, host)
	case state.SubState == "auto-restart" || state.ActiveState == "activating":
		check.Status = platformstatus.Degraded
		check.Message = fmt.Sprintf("%s on %s is restarting", unit, host)
	case state.ActiveState != "active":
		check.Status = platformstatus.Degraded
		check.Message = fmt.Sprintf("%s on %s is %s", unit, host, state.ActiveState)
	case state.UnitFileState == "disabled":
		check.Status = platformstatus.Degraded
		check.Message = fmt.Sprintf("%s on %s is running but disabled", unit, host)
	default:
		check.Status = platformstatus.Healthy
		check.Message = fmt.Sprintf("%s on %s is running", unit, host)
	}
	if check.Status != platformstatus.Healthy {
		check.Logs = r.journal(host, unit)
		check.Commands = r.serviceCommands(host, unit)
	}
	return check
}

func (r *remoteRunner) port(host, name, endpoint string) Check {
	endpointHost, endpointPort, splitErr := net.SplitHostPort(endpoint)
	if splitErr != nil {
		return Check{
			Name: name, Target: host, Status: platformstatus.Unknown,
			Message:  fmt.Sprintf("cannot parse endpoint %s", endpoint),
			Evidence: []string{splitErr.Error()},
		}
	}
	probe := `exec 3<>"/dev/tcp/$1/$2"`
	command := fmt.Sprintf("timeout 3 bash -c %s -- %s %s",
		shellQuote(probe), shellQuote(endpointHost), shellQuote(endpointPort))
	result, err := r.runCommand(host, command)
	check := Check{Name: name, Target: host}
	if err != nil && result.ExitCode == 0 {
		check.Status = platformstatus.Unknown
		check.Message = fmt.Sprintf("cannot test %s from %s", endpoint, host)
		check.Evidence = []string{sanitize(err.Error())}
		return check
	}
	if result.ExitCode == commandNotFoundExitCode {
		check.Status = platformstatus.Unknown
		check.Message = fmt.Sprintf("TCP probe tools are unavailable on %s", host)
		check.Evidence = compactOutput(result.Stdout, result.Stderr)
		return check
	}
	if result.ExitCode != 0 {
		check.Status = platformstatus.Unhealthy
		check.Message = fmt.Sprintf("%s cannot reach %s", host, endpoint)
		check.Evidence = append([]string{endpoint + ": connection failed"}, compactOutput(result.Stdout, result.Stderr)...)
		check.Commands = []string{r.sshCommand(host, command)}
		return check
	}
	check.Status = platformstatus.Healthy
	check.Message = fmt.Sprintf("%s is reachable from %s", endpoint, host)
	return check
}

func (r *remoteRunner) httpHealth(host, name, healthURL string) Check {
	command := fmt.Sprintf("curl -kfsS --connect-timeout 3 --max-time 5 %s", shellQuote(healthURL))
	result, err := r.runCommand(host, command)
	check := Check{Name: name, Target: host}
	if err != nil && result.ExitCode == 0 {
		check.Status = platformstatus.Unknown
		check.Message = fmt.Sprintf("cannot run %s health check on %s", name, host)
		check.Evidence = []string{sanitize(err.Error())}
		return check
	}
	if result.ExitCode == commandNotFoundExitCode {
		check.Status = platformstatus.Unknown
		check.Message = fmt.Sprintf("curl is unavailable on %s", host)
		check.Evidence = compactOutput(result.Stdout, result.Stderr)
		return check
	}
	if result.ExitCode != 0 {
		check.Status = platformstatus.Unhealthy
		check.Message = fmt.Sprintf("%s health check failed on %s", name, host)
		check.Evidence = compactOutput(result.Stdout, result.Stderr)
		check.Commands = []string{r.sshCommand(host, command)}
		return check
	}
	check.Status = platformstatus.Healthy
	check.Message = fmt.Sprintf("%s is healthy on %s", name, host)
	return check
}

func (r *remoteRunner) journal(host, unit string) []string {
	command := fmt.Sprintf("timeout %d journalctl -u %s --since %s --no-pager -n %d -o short-iso",
		remoteCommandTimeout, unit, shellQuote(journalSince), journalLines)
	result, err := r.runCommand(host, command)
	if err != nil || result.ExitCode != 0 {
		return nil
	}
	return selectLogLines(result.Stdout, shownLogs)
}

func (r *remoteRunner) capture(host, command string) []string {
	result, err := r.runCommand(host, command)
	return compactOutput(result.Stdout, result.Stderr, errorString(err))
}

func (r *remoteRunner) serviceCommands(host, unit string) []string {
	return []string{
		r.sshCommand(host, fmt.Sprintf("systemctl status %s --no-pager -l", unit)),
		r.sshCommand(host, fmt.Sprintf("journalctl -u %s --since %s --no-pager", unit, shellQuote(journalSince))),
	}
}

func (r *remoteRunner) sshCommand(host, command string) string {
	user := "root"
	port := defaultSSHPort
	var options []string
	if r.ssh != nil {
		if r.ssh.User != "" {
			user = r.ssh.User
		}
		if r.ssh.Port > 0 {
			port = r.ssh.Port
		}
		if r.ssh.PkFile != "" {
			options = append(options, "-i", shellQuote(r.ssh.PkFile))
		}
	}
	if port != defaultSSHPort {
		options = append(options, "-p", strconv.Itoa(port))
	}
	options = append(options, user+"@"+host, shellDoubleQuote(command))
	return "ssh " + strings.Join(options, " ")
}

func parseServiceState(output string) serviceState {
	values := make(map[string]string)
	for line := range strings.SplitSeq(output, "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if found {
			values[key] = value
		}
	}
	return serviceState{
		LoadState: values["LoadState"], UnitFileState: values["UnitFileState"],
		ActiveState: values["ActiveState"], SubState: values["SubState"],
		Result: values["Result"], ExecMainStatus: values["ExecMainStatus"], NRestarts: values["NRestarts"],
	}
}

func (s *serviceState) evidence() []string {
	values := map[string]string{
		"LoadState": s.LoadState, "UnitFileState": s.UnitFileState, "ActiveState": s.ActiveState,
		"SubState": s.SubState, "Result": s.Result, "ExecMainStatus": s.ExecMainStatus, "NRestarts": s.NRestarts,
	}
	keys := []string{"LoadState", "UnitFileState", "ActiveState", "SubState", "Result", "ExecMainStatus", "NRestarts"}
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		if values[key] != "" {
			result = append(result, key+"="+values[key])
		}
	}
	return result
}

func compactOutput(outputs ...string) []string {
	var lines []string
	for _, output := range outputs {
		output = sanitize(output)
		for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
			if line != "" {
				lines = append(lines, sanitize(line))
			}
		}
	}
	if len(lines) > shownLogs {
		lines = lines[len(lines)-shownLogs:]
	}
	return lines
}

func selectLogLines(output string, limit int) []string {
	all := compactOutput(output)
	var selected []string
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if interestingLogPattern.MatchString(line) {
			selected = append(selected, sanitize(line))
		}
	}
	if len(selected) == 0 {
		selected = all
	}
	if len(selected) > limit {
		selected = selected[len(selected)-limit:]
	}
	return selected
}

func sanitize(value string) string {
	value = stripTerminalControls(value)
	value = pemPrivateKeyPattern.ReplaceAllString(value, "[REDACTED-PRIVATE-KEY]")
	value = authorizationPattern.ReplaceAllString(value, "$1[REDACTED]")
	value = keyValueSecretPattern.ReplaceAllString(value, "$1[REDACTED]")
	value = bearerPattern.ReplaceAllString(value, "Bearer [REDACTED]")
	value = jwtPattern.ReplaceAllString(value, "[REDACTED-JWT]")
	return brokerCredentialPattern.ReplaceAllString(value, "$1[REDACTED]@")
}

func stripTerminalControls(value string) string {
	var result strings.Builder
	result.Grow(len(value))
	for i := 0; i < len(value); i++ {
		character := value[i]
		if character == '\x1b' {
			i = skipEscapeSequence(value, i)
			continue
		}
		if character < 0x20 && character != '\n' && character != '\t' || character == 0x7f {
			continue
		}
		result.WriteByte(character)
	}
	return result.String()
}

func skipEscapeSequence(value string, start int) int {
	if start+1 >= len(value) {
		return start
	}
	switch value[start+1] {
	case '[':
		for i := start + 2; i < len(value); i++ {
			if value[i] >= 0x40 && value[i] <= 0x7e {
				return i
			}
		}
		return len(value) - 1
	case ']':
		for i := start + 2; i < len(value); i++ {
			if value[i] == '\a' {
				return i
			}
			if value[i] == '\x1b' && i+1 < len(value) && value[i+1] == '\\' {
				return i + 1
			}
		}
		return len(value) - 1
	default:
		return start + 1
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func shellDoubleQuote(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`$`, `\$`,
		"`", "\\`",
	)
	return `"` + replacer.Replace(value) + `"`
}

func sortChecks(checks []Check) {
	sort.SliceStable(checks, func(i, j int) bool {
		if checks[i].Target == checks[j].Target {
			return checks[i].Name < checks[j].Name
		}
		return checks[i].Target < checks[j].Target
	})
}
