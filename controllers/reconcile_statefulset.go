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
  yamldiff "github.com/localcorp/phare-controller/pkg/yamldiff"
)

func (r *PhareReconciler) reconcileStatefulSet(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) (ctrl.Result, error) {
  existingStatefulSet := &apps.StatefulSet{}
  desiredStatefulSet := r.desiredStatefulSet(&phare)
  err := r.Get(ctx, req.NamespacedName, existingStatefulSet)

  existingStatefulSetSpec := toYAML(existingStatefulSet)
  desiredStatefulSetSpec := toYAML(desiredStatefulSet)

  if err != nil {
    if errors.IsNotFound(err) {
      // StatefulSet doesn't exist, create it
      if err := r.Create(ctx, desiredStatefulSet); err != nil {
        r.Log.Info("Error creating StatefulSet")
        return ctrl.Result{}, err
      }
      r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "CreatedResource", "Created StatefulSet %s", desiredStatefulSet.Name)
      return ctrl.Result{}, nil
    } else {
      r.Log.Info("Error getting StatefulSet")
      return ctrl.Result{}, err
    }
  } else {
    isValid, desiredMap, modifiedCurrentMap := validator.ValidateYaml(desiredStatefulSetSpec, existingStatefulSetSpec)

    if !isValid {
      r.Log.Info("StatefulSet does not match the desired configuration", "StatefulSet.Namespace", desiredStatefulSet.Namespace, "StatefulSet.Name", desiredStatefulSet.Name)

      map1 := validator.PrintMap(modifiedCurrentMap) // Debugging purposes only
      map2 := validator.PrintMap(desiredMap)         // Debugging purposes only

      diffOutput := yamldiff.Diff(map1, map2) // Debugging purposes only
      fmt.Println(diffOutput)                 // Debugging purposes only

      patch := client.MergeFrom(existingStatefulSet.DeepCopy())
      r.Log.Info("Updating StatefulSet", "StatefulSet.Namespace", existingStatefulSet.Namespace, "StatefulSet.Name", existingStatefulSet.Name)

      // Copy desired StatefulSet's metadata and spec to existingStatefulSet
      existingStatefulSet.ObjectMeta = desiredStatefulSet.ObjectMeta
      existingStatefulSet.Spec = desiredStatefulSet.Spec

      if err := r.Patch(ctx, existingStatefulSet, patch, client.FieldOwner("phare-controller")); err != nil {
        r.Log.Info("Error patching StatefulSet")
        return ctrl.Result{}, err
      }
      return ctrl.Result{}, nil
    } else {
      r.Log.Info("StatefulSet matches the desired configuration", "StatefulSet.Namespace", desiredStatefulSet.Namespace, "StatefulSet.Name", desiredStatefulSet.Name)
    }
  }

  return ctrl.Result{}, nil
}

func (r *PhareReconciler) desiredStatefulSet(phare *pharev1beta1.Phare) *apps.StatefulSet {

  // Keep the same labels at the metadata level
  metadataLabels := map[string]string{
    "app":                          phare.Name,
    "app.kubernetes.io/created-by": "phare-controller",
    // "version":                      phare.Spec.MicroService.Image.Tag, // Use later for rolling updates
  }

  // Only use the "app" label for the spec level
  specLabels := map[string]string{
    "app": phare.Name,
  }

  statefulSet := &apps.StatefulSet{
    ObjectMeta: metav1.ObjectMeta{
      Name:      phare.Name,
      Namespace: phare.Namespace,
      Labels:    metadataLabels,
    },
    Spec: apps.StatefulSetSpec{
      Selector: &metav1.LabelSelector{
        MatchLabels: specLabels,
      },
      Replicas: &phare.Spec.MicroService.ReplicaCount,
      Template: corev1.PodTemplateSpec{
        ObjectMeta: metav1.ObjectMeta{
          Labels: specLabels,
        },
        Spec: corev1.PodSpec{
          Containers: []corev1.Container{
            {
              Name:  phare.Name,
              Image: phare.Spec.MicroService.Image.Repository + ":" + phare.Spec.MicroService.Image.Tag,
            },
          },
          InitContainers: phare.Spec.MicroService.InitContainers,
          Affinity:       phare.Spec.MicroService.Affinity,
          Tolerations:    phare.Spec.MicroService.Tolerations,
          Volumes:        phare.Spec.MicroService.Volumes,
        },
      },
    },
  }

  // Set the owner reference
  // Think about moving this to `statefulSet.ObjectMeta.OwnerReferences`
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

// Move this to pkg/utils or something.
func toYAML(obj interface{}) string {
  data, err := yaml.Marshal(obj)
  if err != nil {
    return fmt.Sprintf("Error marshaling to YAML: %s", err)
  }
  return string(data)
}
