/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/platformstatus"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
)

func TestEtcdEndpointHealthy(t *testing.T) {
	tests := []struct {
		name string
		code int
		body string
		want bool
	}{
		{name: "string true", code: http.StatusOK, body: `{"health":"true"}`, want: true},
		{name: "boolean true", code: http.StatusOK, body: `{"health":true}`, want: true},
		{name: "reported unhealthy", code: http.StatusOK, body: `{"health":"false"}`},
		{name: "invalid response", code: http.StatusOK, body: `ok`},
		{name: "http failure", code: http.StatusServiceUnavailable, body: `{"health":"true"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.code)
				if _, err := writer.Write([]byte(test.body)); err != nil {
					t.Errorf("write response: %v", err)
				}
			}))
			defer server.Close()
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/health", http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			if got := etcdEndpointHealthy(server.Client(), request); got != test.want {
				t.Fatalf("etcdEndpointHealthy() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestEtcdHealthURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		useTLS   bool
		want     string
		wantErr  bool
	}{
		{name: "deployment endpoint with TLS", endpoint: "172.16.131.146:12379", useTLS: true, want: "https://172.16.131.146:12379/health"},
		{name: "deployment endpoint without TLS", endpoint: "127.0.0.1:2379", want: "http://127.0.0.1:2379/health"},
		{name: "explicit scheme", endpoint: "https://etcd.example:2379/", want: "https://etcd.example:2379/health"},
		{name: "empty endpoint", wantErr: true},
		{name: "unsupported scheme", endpoint: "grpc://127.0.0.1:2379", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := etcdHealthURL(test.endpoint, test.useTLS)
			if (err != nil) != test.wantErr {
				t.Fatalf("etcdHealthURL() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("etcdHealthURL() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCountRunningAgents(t *testing.T) {
	now := time.Now()
	duration := int32(240)
	fresh := metav1.NewMicroTime(now.Add(-time.Minute))
	expired := metav1.NewMicroTime(now.Add(-10 * time.Minute))
	nodes := &corev1.NodeList{Items: []corev1.Node{
		{ObjectMeta: metav1.ObjectMeta{Name: "fresh"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "expired"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "missing"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "disabled", Labels: map[string]string{common.LabelNodeDisable: "true"}}},
	}}
	leases := &coordinationv1.LeaseList{Items: []coordinationv1.Lease{
		{ObjectMeta: metav1.ObjectMeta{Name: "fresh", Namespace: nodeLeaseNamespace}, Spec: coordinationv1.LeaseSpec{RenewTime: &fresh, LeaseDurationSeconds: &duration}},
		{ObjectMeta: metav1.ObjectMeta{Name: "expired", Namespace: nodeLeaseNamespace}, Spec: coordinationv1.LeaseSpec{RenewTime: &expired, LeaseDurationSeconds: &duration}},
		{ObjectMeta: metav1.ObjectMeta{Name: "disabled", Namespace: nodeLeaseNamespace}, Spec: coordinationv1.LeaseSpec{RenewTime: &fresh, LeaseDurationSeconds: &duration}},
		{ObjectMeta: metav1.ObjectMeta{Name: "wrong-namespace", Namespace: "default"}, Spec: coordinationv1.LeaseSpec{RenewTime: &fresh, LeaseDurationSeconds: &duration}},
	}}
	total, running := countRunningAgents(nodes, leases, now)
	if total != 3 || running != 1 {
		t.Fatalf("countRunningAgents() = (%d, %d), want (3, 1)", total, running)
	}
}

func TestAggregateServerChecks(t *testing.T) {
	tests := []struct {
		name       string
		checks     []platformstatus.Check
		wantStatus platformstatus.Status
		wantMsg    string
	}{
		{name: "healthy", checks: []platformstatus.Check{
			{Name: "api", Status: platformstatus.Healthy},
			{Name: "controller-manager", Status: platformstatus.Healthy},
			{Name: "static-resource", Status: platformstatus.Healthy},
		}, wantStatus: platformstatus.Healthy, wantMsg: "all server subsystems ready"},
		{name: "static resource degraded", checks: []platformstatus.Check{
			{Name: "api", Status: platformstatus.Healthy},
			{Name: "static-resource", Status: platformstatus.Unhealthy, Message: "resource service is unavailable"},
		}, wantStatus: platformstatus.Degraded, wantMsg: "resource service is unavailable"},
		{name: "core subsystem unhealthy", checks: []platformstatus.Check{
			{Name: "api", Status: platformstatus.Healthy},
			{Name: "controller-manager", Status: platformstatus.Unhealthy, Message: "active leader is not ready"},
		}, wantStatus: platformstatus.Unhealthy, wantMsg: "active leader is not ready"},
		{name: "multiple failures", checks: []platformstatus.Check{
			{Name: "api", Status: platformstatus.Unhealthy},
			{Name: "controller-manager", Status: platformstatus.Unhealthy},
			{Name: "static-resource", Status: platformstatus.Healthy},
		}, wantStatus: platformstatus.Unhealthy, wantMsg: "2/3 subsystems unhealthy"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, message := aggregateServerChecks(test.checks)
			if status != test.wantStatus || message != test.wantMsg {
				t.Fatalf("aggregateServerChecks() = (%q, %q), want (%q, %q)", status, message, test.wantStatus, test.wantMsg)
			}
		})
	}
}

func TestAggregateCount(t *testing.T) {
	tests := []struct {
		name       string
		healthy    int
		total      int
		suffix     string
		skipEmpty  bool
		wantStatus platformstatus.Status
		wantMsg    string
	}{
		{name: "all agents lost", total: 8, suffix: "agents running", skipEmpty: true, wantStatus: platformstatus.Unhealthy, wantMsg: "0/8 agents running"},
		{name: "no agents", suffix: "agents running", skipEmpty: true, wantStatus: platformstatus.Skipped, wantMsg: "no agents registered"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, message := aggregateCount(test.healthy, test.total, test.suffix, test.skipEmpty)
			if status != test.wantStatus || message != test.wantMsg {
				t.Fatalf("aggregateCount() = (%q, %q), want (%q, %q)", status, message, test.wantStatus, test.wantMsg)
			}
		})
	}
}

func TestAggregateEtcdEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		healthy    int
		total      int
		wantStatus platformstatus.Status
		wantMsg    string
	}{
		{name: "all healthy", healthy: 3, total: 3, wantStatus: platformstatus.Healthy, wantMsg: "etcd is healthy"},
		{name: "partially healthy", healthy: 2, total: 3, wantStatus: platformstatus.Degraded, wantMsg: "some etcd endpoints are unavailable"},
		{name: "all unavailable", total: 3, wantStatus: platformstatus.Unhealthy, wantMsg: "etcd is unavailable"},
		{name: "no configuration", wantStatus: platformstatus.Unknown, wantMsg: "no endpoints configured"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, message := aggregateEtcdEndpoints(test.healthy, test.total)
			if status != test.wantStatus || message != test.wantMsg {
				t.Fatalf("aggregateEtcdEndpoints() = (%q, %q), want (%q, %q)", status, message, test.wantStatus, test.wantMsg)
			}
		})
	}
}

func TestEnsureStatusPermission(t *testing.T) {
	role := &iamv1.GlobalRole{Rules: []rbacv1.PolicyRule{{
		APIGroups: []string{"config.kubeclipper.io"},
		Resources: []string{"configz", "components"},
		Verbs:     []string{"get", "list", "watch"},
	}}}
	if changed := ensureStatusPermission(role); !changed {
		t.Fatal("expected role to be updated")
	}
	if !slices.Contains(role.Rules[0].Resources, statusResourceName) {
		t.Fatal("status permission was not added")
	}
	if changed := ensureStatusPermission(role); changed {
		t.Fatal("permission update is not idempotent")
	}
}
