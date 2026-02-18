package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func TestReconcileServiceCreateUpdateDelete(t *testing.T) {
	scheme := testScheme(t)
	phare := basePhare("demo", "default")
	phare.Spec.Service = &corev1.ServiceSpec{
		Ports: []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstrFromInt(80)}},
	}

	r := newTestReconciler(t, scheme, phare)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: phare.Name, Namespace: phare.Namespace}}

	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile create service: %v", err)
	}

	created := &corev1.Service{}
	if err := r.Get(context.Background(), req.NamespacedName, created); err != nil {
		t.Fatalf("service should be created: %v", err)
	}
	if len(created.Spec.Ports) != 1 || created.Spec.Ports[0].Port != 80 {
		t.Fatalf("unexpected created service ports: %#v", created.Spec.Ports)
	}

	currentPhare := &pharev1beta1.Phare{}
	if err := r.Get(context.Background(), req.NamespacedName, currentPhare); err != nil {
		t.Fatalf("get phare for update: %v", err)
	}
	currentPhare.Spec.Service = &corev1.ServiceSpec{
		Ports: []corev1.ServicePort{{Name: "http", Port: 8080, TargetPort: intstrFromInt(8080)}},
	}
	if err := r.Update(context.Background(), currentPhare); err != nil {
		t.Fatalf("update phare with new service spec: %v", err)
	}

	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile update service: %v", err)
	}
	updated := &corev1.Service{}
	if err := r.Get(context.Background(), req.NamespacedName, updated); err != nil {
		t.Fatalf("service should exist after update: %v", err)
	}
	if len(updated.Spec.Ports) != 1 || updated.Spec.Ports[0].Port != 8080 {
		t.Fatalf("expected port 8080 after update, got: %#v", updated.Spec.Ports)
	}

	if err := r.Get(context.Background(), req.NamespacedName, currentPhare); err != nil {
		t.Fatalf("get phare for delete: %v", err)
	}
	currentPhare.Spec.Service = nil
	if err := r.Update(context.Background(), currentPhare); err != nil {
		t.Fatalf("update phare without service: %v", err)
	}

	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile delete service: %v", err)
	}
	err := r.Get(context.Background(), req.NamespacedName, &corev1.Service{})
	if err == nil || !errors.IsNotFound(err) {
		t.Fatalf("expected service to be deleted, got err=%v", err)
	}
}

func TestReconcileServiceUpdatePreservesImmutableFields(t *testing.T) {
	scheme := testScheme(t)
	phare := basePhare("demo", "default")
	phare.Spec.Service = &corev1.ServiceSpec{
		Type:  corev1.ServiceTypeNodePort,
		Ports: []corev1.ServicePort{{Name: "http", Port: 8080, TargetPort: intstrFromInt(8080)}},
	}

	existing := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      phare.Name,
			Namespace: phare.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:              corev1.ServiceTypeNodePort,
			ClusterIP:         "10.0.0.10",
			ClusterIPs:        []string{"10.0.0.10"},
			IPFamilies:        []corev1.IPFamily{corev1.IPv4Protocol},
			LoadBalancerClass: ptrTo("internal-lb"),
			Selector:          map[string]string{"app": "demo"},
			Ports:             []corev1.ServicePort{{Name: "http", Port: 80, NodePort: 30080, TargetPort: intstrFromInt(80)}},
			ExternalIPs:       []string{"1.2.3.4"},
		},
	}

	r := newTestReconciler(t, scheme, phare, existing)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: phare.Name, Namespace: phare.Namespace}}

	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile service immutable preservation: %v", err)
	}

	current := &corev1.Service{}
	if err := r.Get(context.Background(), req.NamespacedName, current); err != nil {
		t.Fatalf("get service after reconcile: %v", err)
	}
	if current.Spec.ClusterIP != "10.0.0.10" {
		t.Fatalf("expected clusterIP to be preserved, got %q", current.Spec.ClusterIP)
	}
	if len(current.Spec.ClusterIPs) != 1 || current.Spec.ClusterIPs[0] != "10.0.0.10" {
		t.Fatalf("expected clusterIPs to be preserved, got %#v", current.Spec.ClusterIPs)
	}
	if len(current.Spec.Ports) != 1 || current.Spec.Ports[0].Port != 8080 {
		t.Fatalf("expected service port updated to 8080, got %#v", current.Spec.Ports)
	}
	if current.Spec.Ports[0].NodePort != 30080 {
		t.Fatalf("expected allocated nodePort to be preserved, got %d", current.Spec.Ports[0].NodePort)
	}
	if current.Spec.LoadBalancerClass == nil || *current.Spec.LoadBalancerClass != "internal-lb" {
		t.Fatalf("expected loadBalancerClass to be preserved, got %#v", current.Spec.LoadBalancerClass)
	}
}

