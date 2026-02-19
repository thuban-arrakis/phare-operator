package controllers

import (
	"testing"

	"github.com/go-logr/logr"
	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := pharev1beta1.AddToScheme(scheme); err != nil {
		t.Fatalf("add phare scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add apps scheme: %v", err)
	}
	if err := gatewayv1beta1.Install(scheme); err != nil {
		t.Fatalf("add gateway scheme: %v", err)
	}

	// Register custom unstructured policy kinds used by reconcile/cleanup logic.
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "networking.gke.io", Version: "v1", Kind: "GCPBackendPolicy"}, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "networking.gke.io", Version: "v1", Kind: "GCPBackendPolicyList"}, &unstructured.UnstructuredList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "networking.gke.io", Version: "v1", Kind: "HealthCheckPolicy"}, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "networking.gke.io", Version: "v1", Kind: "HealthCheckPolicyList"}, &unstructured.UnstructuredList{})

	return scheme
}

func newTestReconciler(t *testing.T, scheme *runtime.Scheme, objs ...client.Object) *PhareReconciler {
	t.Helper()
	builder := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&pharev1beta1.Phare{})
	if len(objs) > 0 {
		builder = builder.WithObjects(objs...)
	}

	return &PhareReconciler{
		Client:   builder.Build(),
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: record.NewFakeRecorder(50),
	}
}

func basePhare(name, namespace string) *pharev1beta1.Phare {
	return &pharev1beta1.Phare{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID("phare-" + name),
		},
		Spec: pharev1beta1.PhareSpec{
			MicroService: pharev1beta1.MicroServiceSpec{
				Kind:         "Deployment",
				ReplicaCount: 1,
				Image: pharev1beta1.ImageSpec{
					Repository: "nginx",
					Tag:        "latest",
				},
				Ports: []corev1.ContainerPort{{ContainerPort: 80}},
			},
		},
	}
}

func ptrInt32(v int32) *int32 {
	return &v
}

func ptrTo(s string) *string {
	return &s
}

func intstrFromInt(v int) intstr.IntOrString {
	return intstr.FromInt(v)
}

func containerExists(items []corev1.Container, name string) bool {
	for _, item := range items {
		if item.Name == name {
			return true
		}
	}
	return false
}

func envVarExists(items []corev1.EnvVar, name string) bool {
	for _, item := range items {
		if item.Name == name {
			return true
		}
	}
	return false
}

func volumeExists(items []corev1.Volume, name string) bool {
	for _, item := range items {
		if item.Name == name {
			return true
		}
	}
	return false
}

func controllerOwnerRef(phare *pharev1beta1.Phare) metav1.OwnerReference {
	return *metav1.NewControllerRef(phare, pharev1beta1.GroupVersion.WithKind("Phare"))
}
