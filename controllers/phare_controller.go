/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/go-logr/logr"
	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
)

// PhareReconciler reconciles a Phare object
type PhareReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=phare.localcorp.internal,resources=phares,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=phare.localcorp.internal,resources=phares/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=phare.localcorp.internal,resources=phares/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.gke.io,resources=gcpbackendpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.gke.io,resources=healthcheckpolicies,verbs=get;list;watch;create;update;patch;delete

// Reconcile moves cluster resources toward the desired Phare spec.
func (r *PhareReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var phare pharev1beta1.Phare

	found, err := r.fetchPhareResource(ctx, req, &phare)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !found {
		return ctrl.Result{}, nil
	}

	if err := r.reconcileResources(ctx, req, phare); err != nil {
		// Best-effort: record Failed status. The original error is returned
		// regardless so the controller requeues even if the status write fails.
		r.updateStatus(ctx, &phare, pharev1beta1.PharePhaseFailed, err.Error()) //nolint:errcheck
		return ctrl.Result{}, err
	}

	// Propagate status-write failures on the success path so the controller
	// requeues instead of silently leaving stale status.
	return ctrl.Result{}, r.updateStatus(ctx, &phare, pharev1beta1.PharePhaseActive, "Successfully reconciled Phare resource")
}

// reconcileResources runs every sub-reconciler in dependency order.
func (r *PhareReconciler) reconcileResources(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) error {
	if err := r.reconcileConfigMap(ctx, phare); err != nil {
		return err
	}
	if err := r.reconcileService(ctx, req, phare); err != nil {
		return err
	}
	if err := r.handleHTTPRoute(ctx, req, phare); err != nil {
		return err
	}
	if err := r.handleGCPBackendPolicy(ctx, req, phare); err != nil {
		return err
	}
	if err := r.handleHealthCheckPolicy(ctx, req, phare); err != nil {
		return err
	}
	return r.reconcileMicroService(ctx, phare)
}

// updateStatus writes phase/message to the Phare status subresource, skipping the
// write when the status is already up-to-date. The returned error should be
// propagated on success paths so the controller requeues on status write failure.
// On error paths it is safe to discard the return value because the original
// reconcile error already causes a requeue.
func (r *PhareReconciler) updateStatus(ctx context.Context, phare *pharev1beta1.Phare, phase pharev1beta1.PharePhase, message string) error {
	if phare.Status.Phase == phase && phare.Status.Message == message {
		return nil
	}
	phare.Status.Phase = phase
	phare.Status.Message = message
	if err := r.Status().Update(ctx, phare); err != nil {
		r.Log.Error(err, "Failed to update Phare status")
		return err
	}
	return nil
}

func (r *PhareReconciler) fetchPhareResource(ctx context.Context, req ctrl.Request, phare *pharev1beta1.Phare) (bool, error) {
	if err := r.Get(ctx, req.NamespacedName, phare); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			return false, nil
		}
		// Error reading the object.
		return false, err
	}
	return true, nil
}

func (r *PhareReconciler) SetupWithManager(mgr ctrl.Manager) error {
	labelFilter := defaultLabelPredicate("app.kubernetes.io/created-by", "phare-controller")
	gcpBackendPolicy := &unstructured.Unstructured{}
	gcpBackendPolicy.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.gke.io",
		Version: "v1",
		Kind:    "GCPBackendPolicy",
	})
	healthCheckPolicy := &unstructured.Unstructured{}
	healthCheckPolicy.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.gke.io",
		Version: "v1",
		Kind:    "HealthCheckPolicy",
	})

	statefulSetPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldStatefulSet, ok1 := e.ObjectOld.(*appsv1.StatefulSet)
			newStatefulSet, ok2 := e.ObjectNew.(*appsv1.StatefulSet)
			if ok1 && ok2 {
				return oldStatefulSet.GetGeneration() != newStatefulSet.GetGeneration()
			}
			// Default to reconcile if we can't cast the objects correctly
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&pharev1beta1.Phare{}).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(labelFilter)).
		Owns(&appsv1.StatefulSet{}, builder.WithPredicates(labelFilter, statefulSetPredicate)). // Apply the predicate here
		Owns(&corev1.Service{}, builder.WithPredicates(labelFilter)).
		Owns(&corev1.ConfigMap{}, builder.WithPredicates(labelFilter)).
		Owns(&gatewayv1beta1.HTTPRoute{}, builder.WithPredicates(labelFilter)).
		Owns(gcpBackendPolicy, builder.WithPredicates(labelFilter)).
		Owns(healthCheckPolicy, builder.WithPredicates(labelFilter)).
		Complete(r)
}

// defaultLabelPredicate filters events by a required label value.
func defaultLabelPredicate(labelKey, labelValue string) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Object.GetLabels()[labelKey] == labelValue
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectNew.GetLabels()[labelKey] == labelValue
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return e.Object.GetLabels()[labelKey] == labelValue
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return e.Object.GetLabels()[labelKey] == labelValue
		},
	}
}
