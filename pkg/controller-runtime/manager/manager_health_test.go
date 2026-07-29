/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package manager

import (
	"testing"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateLease(t *testing.T) {
	now := time.Now()
	duration := int32(15)
	validRenewTime := metav1.NewMicroTime(now.Add(-time.Second))
	expiredRenewTime := metav1.NewMicroTime(now.Add(-20 * time.Second))

	if err := validateLease(&coordinationv1.Lease{Spec: coordinationv1.LeaseSpec{
		RenewTime: &validRenewTime, LeaseDurationSeconds: &duration,
	}}, now); err != nil {
		t.Fatalf("valid lease rejected: %v", err)
	}
	if err := validateLease(&coordinationv1.Lease{Spec: coordinationv1.LeaseSpec{
		RenewTime: &expiredRenewTime, LeaseDurationSeconds: &duration,
	}}, now); err == nil {
		t.Fatal("expired lease was accepted")
	}
}
