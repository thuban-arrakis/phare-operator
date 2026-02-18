package controllers

import (
	"context"
	"testing"

	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	if serviceSpecsDiffer(&base, same, true) {
		t.Fatalf("expected same specs to not differ")
	}

	changed := base.DeepCopy()
	changed.Type = corev1.ServiceTypeNodePort
	if !serviceSpecsDiffer(&base, changed, true) {
		t.Fatalf("expected differing specs to be detected")
	}

	withNodePort := base.DeepCopy()
	withNodePort.Ports[0].NodePort = 30080
	desiredOmitNodePort := base.DeepCopy()
	if serviceSpecsDiffer(withNodePort, desiredOmitNodePort, true) {
		t.Fatalf("expected preserved nodePort to not trigger diff")
	}
}

func TestMergeStringMaps(t *testing.T) {
	existing := map[string]string{"a": "1", "b": "old"}
	desired := map[string]string{"b": "new", "c": "3"}

	merged := mergeStringMaps(existing, desired)
	if merged["a"] != "1" || merged["c"] != "3" {
		t.Fatalf("expected keys from both maps in merge result")
	}
	if merged["b"] != "new" {
		t.Fatalf("expected desired map to override existing key")
	}
}

func TestCopyStringMapPreserveNil(t *testing.T) {
	if out := copyStringMapPreserveNil(nil); out != nil {
		t.Fatalf("expected nil map to stay nil, got %#v", out)
	}
	empty := map[string]string{}
	out := copyStringMapPreserveNil(empty)
	if out == nil || len(out) != 0 {
		t.Fatalf("expected empty non-nil map copy, got %#v", out)
	}
}

func TestStringMapsEqualNilEmpty(t *testing.T) {
	if !stringMapsEqualNilEmpty(nil, map[string]string{}) {
		t.Fatalf("expected nil and empty maps to be considered equal")
	}
	if stringMapsEqualNilEmpty(map[string]string{"a": "1"}, map[string]string{}) {
		t.Fatalf("expected differing maps to be considered different")
	}
}

func TestShouldReallocateNodePorts(t *testing.T) {
	phare := &pharev1beta1.Phare{}
	if shouldReallocateNodePorts(phare) {
		t.Fatalf("expected no reallocation annotation to be false")
	}

	phare.Annotations = map[string]string{reallocateNodePortAnnotation: "true"}
	if !shouldReallocateNodePorts(phare) {
		t.Fatalf("expected true annotation to enable reallocation")
	}

	phare.Annotations[reallocateNodePortAnnotation] = "false"
	if shouldReallocateNodePorts(phare) {
		t.Fatalf("expected false annotation to disable reallocation")
	}
}

func TestSpecMatchesDesired(t *testing.T) {
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
}

func TestServiceAnnotationsFromPhare(t *testing.T) {
	in := map[string]string{
		"owner":                      "team-a",
		reallocateNodePortAnnotation: "true",
	}
	out := serviceAnnotationsFromPhare(in)
	if out == nil {
		t.Fatalf("expected filtered annotations map")
	}
	if out["owner"] != "team-a" {
		t.Fatalf("expected owner annotation to remain, got %#v", out)
	}
	if _, ok := out[reallocateNodePortAnnotation]; ok {
		t.Fatalf("expected control annotation to be removed, got %#v", out)
	}
}

func TestGenerateConfigMapDoesNotMutateSpecConfig(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := pharev1beta1.AddToScheme(scheme); err != nil {
		t.Fatalf("add phare scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}

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

func TestStartupProbeIsAppliedToWorkloads(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := pharev1beta1.AddToScheme(scheme); err != nil {
		t.Fatalf("add phare scheme: %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add apps scheme: %v", err)
	}

	r := &PhareReconciler{Scheme: scheme}

	startup := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/startup",
				Port: intstr.FromInt(8080),
			},
		},
	}

	phare := &pharev1beta1.Phare{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "default",
			UID:       "uid-1",
		},
		Spec: pharev1beta1.PhareSpec{
			MicroService: pharev1beta1.MicroServiceSpec{
				Kind:         "Deployment",
				ReplicaCount: 1,
				Image: pharev1beta1.ImageSpec{
					Repository: "nginx",
					Tag:        "1.27",
				},
				StartupProbe: startup,
			},
		},
	}

	deploy := r.newDeployment(phare)
	if deploy == nil {
		t.Fatalf("expected deployment to be created")
	}
	if deploy.Spec.Template.Spec.Containers[0].StartupProbe == nil {
		t.Fatalf("expected startupProbe on deployment container")
	}

	stateful := r.newStatefulSet(phare)
	if stateful == nil {
		t.Fatalf("expected statefulset to be created")
	}
	if stateful.Spec.Template.Spec.Containers[0].StartupProbe == nil {
		t.Fatalf("expected startupProbe on statefulset container")
	}
}
