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
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/platformstatus"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

func TestServiceState(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		exitCode   int
		err        error
		wantStatus platformstatus.Status
	}{
		{name: "running", output: serviceOutput("active", "running", "enabled"), wantStatus: platformstatus.Healthy},
		{name: "stopped", output: serviceOutput("inactive", "dead", "enabled"), wantStatus: platformstatus.Unhealthy},
		{name: "disabled", output: serviceOutput("active", "running", "disabled"), wantStatus: platformstatus.Degraded},
		{name: "restarting", output: serviceOutput("activating", "auto-restart", "enabled"), wantStatus: platformstatus.Degraded},
		{name: "ssh failure", err: errors.New("connection refused"), wantStatus: platformstatus.Unknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := &remoteRunner{run: func(_ context.Context, _ *sshutils.SSH, _, _ string) (sshutils.Result, error) {
				return sshutils.Result{Stdout: test.output, ExitCode: test.exitCode}, test.err
			}}
			check := runner.service("192.0.2.10", "kc-agent")
			if check.Status != test.wantStatus {
				t.Fatalf("status = %s, want %s", check.Status, test.wantStatus)
			}
			if test.name == "restarting" && !strings.Contains(check.Message, "restarting") {
				t.Fatalf("message = %q, want restarting state", check.Message)
			}
		})
	}
}

func TestNewRemoteRunnerRequiresSSHSettings(t *testing.T) {
	if runner := newRemoteRunner(context.Background(), nil); runner != nil {
		t.Fatalf("newRemoteRunner(nil) = %#v, want nil", runner)
	}
}

func TestPortCommandFailureIsUnhealthy(t *testing.T) {
	var command string
	runner := &remoteRunner{run: func(_ context.Context, _ *sshutils.SSH, _ string, remoteCommand string) (sshutils.Result, error) {
		command = remoteCommand
		return sshutils.Result{ExitCode: 1, Stderr: "connection timed out"}, errors.New("exit status 1")
	}}
	check := runner.port("192.0.2.20", "broker-connectivity", "192.0.2.10:9889")
	if check.Status != platformstatus.Unhealthy {
		t.Fatalf("status = %s, want %s", check.Status, platformstatus.Unhealthy)
	}
	if !strings.Contains(command, `/dev/tcp/$1/$2`) || !strings.Contains(command, "-- '192.0.2.10' '9889'") {
		t.Fatalf("command = %q", command)
	}
}

func TestPortCommandQuotesEndpointAsArguments(t *testing.T) {
	var command string
	runner := &remoteRunner{run: func(_ context.Context, _ *sshutils.SSH, _ string, remoteCommand string) (sshutils.Result, error) {
		command = remoteCommand
		return sshutils.Result{ExitCode: 1}, nil
	}}
	runner.port("192.0.2.20", "broker-connectivity", "[$(touch /tmp/unsafe)]:9889")

	if strings.Contains(command, "/dev/tcp/$(touch") {
		t.Fatalf("endpoint was embedded in shell program: %q", command)
	}
	if !strings.Contains(command, "-- '$(touch /tmp/unsafe)' '9889'") {
		t.Fatalf("endpoint was not passed as quoted arguments: %q", command)
	}
}

func TestStoppedAgentDetectedBeforeHeartbeatExpires(t *testing.T) {
	now := metav1.NewTime(time.Now())
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
		Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{
			Type: corev1.NodeReady, Status: corev1.ConditionTrue, LastHeartbeatTime: now,
		}}},
	}
	heartbeat := agentHeartbeat(agentTarget{ID: "worker-1", IP: "192.0.2.20", Node: node, Deployed: true}, true)
	if heartbeat.Status != platformstatus.Healthy {
		t.Fatalf("heartbeat status = %s", heartbeat.Status)
	}
	runner := &remoteRunner{run: func(_ context.Context, _ *sshutils.SSH, _, command string) (sshutils.Result, error) {
		if strings.HasPrefix(command, "journalctl") {
			return sshutils.Result{}, nil
		}
		return sshutils.Result{Stdout: serviceOutput("inactive", "dead", "enabled")}, nil
	}}
	service := runner.service("192.0.2.20", "kc-agent")
	if service.Status != platformstatus.Unhealthy {
		t.Fatalf("service status = %s, want Unhealthy", service.Status)
	}
}

