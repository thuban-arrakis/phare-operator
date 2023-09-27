package controllers

import (
  "context"
  "fmt"

  apps "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/api/errors"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/utils/pointer"
  ctrl "sigs.k8s.io/controller-runtime"
  "sigs.k8s.io/controller-runtime/pkg/client"
  "sigs.k8s.io/yaml"

  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
  "github.com/localcorp/phare-controller/pkg/validator"
)

func (r *PhareReconciler) reconcileStatefulSet(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) (ctrl.Result, error) {
  existingStatefulSet := &apps.StatefulSet{}
  desired := r.desiredStatefulSet(&phare)
  err := r.Get(ctx, req.NamespacedName, existingStatefulSet)

  a := toYAML(existingStatefulSet.Spec) // Rename it later
  b := toYAML(desired.Spec)             // Rename it later

  if err != nil {
    if errors.IsNotFound(err) {
      // StatefulSet doesn't exist, create it
      if err := r.Create(ctx, desired); err != nil {
        fmt.Println("Error creating StatefulSet")
        return ctrl.Result{}, err
      }
      r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "CreatedResource", "Created StatefulSet %s", desired.Name)
      return ctrl.Result{}, nil
    } else {
      fmt.Println("Error getting StatefulSet")
      return ctrl.Result{}, err
    }
  } else {
    isValid, desiredMap, modifiedCurrentMap := validator.ValidateYaml(b, a)
    // validator.PrintMap("Modified Current Map:", modifiedCurrentMap)
    // validator.PrintMap("Desired Map:", desiredMap)

    if !isValid {
      validator.PrintMap("Modified Current Map:", modifiedCurrentMap)
      validator.PrintMap("Desired Map:", desiredMap)
      patch := client.MergeFrom(existingStatefulSet.DeepCopy())
      r.Log.Info("Updating StatefulSet", "StatefulSet.Namespace", existingStatefulSet.Namespace, "StatefulSet.Name", existingStatefulSet.Name)
      existingStatefulSet.Spec = desired.Spec
      if err := r.Patch(ctx, existingStatefulSet, patch, client.FieldOwner("phare-controller")); err != nil {
        fmt.Println("Error patching StatefulSet")
        return ctrl.Result{}, err
      }
      return ctrl.Result{}, nil
    } else {
      fmt.Println("StatefulSet matches the desired configuration.")
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

func toYAML(obj interface{}) string {
  data, err := yaml.Marshal(obj)
  if err != nil {
    return fmt.Sprintf("Error marshaling to YAML: %s", err)
  }
  return string(data)
}
