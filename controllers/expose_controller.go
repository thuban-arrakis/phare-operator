package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func (r *PhareReconciler) reconcileHttpRoute(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
	existingHttpRoute := &gatewayv1beta1.HTTPRoute{}
	desired := r.desiredHttpRoute(&phare)
	if desired == nil {
		return fmt.Errorf("failed to build HTTPRoute for %s/%s", phare.Namespace, phare.Name)
	}
	err := r.Get(ctx, req.NamespacedName, existingHttpRoute)

	if err != nil {
		if errors.IsNotFound(err) {
			// HTTPRoute doesn't exist, create it
			if err := r.Create(ctx, desired); err != nil {
				return err
			}
			r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "CreatedResource", "Created HTTPRoute %s", desired.Name)
			return nil
		}
		return err
	}

	if !specMatchesDesired(existingHttpRoute.Spec, desired.Spec) ||
		!stringMapsEqualNilEmpty(existingHttpRoute.GetLabels(), desired.GetLabels()) {
		r.Log.Info("HTTPRoute does not match the desired configuration", "HTTPRoute.Namespace", desired.Namespace, "HTTPRoute.Name", desired.Name)

		patch := client.MergeFrom(existingHttpRoute.DeepCopy())
		r.Log.Info("Updating HTTPRoute", "HTTPRoute.Namespace", existingHttpRoute.Namespace, "HTTPRoute.Name", existingHttpRoute.Name)

		// Copy desired spec into the current object before patching.
		existingHttpRoute.Spec = desired.Spec
		existingHttpRoute.ObjectMeta.Labels = copyStringMapPreserveNil(desired.ObjectMeta.Labels)

		if err := r.Patch(ctx, existingHttpRoute, patch, client.FieldOwner("phare-controller")); err != nil {
			return err
		}
		return nil
	}

	r.Log.Info("HTTPRoute matches the desired configuration", "HTTPRoute.Namespace", desired.Namespace, "HTTPRoute.Name", desired.Name)

	return nil
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
				ParentRefs: phare.Spec.ToolChain.HTTPRoute.ParentRefs,
			},
		},
	}

	// Set the controller reference so that we can correlate the HTTPRoute to the Phare resource
	if err := ctrl.SetControllerReference(phare, httpRoute, r.Scheme); err != nil {
		r.Log.Error(err, "Failed to set controller reference for HTTPRoute")
		return nil
	}
	return httpRoute
}

func (r *PhareReconciler) desiredGCPBackendPolicy(phare *pharev1beta1.Phare) *unstructured.Unstructured {
	if phare.Spec.ToolChain == nil || phare.Spec.ToolChain.GCPBackendPolicy == nil {
		return nil
	}

	metadataLabels := map[string]string{
		"app":                          phare.Name,
		"app.kubernetes.io/created-by": "phare-controller",
	}

	spec, err := runtime.DefaultUnstructuredConverter.ToUnstructured(phare.Spec.ToolChain.GCPBackendPolicy)
	if err != nil {
		r.Log.Error(err, "Failed to convert GCPBackendPolicy spec to unstructured map")
		return nil
	}

	gcpBackendPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.gke.io/v1",
			"kind":       "GCPBackendPolicy",
			"metadata": map[string]interface{}{
				"name":      phare.Name,
				"namespace": phare.Namespace,
				"labels":    metadataLabels,
			},
			"spec": spec,
		},
	}
	if err := ctrl.SetControllerReference(phare, gcpBackendPolicy, r.Scheme); err != nil {
		r.Log.Error(err, "Failed to set controller reference for GCPBackendPolicy")
		return nil
	}
	return gcpBackendPolicy
}

func (r *PhareReconciler) reconcileGCPBackendPolicy(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
	existingGCPBackendPolicy := &unstructured.Unstructured{}

	existingGCPBackendPolicy.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.gke.io",
		Version: "v1",
		Kind:    "GCPBackendPolicy",
	})

	desired := r.desiredGCPBackendPolicy(&phare)
	if desired == nil {
		return fmt.Errorf("failed to build GCPBackendPolicy for %s/%s", phare.Namespace, phare.Name)
	}
	err := r.Get(ctx, req.NamespacedName, existingGCPBackendPolicy)

	if err != nil {
		if errors.IsNotFound(err) {
			// GCPBackendPolicy doesn't exist, create it
			if err := r.Create(ctx, desired); err != nil {
				return err
			}
			r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "CreatedResource", "Created GCPBackendPolicy %s", desired.GetName())
			return nil
		}
		return err
	}

	if !specMatchesDesired(existingGCPBackendPolicy.Object["spec"], desired.Object["spec"]) ||
		!stringMapsEqualNilEmpty(existingGCPBackendPolicy.GetLabels(), desired.GetLabels()) {
		r.Log.Info("GCPBackendPolicy does not match the desired configuration", "GCPBackendPolicy.Namespace", desired.GetNamespace(), "GCPBackendPolicy.Name", desired.GetName())

		patch := client.MergeFrom(existingGCPBackendPolicy.DeepCopy())
		r.Log.Info("Updating GCPBackendPolicy", "GCPBackendPolicy.Namespace", existingGCPBackendPolicy.GetNamespace(), "GCPBackendPolicy.Name", existingGCPBackendPolicy.GetName())

		// Copy desired spec into the current object before patching.
		existingGCPBackendPolicy.Object["spec"] = desired.Object["spec"]
		existingGCPBackendPolicy.SetLabels(copyStringMapPreserveNil(desired.GetLabels()))

		if err := r.Patch(ctx, existingGCPBackendPolicy, patch, client.FieldOwner("phare-controller")); err != nil {
			return err
		}
		return nil
	}

	r.Log.Info("GCPBackendPolicy matches the desired configuration", "GCPBackendPolicy.Namespace", desired.GetNamespace(), "GCPBackendPolicy.Name", desired.GetName())
	return nil
}

