package logs

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func TestLogsFormatting(t *testing.T) {
	// Create a sample operation based on the provided JSON structure
	operation := createSampleOperation()

	// Test step title formatting with time
	t.Run("StepTitleFormatting", func(t *testing.T) {
		var buf bytes.Buffer
		startTime := metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, 0, time.UTC)}
		PrintStepTitle(&buf, "installRuntime", "dc90c6de-2b54-4cb7-be5f-e0deb693d499", "successful", startTime)

		output := buf.String()
		t.Logf("Step title output: %s", output)
	})

	// Test node status formatting with IP and duration
	t.Run("NodeStatusFormatting", func(t *testing.T) {
		var buf bytes.Buffer
		duration := 5 * time.Second
		PrintNodeStatus(&buf, "label-studio-k8s", "d27b6a66-87f9-4e55-a264-4497c2a18cd7", "successful", "192.168.10.155", duration)

		output := buf.String()
		t.Logf("Node status output: \n%s", output)
	})

	// Test minimum duration threshold
	t.Run("MinimumDurationThreshold", func(t *testing.T) {
		startAt := metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, 0, time.UTC)}
		endAt := metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, 0, time.UTC)} // Same time

		duration := calculateDuration(startAt, endAt)

		if duration != MinDuration {
			t.Errorf("Expected duration to be %v when times are the same, got %v", MinDuration, duration)
		}
	})

	// Test full operation logs formatting using the sample operation
	t.Run("FullOperationLogsFormatting", func(t *testing.T) {
		var buf bytes.Buffer

		// Get step example
		step := operation.Steps[0]
		stepStatus := string(operation.Status.Status)

		// Get step start time
		startTime := getStepStartTime(operation, step.ID)

		// Display step title
		PrintStepTitle(&buf, step.Name, step.ID, stepStatus, startTime)

		// Process each node
		for _, node := range step.Nodes {
			// Get node status and times
			nodeStatus, startAt, endAt := getNodeStatusWithTime(operation, step.ID, node.ID)

			// Calculate duration using helper function
			duration := calculateDuration(startAt, endAt)

			// Display node status
			PrintNodeStatus(&buf, node.Hostname, node.ID, nodeStatus, node.IPv4, duration)

			// Print example log
			exampleLog := "This is a sample log line\nWith multiple lines\nFor testing purposes"
			for _, line := range SplitLines(exampleLog) {
				PrintLogLine(&buf, line)
			}
		}

		output := buf.String()
		t.Logf("Full operation log output:\n%s", output)
	})

	// Test truncation of long logs
	t.Run("LogTruncation", func(t *testing.T) {
		longLog := "This is a very long log line that should be truncated because it exceeds the maximum allowed length"
		truncated := truncateLog(longLog, 20)

		expectedSuffix := TruncatedSuffix
		if len(truncated) != 20+len(expectedSuffix) || truncated[20:] != expectedSuffix {
			t.Errorf("Expected truncated log to end with '%s', got '%s'", expectedSuffix, truncated)
		}
	})
}

// Simulate calling the modified Run function without actual API calls
func TestSimulatedLogsOutput(t *testing.T) {
	var buf bytes.Buffer
	operation := createSampleOperation()

	// Print all steps in the operation
	for _, step := range operation.Steps {
		// Only show a few steps in the test to keep it manageable
		if step.ID == "dc90c6de-2b54-4cb7-be5f-e0deb693d499" ||
			step.ID == "b96fb1d5-f503-4d2b-baf4-3a9f4e0d73b0" {
			stepStatus := string(operation.Status.Status)
			startTime := getStepStartTime(operation, step.ID)

			PrintStepTitle(&buf, step.Name, step.ID, stepStatus, startTime)

			for _, node := range step.Nodes {
				nodeStatus, startAt, endAt := getNodeStatusWithTime(operation, step.ID, node.ID)

				// Use helper function for duration calculation
				duration := calculateDuration(startAt, endAt)

				PrintNodeStatus(&buf, node.Hostname, node.ID, nodeStatus, node.IPv4, duration)

				// Only show example logs for the first node
				if step.ID == "dc90c6de-2b54-4cb7-be5f-e0deb693d499" {
					sampleLog := "[2025-06-24T16:12:52+08:00] + /usr/bin/sh -c helm upgrade --install kc-csi\nhistory.go:56: [debug] getting history for release kc-csi\nRelease \"kc-csi\" does not exist. Installing it now."
					for _, line := range SplitLines(truncateLog(sampleLog, 200)) {
						PrintLogLine(&buf, line)
					}
				}
			}
		}
	}

	fmt.Println("=============== Sample Log Output ===============")
	fmt.Println(buf.String())
	fmt.Println("================================================")
}

