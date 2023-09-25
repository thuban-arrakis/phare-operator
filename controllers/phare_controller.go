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
  "fmt"

  appsv1 "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/api/errors"
  "k8s.io/apimachinery/pkg/runtime"
  "k8s.io/client-go/tools/record"
  ctrl "sigs.k8s.io/controller-runtime"
  "sigs.k8s.io/controller-runtime/pkg/client"
  "sigs.k8s.io/controller-runtime/pkg/log"

  // TODO(user): Event recorder is required to emit Events.
  // "k8s.io/client-go/tools/record"

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
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Phare object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *PhareReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
  _ = log.FromContext(ctx)

  var phare pharev1beta1.Phare

  // log := r.Log.WithValues("phare", req.NamespacedName)
  // log.Info("Reconciling Phare")

  if err := r.Get(ctx, req.NamespacedName, &phare); err != nil {
    if errors.IsNotFound(err) {
      // Object not found, return. Created objects are automatically garbage collected.
      return ctrl.Result{}, nil
    }
    // Error reading the object - requeue the request.
    return ctrl.Result{}, err
  }

  if err := r.reconcileConfigMap(ctx, phare); err != nil {
    // Handle error and possibly requeue
    return ctrl.Result{}, err
  }

  // Reconcile Service
  if _, err := r.reconcileService(ctx, req, phare); err != nil {
    return ctrl.Result{}, err
  }

  switch phare.Spec.Microservice.Kind {
  case "Deployment":
    // Logic for handling Deployment
    return r.reconcileDeployment(ctx, req, phare)
  case "StatefulSet":
    // Logic for handling StatefulSet
    return r.reconcileStatefulSet(ctx, req, phare)
  default:
    return ctrl.Result{}, fmt.Errorf("unsupported kind: %s", phare.Spec.Microservice.Kind)
  }
}

// SetupWithManager sets up the controller with the Manager.
func (r *PhareReconciler) SetupWithManager(mgr ctrl.Manager) error {
  return ctrl.NewControllerManagedBy(mgr).
    For(&pharev1beta1.Phare{}).
    Owns(&appsv1.Deployment{}).
    Owns(&appsv1.StatefulSet{}).
    Owns(&corev1.Service{}).
    Owns(&corev1.ConfigMap{}).
    Complete(r)
}
