package controllers

import (
  "context"
  "fmt"

  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
  "github.com/localcorp/phare-controller/pkg/validator"
  yamldiff "github.com/localcorp/phare-controller/pkg/yamldiff"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/api/errors"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  ctrl "sigs.k8s.io/controller-runtime"
  "sigs.k8s.io/controller-runtime/pkg/client"
  gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
  "sigs.k8s.io/yaml"
)

func (r *PhareReconciler) reconcileHttpRoute(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) (ctrl.Result, error) {
  existingHttpRoute := &gatewayv1beta1.HTTPRoute{}
  desired := r.desiredHttpRoute(&phare)
  err := r.Get(ctx, req.NamespacedName, existingHttpRoute)

  // Convert existing and desired spec to YAML for comparison
  existingHttpRouteSpecYAML := toYAML(existingHttpRoute.Spec)
  desiredServiceSpecYAML := toYAML(desired.Spec)

  if err != nil {
    if errors.IsNotFound(err) {
      // HTTPRoute doesn't exist, create it
      if err := r.Create(ctx, desired); err != nil {
        return ctrl.Result{}, err
      }
      r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "CreatedResource", "Created HTTPRoute %s", desired.Name)
      return ctrl.Result{}, nil
    }
    return ctrl.Result{}, err
  }

  isValid, desiredMap, modifiedCurrentMap := validator.ValidateYaml(desiredServiceSpecYAML, existingHttpRouteSpecYAML)

  if !isValid {
    r.Log.Info("HTTPRoute does not match the desired configuration", "HTTPRoute.Namespace", desired.Namespace, "HTTPRoute.Name", desired.Name)

    // Log the diff for debugging
    diffOutput := yamldiff.Diff(validator.PrintMap(modifiedCurrentMap), validator.PrintMap(desiredMap))
    r.Log.Info("Diff between current and desired configuration", "diff", diffOutput)

    patch := client.MergeFrom(existingHttpRoute.DeepCopy())
    r.Log.Info("Updating HTTPRoute", "HTTPRoute.Namespace", existingHttpRoute.Namespace, "HTTPRoute.Name", existingHttpRoute.Name)

    // Copy desired service's spec to existingHttpRoute
    existingHttpRoute.Spec = desired.Spec
    // Add or update labels in the existingHttpRoute
    for key, value := range desired.ObjectMeta.Labels {
      existingHttpRoute.ObjectMeta.Labels[key] = value
    }

    if err := r.Patch(ctx, existingHttpRoute, patch, client.FieldOwner("phare-controller")); err != nil {
      return ctrl.Result{}, err
    }
    return ctrl.Result{}, nil
  } else {
    r.Log.Info("HTTPRoute matches the desired configuration", "HTTPRoute.Namespace", desired.Namespace, "HTTPRoute.Name", desired.Name)
  }

  return ctrl.Result{}, nil
}

func (r *PhareReconciler) desiredHttpRoute(phare *pharev1beta1.Phare) *gatewayv1beta1.HTTPRoute {
  // Keep the same labels at the metadata level
  metadataLabels := map[string]string{
    "app":                          phare.Name,
    "app.kubernetes.io/created-by": "phare-controller",
  }

  httpRoute := &gatewayv1beta1.HTTPRoute{
    TypeMeta: metav1.TypeMeta{
      APIVersion: "gateway.networking.k8s.io/v1beta1", // Now it's hard-coded, but it should be a variable or generated
      Kind:       "HTTPRoute",                         // Same here
    },
    ObjectMeta: metav1.ObjectMeta{
      Name:      phare.Name,
      Namespace: phare.Namespace,
      Labels:    metadataLabels,
    },
    Spec: gatewayv1beta1.HTTPRouteSpec{
      Hostnames: phare.Spec.ToolChain.HTTPRoute.Hostnames,
      Rules:     phare.Spec.ToolChain.HTTPRoute.Rules,
      CommonRouteSpec: gatewayv1beta1.CommonRouteSpec{
        ParentRefs: phare.Spec.ToolChain.HTTPRoute.ParentRef,
      },
    },
  }

  // Convert httpRoute to YAML and print it
  yamlData, err := yaml.Marshal(httpRoute)
  if err != nil {
    fmt.Println("Failed to Marshal HTTPRoute to YAML:", err)
    return httpRoute
  }

  fmt.Println("Resulting HTTPRoute YAML:")
  fmt.Println(string(yamlData))

  return httpRoute
}
