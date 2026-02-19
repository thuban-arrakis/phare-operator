package controllers

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestReconcileStatefulSetUpdatesReplicas(t *testing.T) {
	scheme := testScheme(t)
	old := basePhare("demo", "default")
	old.Spec.MicroService.Kind = "StatefulSet"
	old.Spec.MicroService.ReplicaCount = 1

	builder := &PhareReconciler{Scheme: scheme}
	existing := builder.newStatefulSet(old)
	if existing == nil {
		t.Fatalf("expected existing statefulset")
	}

	updated := old.DeepCopy()
	updated.Spec.MicroService.ReplicaCount = 4

	r := newTestReconciler(t, scheme, updated, existing)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: updated.Name, Namespace: updated.Namespace}}
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile statefulset replicas: %v", err)
	}

	current := &appsv1.StatefulSet{}
	if err := r.Get(context.Background(), req.NamespacedName, current); err != nil {
		t.Fatalf("get statefulset after reconcile: %v", err)
	}
	if current.Spec.Replicas == nil || *current.Spec.Replicas != 4 {
		t.Fatalf("expected replicas=4, got %#v", current.Spec.Replicas)
	}
}
