package controllers

import (
  "context"
  "fmt"

  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
  "github.com/localcorp/phare-controller/pkg/validator"
  util "github.com/localcorp/phare-controller/util"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/api/errors"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  ctrl "sigs.k8s.io/controller-runtime"
  "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *PhareReconciler) reconcileService(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) (ctrl.Result, error) {
  existingService := &corev1.Service{}
  desired := r.desiredService(&phare)
  err := r.Get(ctx, req.NamespacedName, existingService)

  a := toYAML(existingService.Spec) // Rename it later
  // fmt.Println("a: ", a)
  b := toYAML(desired.Spec) // Rename it later
  // fmt.Println("b: ", b)

  if err != nil {
    if errors.IsNotFound(err) {
      // Service doesn't exist, create it
      if err := r.Create(ctx, desired); err != nil {
        return ctrl.Result{}, err
      }
      r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "CreatedResource", "Created Service %s", desired.Name)
      return ctrl.Result{}, nil
    } else {
      return ctrl.Result{}, err
    }
  } else {
    isValid, desiredMap, modifiedCurrentMap := validator.ValidateYaml(b, a)
    map1 := validator.PrintMap(modifiedCurrentMap) // Debugging purposes only
    map2 := validator.PrintMap(desiredMap)         // Debugging purposes only
    // validator.PrintMap(desiredMap)                          // Debugging purposes only
    // diffOutput := util.Diff(map1, map2) // Debugging purposes only
    // fmt.Println(diffOutput)             // Debugging purposes only

    if !isValid {
      // validator.PrintMap("Modified Current Map:", modifiedCurrentMap)
      // validator.PrintMap("Desired Map:", desiredMap)
      // d := util.GetDiff(desiredMap, modifiedCurrentMap) // Debug purposes only
      // fmt.Println(util.ToYAML(d, 0))                    // Debug purposes only
      diffOutput := util.Diff(map1, map2) // Debugging purposes only
      fmt.Println(diffOutput)             // Debugging purposes only
      patch := client.MergeFrom(existingService.DeepCopy())
      r.Log.Info("Updating Service", "Service.Namespace", existingService.Namespace, "Service.Name", existingService.Name)
      existingService.Spec = desired.Spec
      if err := r.Patch(ctx, existingService, patch, client.FieldOwner("phare-controller")); err != nil {
        return ctrl.Result{}, err
      }
      return ctrl.Result{}, nil
    } else {
      r.Log.Info("Service matches the desired configuration", "Service.Namespace", desired.Namespace, "Service.Name", desired.Name)
    }
  }

  return ctrl.Result{}, nil
}

func (r *PhareReconciler) desiredService(phare *pharev1beta1.Phare) *corev1.Service {
  labels := map[string]string{
    "app": phare.Name,
  }

  serviceType := corev1.ServiceTypeClusterIP
  if phare.Spec.Service.Type != "" {
    serviceType = phare.Spec.Service.Type
  }

  service := &corev1.Service{
    ObjectMeta: metav1.ObjectMeta{
      Name:        phare.Name,
      Namespace:   phare.Namespace,
      Annotations: phare.Spec.Service.Annotations,
      Labels:      mergeMaps(labels, phare.Spec.Service.Labels),
    },
    Spec: corev1.ServiceSpec{
      Selector: labels,
      Type:     serviceType,
      Ports:    phare.Spec.Service.Ports,
    },
  }

  // Set owner reference for the service to be the Phare object
  if err := ctrl.SetControllerReference(phare, service, r.Scheme); err != nil {
    r.Log.Error(err, "Failed to set controller reference for Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
    return nil
  }

  return service
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