// Create a sample operation based on the provided JSON structure
func createSampleOperation() *v1.Operation {
	// Parse times from the example
	createTime, _ := time.Parse(time.RFC3339, "2025-06-24T08:09:52Z")
	startTime1, _ := time.Parse(time.RFC3339, "2025-06-24T08:09:52Z")
	endTime1, _ := time.Parse(time.RFC3339, "2025-06-24T08:09:57Z")
	startTime2, _ := time.Parse(time.RFC3339, "2025-06-24T08:09:57Z")
	endTime2, _ := time.Parse(time.RFC3339, "2025-06-24T08:09:57Z")

	operation := &v1.Operation{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Operation",
			APIVersion: "core.kubeclipper.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "af08a746-f624-4992-abd1-5f5b6dada965",
			UID:               "3f52c707-0678-4935-806e-45ea93012d60",
			ResourceVersion:   "936",
			CreationTimestamp: metav1.Time{Time: createTime},
			Labels: map[string]string{
				"kubeclipper.io/cluster":           "cdf19a327aab247739c3f1a98",
				"kubeclipper.io/operation":         "CreateCluster",
				"kubeclipper.io/operation-sponsor": "https-192.168.10.155-8080",
				"kubeclipper.io/timeout":           "5400",
				"topology.kubeclipper.io/region":   "mgmt",
			},
			Finalizers: []string{"finalizer.operation.kubeclipper.io"},
		},
		Steps: []v1.Step{
			{
				ID:   "dc90c6de-2b54-4cb7-be5f-e0deb693d499",
				Name: "installRuntime",
				Nodes: []v1.StepNode{
					{
						ID:       "d27b6a66-87f9-4e55-a264-4497c2a18cd7",
						IPv4:     "192.168.10.155",
						NodeIPv4: "192.168.10.155",
						Hostname: "label-studio-k8s",
					},
				},
				Action:  "install",
				Timeout: metav1.Duration{Duration: 10 * time.Minute},
				Commands: []v1.Command{
					{
						Type:          "custom",
						Identity:      "containerd/v1/step",
						CustomCommand: []byte("eyJ2ZXJzaW9uIjoiMS42LjQta2MiLCJvZmZsaW5lIjp0cnVlLCJyb290RGlyIjoiL2RhdGEvY29udGFpbmVyZCIsInJlZ2lzdHJ5IjpbeyJzY2hlbWUiOiJodHRwIiwiaG9zdCI6IjE5Mi4xNjguMTAuMTU1OjUwMDAifSx7InNjaGVtZSI6Imh0dHBzIiwiaG9zdCI6IjE5Mi4xNjguMTAuMTU1OjUwMDAiLCJza2lwVmVyaWZ5Ijp0cnVlfV0sInJlZ2lzdHJ5Q29uZmlnRGlyIjoiIiwibG9jYWxSZWdpc3RyeSI6IjE5Mi4xNjguMTAuMTU1OjUwMDAiLCJrdWJlVmVyc2lvbiI6IiIsInBhdXNlVmVyc2lvbiI6IjMuNiIsInBhdXNlUmVnaXN0cnkiOiJrOHMuZ2NyLmlvIiwiZW5hYmxlU3lzdGVtZENncm91cCI6IiIsInhmc1F1b3RhIjpmYWxzZX0="),
					},
				},
				RetryTimes:     1,
				AutomaticRetry: false,
			},
			{
				ID:   "b96fb1d5-f503-4d2b-baf4-3a9f4e0d73b0",
				Name: "nodeEnvSetup",
				Nodes: []v1.StepNode{
					{
						ID:       "d27b6a66-87f9-4e55-a264-4497c2a18cd7",
						IPv4:     "192.168.10.155",
						NodeIPv4: "192.168.10.155",
						Hostname: "label-studio-k8s",
					},
				},
				Action:  "install",
				Timeout: metav1.Duration{Duration: 10 * time.Second},
				Commands: []v1.Command{
					{
						Type: "shell",
						ShellCommand: []string{
							"/bin/bash",
							"-c",
							"\nsystemctl stop firewalld || true\nsystemctl disable firewalld || true\nsetenforce 0\nsed -i s/^SELINUX=.*$/SELINUX=disabled/ /etc/selinux/config\nmodprobe br_netfilter \u0026\u0026 modprobe nf_conntrack\ncat \u003e /etc/sysctl.d/98-k8s.conf \u003c\u003c EOF\nnet.netfilter.nf_conntrack_tcp_be_liberal = 1\nnet.netfilter.nf_conntrack_tcp_loose = 1\nnet.netfilter.nf_conntrack_max = 524288\nnet.netfilter.nf_conntrack_buckets = 131072\nnet.netfilter.nf_conntrack_tcp_timeout_established = 21600\nnet.netfilter.nf_conntrack_tcp_timeout_time_wait = 120\nnet.ipv4.neigh.default.gc_thresh1 = 1024\nnet.ipv4.neigh.default.gc_thresh2 = 2048\nnet.ipv4.neigh.default.gc_thresh3 = 4096\nvm.max_map_count = 262144\nnet.ipv4.ip_forward = 1\nnet.ipv4.tcp_timestamps = 1\nnet.bridge.bridge-nf-call-ip6tables = 1\nnet.bridge.bridge-nf-call-iptables = 1\nnet.ipv6.conf.all.forwarding=1\nfs.file-max=1048576\nfs.inotify.max_user_instances = 8192\nfs.inotify.max_user_watches = 524288\nEOF\ncat \u003e /etc/security/limits.d/98-k8s.conf \u003c\u003c EOF\n* soft nproc 65535\n* hard nproc 65535\n* soft nofile 65535\n* hard nofile 65535\nEOF\nsysctl --system\nsysctl -p\nswapoff -a\nsed -i /swap/d /etc/fstab",
						},
					},
				},
				RetryTimes:     1,
				AutomaticRetry: false,
			},
		},
		Status: v1.OperationStatus{
			Status: v1.OperationStatusSuccessful,
			Conditions: []v1.OperationCondition{
				{
					StepID: "dc90c6de-2b54-4cb7-be5f-e0deb693d499",
					Status: []v1.StepStatus{
						{
							StartAt: metav1.Time{Time: startTime1},
							EndAt:   metav1.Time{Time: endTime1},
							Node:    "d27b6a66-87f9-4e55-a264-4497c2a18cd7",
							Status:  v1.StepStatusSuccessful,
							Reason:  "run step successfully",
							Message: "run step successfully",
						},
					},
				},
				{
					StepID: "b96fb1d5-f503-4d2b-baf4-3a9f4e0d73b0",
					Status: []v1.StepStatus{
						{
							StartAt: metav1.Time{Time: startTime2},
							EndAt:   metav1.Time{Time: endTime2},
							Node:    "d27b6a66-87f9-4e55-a264-4497c2a18cd7",
							Status:  v1.StepStatusSuccessful,
							Reason:  "run step successfully",
							Message: "run step successfully",
						},
					},
				},
			},
		},
	}

	return operation
}
