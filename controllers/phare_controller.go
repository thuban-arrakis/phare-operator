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
  "time"

  appsv1 "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/api/errors"
  "k8s.io/apimachinery/pkg/runtime"
  "k8s.io/client-go/tools/record"
  ctrl "sigs.k8s.io/controller-runtime"
  "sigs.k8s.io/controller-runtime/pkg/client"
  gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

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
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Phare object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.1/pkg/reconcile
func (r *PhareReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
  var phare pharev1beta1.Phare

  if err := r.fetchPhareResource(ctx, req, &phare); err != nil {
    return ctrl.Result{}, err
  }

  if err := r.reconcileConfigMap(ctx, phare); err != nil {
    return ctrl.Result{}, err
  }

  if err := r.reconcileService(ctx, req, phare); err != nil {
    return ctrl.Result{}, err
  }

  // if err := r.handleHTTPRoute(ctx, req, phare); err != nil {
  //   return ctrl.Result{}, err
  // }

  // if err := r.handleGCPBackendPolicy(ctx, req, phare); err != nil {
  //   return ctrl.Result{}, err
  // }

  if err := r.reconcileMicroService(ctx, phare); err != nil {
    return ctrl.Result{RequeueAfter: time.Minute}, err
  }
  return ctrl.Result{}, nil
}

func (r *PhareReconciler) fetchPhareResource(ctx context.Context, req ctrl.Request, phare *pharev1beta1.Phare) error {
  if err := r.Get(ctx, req.NamespacedName, phare); err != nil {
    if errors.IsNotFound(err) {
      // Object not found, return. Created objects are automatically garbage collected.
      return nil
    }
    // Error reading the object.
    return err
  }
  return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PhareReconciler) SetupWithManager(mgr ctrl.Manager) error {
  return ctrl.NewControllerManagedBy(mgr).
    For(&pharev1beta1.Phare{}).
    Owns(&appsv1.Deployment{}).
    Owns(&appsv1.StatefulSet{}).
    Owns(&corev1.Service{}).
    Owns(&corev1.ConfigMap{}).
    Owns(&gatewayv1beta1.HTTPRoute{}).
    Complete(r)
}
