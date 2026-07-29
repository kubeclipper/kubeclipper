/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package delivery

import (
	"testing"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplyNodeLeaseRenewal(t *testing.T) {
	oldTime := metav1.NewMicroTime(time.Unix(1, 0))
	stored := &coordinationv1.Lease{Spec: coordinationv1.LeaseSpec{RenewTime: &oldTime}}
	holder := "node-1"
	duration := int32(240)
	requested := &coordinationv1.Lease{Spec: coordinationv1.LeaseSpec{
		HolderIdentity: &holder, LeaseDurationSeconds: &duration,
	}}
	now := time.Unix(100, 0)

	applyNodeLeaseRenewal(stored, requested, now)
	if stored.Spec.RenewTime == nil || !stored.Spec.RenewTime.Time.Equal(now) {
		t.Fatalf("renew time = %v, want %v", stored.Spec.RenewTime, now)
	}
	if stored.Spec.HolderIdentity == nil || *stored.Spec.HolderIdentity != holder {
		t.Fatalf("holder identity was not copied")
	}
	if stored.Spec.LeaseDurationSeconds == nil || *stored.Spec.LeaseDurationSeconds != duration {
		t.Fatalf("lease duration was not copied")
	}
}