func TestReconcileServiceCanReallocateNodePortViaAnnotation(t *testing.T) {
	scheme := testScheme(t)
	phare := basePhare("demo", "default")
	phare.Annotations = map[string]string{reallocateNodePortAnnotation: "true"}
	phare.Spec.Service = &corev1.ServiceSpec{
		Type:  corev1.ServiceTypeNodePort,
		Ports: []corev1.ServicePort{{Name: "http", Port: 8080, TargetPort: intstrFromInt(8080)}},
	}

	existing := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      phare.Name,
			Namespace: phare.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeNodePort,
			ClusterIP: "10.0.0.10",
			Selector:  map[string]string{"app": "demo"},
			Ports:     []corev1.ServicePort{{Name: "http", Port: 80, NodePort: 30080, TargetPort: intstrFromInt(80)}},
		},
	}

	r := newTestReconciler(t, scheme, phare, existing)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: phare.Name, Namespace: phare.Namespace}}

	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile service nodeport reallocate: %v", err)
	}

	current := &corev1.Service{}
	if err := r.Get(context.Background(), req.NamespacedName, current); err != nil {
		t.Fatalf("get service after reconcile: %v", err)
	}
	if len(current.Spec.Ports) != 1 {
		t.Fatalf("expected one service port, got %#v", current.Spec.Ports)
	}
	if current.Spec.Ports[0].NodePort != 0 {
		t.Fatalf("expected nodePort to be left unset for reallocation, got %d", current.Spec.Ports[0].NodePort)
	}
}

func TestReconcileServiceMetadataIsAuthoritative(t *testing.T) {
	scheme := testScheme(t)
	phare := basePhare("demo", "default")
	phare.Annotations = map[string]string{"desired": "true"}
	phare.Labels = map[string]string{"team": "core"}
	phare.Spec.Service = &corev1.ServiceSpec{
		Type:  corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstrFromInt(80)}},
	}

	existing := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        phare.Name,
			Namespace:   phare.Namespace,
			Annotations: map[string]string{"stale": "yes"},
			Labels:      map[string]string{"app": "demo", "stale": "yes"},
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.0.0.10",
			Selector:  map[string]string{"app": "demo"},
			Ports:     []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstrFromInt(80)}},
		},
	}

	r := newTestReconciler(t, scheme, phare, existing)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: phare.Name, Namespace: phare.Namespace}}

	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile service metadata authoritative: %v", err)
	}

	current := &corev1.Service{}
	if err := r.Get(context.Background(), req.NamespacedName, current); err != nil {
		t.Fatalf("get service after reconcile: %v", err)
	}
	if _, ok := current.Annotations["stale"]; ok {
		t.Fatalf("expected stale annotation to be removed, got %#v", current.Annotations)
	}
	if current.Annotations["desired"] != "true" {
		t.Fatalf("expected desired annotation to be applied, got %#v", current.Annotations)
	}
	if _, ok := current.Labels["stale"]; ok {
		t.Fatalf("expected stale label to be removed, got %#v", current.Labels)
	}
}

func TestReconcileCleansUpOwnedPoliciesWhenNotConfigured(t *testing.T) {
	scheme := testScheme(t)
	phare := basePhare("demo", "default")

	gcp := &unstructured.Unstructured{}
	gcp.SetGroupVersionKind(schema.GroupVersionKind{Group: "networking.gke.io", Version: "v1", Kind: "GCPBackendPolicy"})
	gcp.SetName(phare.Name)
	gcp.SetNamespace(phare.Namespace)
	gcp.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: pharev1beta1.GroupVersion.String(),
		Kind:       "Phare",
		Name:       phare.Name,
		UID:        phare.UID,
	}})
	gcp.Object["spec"] = map[string]interface{}{"default": map[string]interface{}{}}

	health := &unstructured.Unstructured{}
	health.SetGroupVersionKind(schema.GroupVersionKind{Group: "networking.gke.io", Version: "v1", Kind: "HealthCheckPolicy"})
	health.SetName(phare.Name)
	health.SetNamespace(phare.Namespace)
	health.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: pharev1beta1.GroupVersion.String(),
		Kind:       "Phare",
		Name:       phare.Name,
		UID:        phare.UID,
	}})
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
	gcp.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: pharev1beta1.GroupVersion.String(),
		Kind:       "Phare",
		Name:       phare.Name,
		UID:        phare.UID,
	}})
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