func (r *PhareReconciler) desiredHealthCheckPolicy(phare *pharev1beta1.Phare) *unstructured.Unstructured {
	if phare.Spec.ToolChain == nil || phare.Spec.ToolChain.HealthCheckPolicy == nil {
		return nil
	}

	metadataLabels := map[string]string{
		"app":                          phare.Name,
		"app.kubernetes.io/created-by": "phare-controller",
	}

	spec, err := runtime.DefaultUnstructuredConverter.ToUnstructured(phare.Spec.ToolChain.HealthCheckPolicy)
	if err != nil {
		r.Log.Error(err, "Failed to convert HealthCheckPolicy spec to unstructured map")
		return nil
	}

	// Build HealthCheckPolicy from the Phare spec.
	healthCheckPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.gke.io/v1",
			"kind":       "HealthCheckPolicy",
			"metadata": map[string]interface{}{
				"name":      phare.Name,
				"namespace": phare.Namespace,
				"labels":    metadataLabels,
			},
			"spec": spec,
		},
	}
	if err := ctrl.SetControllerReference(phare, healthCheckPolicy, r.Scheme); err != nil {
		r.Log.Error(err, "Failed to set controller reference for HealthCheckPolicy")
		return nil
	}
	return healthCheckPolicy
}

func (r *PhareReconciler) reconcileHealthCheckPolicy(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
	existingHealthCheckPolicy := &unstructured.Unstructured{}

	existingHealthCheckPolicy.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.gke.io",
		Version: "v1",
		Kind:    "HealthCheckPolicy",
	})

	desired := r.desiredHealthCheckPolicy(&phare)
	if desired == nil {
		return fmt.Errorf("failed to build HealthCheckPolicy for %s/%s", phare.Namespace, phare.Name)
	}
	err := r.Get(ctx, req.NamespacedName, existingHealthCheckPolicy)

	if err != nil {
		if errors.IsNotFound(err) {
			// HealthCheckPolicy doesn't exist, create it
			if err := r.Create(ctx, desired); err != nil {
				return err
			}
			r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "CreatedResource", "Created HealthCheckPolicy %s", desired.GetName())
			return nil
		}
		return err
	}

	if !specMatchesDesired(existingHealthCheckPolicy.Object["spec"], desired.Object["spec"]) ||
		!stringMapsEqualNilEmpty(existingHealthCheckPolicy.GetLabels(), desired.GetLabels()) {
		r.Log.Info("HealthCheckPolicy does not match the desired configuration", "HealthCheckPolicy.Namespace", desired.GetNamespace(), "HealthCheckPolicy.Name", desired.GetName())

		patch := client.MergeFrom(existingHealthCheckPolicy.DeepCopy())
		r.Log.Info("Updating HealthCheckPolicy", "HealthCheckPolicy.Namespace", existingHealthCheckPolicy.GetNamespace(), "HealthCheckPolicy.Name", existingHealthCheckPolicy.GetName())

		// Copy desired spec into the current object before patching.
		existingHealthCheckPolicy.Object["spec"] = desired.Object["spec"]
		existingHealthCheckPolicy.SetLabels(copyStringMapPreserveNil(desired.GetLabels()))

		if err := r.Patch(ctx, existingHealthCheckPolicy, patch, client.FieldOwner("phare-controller")); err != nil {
			return err
		}
		return nil
	}

	r.Log.Info("HealthCheckPolicy matches the desired configuration", "HealthCheckPolicy.Namespace", desired.GetNamespace(), "HealthCheckPolicy.Name", desired.GetName())

	return nil
}

func specMatchesDesired(existingSpec, desiredSpec interface{}) bool {
	return reflect.DeepEqual(canonicalizeSpec(existingSpec), canonicalizeSpec(desiredSpec))
}

func canonicalizeSpec(spec interface{}) interface{} {
	if spec == nil {
		return nil
	}
	b, err := json.Marshal(spec)
	if err != nil {
		return spec
	}
	var out interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return spec
	}
	return out
}
