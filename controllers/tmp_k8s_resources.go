package controllers

import (
	"context"
	"fmt"

	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func (r *PhareReconciler) handleHTTPRoute(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
	if phare.Spec.ToolChain != nil && phare.Spec.ToolChain.HTTPRoute != nil {
		return r.reconcileHttpRoute(ctx, req, phare)
	}
	return r.cleanupHTTPRoute(ctx, phare)
}

func (r *PhareReconciler) cleanupHTTPRoute(ctx context.Context, phare pharev1beta1.Phare) error {
	httpRoute := &gatewayv1beta1.HTTPRoute{}
	if deleted, err := r.deleteIfOwned(ctx, httpRoute, phare.Name, phare.Namespace, &phare); err != nil {
		return err
	} else if deleted {
		r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "DeletedResource", "Deleted HTTPRoute %s", phare.Name)
	}
	return nil
}

func (r *PhareReconciler) handleGCPBackendPolicy(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
	if phare.Spec.ToolChain != nil && phare.Spec.ToolChain.GCPBackendPolicy != nil {
		return r.reconcileGCPBackendPolicy(ctx, req, phare)
	}
	return r.cleanupGCPBackendPolicy(ctx, phare)
}

func (r *PhareReconciler) cleanupGCPBackendPolicy(ctx context.Context, phare pharev1beta1.Phare) error {
	gcpBackendPolicy := &unstructured.Unstructured{}
	gcpBackendPolicy.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.gke.io",
		Version: "v1",
		Kind:    "GCPBackendPolicy",
	})
	if deleted, err := r.deleteIfOwned(ctx, gcpBackendPolicy, phare.Name, phare.Namespace, &phare); err != nil {
		return err
	} else if deleted {
		r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "DeletedResource", "Deleted GCPBackendPolicy %s", phare.Name)
	}
	return nil
}

func (r *PhareReconciler) handleHealthCheckPolicy(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
	if phare.Spec.ToolChain != nil && phare.Spec.ToolChain.HealthCheckPolicy != nil {
		return r.reconcileHealthCheckPolicy(ctx, req, phare)
	}
	return r.cleanupHealthCheckPolicy(ctx, phare)
}

func (r *PhareReconciler) cleanupHealthCheckPolicy(ctx context.Context, phare pharev1beta1.Phare) error {
	healthCheckPolicy := &unstructured.Unstructured{}
	healthCheckPolicy.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.gke.io",
		Version: "v1",
		Kind:    "HealthCheckPolicy",
	})
	if deleted, err := r.deleteIfOwned(ctx, healthCheckPolicy, phare.Name, phare.Namespace, &phare); err != nil {
		return err
	} else if deleted {
		r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "DeletedResource", "Deleted HealthCheckPolicy %s", phare.Name)
	}
	return nil
}

func (r *PhareReconciler) reconcileMicroService(ctx context.Context, phare pharev1beta1.Phare) error {
	switch phare.Spec.MicroService.Kind {
	case "Deployment":
		// Remove a stale StatefulSet left over from a previous Kind value.
		if err := r.deleteIfExists(ctx, &appsv1.StatefulSet{}, phare.Name, phare.Namespace, &phare); err != nil {
			return err
		}
		r.Log.Info("Reconciling Deployment", "Deployment.Namespace", phare.Namespace, "Deployment.Name", phare.Name)
		return r.reconcileDeployment(ctx, phare)
	case "StatefulSet":
		// Remove a stale Deployment left over from a previous Kind value.
		if err := r.deleteIfExists(ctx, &appsv1.Deployment{}, phare.Name, phare.Namespace, &phare); err != nil {
			return err
		}
		r.Log.Info("Reconciling StatefulSet", "StatefulSet.Namespace", phare.Namespace, "StatefulSet.Name", phare.Name)
		return r.reconcileStatefulSet(ctx, phare)
	default:
		return fmt.Errorf("unsupported kind: %s", phare.Spec.MicroService.Kind)
	}
}

// deleteIfExists deletes the named object if it exists and is owned by phare.
// NotFound is tolerated on both Get and Delete to handle concurrent deletion (TOCTOU).
func (r *PhareReconciler) deleteIfExists(ctx context.Context, obj client.Object, name, namespace string, phare *pharev1beta1.Phare) error {
	if err := r.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, obj); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if !metav1.IsControlledBy(obj, phare) {
		return nil
	}
	if err := r.Delete(ctx, obj); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// deleteIfOwned deletes the named object only when it is controlled by the Phare.
// This protects unrelated resources that happen to share labels/name conventions.
func (r *PhareReconciler) deleteIfOwned(ctx context.Context, obj client.Object, name, namespace string, phare *pharev1beta1.Phare) (bool, error) {
	if err := r.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, obj); err != nil {
		if apimeta.IsNoMatchError(err) || errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if !metav1.IsControlledBy(obj, phare) {
		return false, nil
	}
	if err := r.Delete(ctx, obj); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
