// Copyright 2026 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	extensionsv1beta1 "sigs.k8s.io/agent-sandbox/extensions/api/v1beta1"
	asmetrics "sigs.k8s.io/agent-sandbox/internal/metrics"
)

func TestSandboxClaimDefaulter_StampsWhenAbsent(t *testing.T) {
	fixed := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	d := &SandboxClaimDefaulter{now: func() time.Time { return fixed }}

	claim := &extensionsv1beta1.SandboxClaim{}
	if err := d.Default(context.Background(), claim); err != nil {
		t.Fatalf("Default returned error: %v", err)
	}

	got := claim.Annotations[asmetrics.WebhookAnnotation]
	want := fixed.Format(time.RFC3339Nano)
	if got != want {
		t.Errorf("WebhookAnnotation = %q, want %q", got, want)
	}
}

func TestSandboxClaimDefaulter_PreservesExisting(t *testing.T) {
	existing := "2020-01-01T00:00:00Z"
	d := &SandboxClaimDefaulter{now: func() time.Time { return time.Now() }}

	claim := &extensionsv1beta1.SandboxClaim{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{asmetrics.WebhookAnnotation: existing},
		},
	}
	if err := d.Default(context.Background(), claim); err != nil {
		t.Fatalf("Default returned error: %v", err)
	}

	if got := claim.Annotations[asmetrics.WebhookAnnotation]; got != existing {
		t.Errorf("WebhookAnnotation = %q, want preserved %q", got, existing)
	}
}

func TestSandboxClaimDefaulter_NilClockUsesTimeNow(t *testing.T) {
	d := &SandboxClaimDefaulter{}
	claim := &extensionsv1beta1.SandboxClaim{}

	before := time.Now().UTC()
	if err := d.Default(context.Background(), claim); err != nil {
		t.Fatalf("Default returned error: %v", err)
	}
	after := time.Now().UTC()

	stamp, err := time.Parse(time.RFC3339Nano, claim.Annotations[asmetrics.WebhookAnnotation])
	if err != nil {
		t.Fatalf("stamp not RFC3339Nano: %v", err)
	}
	if stamp.Before(before) || stamp.After(after) {
		t.Errorf("stamp %v not within [%v, %v]", stamp, before, after)
	}
}
