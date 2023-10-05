package controllers

import (
  "context"
  "reflect"

  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
  corev1 "k8s.io/api/core/v1"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  ctrl "sigs.k8s.io/controller-runtime"
  "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *PhareReconciler) reconcileService(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
  existingService := &corev1.Service{}
  err := r.Get(ctx, req.NamespacedName, existingService)
  serviceNotFound := err != nil && client.IgnoreNotFound(err) == nil

  if phare.Spec.Service == nil {
    if !serviceNotFound {
      return r.Delete(ctx, existingService)
    }
    return nil
  }

  desiredService := r.desiredService(&phare)

  if serviceNotFound {
    return r.createService(ctx, desiredService)
  }

  if serviceSpecsDiffer(&existingService.Spec, &desiredService.Spec) {
    existingService.Spec = desiredService.Spec
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

  // Keep the same labels at the metadata level
  metadataLabels := map[string]string{
    "app":                          phare.Name,
    "app.kubernetes.io/created-by": "phare-controller",
    // "version":                      phare.Spec.Microservice.Image.Tag // Use later for rolling updates
  }

  service := &corev1.Service{
    ObjectMeta: metav1.ObjectMeta{
      Name:        phare.Name,
      Namespace:   phare.Namespace,
      Annotations: phare.Annotations,
      Labels:      mergeMaps(metadataLabels, phare.Labels), // Note: This will override your static metadataLabels if the same keys are used in phare.Spec.Service.Labels
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

// If you want to compare the ServiceSpec fields, you can use this function
// to check if the existing and desired ServiceSpecs differ.
// Func returns true if the ServiceSpecs differ, false otherwise.
func serviceSpecsDiffer(existing, desired *corev1.ServiceSpec) bool {
  if !reflect.DeepEqual(existing.Ports, desired.Ports) {
    return true
  }

  if !reflect.DeepEqual(existing.Selector, desired.Selector) {
    return true
  }

  if !reflect.DeepEqual(existing.Type, desired.Type) {
    return true
  }

  // Add comparisons for other fields you care about

  return false
}

func mergeMaps(map1, map2 map[string]string) map[string]string {
  result := make(map[string]string)
  for k, v := range map1 {
    result[k] = v
  }
  for k, v := range map2 {
    result[k] = v
  }
  return result
}
