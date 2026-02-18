package controllers

import (
	"context"
	"fmt"

	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	"github.com/localcorp/phare-controller/pkg/validator"
	yamldiff "github.com/localcorp/phare-controller/pkg/yamldiff"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
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
		if existingHttpRoute.ObjectMeta.Labels == nil {
			existingHttpRoute.ObjectMeta.Labels = map[string]string{}
		}
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

	// Set the controller reference so that we can correlate the HTTPRoute to the Phare resource
	if err := ctrl.SetControllerReference(phare, httpRoute, r.Scheme); err != nil {
		r.Log.Error(err, "Failed to set controller reference for HTTPRoute")
		return httpRoute
	}
	return httpRoute
}

func (r *PhareReconciler) desiredGCPBackendPolicy(phare *pharev1beta1.Phare) *unstructured.Unstructured {
	metadataLabels := map[string]string{
		"kustomize.toolkit.fluxcd.io/name":      "apps",
		"kustomize.toolkit.fluxcd.io/namespace": "flux-system",
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
			"spec": phare.Spec.ToolChain.GCPBackendPolicy,
		},
	}
	if err := ctrl.SetControllerReference(phare, gcpBackendPolicy, r.Scheme); err != nil {
		r.Log.Error(err, "Failed to set controller reference for GCPBackendPolicy")
		return gcpBackendPolicy
	}
	return gcpBackendPolicy
}

func (r *PhareReconciler) reconcileGCPBackendPolicy(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) (ctrl.Result, error) {
	existingGCPBackendPolicy := &unstructured.Unstructured{}

	existingGCPBackendPolicy.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.gke.io",
		Version: "v1",
		Kind:    "GCPBackendPolicy",
	})

	desired := r.desiredGCPBackendPolicy(&phare)
	err := r.Get(ctx, req.NamespacedName, existingGCPBackendPolicy)

	// Convert existing and desired spec to YAML for comparison
	existingGCPBackendPolicySpecYAML := toYAML(existingGCPBackendPolicy.Object["spec"])
	desiredGCPBackendPolicySpecYAML := toYAML(desired.Object["spec"])

	if err != nil {
		if errors.IsNotFound(err) {
			// GCPBackendPolicy doesn't exist, create it
			if err := r.Create(ctx, desired); err != nil {
				return ctrl.Result{}, err
			}
			r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "CreatedResource", "Created GCPBackendPolicy %s", desired.GetName())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	isValid, desiredMap, modifiedCurrentMap := validator.ValidateYaml(desiredGCPBackendPolicySpecYAML, existingGCPBackendPolicySpecYAML)

	if !isValid {
		r.Log.Info("GCPBackendPolicy does not match the desired configuration", "GCPBackendPolicy.Namespace", desired.GetNamespace(), "GCPBackendPolicy.Name", desired.GetName())

		// Log the diff for debugging
		diffOutput := yamldiff.Diff(validator.PrintMap(modifiedCurrentMap), validator.PrintMap(desiredMap))
		r.Log.Info("Diff between current and desired configuration", "diff", diffOutput)

		patch := client.MergeFrom(existingGCPBackendPolicy.DeepCopy())
		r.Log.Info("Updating GCPBackendPolicy", "GCPBackendPolicy.Namespace", existingGCPBackendPolicy.GetNamespace(), "GCPBackendPolicy.Name", existingGCPBackendPolicy.GetName())

		// Copy desired service's spec to existingGCPBackendPolicy
		existingGCPBackendPolicy.Object["spec"] = desired.Object["spec"]
		// Add or update labels in the existingGCPBackendPolicy
		existingGCPBackendPolicy.SetLabels(mergeLabelMaps(existingGCPBackendPolicy.GetLabels(), desired.GetLabels()))

		if err := r.Patch(ctx, existingGCPBackendPolicy, patch, client.FieldOwner("phare-controller")); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	r.Log.Info("GCPBackendPolicy matches the desired configuration", "GCPBackendPolicy.Namespace", desired.GetNamespace(), "GCPBackendPolicy.Name", desired.GetName())
	return ctrl.Result{}, nil
}

func (r *PhareReconciler) desiredHealthCheckPolicy(phare *pharev1beta1.Phare) *unstructured.Unstructured {
	if phare.Spec.ToolChain == nil || phare.Spec.ToolChain.HealthCheckPolicy == nil {
		return nil
	}

	// metadataLabels := map[string]string{
	//   // Define your labels here
	// }

	// Initialize the HealthCheckPolicy
	healthCheckPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.gke.io/v1",
			"kind":       "HealthCheckPolicy",
			"metadata": map[string]interface{}{
				"name":      phare.Name,
				"namespace": phare.Namespace,
				// "labels":    metadataLabels,
			},
			"spec": phare.Spec.ToolChain.HealthCheckPolicy,
		},
	}
	if err := ctrl.SetControllerReference(phare, healthCheckPolicy, r.Scheme); err != nil {
		r.Log.Error(err, "Failed to set controller reference for HealthCheckPolicy")
		return healthCheckPolicy
	}
	return healthCheckPolicy
}

