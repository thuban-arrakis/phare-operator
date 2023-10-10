package controllers

import (
  "context"
  "fmt"

  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
  "k8s.io/apimachinery/pkg/runtime/schema"
  ctrl "sigs.k8s.io/controller-runtime"
  "sigs.k8s.io/controller-runtime/pkg/client"
  gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// func (r *PhareReconciler) handleConfigMap(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
// 	if phare.Spec.ToolChain.Config != nil {
// 		return r.reconcileConfigMap(ctx, phare)
// 	} else {
// 		return r.cleanupConfigMap(ctx, phare)
// 	}
// }

// func (r *PhareReconciler) cleanupConfigMap(ctx context.Context, phare pharev1beta1.Phare) error {
// 	configMapList := &corev1.ConfigMapList{}
// 	if err := r.List(ctx, configMapList, client.InNamespace(phare.Namespace)); err != nil {
// 		return err
// 	}

// 	for _, configMap := range configMapList.Items {
// 		for _, ownerRef := range configMap.OwnerReferences {
// 			if ownerRef.UID == phare.UID {
// 				if err := r.Delete(ctx, &configMap); err != nil {
// 					r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "DeletedResource", "Deleted ConfigMap %s", configMap.Name)
// 					return err
// 				}
// 			}
// 		}
// 	}
// 	return nil
// }

func (r *PhareReconciler) handleHTTPRoute(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
  if phare.Spec.ToolChain != nil && phare.Spec.ToolChain.HTTPRoute != nil {
    _, err := r.reconcileHttpRoute(ctx, req, phare)
    return err
  } else {
    return r.cleanupHTTPRoute(ctx, phare)
  }
}

func (r *PhareReconciler) cleanupHTTPRoute(ctx context.Context, phare pharev1beta1.Phare) error {
  httpRouteList := &gatewayv1beta1.HTTPRouteList{}
  if err := r.List(ctx, httpRouteList, client.InNamespace(phare.Namespace)); err != nil {
    return err
  }

  for _, httpRoute := range httpRouteList.Items {
    for _, ownerRef := range httpRoute.OwnerReferences {
      if ownerRef.UID == phare.UID {
        if err := r.Delete(ctx, &httpRoute); err != nil {
          r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "DeletedResource", "Deleted HTTPRoute %s", phare.Name)
          return err
        }
      }
    }
  }
  return nil
}

func (r *PhareReconciler) handleGCPBackendPolicy(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
  if phare.Spec.ToolChain != nil && phare.Spec.ToolChain.GCPBackendPolicy != nil {
    _, err := r.reconcileGCPBackendPolicy(ctx, req, phare)
    return err
  } else {
    return r.cleanupGCPBackendPolicy(ctx, phare)
  }
}

func (r *PhareReconciler) cleanupGCPBackendPolicy(ctx context.Context, phare pharev1beta1.Phare) error {
  gcpBackendPolicyList := &unstructured.UnstructuredList{}

  gcpBackendPolicyList.SetGroupVersionKind(schema.GroupVersionKind{
    Group:   "networking.gke.io",
    Version: "v1",
    Kind:    "GCPBackendPolicy",
  })

  if err := r.List(ctx, gcpBackendPolicyList, client.InNamespace(phare.Namespace)); err != nil {
    return err
  }

  for _, gcpBackendPolicy := range gcpBackendPolicyList.Items {
    r.Log.Info("Processing GCPBackendPolicy", "name", gcpBackendPolicy.GetName(), "namespace", gcpBackendPolicy.GetNamespace(), "kind", gcpBackendPolicy.GetKind())
    for _, ownerRef := range gcpBackendPolicy.GetOwnerReferences() {
      if ownerRef.UID == phare.UID {
        if err := r.Delete(ctx, &gcpBackendPolicy); err != nil {
          r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "DeletedResource", "Deleted GCPBackendPolicy %s", phare.Name)
          return err
        }
      }
    }
  }
  return nil
}

// func (r *PhareReconciler) handleService(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) (ctrl.Result, error) {
// 	if phare.Spec.Service != nil {
// 		return r.reconcileService(ctx, req, phare)
// 	} else {
// 		return ctrl.Result{}, r.cleanupService(ctx, phare)
// 	}
// }

// func (r *PhareReconciler) cleanupService(ctx context.Context, phare pharev1beta1.Phare) error {
// 	serviceList := &corev1.ServiceList{}
// 	if err := r.List(ctx, serviceList, client.InNamespace(phare.Namespace)); err != nil {
// 		return err
// 	}

// 	for _, service := range serviceList.Items {
// 		for _, ownerRef := range service.OwnerReferences {
// 			if ownerRef.UID == phare.UID {
// 				if err := r.Delete(ctx, &service); err != nil {
// 					r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "DeletedResource", "Deleted Service %s", phare.Name)
// 					return err
// 				}
// 			}
// 		}
// 	}
// 	return nil
// }

// TODO: Should be optimized to use a switch statement.
// Like, add some deletion logic to the cleanup old resources.
func (r *PhareReconciler) reconcileMicroService(ctx context.Context, phare pharev1beta1.Phare) error {
  switch phare.Spec.MicroService.Kind {
  case "Deployment":
    r.Log.Info("Reconciling Deployment") // TODO: remove this
    return r.reconcileDeployment(ctx, phare)
  case "StatefulSet":
    r.Log.Info("Reconciling StatefulSet") // TODO: remove this
    return r.reconcileStatefulSet(ctx, phare)
  default:
    return fmt.Errorf("unsupported kind: %s", phare.Spec.MicroService.Kind)
  }
}
