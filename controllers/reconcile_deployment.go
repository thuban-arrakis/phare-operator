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

func (r *PhareReconciler) reconcileDeployment(ctx context.Context, phare pharev1beta1.Phare) error {
  desiredDeployment := r.newDeployment(&phare)
  existingDeployment := &appsv1.Deployment{}
  err := r.Get(ctx, client.ObjectKey{Name: desiredDeployment.Name, Namespace: phare.Namespace}, existingDeployment)

  if err != nil && errors.IsNotFound(err) {
    if createErr := r.Create(ctx, desiredDeployment); createErr != nil {
      return createErr
    }
  } else if err == nil {
    // Make a deep copy of existingDeployment to store the original state
    originalDeployment := existingDeployment.DeepCopy()

    // Modify the existingDeployment in memory
    r.mergeDeployments(desiredDeployment, existingDeployment)

    // define the ignored fields for containers
    var IgnoreContainerFields = cmpopts.IgnoreFields(corev1.Container{}, "TerminationMessagePath", "TerminationMessagePolicy", "ImagePullPolicy")

    // Use cmp.Diff to determine differences
    diff := cmp.Diff(originalDeployment, existingDeployment, IgnoreContainerFields)
    // println("Diff: ", diff) // TODO: remove this, it's just for debugging
    if diff != "" {
      // Calculate the patch using MergeFrom if differences are detected
      patch := client.MergeFrom(originalDeployment)
      if patchErr := r.Patch(ctx, existingDeployment, patch); patchErr != nil {
        println("Error patching Deployment: ", patchErr)
        return patchErr
      }
      println("Deployment patched successfully")
    } else {
      println("No changes detected") // TODO: remove this, it's just for debugging
    }
  } else {
    // Handle other potential errors
    return err
  }

  return nil
}

// This is a mess, but at this point I've not found a better way to do this.
// Maybe I should just use the k8s client-go library directly?
// Or wrap Statefulset/Deployment pod template spec in a struct and use that?
// Anywaways, it works for now.
// NOTE: Now cmp.Diff is used to determine differences with `cmpopts.IgnoreFields`, so it must be some overhead.
func (r *PhareReconciler) mergeDeployments(desiredDeployment, existingDeployment *appsv1.Deployment) {
  existingDeployment.Spec.Template.Spec.Containers[0].Image = desiredDeployment.Spec.Template.Spec.Containers[0].Image
  existingDeployment.Spec.Template.Spec.Containers[0].Command = desiredDeployment.Spec.Template.Spec.Containers[0].Command
  existingDeployment.Spec.Template.Spec.Containers[0].Args = desiredDeployment.Spec.Template.Spec.Containers[0].Args
  existingDeployment.Spec.Template.Spec.Containers[0].Env = desiredDeployment.Spec.Template.Spec.Containers[0].Env
  existingDeployment.Spec.Template.Spec.Containers[0].EnvFrom = desiredDeployment.Spec.Template.Spec.Containers[0].EnvFrom
  existingDeployment.Spec.Template.Spec.Containers[0].Ports = desiredDeployment.Spec.Template.Spec.Containers[0].Ports
  existingDeployment.Spec.Template.Spec.Containers[0].Resources = desiredDeployment.Spec.Template.Spec.Containers[0].Resources
  existingDeployment.Spec.Template.Spec.Containers[0].VolumeMounts = desiredDeployment.Spec.Template.Spec.Containers[0].VolumeMounts
  existingDeployment.Spec.Template.Spec.InitContainers = desiredDeployment.Spec.Template.Spec.InitContainers
  existingDeployment.Spec.Template.Spec.Affinity = desiredDeployment.Spec.Template.Spec.Affinity
  existingDeployment.Spec.Template.Spec.Tolerations = desiredDeployment.Spec.Template.Spec.Tolerations
  existingDeployment.Spec.Template.Spec.Volumes = desiredDeployment.Spec.Template.Spec.Volumes
}

func (r *PhareReconciler) newDeployment(phare *pharev1beta1.Phare) *appsv1.Deployment {
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

  // Preserve the default mode of the volumes.
  for i := range deployment.Spec.Template.Spec.Volumes {
    UpdateVolume(&deployment.Spec.Template.Spec.Volumes[i], 420) // or any default mode you want
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

// UpdateVolume updates the default mode of a volume. And that's it. Really.
// NOTE: Can be optimized by using a pointer to the default mode.
// Or rid of at all, since we can use IgnoreFields in cmp.Diff.
func UpdateVolume(volume *corev1.Volume, defaultMode int32) {
  if volume.Secret != nil {
    volume.Secret.DefaultMode = &defaultMode
  } else if volume.ConfigMap != nil {
    volume.ConfigMap.DefaultMode = &defaultMode
  }
}
