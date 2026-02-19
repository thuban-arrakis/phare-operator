package controllers

import (
	"context"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func TestDeleteIfExistsSkipsNonOwnedStatefulSet(t *testing.T) {
	scheme := testScheme(t)
	phare := basePhare("demo", "default")

	// A StatefulSet with the same name but owned by a different controller.
	otherSS := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      phare.Name,
			Namespace: phare.Namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1",
				Kind:       "ReplicaSet",
				Name:       "other",
				UID:        "other-uid",
				Controller: func() *bool { v := true; return &v }(),
			}},
		},
	}

	r := newTestReconciler(t, scheme, phare, otherSS)
	if err := r.deleteIfExists(context.Background(), &appsv1.StatefulSet{}, phare.Name, phare.Namespace, phare); err != nil {
		t.Fatalf("deleteIfExists returned unexpected error: %v", err)
	}

	// The StatefulSet must still be present.
	remaining := &appsv1.StatefulSet{}
	if err := r.Get(context.Background(), client.ObjectKey{Name: phare.Name, Namespace: phare.Namespace}, remaining); err != nil {
		t.Fatalf("expected non-owned StatefulSet to remain, got: %v", err)
	}
}

func TestReconcileStatefulSetEmitsWarningOnVCTDrift(t *testing.T) {
	scheme := testScheme(t)
	phare := basePhare("demo", "default")
	phare.Spec.MicroService.Kind = "StatefulSet"

	r := newTestReconciler(t, scheme, phare)
	fakeRecorder := record.NewFakeRecorder(10)
	r.Recorder = fakeRecorder

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: phare.Name, Namespace: phare.Namespace}}

	// First reconcile: creates the StatefulSet with empty VCTs.
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("first reconcile: %v", err)
	}

	// Re-fetch so we have the latest resourceVersion after the status update.
	if err := r.Get(context.Background(), req.NamespacedName, phare); err != nil {
		t.Fatalf("re-fetch phare: %v", err)
	}

	// Add a VolumeClaimTemplate to the Phare spec to simulate drift.
	phare.Spec.MicroService.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "data"},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
		},
	}
	if err := r.Update(context.Background(), phare); err != nil {
		t.Fatalf("update phare: %v", err)
	}

	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("second reconcile: %v", err)
	}

	// Expect a Warning event with reason ImmutableField about VolumeClaimTemplates.
	select {
	case msg := <-fakeRecorder.Events:
		if !strings.Contains(msg, "Warning") || !strings.Contains(msg, "ImmutableField") {
			t.Fatalf("expected Warning ImmutableField event, got: %q", msg)
		}
	default:
		t.Fatal("expected Warning event for VolumeClaimTemplates drift but got none")
	}
}
