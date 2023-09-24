package controllers

import (
  "context"

  apps "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/api/errors"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  ctrl "sigs.k8s.io/controller-runtime"

  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
)

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
