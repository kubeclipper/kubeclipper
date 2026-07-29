/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package natsio

import (
	"context"
	"testing"
	"time"
)

func TestEmbeddedMonitoringUsesRandomLoopbackPort(t *testing.T) {
	client, ok := NewNats(NewOptions()).(*Client)
	if !ok {
		t.Fatal("NewNats did not return *Client")
	}
	if client.serverOptions.HTTPHost != "127.0.0.1" {
		t.Fatalf("monitor host = %q, want loopback", client.serverOptions.HTTPHost)
	}
	if client.serverOptions.HTTPPort != -1 {
		t.Fatalf("monitor port = %d, want random port sentinel -1", client.serverOptions.HTTPPort)
	}
}

func TestEmbeddedNATSHealth(t *testing.T) {
	opts := NewOptions()
	opts.Server.Host = "127.0.0.1"
	opts.Server.Port = -1
	opts.Server.Cluster.Host = "127.0.0.1"
	opts.Server.Cluster.Port = 0
	opts.Server.Cluster.LeaderHost = ""

	client, ok := NewNats(opts).(*Client)
	if !ok {
		t.Fatal("NewNats did not return *Client")
	}
	stopCh := make(chan struct{})
	defer close(stopCh)
	if err := client.RunServer(stopCh); err != nil {
		t.Fatal(err)
	}
	if !client.server.ReadyForConnections(time.Second) {
		t.Fatal("embedded NATS server did not become ready")
	}
	client.url = client.server.Addr().String()
	if err := client.InitConn(stopCh); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Health(ctx); err != nil {
		t.Fatalf("Health() error = %v", err)
	}
}
