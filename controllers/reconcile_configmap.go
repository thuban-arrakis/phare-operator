package controllers

import (
  "context"
  "crypto/sha256"
  "fmt"
  "sort"

  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
  tpl "github.com/localcorp/phare-controller/pkg/go-templates"

  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/api/errors"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/apimachinery/pkg/types"
  ctrl "sigs.k8s.io/controller-runtime"
  "sigs.k8s.io/controller-runtime/pkg/client"
)

// The main function to reconcile the ConfigMap
// If the ConfigMap doesn't exist, create it
// If the ConfigMap exists, update it if necessary
// If the ConfigMap exists but is not specified in the Phare CR, delete it.
func (r *PhareReconciler) reconcileConfigMap(ctx context.Context, phare pharev1beta1.Phare) error {
  // 1. Check if ToolChain and Config are non-nil.
  if phare.Spec.ToolChain == nil || phare.Spec.ToolChain.Config == nil {
    existingConfigMap := &corev1.ConfigMap{}
    err := r.Get(ctx, client.ObjectKey{Name: phare.Name + "-config", Namespace: phare.Namespace}, existingConfigMap)

    // If an existing ConfigMap was found, delete it.
    if err == nil {
      if deleteErr := r.Delete(ctx, existingConfigMap); deleteErr != nil {
        return deleteErr
      }
      r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "DeletedResource", "Deleted ConfigMap %s", existingConfigMap.Name)
    }
    return nil
  }

  // 2. Generate the desired ConfigMap as ToolChain and Config are non-nil.
  desiredConfigMap := r.generateConfigMap(phare)

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
    r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "UpdatedResource", "Updated ConfigMap %s", desiredConfigMap.Name)
    if updateErr := r.Update(ctx, existingConfigMap); updateErr != nil {
      return updateErr
    }
  } else if err != nil {
    // Some other error occurred while fetching the ConfigMap.
    return err
  }

  return nil
}

// Generates a ConfigMap object based on the Phare CR
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
    Data: phare.Spec.ToolChain.Config,
  }

  // Go-templates support
  for key, tmplValue := range phare.Spec.ToolChain.Config {
    processedValue, err := tpl.ProcessTemplate(tmplValue, phare.ObjectMeta)
    if err != nil {
      fmt.Println("Error processing template: ", err)
    }
    configMap.Data[key] = processedValue
  }

  // Set Phare CR as the owner of this ConfigMap
  ctrl.SetControllerReference(&phare, configMap, r.Scheme)

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

// Utility function to hash the data of a ConfigMap
// CAUTION!: This function will return an error if the ConfigMap doesn't exist.

func (r *PhareReconciler) hashConfigMapData(configMapName string, namespace string) (string, error) {
  cm := &corev1.ConfigMap{}
  err := r.Get(context.TODO(), types.NamespacedName{Name: configMapName, Namespace: namespace}, cm)
  if err != nil {
    return "", err
  }

  keys := make([]string, 0, len(cm.Data))
  for k := range cm.Data {
    keys = append(keys, k)
  }
  sort.Strings(keys)

  hashData := ""
  for _, k := range keys {
    hashData = hashData + k + cm.Data[k]
  }

  return fmt.Sprintf("%x", sha256.Sum256([]byte(hashData))), nil
}