func TestReconcileDeploymentPreservesMutatedFields(t *testing.T) {
	scheme := testScheme(t)
	phare := basePhare("demo", "default")
	phare.Spec.MicroService.Env = []corev1.EnvVar{{Name: "APP_MODE", Value: "prod"}}

	builder := &PhareReconciler{Scheme: scheme}
	existing := builder.newDeployment(phare)
	if existing == nil {
		t.Fatalf("expected base deployment")
	}

	// Simulate webhook/controller sidecar mutation that should be preserved.
	existing.Spec.Template.Spec.Containers = append(existing.Spec.Template.Spec.Containers, corev1.Container{
		Name:  "istio-proxy",
		Image: "proxyv2:latest",
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "injected-volume",
			MountPath: "/var/run/istio",
		}},
	})
	existing.Spec.Template.Spec.Volumes = append(existing.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: "injected-volume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
	existing.Status.AvailableReplicas = 1
	existing.ManagedFields = append(existing.ManagedFields, metav1.ManagedFieldsEntry{
		Manager:   "istio-sidecar-injector",
		Operation: metav1.ManagedFieldsOperationUpdate,
	})

	r := newTestReconciler(t, scheme, phare, existing)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: phare.Name, Namespace: phare.Namespace}}

	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile deployment preserve mutations: %v", err)
	}

	current := &appsv1.Deployment{}
	if err := r.Get(context.Background(), req.NamespacedName, current); err != nil {
		t.Fatalf("get deployment after reconcile: %v", err)
	}

	if !containerExists(current.Spec.Template.Spec.Containers, "istio-proxy") {
		t.Fatalf("expected injected sidecar to be preserved")
	}
	if !volumeExists(current.Spec.Template.Spec.Volumes, "injected-volume") {
		t.Fatalf("expected injected volume to be preserved")
	}
	if !envVarExists(current.Spec.Template.Spec.Containers[0].Env, "APP_MODE") {
		t.Fatalf("expected desired env var to remain present")
	}
}

func TestReconcileDeploymentRemovesStaleManagedFields(t *testing.T) {
	scheme := testScheme(t)

	old := basePhare("demo", "default")
	old.Spec.MicroService.Env = []corev1.EnvVar{
		{Name: "KEEP", Value: "1"},
		{Name: "REMOVE", Value: "1"},
	}
	old.Spec.MicroService.Affinity = &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{{
					MatchExpressions: []corev1.NodeSelectorRequirement{{
						Key:      "kubernetes.io/os",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"linux"},
					}},
				}},
			},
		},
	}

	builder := &PhareReconciler{Scheme: scheme}
	existing := builder.newDeployment(old)
	if existing == nil {
		t.Fatalf("expected existing deployment")
	}

	updated := old.DeepCopy()
	updated.Spec.MicroService.Env = []corev1.EnvVar{
		{Name: "KEEP", Value: "1"},
	}
	updated.Spec.MicroService.Affinity = nil

	r := newTestReconciler(t, scheme, updated, existing)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: updated.Name, Namespace: updated.Namespace}}
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile stale field removal: %v", err)
	}

	current := &appsv1.Deployment{}
	if err := r.Get(context.Background(), req.NamespacedName, current); err != nil {
		t.Fatalf("get deployment after reconcile: %v", err)
	}
	if envVarExists(current.Spec.Template.Spec.Containers[0].Env, "REMOVE") {
		t.Fatalf("expected stale env var to be removed")
	}
	if current.Spec.Template.Spec.Affinity != nil {
		t.Fatalf("expected affinity to be cleared when removed from spec")
	}
}

func TestReconcileDeploymentUpdatesReplicas(t *testing.T) {
	scheme := testScheme(t)
	old := basePhare("demo", "default")
	old.Spec.MicroService.ReplicaCount = 1

	builder := &PhareReconciler{Scheme: scheme}
	existing := builder.newDeployment(old)
	if existing == nil {
		t.Fatalf("expected existing deployment")
	}

	updated := old.DeepCopy()
	updated.Spec.MicroService.ReplicaCount = 3

	r := newTestReconciler(t, scheme, updated, existing)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: updated.Name, Namespace: updated.Namespace}}
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile deployment replicas: %v", err)
	}

	current := &appsv1.Deployment{}
	if err := r.Get(context.Background(), req.NamespacedName, current); err != nil {
		t.Fatalf("get deployment after reconcile: %v", err)
	}
	if current.Spec.Replicas == nil || *current.Spec.Replicas != 3 {
		t.Fatalf("expected replicas=3, got %#v", current.Spec.Replicas)
	}
}

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

func intstrFromInt(v int) intstr.IntOrString {
	return intstr.FromInt(v)
}

func ptrTo(s string) *string {
	return &s
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