func (r *PhareReconciler) reconcileHealthCheckPolicy(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) (ctrl.Result, error) {
	existingHealthCheckPolicy := &unstructured.Unstructured{}

	existingHealthCheckPolicy.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.gke.io",
		Version: "v1",
		Kind:    "HealthCheckPolicy",
	})

	desired := r.desiredHealthCheckPolicy(&phare)
	if desired == nil {
		return ctrl.Result{}, nil
	}
	err := r.Get(ctx, req.NamespacedName, existingHealthCheckPolicy)

	// Convert existing and desired spec to YAML for comparison
	existingHealthCheckPolicySpecYAML := toYAML(existingHealthCheckPolicy.Object["spec"])
	desiredHealthCheckPolicySpecYAML := toYAML(desired.Object["spec"])

	if err != nil {
		if errors.IsNotFound(err) {
			// HealthCheckPolicy doesn't exist, create it
			if err := r.Create(ctx, desired); err != nil {
				return ctrl.Result{}, err
			}
			r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "CreatedResource", "Created HealthCheckPolicy %s", desired.GetName())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	isValid, desiredMap, modifiedCurrentMap := validator.ValidateYaml(desiredHealthCheckPolicySpecYAML, existingHealthCheckPolicySpecYAML)

	if !isValid {
		r.Log.Info("HealthCheckPolicy does not match the desired configuration", "HealthCheckPolicy.Namespace", desired.GetNamespace(), "HealthCheckPolicy.Name", desired.GetName())

		// Log the diff for debugging
		diffOutput := yamldiff.Diff(validator.PrintMap(modifiedCurrentMap), validator.PrintMap(desiredMap))
		r.Log.Info("Diff between current and desired configuration", "diff", diffOutput)

		patch := client.MergeFrom(existingHealthCheckPolicy.DeepCopy())
		r.Log.Info("Updating HealthCheckPolicy", "HealthCheckPolicy.Namespace", existingHealthCheckPolicy.GetNamespace(), "HealthCheckPolicy.Name", existingHealthCheckPolicy.GetName())

		// Copy desired service's spec to existingHealthCheckPolicy
		existingHealthCheckPolicy.Object["spec"] = desired.Object["spec"]
		// Add or update labels in the existingHealthCheckPolicy
		existingHealthCheckPolicy.SetLabels(mergeLabelMaps(existingHealthCheckPolicy.GetLabels(), desired.GetLabels()))

		if err := r.Patch(ctx, existingHealthCheckPolicy, patch, client.FieldOwner("phare-controller")); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	} else {
		r.Log.Info("HealthCheckPolicy matches the desired configuration", "HealthCheckPolicy.Namespace", desired.GetNamespace(), "HealthCheckPolicy.Name", desired.GetName())
	}

	return ctrl.Result{}, nil
}

func mergeLabelMaps(existing map[string]string, desired map[string]string) map[string]string {
	merged := make(map[string]string, len(existing)+len(desired))
	for key, value := range existing {
		merged[key] = value
	}
	for key, value := range desired {
		merged[key] = value
	}
	return merged
}

// Move this to pkg/utils or something.
func toYAML(obj interface{}) string {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("Error marshaling to YAML: %s", err)
	}
	return string(data)
}