func TestAgentRegistrationUnknownWhenAPIInventoryUnavailable(t *testing.T) {
	target := agentTarget{ID: "worker-1", IP: "192.0.2.20", Deployed: true}
	check := agentHeartbeat(target, false)
	if check.Status != platformstatus.Skipped {
		t.Fatalf("status = %s, want Skipped", check.Status)
	}
	check = agentHeartbeat(target, true)
	if check.Status != platformstatus.Unhealthy {
		t.Fatalf("status with loaded inventory = %s, want Unhealthy", check.Status)
	}
}

func TestMergeAgentTargets(t *testing.T) {
	deployConfig := options.NewDeployOptions()
	deployConfig.Agents["192.0.2.10"] = options.Metadata{AgentID: "worker-1"}
	nodes := []corev1.Node{
		{ObjectMeta: metav1.ObjectMeta{Name: "worker-1"}, Status: corev1.NodeStatus{Ipv4DefaultIP: "192.0.2.10"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "worker-2", Labels: map[string]string{common.LabelNodeDisable: "true"}}, Status: corev1.NodeStatus{Ipv4DefaultIP: "192.0.2.11"}},
	}
	targets := mergeAgentTargets(deployConfig, nodes)
	if len(targets) != 2 {
		t.Fatalf("targets = %d, want 2", len(targets))
	}
	if !targets[0].Deployed || targets[0].Node == nil {
		t.Fatalf("deployed target was not reconciled: %#v", targets[0])
	}
	if !targets[1].Disabled {
		t.Fatalf("disabled node was not preserved: %#v", targets[1])
	}
}

func TestSanitize(t *testing.T) {
	input := "\x1b[31m\x1b]52;c;Y2xpcGJvYXJk\a" +
		`password=hunter2 {"token":"abc.def"} Authorization:Bearer abc.def amqp://admin:secret@example:5672 ` +
		"-----BEGIN PRIVATE KEY-----\nsensitive\n-----END PRIVATE KEY-----"
	result := sanitize(input)
	for _, secret := range []string{"hunter2", "abc.def", "admin:secret", "sensitive", "\x1b", "Y2xpcGJvYXJk"} {
		if strings.Contains(result, secret) {
			t.Errorf("sanitized output contains %q: %s", secret, result)
		}
	}
}

func TestEtcdSummaryUsesNativeEndpointHealth(t *testing.T) {
	checks := []Check{
		{Name: "kc-etcd-service", Status: platformstatus.Healthy},
		{Name: "endpoint-health", Status: platformstatus.Healthy},
	}
	if got, want := etcdSummary([]string{"192.0.2.10"}, checks, nil), "1/1 members healthy"; got != want {
		t.Fatalf("etcd summary = %q, want %q", got, want)
	}
}

func TestPadRightUsesDisplayCharacters(t *testing.T) {
	if got, want := padRight("✓ Healthy", statusColumnWidth), "✓ Healthy   "; got != want {
		t.Fatalf("padded status = %q, want %q", got, want)
	}
}

func TestProgressReporterShowsEachCompletedComponent(t *testing.T) {
	var output bytes.Buffer
	progress := &progressReporter{out: &output, enabled: true}
	progress.start("kc-server")
	progress.complete(Component{
		Name: "kc-server", Status: platformstatus.Healthy, Message: "all server subsystems ready",
	})

	text := output.String()
	for _, expected := range []string{
		"Running KubeClipper diagnostics...",
		"checking...",
		"kc-server",
		"Healthy",
		"all server subsystems ready",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("progress output does not contain %q:\n%s", expected, text)
		}
	}
}

