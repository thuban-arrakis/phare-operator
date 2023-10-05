package controllers

import (
  "context"
  "fmt"
  "reflect"

  appsv1 "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/api/errors"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/client-go/util/retry"
  "k8s.io/utils/pointer"
  ctrl "sigs.k8s.io/controller-runtime"
  "sigs.k8s.io/controller-runtime/pkg/client"

  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
)

func (r *PhareReconciler) reconcileDeployment(ctx context.Context, phare pharev1beta1.Phare) error {
  desiredDeployment := r.desiredDeployment(&phare)
  existingDeployment := &appsv1.Deployment{}
  err := r.Get(ctx, client.ObjectKey{Name: desiredDeployment.Name, Namespace: phare.Namespace}, existingDeployment)

  if err != nil && errors.IsNotFound(err) {
    if createErr := r.Create(ctx, desiredDeployment); createErr != nil {
      return createErr
    }
  } else if err == nil {
    if !deploymentsAreEqual(existingDeployment, desiredDeployment) {
      updateFunc := func() error {
        // Fetch the most recent version of the deployment
        if getErr := r.Get(ctx, client.ObjectKey{Name: desiredDeployment.Name, Namespace: phare.Namespace}, existingDeployment); getErr != nil {
          return getErr
        }

        // Modify existingDeployment based on desiredDeployment
        // Workaround for `the object has been modified; please apply your changes to the latest version and try again`.
        desiredDeployment.ResourceVersion = existingDeployment.ResourceVersion
        // Temp solution. Need to find a better way to update the deployment. Maybe use client.MergeFrom? Or construct pod template from scratch.
        existingDeployment.Spec.Replicas = desiredDeployment.Spec.Replicas // is this necessary?
        existingDeployment.Spec.Template.Spec.Containers[0].Image = desiredDeployment.Spec.Template.Spec.Containers[0].Image
        existingDeployment.Spec.Template.Spec.Containers[0].Command = desiredDeployment.Spec.Template.Spec.Containers[0].Command // so
        existingDeployment.Spec.Template.Spec.Containers[0].Args = desiredDeployment.Spec.Template.Spec.Containers[0].Args       // stupid
        existingDeployment.Spec.Template.Spec.Containers[0].Ports = desiredDeployment.Spec.Template.Spec.Containers[0].Ports     // redo
        existingDeployment.Spec.Template.Spec.Containers[0].Resources = desiredDeployment.Spec.Template.Spec.Containers[0].Resources
        existingDeployment.Spec.Template.Spec.Containers[0].Env = desiredDeployment.Spec.Template.Spec.Containers[0].Env
        existingDeployment.Spec.Template.Spec.Volumes = desiredDeployment.Spec.Template.Spec.Volumes // "Granular update", yea right
        existingDeployment.Spec.Template.Spec.Containers[0].VolumeMounts = desiredDeployment.Spec.Template.Spec.Containers[0].VolumeMounts

        existingDeployment.Spec.Template.Spec.Affinity = desiredDeployment.Spec.Template.Spec.Affinity

        // Update and return any error
        return r.Patch(ctx, existingDeployment, client.MergeFrom(desiredDeployment)) // This is last modified. Todo: check if it works.
      }
      if retryErr := retry.RetryOnConflict(retry.DefaultRetry, updateFunc); retryErr != nil {
        r.Log.Error(retryErr, "Failed to update Deployment after retrying", "Deployment.Namespace", existingDeployment.Namespace, "Deployment.Name", existingDeployment.Name)
        return retryErr
      }
    }
  } else {
    // Handle other potential errors
    return err
  }

  return nil
}

