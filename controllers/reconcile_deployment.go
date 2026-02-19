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
	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
)

// configVolumeMountPath is the container path where the managed ConfigMap is mounted.
const configVolumeMountPath = "/etc/phare/config"

func (r *PhareReconciler) reconcileDeployment(ctx context.Context, phare pharev1beta1.Phare) error {
	desiredDeployment := r.newDeployment(&phare)
	if desiredDeployment == nil {
		return fmt.Errorf("failed to build desired Deployment for %s/%s", phare.Namespace, phare.Name)
	}

	// Inject ConfigMap hash annotation using the reconcile context so that pod
	// templates are rolled when config changes.
	if phare.Spec.ToolChain != nil && len(phare.Spec.ToolChain.Config) > 0 {
		hash, err := r.hashConfigMapData(ctx, phare.Name+"-config", phare.Namespace)
		if err != nil {
			r.Log.Error(err, "Error hashing ConfigMap data", "ConfigMap.Namespace", phare.Namespace, "ConfigMap.Name", phare.Name+"-config")
		}
		if desiredDeployment.Spec.Template.Annotations == nil {
			desiredDeployment.Spec.Template.Annotations = make(map[string]string)
		}
		desiredDeployment.Spec.Template.Annotations["checksum/config-files"] = hash
	}

	existingDeployment := &appsv1.Deployment{}
	err := r.Get(ctx, client.ObjectKey{Name: desiredDeployment.Name, Namespace: phare.Namespace}, existingDeployment)

	if err != nil && errors.IsNotFound(err) {
		if createErr := r.Create(ctx, desiredDeployment); createErr != nil {
			return createErr
		}
	} else if err == nil {
		// Keep a copy so we can generate a patch only when something changed.
		originalDeployment := existingDeployment.DeepCopy()

		// Merge the desired values into the current object.
		r.mergeDeployments(desiredDeployment, existingDeployment)

		// Compare old and new objects before patching.
		diff := cmp.Diff(originalDeployment, existingDeployment, podTemplateCompareOptions())
		if diff != "" {
			// Patch only if we found changes.
			patch := client.MergeFrom(originalDeployment)
			if patchErr := r.Patch(ctx, existingDeployment, patch); patchErr != nil {
				r.Log.Error(patchErr, "Error patching Deployment", "Deployment.Namespace", existingDeployment.Namespace, "Deployment.Name", existingDeployment.Name)
				return patchErr
			}
			r.Log.Info("Deployment patched successfully", "Deployment.Namespace", existingDeployment.Namespace, "Deployment.Name", existingDeployment.Name)
		} else {
			r.Log.Info("No changes detected", "Deployment.Namespace", existingDeployment.Namespace, "Deployment.Name", existingDeployment.Name)
		}
	} else {
		return err
	}

	return nil
}

// mergeDeployments applies controller-managed fields while keeping injected sidecars
// and related volumes that are still in use.
func (r *PhareReconciler) mergeDeployments(desiredDeployment, existingDeployment *appsv1.Deployment) {
	existingDeployment.Spec.Replicas = desiredDeployment.Spec.Replicas
	existingDeployment.Spec.Template.Labels = copyStringMapPreserveNil(desiredDeployment.Spec.Template.Labels)
	existingDeployment.Spec.Template.Annotations = copyStringMapPreserveNil(desiredDeployment.Spec.Template.Annotations)

	spec := &existingDeployment.Spec.Template.Spec
	desired := &desiredDeployment.Spec.Template.Spec

	spec.Containers = mergeContainersPreservingUnknown(spec.Containers, desired.Containers)
	spec.InitContainers = mergeContainersPreservingUnknown(spec.InitContainers, desired.InitContainers)
	spec.Volumes = mergeVolumesRespectingMountedNames(spec.Volumes, desired.Volumes, spec.Containers, spec.InitContainers)
	spec.Tolerations = desired.Tolerations
	spec.Affinity = desired.Affinity
}

