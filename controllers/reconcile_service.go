package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// reallocateNodePortAnnotation is an opt-in control flag on the Phare resource.
// When true, the controller does not preserve existing NodePort values.
const reallocateNodePortAnnotation = "phare.localcorp.internal/reallocate-nodeport"

// reconcileService creates, updates, or deletes the Service for a Phare resource.
func (r *PhareReconciler) reconcileService(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
	existingService := &corev1.Service{}
	err := r.Get(ctx, req.NamespacedName, existingService)

	// If there's no service specified in the CR and the service doesn't exist in the cluster, just return
	if phare.Spec.Service == nil && errors.IsNotFound(err) {
		return nil
	}

	// If there's no service specified in the CR and the get failed for another reason, return error
	if phare.Spec.Service == nil && err != nil {
		return err
	}

	// If there's no service specified in the CR but the service exists in the cluster, delete it
	if phare.Spec.Service == nil {
		return r.Delete(ctx, existingService)
	}

	desiredService := r.desiredService(&phare)
	if desiredService == nil {
		return fmt.Errorf("failed to build desired Service for %s/%s", phare.Namespace, phare.Name)
	}

	// If the service doesn't exist in the cluster, create it
	if errors.IsNotFound(err) {
		return r.createService(ctx, desiredService)
	}

	// Propagate get failures instead of reconciling against an empty object.
	if err != nil {
		return err
	}

	// Update when spec or controller-managed metadata changed.
	preserveNodePort := !shouldReallocateNodePorts(&phare)
	if serviceSpecsDiffer(&existingService.Spec, &desiredService.Spec, preserveNodePort) ||
		!stringMapsEqualNilEmpty(existingService.Labels, desiredService.Labels) ||
		!stringMapsEqualNilEmpty(existingService.Annotations, desiredService.Annotations) {
		existingService.Spec = mergeServiceSpecPreservingImmutable(
			existingService.Spec,
			desiredService.Spec,
			preserveNodePort,
		)
		existingService.Labels = copyStringMapPreserveNil(desiredService.Labels)
		existingService.Annotations = copyStringMapPreserveNil(desiredService.Annotations)
		return r.updateService(ctx, existingService)
	}

	return nil
}

func (r *PhareReconciler) createService(ctx context.Context, service *corev1.Service) error {
	if err := r.Create(ctx, service); err != nil {
		return err
	}
	r.Log.Info("Service created successfully", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
	return nil
}

func (r *PhareReconciler) updateService(ctx context.Context, service *corev1.Service) error {
	if err := r.Update(ctx, service); err != nil {
		return err
	}
	r.Log.Info("Service updated successfully", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
	return nil
}

func (r *PhareReconciler) desiredService(phare *pharev1beta1.Phare) *corev1.Service {

	// Base labels for resources created by this controller.
	metadataLabels := map[string]string{
		"app":                          phare.Name,
		"app.kubernetes.io/created-by": "phare-controller",
		// "version":                      phare.Spec.Microservice.Image.Tag // Use later for rolling updates
	}

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        phare.Name,
			Namespace:   phare.Namespace,
			Annotations: serviceAnnotationsFromPhare(phare.Annotations),
			Labels:      mergeStringMaps(metadataLabels, phare.Labels),
		},
		Spec: *phare.Spec.Service,
	}

	// Set the service type to ClusterIP if it's not set.
	if service.Spec.Type == "" {
		service.Spec.Type = corev1.ServiceTypeClusterIP
	}

	// Set the service selector to match the labels in the Phare object
	// This will let Kubernetes know that the Service is related to the Phare object
	// and thus will be deleted when the Phare object is deleted.
	service.Spec.Selector = map[string]string{
		"app": phare.Name,
	}

	// Set owner reference for the Service to be the Phare object
	// https://book.kubebuilder.io/reference/using-finalizers.html#finalizer-owners.
	if err := ctrl.SetControllerReference(phare, service, r.Scheme); err != nil {
		r.Log.Error(err, "Failed to set controller reference for Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		return nil
	}

	return service
}

// serviceSpecsDiffer compares managed fields after applying preservation rules.
func serviceSpecsDiffer(existing, desired *corev1.ServiceSpec, preserveNodePort bool) bool {
	normalizedDesired := mergeServiceSpecPreservingImmutable(*existing, *desired, preserveNodePort)

	if !reflect.DeepEqual(existing.Ports, normalizedDesired.Ports) {
		return true
	}

	if !reflect.DeepEqual(existing.Selector, normalizedDesired.Selector) {
		return true
	}

	if !reflect.DeepEqual(existing.Type, normalizedDesired.Type) {
		return true
	}

	return false
}

func mergeServiceSpecPreservingImmutable(existing, desired corev1.ServiceSpec, preserveNodePort bool) corev1.ServiceSpec {
	merged := *desired.DeepCopy()

	merged.ClusterIP = existing.ClusterIP
	merged.ClusterIPs = append([]string(nil), existing.ClusterIPs...)
	merged.IPFamilies = append([]corev1.IPFamily(nil), existing.IPFamilies...)
	merged.IPFamilyPolicy = existing.IPFamilyPolicy
	merged.HealthCheckNodePort = existing.HealthCheckNodePort
	merged.LoadBalancerClass = existing.LoadBalancerClass

	if preserveNodePort && len(existing.Ports) > 0 {
		existingByKey := make(map[string]corev1.ServicePort, len(existing.Ports))
		for _, p := range existing.Ports {
			// Port identity prefers name; fallback is protocol/port.
			// Renaming a port (named<->unnamed) changes this identity and may not preserve NodePort.
			key := p.Name
			if key == "" {
				key = fmt.Sprintf("%s/%d", p.Protocol, p.Port)
			}
			existingByKey[key] = p
		}
		for i := range merged.Ports {
			key := merged.Ports[i].Name
			if key == "" {
				key = fmt.Sprintf("%s/%d", merged.Ports[i].Protocol, merged.Ports[i].Port)
			}
			if current, ok := existingByKey[key]; ok && merged.Ports[i].NodePort == 0 {
				merged.Ports[i].NodePort = current.NodePort
			}
		}
	}

	return merged
}

func shouldReallocateNodePorts(phare *pharev1beta1.Phare) bool {
	if phare == nil {
		return false
	}
	v, ok := phare.Annotations[reallocateNodePortAnnotation]
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func serviceAnnotationsFromPhare(in map[string]string) map[string]string {
	out := copyStringMapPreserveNil(in)
	if out == nil {
		return nil
	}
	delete(out, reallocateNodePortAnnotation)
	if len(out) == 0 {
		return nil
	}
	return out
}
