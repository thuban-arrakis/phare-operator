package controllers

import (
	"context"
	"testing"

	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSpecMatchesDesired(t *testing.T) {
	if !specMatchesDesired(nil, nil) {
		t.Fatalf("expected nil specs to match")
	}
	if specMatchesDesired(nil, map[string]interface{}{}) {
		t.Fatalf("expected nil and empty map to differ")
	}
	if !specMatchesDesired(map[string]interface{}{}, map[string]interface{}{}) {
		t.Fatalf("expected empty maps to match")
	}

	existing := map[string]interface{}{
		"default": map[string]interface{}{
			"timeoutSec": int64(30),
			"extra":      "stale",
		},
	}
	desired := map[string]interface{}{
		"default": map[string]interface{}{
			"timeoutSec": int(30),
		},
	}
	if specMatchesDesired(existing, desired) {
		t.Fatalf("expected stale extra field to be detected as drift")
	}

	existingNoExtra := map[string]interface{}{
		"default": map[string]interface{}{
			"timeoutSec": int64(30),
		},
	}
	if !specMatchesDesired(existingNoExtra, desired) {
		t.Fatalf("expected numeric type normalization to match int and int64")
	}

	existingMissingField := map[string]interface{}{
		"default": map[string]interface{}{
			"timeoutSec": int64(30),
		},
	}
	desiredWithNewField := map[string]interface{}{
		"default": map[string]interface{}{
			"timeoutSec": int(30),
			"port":       int(8080),
		},
	}
	if specMatchesDesired(existingMissingField, desiredWithNewField) {
		t.Fatalf("expected new field in desired to be detected as drift")
	}
}

func TestReconcileCleansUpOwnedPoliciesWhenNotConfigured(t *testing.T) {
	scheme := testScheme(t)
	phare := basePhare("demo", "default")

	ownedLabels := map[string]string{"app": phare.Name, "app.kubernetes.io/created-by": "phare-controller"}

	gcp := &unstructured.Unstructured{}
	gcp.SetGroupVersionKind(schema.GroupVersionKind{Group: "networking.gke.io", Version: "v1", Kind: "GCPBackendPolicy"})
	gcp.SetName(phare.Name)
	gcp.SetNamespace(phare.Namespace)
	gcp.SetLabels(ownedLabels)
	gcp.SetOwnerReferences([]metav1.OwnerReference{controllerOwnerRef(phare)})
	gcp.Object["spec"] = map[string]interface{}{"default": map[string]interface{}{}}

	health := &unstructured.Unstructured{}
	health.SetGroupVersionKind(schema.GroupVersionKind{Group: "networking.gke.io", Version: "v1", Kind: "HealthCheckPolicy"})
	health.SetName(phare.Name)
	health.SetNamespace(phare.Namespace)
	health.SetLabels(ownedLabels)
	health.SetOwnerReferences([]metav1.OwnerReference{controllerOwnerRef(phare)})
	health.Object["spec"] = map[string]interface{}{"default": map[string]interface{}{}}

	r := newTestReconciler(t, scheme, phare, gcp, health)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: phare.Name, Namespace: phare.Namespace}}

	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile policy cleanup: %v", err)
	}

	err := r.Get(context.Background(), client.ObjectKey{Name: phare.Name, Namespace: phare.Namespace}, gcp)
	if err == nil || !errors.IsNotFound(err) {
		t.Fatalf("expected GCPBackendPolicy to be deleted, got err=%v", err)
	}

	err = r.Get(context.Background(), client.ObjectKey{Name: phare.Name, Namespace: phare.Namespace}, health)
	if err == nil || !errors.IsNotFound(err) {
		t.Fatalf("expected HealthCheckPolicy to be deleted, got err=%v", err)
	}
}

func TestCleanupPolicySkipsResourceNotOwnedByPhare(t *testing.T) {
	scheme := testScheme(t)
	phare := basePhare("demo", "default")

	// Matches naming/labels but not owner reference.
	gcp := &unstructured.Unstructured{}
	gcp.SetGroupVersionKind(schema.GroupVersionKind{Group: "networking.gke.io", Version: "v1", Kind: "GCPBackendPolicy"})
	gcp.SetName(phare.Name)
	gcp.SetNamespace(phare.Namespace)
	gcp.SetLabels(map[string]string{"app": phare.Name, "app.kubernetes.io/created-by": "phare-controller"})
	isController := true
	gcp.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: pharev1beta1.GroupVersion.String(),
		Kind:       "Phare",
		Name:       "other",
		UID:        "other-uid",
		Controller: &isController,
	}})
	gcp.Object["spec"] = map[string]interface{}{"default": map[string]interface{}{}}

	r := newTestReconciler(t, scheme, phare, gcp)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: phare.Name, Namespace: phare.Namespace}}

	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile should not fail: %v", err)
	}
	if err := r.Get(context.Background(), req.NamespacedName, gcp); err != nil {
		t.Fatalf("expected non-owned policy to remain, got: %v", err)
	}
}

