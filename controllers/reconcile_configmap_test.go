package controllers

import (
	"context"
	"testing"

	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGenerateConfigMapDoesNotMutateSpecConfig(t *testing.T) {
	scheme := testScheme(t)

	r := &PhareReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme: scheme,
	}

	phare := pharev1beta1.Phare{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "default",
		},
		Spec: pharev1beta1.PhareSpec{
			ToolChain: &pharev1beta1.ToolChainSpec{
				Config: pharev1beta1.ConfigSpec{
					"raw":       "plain",
					"templated": "/{{ .Name }}/ready",
				},
			},
		},
	}

	cm := r.generateConfigMap(phare)
	if cm.Data["templated"] != "/app/ready" {
		t.Fatalf("expected templated value to be rendered, got %q", cm.Data["templated"])
	}
	if phare.Spec.ToolChain.Config["templated"] != "/{{ .Name }}/ready" {
		t.Fatalf("expected source spec config to remain unchanged, got %q", phare.Spec.ToolChain.Config["templated"])
	}
}

// TestReconcileConfigMapDeleteSucceeds verifies that reconcileConfigMap deletes
// an existing ConfigMap when ToolChain is removed from the Phare spec, exercising
// the Delete code path including the IsNotFound guard added for TOCTOU safety.
func TestReconcileConfigMapDeleteSucceeds(t *testing.T) {
	scheme := testScheme(t)

	// Phare with no ToolChain — reconciler must delete the ConfigMap.
	phare := basePhare("app", "default")

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-config",
			Namespace: "default",
		},
	}
	r := &PhareReconciler{
		Client:   fake.NewClientBuilder().WithScheme(scheme).WithObjects(phare, cm).Build(),
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(10),
	}

	// reconcileConfigMap should delete the ConfigMap and return no error.
	if err := r.reconcileConfigMap(context.Background(), *phare); err != nil {
		t.Fatalf("expected no error on ConfigMap deletion, got: %v", err)
	}

	// ConfigMap should now be absent.
	if err := r.Get(context.Background(), client.ObjectKey{Name: "app-config", Namespace: "default"}, &corev1.ConfigMap{}); err == nil {
		t.Fatal("expected ConfigMap to be deleted")
	}
}
