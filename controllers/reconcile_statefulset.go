package controllers

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/google/go-cmp/cmp"
	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	tpl "github.com/localcorp/phare-controller/pkg/go-templates"
)

func (r *PhareReconciler) reconcileStatefulSet(ctx context.Context, phare pharev1beta1.Phare) error {
	desiredStatefulSet := r.newStatefulSet(&phare)
	if desiredStatefulSet == nil {
		return fmt.Errorf("failed to build desired StatefulSet for %s/%s", phare.Namespace, phare.Name)
	}

	// Inject ConfigMap hash annotation using the reconcile context so that pod
	// templates are rolled when config changes.
	if phare.Spec.ToolChain != nil && len(phare.Spec.ToolChain.Config) > 0 {
		hash, err := r.hashConfigMapData(ctx, phare.Name+"-config", phare.Namespace)
		if err != nil {
			return fmt.Errorf("hash configmap %s: %w", phare.Name+"-config", err)
		}
		if desiredStatefulSet.Spec.Template.Annotations == nil {
			desiredStatefulSet.Spec.Template.Annotations = make(map[string]string)
		}
		desiredStatefulSet.Spec.Template.Annotations["checksum/config-files"] = hash
	}

	existingStatefulSet := &appsv1.StatefulSet{}
	err := r.Get(ctx, client.ObjectKey{Name: desiredStatefulSet.Name, Namespace: phare.Namespace}, existingStatefulSet)

	if err != nil && errors.IsNotFound(err) {
		if createErr := r.Create(ctx, desiredStatefulSet); createErr != nil {
			return createErr
		}
	} else if err == nil {
		// Warn when the user changed VolumeClaimTemplates in the Phare spec: the field
		// is immutable on StatefulSets and the change cannot be applied without a
		// delete+recreate. Emit a Warning event so operators are not left guessing.
		if !vctEqual(existingStatefulSet.Spec.VolumeClaimTemplates, desiredStatefulSet.Spec.VolumeClaimTemplates) {
			r.Log.Info("VolumeClaimTemplates differ but are immutable after creation; changes ignored",
				"StatefulSet.Namespace", existingStatefulSet.Namespace, "StatefulSet.Name", existingStatefulSet.Name)
			r.Recorder.Eventf(&phare, corev1.EventTypeWarning, "ImmutableField",
				"VolumeClaimTemplates for StatefulSet %s cannot be changed after creation; delete and recreate to apply new templates",
				existingStatefulSet.Name)
		}

		// Keep a copy so we can patch only when something changed.
		originalStatefulSet := existingStatefulSet.DeepCopy()
		r.mergeStatefulSets(desiredStatefulSet, existingStatefulSet)

		diff := cmp.Diff(originalStatefulSet, existingStatefulSet, podTemplateCompareOptions())
		if diff != "" {
			patch := client.MergeFrom(originalStatefulSet)
			if patchErr := r.Patch(ctx, existingStatefulSet, patch); patchErr != nil {
				r.Log.Error(patchErr, "Error patching StatefulSet", "StatefulSet.Namespace", existingStatefulSet.Namespace, "StatefulSet.Name", existingStatefulSet.Name)
				return patchErr
			}
			r.Log.Info("StatefulSet patched successfully", "StatefulSet.Namespace", existingStatefulSet.Namespace, "StatefulSet.Name", existingStatefulSet.Name)
		} else {
			r.Log.Info("No changes detected", "StatefulSet.Namespace", existingStatefulSet.Namespace, "StatefulSet.Name", existingStatefulSet.Name)
		}
	} else {
		return err
	}

	return nil
}

func (r *PhareReconciler) mergeStatefulSets(desiredStatefulSet, existingStatefulSet *appsv1.StatefulSet) {
	existingStatefulSet.Spec.Replicas = desiredStatefulSet.Spec.Replicas
	existingStatefulSet.Spec.Template.Labels = copyStringMapPreserveNil(desiredStatefulSet.Spec.Template.Labels)
	existingStatefulSet.Spec.Template.Annotations = copyStringMapPreserveNil(desiredStatefulSet.Spec.Template.Annotations)

	spec := &existingStatefulSet.Spec.Template.Spec
	desired := &desiredStatefulSet.Spec.Template.Spec

	spec.Containers = mergeContainersPreservingUnknown(spec.Containers, desired.Containers)
	spec.InitContainers = mergeContainersPreservingUnknown(spec.InitContainers, desired.InitContainers)
	spec.Volumes = mergeVolumesRespectingMountedNames(spec.Volumes, desired.Volumes, spec.Containers, spec.InitContainers)
	spec.Tolerations = desired.Tolerations
	spec.Affinity = desired.Affinity
	// VolumeClaimTemplates are immutable after StatefulSet creation; never patch them.
}

func vctEqual(a, b []corev1.PersistentVolumeClaim) bool {
	return reflect.DeepEqual(a, b)
}

func (r *PhareReconciler) newStatefulSet(phare *pharev1beta1.Phare) *appsv1.StatefulSet {
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

	statefulSet := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      phare.Name,
			Namespace: phare.Namespace,
			Labels:    metadataLabels,
		},
		Spec: appsv1.StatefulSetSpec{
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
			VolumeClaimTemplates: phare.Spec.MicroService.VolumeClaimTemplates,
		},
	}

	// Set owner reference for the StatefulSet to be the Phare object.
	if err := ctrl.SetControllerReference(phare, statefulSet, r.Scheme); err != nil {
		r.Log.Error(err, "Failed to set controller reference for StatefulSet", "StatefulSet.Namespace", statefulSet.Namespace, "StatefulSet.Name", statefulSet.Name)
		return nil
	}

	// Add config volume only when toolchain config exists.
	if phare.Spec.ToolChain != nil && len(phare.Spec.ToolChain.Config) > 0 {
		addConfigVolumeToSpec(&statefulSet.Spec.Template, phare.Name+"-config")
	}

	// Set default file mode for Secret/ConfigMap volumes.
	for i := range statefulSet.Spec.Template.Spec.Volumes {
		UpdateVolume(&statefulSet.Spec.Template.Spec.Volumes[i], 420)
	}

	// Render liveness probe templates if present.
	if err := tpl.ProcessLivenessProbeTemplate(statefulSet.Spec.Template.Spec.Containers[0].LivenessProbe, phare.ObjectMeta); err != nil {
		r.Log.Error(err, "Error processing liveness probe template")
		return nil
	}

	return statefulSet
}
