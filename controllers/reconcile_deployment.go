package controllers

import (
  "context"
  "fmt"
  "reflect"

  apps "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/api/errors"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  ctrl "sigs.k8s.io/controller-runtime"

  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
)

func (r *PhareReconciler) reconcileDeployment(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) (ctrl.Result, error) {
  existingDeployment := &apps.Deployment{}
  err := r.Get(ctx, req.NamespacedName, existingDeployment)
  if err != nil && errors.IsNotFound(err) {
    // Deployment doesn't exist, so create it:

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
    return ctrl.Result{}, err // An error other than "not found" occurred.
  }

  // If we reach here, it means the Deployment exists.
  // Check if it needs to be updated:

  desired := r.desiredDeployment(&phare)
  if !reflect.DeepEqual(existingDeployment.Spec, desired.Spec) {
    existingDeployment.Spec = desired.Spec
    if err := r.Update(ctx, existingDeployment); err != nil {
      return ctrl.Result{}, err
    }
  }

  return ctrl.Result{}, nil
}

func (r *PhareReconciler) desiredDeployment(phare *pharev1beta1.Phare) *apps.Deployment {
  replicaCount := phare.Spec.Microservice.ReplicaCount

  labels := map[string]string{
    "app": phare.Name,
  }

  deployment := &apps.Deployment{
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

  // Check if the spec.config is not empty
  if phare.Spec.Config != nil && len(phare.Spec.Config) > 0 {
    // Add ConfigMap as a volume to the Pod template
    deploymentVolume := corev1.Volume{
      Name: "config-volume",
      VolumeSource: corev1.VolumeSource{
        ConfigMap: &corev1.ConfigMapVolumeSource{
          LocalObjectReference: corev1.LocalObjectReference{
            Name: phare.Name + "-config",
          },
        },
      },
    }
    configMapDataHash, err := r.hashConfigMapData(phare.Name+"-config", phare.Namespace)
    if err != nil {
      fmt.Println("Error hashing config map data")
      // Handle error, maybe return an error or log it
    }

    if deployment.Spec.Template.Annotations == nil {
      deployment.Spec.Template.Annotations = make(map[string]string)
    }
    deployment.Spec.Template.Annotations["checksum/config-files"] = configMapDataHash
    deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, deploymentVolume)

    // Mount the volume to the container
    volumeMount := corev1.VolumeMount{
      Name:      "config-volume",
      MountPath: "/path/to/mount", // Adjust this to your desired mount path
    }
    deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMount)
  }

  return deployment
}
