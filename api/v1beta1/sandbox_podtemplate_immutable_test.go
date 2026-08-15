//go:build integration
// +build integration

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

package v1beta1_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	sandboxv1beta1 "sigs.k8s.io/agent-sandbox/api/v1beta1"
)

// TestSandboxPodTemplateImmutable asserts that spec.podTemplate is rejected
// on update by the CRD-level CEL rule added alongside volumeClaimTemplates'
// immutability rule. It spins up a local envtest apiserver with the CRDs
// loaded (no controller, no live cluster) so it exercises admission only,
// mirroring TestSandboxVolumeClaimTemplatesImmutable in test/e2e.
func TestSandboxPodTemplateImmutable(t *testing.T) {
	crdPath := filepath.Join("..", "..", "k8s", "crds")

	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{crdPath},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	require.NoError(t, err)
	defer func() { require.NoError(t, testEnv.Stop()) }()

	scheme := runtime.NewScheme()
	require.NoError(t, sandboxv1beta1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
	require.NoError(t, err)

	ctx := t.Context()

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("sandbox-podtemplate-immutable-%d", time.Now().UnixNano())}}
	require.NoError(t, k8sClient.Create(ctx, ns))

	pausePod := sandboxv1beta1.PodTemplate{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "pause", Image: "registry.k8s.io/pause:3.10"}}}}
	otherPod := sandboxv1beta1.PodTemplate{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "pause", Image: "registry.k8s.io/pause:3.9"}}}}

	sb := &sandboxv1beta1.Sandbox{
		ObjectMeta: metav1.ObjectMeta{Name: "podtemplate-immutable", Namespace: ns.Name},
		Spec: sandboxv1beta1.SandboxSpec{SandboxBlueprint: sandboxv1beta1.SandboxBlueprint{
			PodTemplate: pausePod,
		}},
	}
	require.NoError(t, k8sClient.Create(ctx, sb))

	latest := &sandboxv1beta1.Sandbox{}
	require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Name: sb.Name, Namespace: ns.Name}, latest))
	latest.Spec.PodTemplate = otherPod

	err = k8sClient.Update(ctx, latest)
	require.Error(t, err)
	require.Contains(t, err.Error(), "podTemplate.spec is immutable")
}
