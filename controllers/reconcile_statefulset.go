package controllers

import (
  "context"
  "fmt"

  appsv1 "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/api/errors"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/utils/pointer"
  ctrl "sigs.k8s.io/controller-runtime"
  "sigs.k8s.io/controller-runtime/pkg/client"

  "github.com/google/go-cmp/cmp"
  "github.com/google/go-cmp/cmp/cmpopts"
  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
)

func (r *PhareReconciler) reconcileStatefulSet(ctx context.Context, phare pharev1beta1.Phare) error {
  desiredStatefulSet := r.newStatefulSet(&phare)
  existingStatefulSet := &appsv1.StatefulSet{}
  err := r.Get(ctx, client.ObjectKey{Name: desiredStatefulSet.Name, Namespace: phare.Namespace}, existingStatefulSet)

  if err != nil && errors.IsNotFound(err) {
    if createErr := r.Create(ctx, desiredStatefulSet); createErr != nil {
      return createErr
    }
  } else if err == nil {
    originalStatefulSet := existingStatefulSet.DeepCopy()
    r.mergeStatefulSets(desiredStatefulSet, existingStatefulSet)

    // define the ignored fields for containers
    var IgnoreContainerFields = cmpopts.IgnoreFields(corev1.Container{}, "TerminationMessagePath", "TerminationMessagePolicy", "ImagePullPolicy")

    diff := cmp.Diff(originalStatefulSet, existingStatefulSet, IgnoreContainerFields)
    println("Diff: ", diff) // TODO: remove this, it's just for debugging
    if diff != "" {
      patch := client.MergeFrom(originalStatefulSet)
      if patchErr := r.Patch(ctx, existingStatefulSet, patch); patchErr != nil {
        println("Error patching StatefulSet: ", patchErr)
        return patchErr
      }
      println("StatefulSet patched successfully")
    } else {
      println("No changes detected in StatefulSet")
    }
  } else {
    return err
  }

  return nil
}

func (r *PhareReconciler) mergeStatefulSets(desiredStatefulSet, existingStatefulSet *appsv1.StatefulSet) {

  existingStatefulSet.Spec.Template.Spec.Containers[0].Image = desiredStatefulSet.Spec.Template.Spec.Containers[0].Image
  existingStatefulSet.Spec.Template.Spec.Containers[0].Command = desiredStatefulSet.Spec.Template.Spec.Containers[0].Command
  existingStatefulSet.Spec.Template.Spec.Containers[0].Args = desiredStatefulSet.Spec.Template.Spec.Containers[0].Args
  existingStatefulSet.Spec.Template.Spec.Containers[0].Env = desiredStatefulSet.Spec.Template.Spec.Containers[0].Env
  existingStatefulSet.Spec.Template.Spec.Containers[0].EnvFrom = desiredStatefulSet.Spec.Template.Spec.Containers[0].EnvFrom
  existingStatefulSet.Spec.Template.Spec.Containers[0].Ports = desiredStatefulSet.Spec.Template.Spec.Containers[0].Ports
  existingStatefulSet.Spec.Template.Spec.Containers[0].Resources = desiredStatefulSet.Spec.Template.Spec.Containers[0].Resources
  existingStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts = desiredStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts
  existingStatefulSet.Spec.Template.Spec.InitContainers = desiredStatefulSet.Spec.Template.Spec.InitContainers
  existingStatefulSet.Spec.Template.Spec.Affinity = desiredStatefulSet.Spec.Template.Spec.Affinity
  existingStatefulSet.Spec.Template.Spec.Tolerations = desiredStatefulSet.Spec.Template.Spec.Tolerations
  existingStatefulSet.Spec.Template.Spec.Volumes = desiredStatefulSet.Spec.Template.Spec.Volumes

}

func (r *PhareReconciler) newStatefulSet(phare *pharev1beta1.Phare) *appsv1.StatefulSet {

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

  statefulSet := &appsv1.StatefulSet{
    ObjectMeta: metav1.ObjectMeta{
      Name:      phare.Name,
      Namespace: phare.Namespace,
      Labels:    metadataLabels,
    },
    Spec: appsv1.StatefulSetSpec{
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
              Name:         phare.Name,
              Image:        phare.Spec.MicroService.Image.Repository + ":" + phare.Spec.MicroService.Image.Tag,
              VolumeMounts: phare.Spec.MicroService.VolumeMounts,
              Command:      phare.Spec.MicroService.Command,
              Args:         phare.Spec.MicroService.Args,
              Env:          phare.Spec.MicroService.Env,
              EnvFrom:      phare.Spec.MicroService.EnvFrom,
              Ports:        phare.Spec.MicroService.Ports,
              Resources:    phare.Spec.MicroService.ResourceRequirements,
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

  // Set owner reference for the statefulset to be the Phare object
  // https://book.kubebuilder.io/reference/using-finalizers.html#finalizer-owners.
  if err := ctrl.SetControllerReference(phare, statefulSet, r.Scheme); err != nil {
    r.Log.Error(err, "Failed to set controller reference for statefulset", "Statefulset.Namespace", statefulSet.Namespace, "Statefulset.Name", statefulSet.Name)
    return nil
  }

  // Check if the Spec.Toolchain.Config is not empty and add the ConfigMap volume
  if phare.Spec.ToolChain.Config != nil && len(phare.Spec.ToolChain.Config) > 0 {
    r.addConfigVolumeToStatefulset(statefulSet, phare)
  }

  // Preserve the default mode of the volumes.
  for i := range statefulSet.Spec.Template.Spec.Volumes {
    UpdateVolume(&statefulSet.Spec.Template.Spec.Volumes[i], 420) // or any default mode you want
  }

  return statefulSet
}

func (r *PhareReconciler) addConfigVolumeToStatefulset(statefulSet *appsv1.StatefulSet, phare *pharev1beta1.Phare) {
  statefulSetVolume := corev1.Volume{
    Name: "config-volume",
    VolumeSource: corev1.VolumeSource{
      ConfigMap: &corev1.ConfigMapVolumeSource{
        LocalObjectReference: corev1.LocalObjectReference{
          Name: phare.Name + "-config",
        },
        DefaultMode: pointer.Int32(420),
        Optional:    pointer.Bool(false),
      },
    },
  }
  configMapDataHash, err := r.hashConfigMapData(phare.Name+"-config", phare.Namespace)
  if err != nil {
    r.Log.Info(fmt.Sprintf("Error hashing ConfigMap data: %s", err))
    // Handle error, maybe return an error or log it
  }

  if statefulSet.Spec.Template.Annotations == nil {
    statefulSet.Spec.Template.Annotations = make(map[string]string)
  }
  statefulSet.Spec.Template.Annotations["checksum/config-files"] = configMapDataHash

  // Prepend the volume to the beginning of the Volumes slice
  statefulSet.Spec.Template.Spec.Volumes = append([]corev1.Volume{statefulSetVolume}, statefulSet.Spec.Template.Spec.Volumes...)

  // Prepend the volume mount for the container
  volumeMount := corev1.VolumeMount{
    Name:      "config-volume",
    MountPath: "/path/to/mount",
  }
  statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts = append([]corev1.VolumeMount{volumeMount}, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts...)
}