func (r *PhareReconciler) desiredDeployment(phare *pharev1beta1.Phare) *appsv1.Deployment {

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

  deployment := &appsv1.Deployment{
    ObjectMeta: metav1.ObjectMeta{
      Name:      phare.Name,
      Namespace: phare.Namespace,
      Labels:    metadataLabels,
    },
    Spec: appsv1.DeploymentSpec{
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

  // Set owner reference for the Deployment to be the Phare object
  // https://book.kubebuilder.io/reference/using-finalizers.html#finalizer-owners.
  if err := ctrl.SetControllerReference(phare, deployment, r.Scheme); err != nil {
    r.Log.Error(err, "Failed to set controller reference for Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
    return nil
  }

  // Check if the Spec.Toolchain.Config is not empty and add the ConfigMap volume
  if phare.Spec.ToolChain.Config != nil && len(phare.Spec.ToolChain.Config) > 0 {
    r.addConfigVolumeToDeployment(deployment, phare)
  }
  return deployment
}

func (r *PhareReconciler) addConfigVolumeToDeployment(deployment *appsv1.Deployment, phare *pharev1beta1.Phare) {
  deploymentVolume := corev1.Volume{
    Name: "config-volume",
    VolumeSource: corev1.VolumeSource{
      ConfigMap: &corev1.ConfigMapVolumeSource{
        LocalObjectReference: corev1.LocalObjectReference{
          Name: phare.Name + "-config",
        },
        Optional: pointer.Bool(false),
      },
    },
  }
  configMapDataHash, err := r.hashConfigMapData(phare.Name+"-config", phare.Namespace)
  if err != nil {
    r.Log.Info(fmt.Sprintf("Error hashing ConfigMap data: %s", err))
    // Handle error, maybe return an error or log it
  }

  if deployment.Spec.Template.Annotations == nil {
    deployment.Spec.Template.Annotations = make(map[string]string)
  }
  deployment.Spec.Template.Annotations["checksum/config-files"] = configMapDataHash

  // Prepend the volume to the beginning of the Volumes slice
  deployment.Spec.Template.Spec.Volumes = append([]corev1.Volume{deploymentVolume}, deployment.Spec.Template.Spec.Volumes...)

  // Prepend the volume mount for the container
  volumeMount := corev1.VolumeMount{
    Name:      "config-volume",
    MountPath: "/path/to/mount",
  }
  deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append([]corev1.VolumeMount{volumeMount}, deployment.Spec.Template.Spec.Containers[0].VolumeMounts...)
}

func deploymentsAreEqual(existingDeployment, desiredDeployment *appsv1.Deployment) bool {
  // Compare replicas
  if *existingDeployment.Spec.Replicas != *desiredDeployment.Spec.Replicas {
    return false
  }

  // Compare container image, environment variables, etc.
  // Assuming only a single container here for simplicity, you might need to iterate through containers if more than one
  if existingDeployment.Spec.Template.Spec.Containers[0].Image != desiredDeployment.Spec.Template.Spec.Containers[0].Image {
    return false
  }

  // Compare command using DeepEqual here for simplicity
  if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[0].Command, desiredDeployment.Spec.Template.Spec.Containers[0].Command) {
    return false
  }

  // Compare ports using DeepEqual here for simplicity
  if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[0].Ports, desiredDeployment.Spec.Template.Spec.Containers[0].Ports) {
    return false
  }

  // Compare resources using DeepEqual here for simplicity
  if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[0].Resources, desiredDeployment.Spec.Template.Spec.Containers[0].Resources) {
    return false
  }

  // Using DeepEqual here for environment variables as it's a slice of key-value pairs
  if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[0].Env, desiredDeployment.Spec.Template.Spec.Containers[0].Env) {
    return false
  }

  // Compare volumes using the volumesAreEqual function
  if !volumesAreEqual(existingDeployment.Spec.Template.Spec.Volumes, desiredDeployment.Spec.Template.Spec.Volumes) {
    return false
  }

  // Compare volume mounts
  // Again assuming only a single container for simplicity
  if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[0].VolumeMounts, desiredDeployment.Spec.Template.Spec.Containers[0].VolumeMounts) {
    return false
  }

  // Add more comparisons as required...
  return true
}

// Helper function to check if the ConfigMap volume source is equal
// in config volume sources of two volumes.
func volumesAreEqual(volume1, volume2 []corev1.Volume) bool {
  // Helper function to get ConfigMap volume source for volume named "config-volume"
  getConfigVolumeSource := func(volumes []corev1.Volume) *corev1.ConfigMapVolumeSource {
    for _, v := range volumes {
      if v.Name == "config-volume" {
        return v.ConfigMap
      }
    }
    return nil
  }

  configVolumeSource1 := getConfigVolumeSource(volume1)
  configVolumeSource2 := getConfigVolumeSource(volume2)

  // If both are nil, they are equal
  if configVolumeSource1 == nil && configVolumeSource2 == nil {
    return true
  }

  // If one is nil and the other isn't, they aren't equal
  if configVolumeSource1 == nil || configVolumeSource2 == nil {
    return false
  }

  // Now compare relevant fields of the ConfigMap volume sources, excluding defaultMode
  return configVolumeSource1.Name == configVolumeSource2.Name &&
    configVolumeSource1.Optional == configVolumeSource2.Optional
  // Add more fields if needed, but keep excluding defaultMode
}
