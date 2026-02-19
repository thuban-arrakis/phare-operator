package controllers

import (
	"context"
	"testing"

	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFetchPhareResourceNotFound(t *testing.T) {
	scheme := testScheme(t)

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
	scheme := testScheme(t)

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

func TestDefaultLabelPredicateUpdateTriggersOnLabelRemoval(t *testing.T) {
	pred := defaultLabelPredicate("app.kubernetes.io/created-by", "phare-controller")

	oldObj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cfg",
			Namespace: "default",
			Labels:    map[string]string{"app.kubernetes.io/created-by": "phare-controller"},
		},
	}
	newObj := oldObj.DeepCopy()
	newObj.Labels = map[string]string{}

	if !pred.Update(event.UpdateEvent{ObjectOld: oldObj, ObjectNew: newObj}) {
		t.Fatalf("expected update predicate to trigger when managed label is removed")
	}
}
