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

  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
  "github.com/localcorp/phare-controller/pkg/validator"
  yamldiff "github.com/localcorp/phare-controller/pkg/yamldiff"
)

func (r *PhareReconciler) reconcileDeployment(ctx context.Context, req ctrl.Request, phare pharev1beta1.Phare) (ctrl.Result, error) {
  existingDeployment := &apps.Deployment{}
  desiredDeployment := r.desiredDeployment(&phare)
  err := r.Get(ctx, req.NamespacedName, existingDeployment)

  existingDeploymentSpec := toYAML(existingDeployment)
  desiredDeploymentSpec := toYAML(desiredDeployment)

  if err != nil {
    if errors.IsNotFound(err) {
      // Deployment doesn't exist, create it
      if err := r.Create(ctx, desiredDeployment); err != nil {
        r.Log.Info("Error creating Deployment")
        return ctrl.Result{}, err
      }
      r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "CreatedResource", "Created Deployment %s", desiredDeployment.Name)
      return ctrl.Result{}, nil
    } else {
      r.Log.Info("Error getting Deployment")
      return ctrl.Result{}, err
    }
  } else {
    isValid, desiredMap, modifiedCurrentMap := validator.ValidateYaml(desiredDeploymentSpec, existingDeploymentSpec)

    if !isValid {
      r.Log.Info("Deployment does not match the desired configuration", "Deployment.Namespace", desiredDeployment.Namespace, "Deployment.Name", desiredDeployment.Name)

      map1 := validator.PrintMap(modifiedCurrentMap) // Debugging purposes only
      map2 := validator.PrintMap(desiredMap)         // Debugging purposes only

      diffOutput := yamldiff.Diff(map1, map2) // Debugging purposes only
      fmt.Println(diffOutput)                 // Debugging purposes only

      patch := client.MergeFrom(existingDeployment.DeepCopy())
      r.Log.Info("Updating Deployment", "Deployment.Namespace", existingDeployment.Namespace, "Deployment.Name", existingDeployment.Name)

      // Copy desired Deployment's metadata and spec to existingDeployment
      existingDeployment.ObjectMeta = desiredDeployment.ObjectMeta
      existingDeployment.Spec = desiredDeployment.Spec

      if err := r.Patch(ctx, existingDeployment, patch, client.FieldOwner("phare-controller")); err != nil {
        r.Log.Info("Error patching Deployment")
        return ctrl.Result{}, err
      }
      return ctrl.Result{}, nil
    } else {
      r.Log.Info("Deployment matches the desired configuration", "Deployment.Namespace", desiredDeployment.Namespace, "Deployment.Name", desiredDeployment.Name)
    }
  }

  return ctrl.Result{}, nil
}

func (r *PhareReconciler) desiredDeployment(phare *pharev1beta1.Phare) *apps.Deployment {

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

  deployment := &apps.Deployment{
    ObjectMeta: metav1.ObjectMeta{
      Name:      phare.Name,
      Namespace: phare.Namespace,
      Labels:    metadataLabels,
    },
    Spec: apps.DeploymentSpec{
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
  // Think about moving this to `Deployment.ObjectMeta.OwnerReferences`
  deployment.OwnerReferences = []metav1.OwnerReference{
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