func TestCleanupOwnedPoliciesEvenWhenLabelsDrifted(t *testing.T) {
	scheme := testScheme(t)
	phare := basePhare("demo", "default")

	gcp := &unstructured.Unstructured{}
	gcp.SetGroupVersionKind(schema.GroupVersionKind{Group: "networking.gke.io", Version: "v1", Kind: "GCPBackendPolicy"})
	gcp.SetName(phare.Name)
	gcp.SetNamespace(phare.Namespace)
	gcp.SetOwnerReferences([]metav1.OwnerReference{controllerOwnerRef(phare)})
	gcp.Object["spec"] = map[string]interface{}{"default": map[string]interface{}{}}

	health := &unstructured.Unstructured{}
	health.SetGroupVersionKind(schema.GroupVersionKind{Group: "networking.gke.io", Version: "v1", Kind: "HealthCheckPolicy"})
	health.SetName(phare.Name)
	health.SetNamespace(phare.Namespace)
	health.SetOwnerReferences([]metav1.OwnerReference{controllerOwnerRef(phare)})
	health.Object["spec"] = map[string]interface{}{"default": map[string]interface{}{}}

	r := newTestReconciler(t, scheme, phare, gcp, health)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: phare.Name, Namespace: phare.Namespace}}

	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile policy cleanup with drifted labels: %v", err)
	}

	err := r.Get(context.Background(), client.ObjectKey{Name: phare.Name, Namespace: phare.Namespace}, gcp)
	if err == nil || !errors.IsNotFound(err) {
		t.Fatalf("expected owned GCPBackendPolicy to be deleted despite label drift, got err=%v", err)
	}
	err = r.Get(context.Background(), client.ObjectKey{Name: phare.Name, Namespace: phare.Namespace}, health)
	if err == nil || !errors.IsNotFound(err) {
		t.Fatalf("expected owned HealthCheckPolicy to be deleted despite label drift, got err=%v", err)
	}
}

func TestReconcileGCPBackendPolicyRemovesStaleSpecFields(t *testing.T) {
	scheme := testScheme(t)
	phare := basePhare("demo", "default")
	phare.Spec.ToolChain = &pharev1beta1.ToolChainSpec{
		GCPBackendPolicy: &pharev1beta1.GCPBackendPolicySpec{
			Default: pharev1beta1.GCPBackendPolicyDefaultSpec{
				TimeoutSec: 60,
			},
			TargetRef: pharev1beta1.GCPBackendPolicyTargetRefSpec{
				Group: "",
				Kind:  "Service",
				Name:  "demo",
			},
		},
	}

	gcp := &unstructured.Unstructured{}
	gcp.SetGroupVersionKind(schema.GroupVersionKind{Group: "networking.gke.io", Version: "v1", Kind: "GCPBackendPolicy"})
	gcp.SetName(phare.Name)
	gcp.SetNamespace(phare.Namespace)
	gcp.SetOwnerReferences([]metav1.OwnerReference{controllerOwnerRef(phare)})
	gcp.SetLabels(map[string]string{"stale": "label"})
	gcp.Object["spec"] = map[string]interface{}{
		"default": map[string]interface{}{
			"timeoutSec": 60,
			"logging": map[string]interface{}{
				"enabled":    true,
				"sampleRate": 100,
			},
		},
		"targetRef": map[string]interface{}{
			"group": "",
			"kind":  "Service",
			"name":  "demo",
		},
	}

	r := newTestReconciler(t, scheme, phare, gcp)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: phare.Name, Namespace: phare.Namespace}}

	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile gcp backend policy strict compare: %v", err)
	}

	current := &unstructured.Unstructured{}
	current.SetGroupVersionKind(schema.GroupVersionKind{Group: "networking.gke.io", Version: "v1", Kind: "GCPBackendPolicy"})
	if err := r.Get(context.Background(), req.NamespacedName, current); err != nil {
		t.Fatalf("get gcp backend policy after reconcile: %v", err)
	}
	spec, ok := current.Object["spec"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected object spec map, got %#v", current.Object["spec"])
	}
	def, ok := spec["default"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected default spec map, got %#v", spec["default"])
	}
	logging, ok := def["logging"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected logging field to be present with zero values, got %#v", def["logging"])
	}
	if logging["enabled"] != false || logging["sampleRate"] != int64(0) {
		t.Fatalf("expected stale logging values to be reset, got %#v", logging)
	}
	if current.GetLabels()["stale"] != "" {
		t.Fatalf("expected stale labels to be removed, got %#v", current.GetLabels())
	}
}
