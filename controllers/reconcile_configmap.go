package controllers

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	tpl "github.com/localcorp/phare-controller/pkg/go-templates"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// reconcileConfigMap creates, updates, or deletes the managed ConfigMap.
func (r *PhareReconciler) reconcileConfigMap(ctx context.Context, phare pharev1beta1.Phare) error {
	// 1. Check if ToolChain and Config are non-nil.
	if phare.Spec.ToolChain == nil || phare.Spec.ToolChain.Config == nil {
		existingConfigMap := &corev1.ConfigMap{}
		err := r.Get(ctx, client.ObjectKey{Name: phare.Name + "-config", Namespace: phare.Namespace}, existingConfigMap)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}

		// If an existing ConfigMap was found, delete it.
		if err == nil {
			if deleteErr := r.Delete(ctx, existingConfigMap); deleteErr != nil && !errors.IsNotFound(deleteErr) {
				return deleteErr
			}
			r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "DeletedResource", "Deleted ConfigMap %s", existingConfigMap.Name)
		}
		return nil
	}

	// 2. Generate the desired ConfigMap as ToolChain and Config are non-nil.
	desiredConfigMap := r.generateConfigMap(phare)
	if desiredConfigMap == nil {
		return fmt.Errorf("failed to build desired ConfigMap for %s/%s", phare.Namespace, phare.Name)
	}

	// 3. Check the existence and state of the current ConfigMap.
	existingConfigMap := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{Name: desiredConfigMap.Name, Namespace: phare.Namespace}, existingConfigMap)

	if errors.IsNotFound(err) {
		// ConfigMap doesn't exist, create it.
		if err = r.Create(ctx, desiredConfigMap); err != nil {
			return err
		}
		r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "CreatedResource", "Created ConfigMap %s", desiredConfigMap.Name)
	} else if err == nil && !isDataEqual(existingConfigMap.Data, desiredConfigMap.Data) {
		// ConfigMap exists and differs from desired, update it.
		existingConfigMap.Data = desiredConfigMap.Data
		if updateErr := r.Update(ctx, existingConfigMap); updateErr != nil {
			return updateErr
		}
		r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "UpdatedResource", "Updated ConfigMap %s", desiredConfigMap.Name)
	} else if err != nil {
		// Some other error occurred while fetching the ConfigMap.
		return err
	}

	return nil
}

// generateConfigMap builds the desired ConfigMap from the Phare spec.
func (r *PhareReconciler) generateConfigMap(phare pharev1beta1.Phare) *corev1.ConfigMap {

	metadataLabels := map[string]string{
		"app":                          phare.Name,
		"app.kubernetes.io/created-by": "phare-controller",
		// "version":                      phare.Spec.MicroService.Image.Tag, // Use later for rolling updates
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      phare.Name + "-config",
			Namespace: phare.Namespace,
			Labels:    metadataLabels,
		},
		Data: copyStringMap(phare.Spec.ToolChain.Config),
	}

	// Go-templates support
	for key, tmplValue := range phare.Spec.ToolChain.Config {
		processedValue, err := tpl.ProcessTemplate(tmplValue, phare.ObjectMeta)
		if err != nil {
			r.Log.Error(err, "Error processing ConfigMap template", "ConfigMap.Name", configMap.Name, "Config.Key", key)
			continue
		}
		configMap.Data[key] = processedValue
	}

	// Set Phare CR as the owner of this ConfigMap
	if err := ctrl.SetControllerReference(&phare, configMap, r.Scheme); err != nil {
		r.Log.Error(err, "Failed to set controller reference for ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
		return nil
	}

	return configMap
}

// Utility function to compare map data
// Returns true if the maps are equal, false otherwise
func isDataEqual(map1, map2 map[string]string) bool {
	if len(map1) != len(map2) {
		return false
	}
	for k, v1 := range map1 {
		if v2, ok := map2[k]; !ok || v1 != v2 {
			return false
		}
	}
	return true
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// hashConfigMapData returns a deterministic SHA-256 hash of ConfigMap data.
// It returns an error if the ConfigMap does not exist.
// Data is encoded as "key=value\n" pairs to avoid ambiguous concatenation.
// Note: changing this encoding can cause a one-time checksum change and rollout.
func (r *PhareReconciler) hashConfigMapData(ctx context.Context, configMapName string, namespace string) (string, error) {
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: namespace}, cm); err != nil {
		return "", err
	}

	keys := make([]string, 0, len(cm.Data))
	for k := range cm.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(cm.Data[k])
		sb.WriteByte('\n')
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(sb.String()))), nil
}
