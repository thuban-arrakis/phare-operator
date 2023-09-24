package controllers

import (
  "context"
  "reflect"

  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/api/errors"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  ctrl "sigs.k8s.io/controller-runtime"

  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
)

func (r *PhareReconciler) reconcileService(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) (ctrl.Result, error) {
  existingService := &corev1.Service{}
  err := r.Get(ctx, req.NamespacedName, existingService)
  if err != nil && errors.IsNotFound(err) {
    // Handle the scenario where the service does not exist

    desiredService := r.desiredService(&phare)
    if err := r.Create(ctx, desiredService); err != nil {
      return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
  } else if err != nil {
    return ctrl.Result{}, err
  }

  // Compare the existing service with the desired state
  desired := r.desiredService(&phare)
  if !reflect.DeepEqual(existingService.Spec, desired.Spec) {
    existingService.Spec = desired.Spec
    if err := r.Update(ctx, existingService); err != nil {
      return ctrl.Result{}, err
    }
  }

  return ctrl.Result{}, nil
}

func (r *PhareReconciler) desiredService(phare *pharev1beta1.Phare) *corev1.Service {
  // Define the desired state of the Service

  labels := map[string]string{
    "app": phare.Name,
  }

  // Default service type is ClusterIP, but can be overridden by the Phare spec
  serviceType := corev1.ServiceTypeClusterIP
  if phare.Spec.Service.Type != "" {
    serviceType = phare.Spec.Service.Type
  }

  service := &corev1.Service{
    ObjectMeta: metav1.ObjectMeta{
      Name:        phare.Name,
      Namespace:   phare.Namespace,
      Annotations: phare.Spec.Service.Annotations,
      Labels:      mergeMaps(labels, phare.Spec.Service.Labels), // Merge custom labels with default ones
    },
    Spec: corev1.ServiceSpec{
      Selector: labels,
      Type:     serviceType,
      Ports:    phare.Spec.Service.Ports,
    },
  }

  return service
}

// Helper function to merge two maps. In case of conflicts, values from the second map overwrite those in the first.
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
