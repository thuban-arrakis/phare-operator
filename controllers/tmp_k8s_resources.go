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

func (r *PhareReconciler) handleHTTPRoute(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
	if phare.Spec.ToolChain != nil && phare.Spec.ToolChain.HTTPRoute != nil {
		_, err := r.reconcileHttpRoute(ctx, req, phare)
		return err
	}
	return r.cleanupHTTPRoute(ctx, phare)
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
					return err
				}
				r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "DeletedResource", "Deleted HTTPRoute %s", phare.Name)
			}
		}
	}
	return nil
}

func (r *PhareReconciler) handleGCPBackendPolicy(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
	if phare.Spec.ToolChain != nil && phare.Spec.ToolChain.GCPBackendPolicy != nil {
		_, err := r.reconcileGCPBackendPolicy(ctx, req, phare)
		return err
	}
	return r.cleanupGCPBackendPolicy(ctx, phare)
}

func (r *PhareReconciler) cleanupGCPBackendPolicy(ctx context.Context, phare pharev1beta1.Phare) error {
	gcpBackendPolicyList := &unstructured.UnstructuredList{}
	gcpBackendPolicyList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.gke.io",
		Version: "v1",
		Kind:    "GCPBackendPolicyList",
	})

	if err := r.List(ctx, gcpBackendPolicyList, client.InNamespace(phare.Namespace)); err != nil {
		return err
	}

	for _, gcpBackendPolicy := range gcpBackendPolicyList.Items {
		r.Log.Info("Processing GCPBackendPolicy", "name", gcpBackendPolicy.GetName(), "namespace", gcpBackendPolicy.GetNamespace(), "kind", gcpBackendPolicy.GetKind())
		for _, ownerRef := range gcpBackendPolicy.GetOwnerReferences() {
			if ownerRef.UID == phare.UID {
				if err := r.Delete(ctx, &gcpBackendPolicy); err != nil {
					return err
				}
				r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "DeletedResource", "Deleted GCPBackendPolicy %s", phare.Name)
			}
		}
	}
	return nil
}

func (r *PhareReconciler) handleHealthCheckPolicy(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
	if phare.Spec.ToolChain != nil && phare.Spec.ToolChain.HealthCheckPolicy != nil {
		_, err := r.reconcileHealthCheckPolicy(ctx, req, phare)
		return err
	}
	return r.cleanupHealthCheckPolicy(ctx, phare)
}

func (r *PhareReconciler) cleanupHealthCheckPolicy(ctx context.Context, phare pharev1beta1.Phare) error {
	healthCheckPolicyList := &unstructured.UnstructuredList{}
	healthCheckPolicyList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.gke.io",
		Version: "v1",
		Kind:    "HealthCheckPolicyList",
	})

	if err := r.List(ctx, healthCheckPolicyList, client.InNamespace(phare.Namespace)); err != nil {
		return err
	}

	for _, healthCheckPolicy := range healthCheckPolicyList.Items {
		for _, ownerRef := range healthCheckPolicy.GetOwnerReferences() {
			if ownerRef.UID == phare.UID {
				if err := r.Delete(ctx, &healthCheckPolicy); err != nil {
					return err
				}
				r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "DeletedResource", "Deleted HealthCheckPolicy %s", phare.Name)
			}
		}
	}
	return nil
}

func (r *PhareReconciler) reconcileMicroService(ctx context.Context, phare pharev1beta1.Phare) error {
	switch phare.Spec.MicroService.Kind {
	case "Deployment":
		r.Log.Info("Reconciling Deployment", "Deployment.Namespace", phare.Namespace, "Deployment.Name", phare.Name)
		return r.reconcileDeployment(ctx, phare)
	case "StatefulSet":
		r.Log.Info("Reconciling StatefulSet", "StatefulSet.Namespace", phare.Namespace, "StatefulSet.Name", phare.Name)
		return r.reconcileStatefulSet(ctx, phare)
	default:
		return fmt.Errorf("unsupported kind: %s", phare.Spec.MicroService.Kind)
	}
}
