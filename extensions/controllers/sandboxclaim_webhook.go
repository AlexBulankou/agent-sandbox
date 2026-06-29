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
	"time"

	extensionsv1beta1 "sigs.k8s.io/agent-sandbox/extensions/api/v1beta1"
	asmetrics "sigs.k8s.io/agent-sandbox/internal/metrics"
)

// SandboxClaimDefaulter stamps the webhook-first-observed-at annotation on a
// SandboxClaim at admission time.
//
// This is the production writer that the agent_sandbox_claim_startup_latency_ms
// histogram (the true end-to-end TTFE: claim creation -> Sandbox Ready) depends
// on. recordClaimStartupLatency skips the metric when WebhookAnnotation is
// absent, so without an admission-time stamp the histogram is never observed and
// reads empty (dataItems: null). Stamping here, before the object is persisted,
// captures the earliest server-side observation of the claim.
//
// The defaulter is idempotent (stamps only when the annotation is missing, so a
// re-admission or a client-supplied value is preserved) and is pure
// observability: it only adds a timestamp annotation and never rejects or
// mutates claim spec. Its webhook is therefore registered failurePolicy: Ignore
// (see ensureSandboxClaimMutatingWebhook) so a webhook outage can never block
// claim creation.
type SandboxClaimDefaulter struct {
	// now is injectable for tests; nil defaults to time.Now.
	now func() time.Time
}

// Default implements sigs.k8s.io/controller-runtime/pkg/webhook/admission.Defaulter[*SandboxClaim].
func (d *SandboxClaimDefaulter) Default(_ context.Context, claim *extensionsv1beta1.SandboxClaim) error {
	nowFn := d.now
	if nowFn == nil {
		nowFn = time.Now
	}
	if claim.Annotations == nil {
		claim.Annotations = map[string]string{}
	}
	if claim.Annotations[asmetrics.WebhookAnnotation] == "" {
		claim.Annotations[asmetrics.WebhookAnnotation] = nowFn().UTC().Format(time.RFC3339Nano)
	}
	return nil
}
