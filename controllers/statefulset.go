package controllers

import (
  "context"
  "fmt"
  "reflect"

  apps "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/api/errors"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/utils/pointer"
  ctrl "sigs.k8s.io/controller-runtime"
  "sigs.k8s.io/controller-runtime/pkg/client"

  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
)

func (r *PhareReconciler) reconcileStatefulSet(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) (ctrl.Result, error) {
  existingStatefulSet := &apps.StatefulSet{}
  err := r.Get(ctx, req.NamespacedName, existingStatefulSet)

  if err != nil && errors.IsNotFound(err) {
    // Before creating a new StatefulSet:
    phare.Status.Phase = PharePhaseReconciling
    phare.Status.Message = "Creating StatefulSet."
    r.Log.Info("Creating StatefulSet")
    if err := r.Status().Update(ctx, &phare); err != nil {
      return ctrl.Result{}, err
    }

    // Define a new StatefulSet
    sts := r.desiredStatefulSet(&phare)
    r.Log.Info("Creating StatefulSet", "StatefulSet.Namespace", sts.Namespace, "StatefulSet.Name", sts.Name)
    if err := r.Create(ctx, sts); err != nil {
      return ctrl.Result{}, err
    }

    // After creating a new StatefulSet:
    phare.Status.Phase = PharePhaseActive
    phare.Status.Message = "StatefulSet created successfully."
    r.Log.Info("StatefulSet created successfully")
    if err := r.Status().Update(ctx, &phare); err != nil {
      return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
  } else if err != nil {
    return ctrl.Result{}, err
  }

  desired := r.desiredStatefulSet(&phare)
  if !reflect.DeepEqual(existingStatefulSet.Spec, desired.Spec) {
    patch := client.MergeFrom(existingStatefulSet.DeepCopy())
    r.Log.Info("Updating StatefulSet", "StatefulSet.Namespace", existingStatefulSet.Namespace, "StatefulSet.Name", existingStatefulSet.Name)
    existingStatefulSet.Spec = desired.Spec
    if err := r.Patch(ctx, existingStatefulSet, patch, client.FieldOwner("phare-controller")); err != nil {
      return ctrl.Result{}, err
    }
  }

  return ctrl.Result{}, nil
}

func (r *PhareReconciler) desiredStatefulSet(phare *pharev1beta1.Phare) *apps.StatefulSet {
  replicaCount := phare.Spec.Microservice.ReplicaCount

  labels := map[string]string{
    "app": phare.Name,
  }

  statefulSet := &apps.StatefulSet{
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

  // Set the owner reference
  statefulSet.OwnerReferences = []metav1.OwnerReference{
    {
      APIVersion: phare.APIVersion, // Ensure this matches the API version of your Phare CRD
      Kind:       phare.Kind,       // Typically this would be "Phare"
      Name:       phare.Name,
      UID:        phare.UID,
      Controller: pointer.Bool(true),
    },
  }

  // Check if the spec.config is not empty
  if phare.Spec.Config != nil && len(phare.Spec.Config) > 0 {
    // Add ConfigMap as a volume to the Pod template
    statefulSetVolume := corev1.Volume{
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

    if statefulSet.Spec.Template.Annotations == nil {
      statefulSet.Spec.Template.Annotations = make(map[string]string)
    }
    statefulSet.Spec.Template.Annotations["checksum/config-files"] = configMapDataHash
    statefulSet.Spec.Template.Spec.Volumes = append(statefulSet.Spec.Template.Spec.Volumes, statefulSetVolume)

    // Mount the volume to the container
    volumeMount := corev1.VolumeMount{
      Name:      "config-volume",
      MountPath: "/path/to/mount", // Adjust this to your desired mount path
    }
    statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts = append(statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMount)
  }

  return statefulSet
}
