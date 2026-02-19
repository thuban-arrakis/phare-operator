package controllers

import (
	"context"
	"testing"

	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestStartupProbeIsAppliedToWorkloads(t *testing.T) {
	scheme := testScheme(t)
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
