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

  // "github.com/google/go-cmp/cmp"

  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
)

func (r *PhareReconciler) reconcileStatefulSet(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) (ctrl.Result, error) {
  // 1. Fetch the current StatefulSet from the cluster.
  currentSts := &apps.StatefulSet{}
  err := r.Get(ctx, req.NamespacedName, currentSts)
  if err != nil && errors.IsNotFound(err) {
    // The StatefulSet does not exist. Create it.
    newSts := r.buildStatefulSetFromPhare(phare)
    if err := r.Create(ctx, newSts); err != nil {
      return ctrl.Result{}, err
    }
    return ctrl.Result{Requeue: true}, nil
  } else if err != nil {
    return ctrl.Result{}, err
  }

  // 2. Convert the Phare CR spec to a desired StatefulSet spec.
  desiredStsSpec := r.convertPhareSpecToSts(phare)

  // 3. Compare the desired StatefulSet spec with the current one.
  if !reflect.DeepEqual(desiredStsSpec, currentSts.Spec) {
    currentSts.Spec = desiredStsSpec
    if err := r.Update(ctx, currentSts); err != nil {
      return ctrl.Result{}, err
    }
    return ctrl.Result{Requeue: true}, nil
  }

  return ctrl.Result{}, nil
}

func (r *PhareReconciler) buildStatefulSetFromPhare(phare pharev1beta1.Phare) *apps.StatefulSet {
  // Define the labels to be used for the StatefulSet
  labels := map[string]string{
    "app": phare.Name,
    // add any other labels if needed
  }

  // Define the StatefulSet
  statefulSet := &apps.StatefulSet{
    ObjectMeta: metav1.ObjectMeta{
      Name:      phare.Name,
      Namespace: phare.Namespace,
      Labels:    labels,
    },
    Spec: apps.StatefulSetSpec{
      Replicas: &phare.Spec.Microservice.ReplicaCount,
      Selector: &metav1.LabelSelector{
        MatchLabels: labels,
      },
      Template: corev1.PodTemplateSpec{
        ObjectMeta: metav1.ObjectMeta{
          Labels: labels,
        },
        Spec: corev1.PodSpec{
          Containers: []corev1.Container{
            {
              Name:            phare.Name,
              Image:           fmt.Sprintf("%s:%s", phare.Spec.Microservice.Image.Repository, phare.Spec.Microservice.Image.Tag),
              ImagePullPolicy: corev1.PullPolicy(phare.Spec.Microservice.ImagePullPolicy),
              // Add other container fields as needed based on your MicroserviceSpec
            },
          },
          // Add other PodSpec fields as needed
        },
      },
      // Add other StatefulSetSpec fields as needed
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
      return nil
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

func (r *PhareReconciler) convertPhareSpecToSts(phare pharev1beta1.Phare) apps.StatefulSetSpec {
  // Define the labels to be used for the StatefulSet
  labels := map[string]string{
    "app": phare.Name,
    // add other labels if needed
  }

  // Construct the StatefulSetSpec based on the PhareSpec
  stsSpec := apps.StatefulSetSpec{
    Replicas: &phare.Spec.Microservice.ReplicaCount,
    Selector: &metav1.LabelSelector{
      MatchLabels: labels,
    },
    Template: corev1.PodTemplateSpec{
      ObjectMeta: metav1.ObjectMeta{
        Labels: labels,
      },
      Spec: corev1.PodSpec{
        Containers: []corev1.Container{
          {
            Name:            phare.Name,
            Image:           fmt.Sprintf("%s:%s", phare.Spec.Microservice.Image.Repository, phare.Spec.Microservice.Image.Tag),
            ImagePullPolicy: corev1.PullPolicy(phare.Spec.Microservice.ImagePullPolicy),
            // Add other container fields as needed based on your MicroserviceSpec
          },
        },
        // Add other PodSpec fields if needed, like volumes, etc.
      },
    },
    // Add any other StatefulSetSpec configurations if needed, like volumeClaimTemplates, etc.
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
      return stsSpec
    }

    if stsSpec.Template.Annotations == nil {
      stsSpec.Template.Annotations = make(map[string]string)
    }
    stsSpec.Template.Annotations["checksum/config-files"] = configMapDataHash
    stsSpec.Template.Spec.Volumes = append(stsSpec.Template.Spec.Volumes, statefulSetVolume)

    // Mount the volume to the container
    volumeMount := corev1.VolumeMount{
      Name:      "config-volume",
      MountPath: "/path/to/mount", // Adjust this to your desired mount path
    }
    stsSpec.Template.Spec.Containers[0].VolumeMounts = append(stsSpec.Template.Spec.Containers[0].VolumeMounts, volumeMount)
  }

  return stsSpec
}
