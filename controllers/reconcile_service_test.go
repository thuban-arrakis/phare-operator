package controllers

import (
	"context"
	"reflect"
	"testing"

	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestDesiredServiceDefaultsAndOwnerRef(t *testing.T) {
	scheme := testScheme(t)
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

	existingWithPort := base.DeepCopy()
	existingWithPort.Ports[0].NodePort = 30080
	desiredWithDifferentPort := base.DeepCopy()
	desiredWithDifferentPort.Ports[0].NodePort = 30090
	if !serviceSpecsDiffer(existingWithPort, desiredWithDifferentPort, false) {
		t.Fatalf("expected nodePort difference to be detected when preservation is disabled")
	}

	existingSession := base.DeepCopy()
	existingSession.SessionAffinity = corev1.ServiceAffinityNone
	desiredSession := base.DeepCopy()
	desiredSession.SessionAffinity = corev1.ServiceAffinityClientIP
	if !serviceSpecsDiffer(existingSession, desiredSession, true) {
		t.Fatalf("expected sessionAffinity change to be detected")
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

func TestServiceAnnotationsFromPhare(t *testing.T) {
	if out := serviceAnnotationsFromPhare(nil); out != nil {
		t.Fatalf("expected nil input to return nil, got %#v", out)
	}

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

	onlyControl := map[string]string{reallocateNodePortAnnotation: "true"}
	if out := serviceAnnotationsFromPhare(onlyControl); out != nil {
		t.Fatalf("expected nil when only control annotation present, got %#v", out)
	}
}

func TestNormalizeServiceSpecForDiff(t *testing.T) {
	boolTrue := true
	intTrafficPolicy := corev1.ServiceInternalTrafficPolicyLocal
	clientIPAffinity := corev1.ServiceAffinityClientIP
	clientIPCfg := &corev1.SessionAffinityConfig{
		ClientIP: &corev1.ClientIPConfig{
			TimeoutSeconds: ptrInt32(10800),
		},
	}

	existing := corev1.ServiceSpec{
		ExternalIPs:                   []string{"1.2.3.4"},
		LoadBalancerSourceRanges:      []string{"10.0.0.0/24"},
		SessionAffinityConfig:         clientIPCfg,
		InternalTrafficPolicy:         &intTrafficPolicy,
		AllocateLoadBalancerNodePorts: &boolTrue,
		SessionAffinity:               clientIPAffinity,
		ExternalTrafficPolicy:         corev1.ServiceExternalTrafficPolicyTypeLocal,
		LoadBalancerIP:                "34.1.2.3",
		ExternalName:                  "example.org",
	}

	t.Run("nil desired slices keep existing", func(t *testing.T) {
		desired := corev1.ServiceSpec{}
		got := normalizeServiceSpecForDiff(existing, desired)
		if !reflect.DeepEqual(got.ExternalIPs, existing.ExternalIPs) {
			t.Fatalf("expected externalIPs to be kept, got %#v", got.ExternalIPs)
		}
		if !reflect.DeepEqual(got.LoadBalancerSourceRanges, existing.LoadBalancerSourceRanges) {
			t.Fatalf("expected loadBalancerSourceRanges to be kept, got %#v", got.LoadBalancerSourceRanges)
		}
	})

	t.Run("empty desired slices clear existing", func(t *testing.T) {
		desired := corev1.ServiceSpec{
			ExternalIPs:              []string{},
			LoadBalancerSourceRanges: []string{},
		}
		got := normalizeServiceSpecForDiff(existing, desired)
		if got.ExternalIPs == nil || len(got.ExternalIPs) != 0 {
			t.Fatalf("expected empty externalIPs to remain empty, got %#v", got.ExternalIPs)
		}
		if got.LoadBalancerSourceRanges == nil || len(got.LoadBalancerSourceRanges) != 0 {
			t.Fatalf("expected empty loadBalancerSourceRanges to remain empty, got %#v", got.LoadBalancerSourceRanges)
		}
	})

	t.Run("nil desired pointers keep existing", func(t *testing.T) {
		desired := corev1.ServiceSpec{}
		got := normalizeServiceSpecForDiff(existing, desired)
		if got.SessionAffinityConfig == nil || got.SessionAffinityConfig.ClientIP == nil {
			t.Fatalf("expected sessionAffinityConfig to be kept, got %#v", got.SessionAffinityConfig)
		}
		if got.InternalTrafficPolicy == nil || *got.InternalTrafficPolicy != *existing.InternalTrafficPolicy {
			t.Fatalf("expected internalTrafficPolicy to be kept, got %#v", got.InternalTrafficPolicy)
		}
		if got.AllocateLoadBalancerNodePorts == nil || *got.AllocateLoadBalancerNodePorts != *existing.AllocateLoadBalancerNodePorts {
			t.Fatalf("expected allocateLoadBalancerNodePorts to be kept, got %#v", got.AllocateLoadBalancerNodePorts)
		}
	})

	t.Run("empty desired strings keep existing", func(t *testing.T) {
		desired := corev1.ServiceSpec{
			SessionAffinity:       "",
			ExternalTrafficPolicy: "",
			LoadBalancerIP:        "",
			ExternalName:          "",
		}
		got := normalizeServiceSpecForDiff(existing, desired)
		if got.SessionAffinity != existing.SessionAffinity {
			t.Fatalf("expected sessionAffinity to be kept, got %q", got.SessionAffinity)
		}
		if got.ExternalTrafficPolicy != existing.ExternalTrafficPolicy {
			t.Fatalf("expected externalTrafficPolicy to be kept, got %q", got.ExternalTrafficPolicy)
		}
		if got.LoadBalancerIP != existing.LoadBalancerIP {
			t.Fatalf("expected loadBalancerIP to be kept, got %q", got.LoadBalancerIP)
		}
		if got.ExternalName != existing.ExternalName {
			t.Fatalf("expected externalName to be kept, got %q", got.ExternalName)
		}
	})
}

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
	if _, ok := current.Annotations[reallocateNodePortAnnotation]; ok {
		t.Fatalf("expected control annotation not to be copied to Service, got %#v", current.Annotations)
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
