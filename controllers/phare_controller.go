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

  apps "k8s.io/api/apps/v1"
  appsv1 "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/api/errors"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/apimachinery/pkg/runtime"
  ctrl "sigs.k8s.io/controller-runtime"
  "sigs.k8s.io/controller-runtime/pkg/client"
  "sigs.k8s.io/controller-runtime/pkg/handler"
  "sigs.k8s.io/controller-runtime/pkg/log"
  "sigs.k8s.io/controller-runtime/pkg/source"

  "github.com/go-logr/logr"
  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
)

// PhareReconciler reconciles a Phare object
type PhareReconciler struct {
  client.Client
  Log    logr.Logger
  Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=phare.localcorp.internal,resources=phares,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=phare.localcorp.internal,resources=phares/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=phare.localcorp.internal,resources=phares/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

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

  // log := r.Log.WithValues("phare", req.NamespacedName)
  // log.Info("Reconciling Phare")

  var phare pharev1beta1.Phare
  if err := r.Get(ctx, req.NamespacedName, &phare); err != nil {
    if errors.IsNotFound(err) {
      // Object not found, return. Created objects are automatically garbage collected.
      return ctrl.Result{}, nil
    }
    // Error reading the object - requeue the request.
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
    Watches(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForObject{}).
    Watches(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForObject{}).
    Complete(r)
}

func (r *PhareReconciler) reconcileDeployment(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) (ctrl.Result, error) {
  // Your existing logic for handling Deployment
  deployment := &apps.Deployment{}
  err := r.Get(ctx, req.NamespacedName, deployment)
  if err != nil && errors.IsNotFound(err) {
    // Before creating a new Deployment:
    phare.Status.Phase = PharePhaseReconciling
    phare.Status.Message = "Creating Deployment."
    if err := r.Status().Update(ctx, &phare); err != nil {
      return ctrl.Result{}, err
    }

    // Define a new Deployment
    dep := r.desiredDeployment(&phare)
    if err := r.Create(ctx, dep); err != nil {
      return ctrl.Result{}, err
    }

    // After creating a new Deployment:
    phare.Status.Phase = PharePhaseActive
    phare.Status.Message = "Deployment created successfully."
    if err := r.Status().Update(ctx, &phare); err != nil {
      return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
  } else if err != nil {
    return ctrl.Result{}, err
  }

  // TODO: Update the Deployment if necessary...

  return ctrl.Result{}, nil
}

func (r *PhareReconciler) reconcileStatefulSet(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) (ctrl.Result, error) {
  // Your existing logic for handling StatefulSet
  statefulSet := &apps.StatefulSet{}
  err := r.Get(ctx, req.NamespacedName, statefulSet)
  if err != nil && errors.IsNotFound(err) {
    // Before creating a new StatefulSet:
    phare.Status.Phase = PharePhaseReconciling
    phare.Status.Message = "Creating StatefulSet."
    if err := r.Status().Update(ctx, &phare); err != nil {
      return ctrl.Result{}, err
    }

    // Define a new StatefulSet
    sts := r.desiredStatefulSet(&phare)
    if err := r.Create(ctx, sts); err != nil {
      return ctrl.Result{}, err
    }

    // After creating a new StatefulSet:
    phare.Status.Phase = PharePhaseActive
    phare.Status.Message = "StatefulSet created successfully."
    if err := r.Status().Update(ctx, &phare); err != nil {
      return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
  } else if err != nil {
    return ctrl.Result{}, err
  }

  // TODO: Update the StatefulSet if necessary...
  return ctrl.Result{}, nil
}

func (r *PhareReconciler) desiredDeployment(phare *pharev1beta1.Phare) *apps.Deployment {
  replicaCount := phare.Spec.Microservice.ReplicaCount

  labels := map[string]string{
    "app": phare.Name,
  }

  // log := r.Log.WithValues("phare", phare.Name)
  // log.Info("Generating desired Deployment for Phare", "Image", phare.Spec.Microservice.Image.Repository+":"+phare.Spec.Microservice.Image.Tag, "ReplicaCount", replicaCount)

  return &apps.Deployment{
    ObjectMeta: metav1.ObjectMeta{
      Name:      phare.Name,
      Namespace: phare.Namespace,
    },
    Spec: apps.DeploymentSpec{
      Selector: &metav1.LabelSelector{
        MatchLabels: labels,
      },
      Replicas: &replicaCount,
      Template: corev1.PodTemplateSpec{
        ObjectMeta: metav1.ObjectMeta{
          Labels: labels,
        },
        Spec: corev1.PodSpec{
          Containers: []corev1.Container{
            {
              Name:  phare.Name,
              Image: phare.Spec.Microservice.Image.Repository + ":" + phare.Spec.Microservice.Image.Tag,
            },
          },
        },
      },
    },
  }
}

func (r *PhareReconciler) desiredStatefulSet(phare *pharev1beta1.Phare) *apps.StatefulSet {
  replicaCount := phare.Spec.Microservice.ReplicaCount

  labels := map[string]string{
    "app": phare.Name,
  }

  // log := r.Log.WithValues("phare", phare.Name)
  // log.Info("Generating desired StatefulSet for Phare", "Image", phare.Spec.Microservice.Image.Repository+":"+phare.Spec.Microservice.Image.Tag, "ReplicaCount", replicaCount)

  return &apps.StatefulSet{
    ObjectMeta: metav1.ObjectMeta{
      Name:      phare.Name,
      Namespace: phare.Namespace,
    },
    Spec: apps.StatefulSetSpec{
      Selector: &metav1.LabelSelector{
        MatchLabels: labels,
      },
      Replicas: &replicaCount,
      Template: corev1.PodTemplateSpec{
        ObjectMeta: metav1.ObjectMeta{
          Labels: labels,
        },
        Spec: corev1.PodSpec{
          Containers: []corev1.Container{
            {
              Name:  phare.Name,
              Image: phare.Spec.Microservice.Image.Repository + ":" + phare.Spec.Microservice.Image.Tag,
            },
          },
        },
      },
    },
  }
}