func TestPrintReportExpandsOnlyProblems(t *testing.T) {
	report := newReport(time.Now(), time.Second, []Component{
		{Name: "kcctl", Message: "API reachable", Checks: []Check{{Name: "api", Status: platformstatus.Healthy, Message: "healthy detail"}}},
		{Name: "kc-agent", Message: "0/1 agents healthy", Checks: []Check{{
			Name: "service", Target: "worker-1", Status: platformstatus.Unhealthy,
			Message: "kc-agent on worker-1 is not running", Evidence: []string{"ActiveState=inactive"},
		}}},
	})
	var output bytes.Buffer
	if err := printReport(&output, report); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, expected := range []string{
		"Overall: Unhealthy",
		"[OK]  kcctl",
		"[FAIL] kc-agent",
		"Problems",
		"ActiveState=inactive",
		"Summary: 1 passed, 0 warnings, 1 failed, 0 skipped",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("output does not contain %q:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "healthy detail") {
		t.Errorf("healthy check details should be collapsed:\n%s", text)
	}
}

func TestCountChecksSeparatesWarnings(t *testing.T) {
	report := newReport(time.Now(), 0, []Component{{Checks: []Check{
		{Status: platformstatus.Healthy},
		{Status: platformstatus.Degraded},
		{Status: platformstatus.Unhealthy},
		{Status: platformstatus.Unknown},
		{Status: platformstatus.Skipped},
	}}})
	passed, warnings, failed, skipped := countChecks(report)
	if passed != 1 || warnings != 1 || failed != 2 || skipped != 1 {
		t.Fatalf("countChecks() = (%d, %d, %d, %d), want (1, 1, 2, 1)", passed, warnings, failed, skipped)
	}
}

func TestDoctorCommandHasNoBusinessFlags(t *testing.T) {
	command := NewCmdDoctor(options.IOStreams{Out: &bytes.Buffer{}})
	if command.Flags().HasFlags() {
		t.Fatalf("doctor command unexpectedly defines flags: %s", command.Flags().FlagUsages())
	}
}

func TestAgentSummaryExplainsStaleAPIStatus(t *testing.T) {
	targets := []agentTarget{{ID: "worker-1", Deployed: true}}
	checks := []Check{{Name: "kc-agent-service", Status: platformstatus.Unhealthy}}
	status := &platformstatus.Component{
		Name: "kc-agent", Status: platformstatus.Healthy, Message: "1/1 agents running",
	}

	got := agentSummary(targets, checks, status)
	want := "0/1 agent services running; API still reports 1/1 agents running"
	if got != want {
		t.Fatalf("agent summary = %q, want %q", got, want)
	}
}

func TestLoadDeployConfigIgnoresUnrelatedDurations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deploy-config.yaml")
	data := []byte(`
serverIPs: [192.0.2.10]
agents:
  192.0.2.20:
    agentID: worker-1
ssh:
  user: root
  port: 22
audit:
  retentionPeriod: 168h
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := loadDeployConfig(path)
	if err != nil {
		t.Fatalf("load deploy config: %v", err)
	}
	if len(config.ServerIPs) != 1 || config.ServerIPs[0] != "192.0.2.10" {
		t.Fatalf("serverIPs = %v", config.ServerIPs)
	}
	if !config.Agents.ExistsByID("worker-1") {
		t.Fatalf("agent was not loaded: %v", config.Agents)
	}
}

func TestDoctorUsesLocalDeployConfigWhenAPIConfigIsUnavailable(t *testing.T) {
	deployConfigPath := filepath.Join(t.TempDir(), "deploy-config.yaml")
	data := []byte(`
serverIPs: [192.0.2.10]
ssh:
  user: root
  port: 22
`)
	if err := os.WriteFile(deployConfigPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	o := &Options{
		configPath:       filepath.Join(t.TempDir(), "missing-config"),
		deployConfigPath: deployConfigPath,
	}
	state := &diagnosticState{}
	component := o.checkLocalAccess(context.Background(), state)

	if state.deployConfig == nil || state.remote == nil {
		t.Fatalf("local deploy config was not loaded: state=%#v", state)
	}
	if check := componentCheck(component, deployConfigCheck); check == nil || check.Status != platformstatus.Degraded {
		t.Fatalf("deploy config fallback check = %#v", check)
	}
}

func componentCheck(component Component, name string) *Check {
	for i := range component.Checks {
		if component.Checks[i].Name == name {
			return &component.Checks[i]
		}
	}
	return nil
}

func serviceOutput(active, sub, unitFile string) string {
	return "LoadState=loaded\n" +
		"UnitFileState=" + unitFile + "\n" +
		"ActiveState=" + active + "\n" +
		"SubState=" + sub + "\n" +
		"Result=success\nExecMainStatus=0\nNRestarts=0\n"
}