func (r *PhareReconciler) newDeployment(phare *pharev1beta1.Phare) *appsv1.Deployment {
	// Base labels for resources created by this controller.
	metadataLabels := map[string]string{
		"app":                          phare.Name,
		"app.kubernetes.io/created-by": "phare-controller",
	}

	// Default pod labels and annotations.
	podLabels := map[string]string{
		"app": phare.Name,
	}
	for key, value := range phare.Spec.MicroService.PodLabels {
		podLabels[key] = value
	}

	podAnnotations := map[string]string{}
	for key, value := range phare.Spec.MicroService.PodAnnotations {
		podAnnotations[key] = value
	}

	containers := []corev1.Container{
		{
			Name:           phare.Name,
			Image:          phare.Spec.MicroService.Image.Repository + ":" + phare.Spec.MicroService.Image.Tag,
			VolumeMounts:   phare.Spec.MicroService.VolumeMounts,
			Command:        phare.Spec.MicroService.Command,
			Args:           phare.Spec.MicroService.Args,
			Env:            phare.Spec.MicroService.Env,
			EnvFrom:        phare.Spec.MicroService.EnvFrom,
			Ports:          phare.Spec.MicroService.Ports,
			Resources:      phare.Spec.MicroService.ResourceRequirements,
			LivenessProbe:  phare.Spec.MicroService.LivenessProbe,
			ReadinessProbe: phare.Spec.MicroService.ReadinessProbe,
			StartupProbe:   phare.Spec.MicroService.StartupProbe,
		},
	}
	containers = append(containers, phare.Spec.MicroService.ExtraContainers...)

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      phare.Name,
			Namespace: phare.Namespace,
			Labels:    metadataLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Replicas: &phare.Spec.MicroService.ReplicaCount,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					Containers:     containers,
					InitContainers: phare.Spec.MicroService.InitContainers,
					Affinity:       phare.Spec.MicroService.Affinity,
					Tolerations:    phare.Spec.MicroService.Tolerations,
					Volumes:        phare.Spec.MicroService.Volumes,
				},
			},
		},
	}

	// Set owner reference for the Deployment to be the Phare object.
	if err := ctrl.SetControllerReference(phare, deployment, r.Scheme); err != nil {
		r.Log.Error(err, "Failed to set controller reference for Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		return nil
	}

	// Add config volume only when toolchain config exists.
	if phare.Spec.ToolChain != nil && len(phare.Spec.ToolChain.Config) > 0 {
		addConfigVolumeToSpec(&deployment.Spec.Template, phare.Name+"-config")
	}

	// Set default file mode for Secret/ConfigMap volumes.
	for i := range deployment.Spec.Template.Spec.Volumes {
		UpdateVolume(&deployment.Spec.Template.Spec.Volumes[i], 420)
	}

	return deployment
}

// addConfigVolumeToSpec mounts the managed ConfigMap into the first container of
// the given pod template. Hash annotation injection is handled separately by the
// reconcile functions so they can use the request context.
func addConfigVolumeToSpec(template *corev1.PodTemplateSpec, configMapName string) {
	vol := corev1.Volume{
		Name: "config-volume",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
				DefaultMode: pointer.Int32(420),
				Optional:    pointer.Bool(false),
			},
		},
	}
	template.Spec.Volumes = append([]corev1.Volume{vol}, template.Spec.Volumes...)

	mount := corev1.VolumeMount{
		Name:      "config-volume",
		MountPath: configVolumeMountPath,
	}
	template.Spec.Containers[0].VolumeMounts = append([]corev1.VolumeMount{mount}, template.Spec.Containers[0].VolumeMounts...)
}

// UpdateVolume sets the default mode for Secret and ConfigMap volumes.
func UpdateVolume(volume *corev1.Volume, defaultMode int32) {
	if volume.Secret != nil {
		volume.Secret.DefaultMode = &defaultMode
	} else if volume.ConfigMap != nil {
		volume.ConfigMap.DefaultMode = &defaultMode
	}
}
