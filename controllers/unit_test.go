package controllers

import (
	"context"
	"testing"

	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFetchPhareResourceNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := pharev1beta1.AddToScheme(scheme); err != nil {
		t.Fatalf("add phare scheme: %v", err)
	}

	r := &PhareReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme: scheme,
	}

	var phare pharev1beta1.Phare
	found, err := r.fetchPhareResource(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "missing", Namespace: "default"},
	}, &phare)
	if err != nil {
		t.Fatalf("fetch should not return error on not found: %v", err)
	}
	if found {
		t.Fatalf("expected found=false for missing resource")
	}
}

func TestFetchPhareResourceFound(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := pharev1beta1.AddToScheme(scheme); err != nil {
		t.Fatalf("add phare scheme: %v", err)
	}

	obj := &pharev1beta1.Phare{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "default",
		},
	}

	r := &PhareReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj).Build(),
		Scheme: scheme,
	}

	var phare pharev1beta1.Phare
	found, err := r.fetchPhareResource(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "app", Namespace: "default"},
	}, &phare)
	if err != nil {
		t.Fatalf("fetch should not fail: %v", err)
	}
	if !found {
		t.Fatalf("expected found=true for existing resource")
	}
	if phare.Name != "app" {
		t.Fatalf("unexpected object loaded: got %q", phare.Name)
	}
}

func TestDesiredServiceDefaultsAndOwnerRef(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := pharev1beta1.AddToScheme(scheme); err != nil {
		t.Fatalf("add phare scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}

	r := &PhareReconciler{Scheme: scheme}

	phare := &pharev1beta1.Phare{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "default",
			UID:       "uid-1",
		},
		Spec: pharev1beta1.PhareSpec{
			Service: &corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{Name: "http", Port: 80}},
			},
		},
	}

	svc := r.desiredService(phare)
	if svc == nil {
		t.Fatalf("expected desired service to be created")
	}
	if svc.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Fatalf("expected default type ClusterIP, got %q", svc.Spec.Type)
	}
	if got := svc.Spec.Selector["app"]; got != "app" {
		t.Fatalf("expected selector app=app, got %q", got)
	}
	if len(svc.OwnerReferences) == 0 {
		t.Fatalf("expected owner reference to be set")
	}
}

func TestServiceSpecsDiffer(t *testing.T) {
	base := corev1.ServiceSpec{
		Type:     corev1.ServiceTypeClusterIP,
		Selector: map[string]string{"app": "a"},
		Ports:    []corev1.ServicePort{{Name: "http", Port: 80}},
	}

	same := base.DeepCopy()
	if serviceSpecsDiffer(&base, same) {
		t.Fatalf("expected same specs to not differ")
	}

	changed := base.DeepCopy()
	changed.Type = corev1.ServiceTypeNodePort
	if !serviceSpecsDiffer(&base, changed) {
		t.Fatalf("expected differing specs to be detected")
	}
}

func TestMergeLabelMaps(t *testing.T) {
	existing := map[string]string{"a": "1", "b": "old"}
	desired := map[string]string{"b": "new", "c": "3"}

	merged := mergeLabelMaps(existing, desired)
	if merged["a"] != "1" || merged["c"] != "3" {
		t.Fatalf("expected keys from both maps in merge result")
	}
	if merged["b"] != "new" {
		t.Fatalf("expected desired map to override existing key")
	}
}
