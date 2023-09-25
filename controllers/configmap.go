package controllers

import (
  "context"
  "crypto/sha256"
  "fmt"
  "sort"

  pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/api/errors"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/apimachinery/pkg/types"
  ctrl "sigs.k8s.io/controller-runtime"
  "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *PhareReconciler) reconcileConfigMap(ctx context.Context, phare pharev1beta1.Phare) error {

  // 1. Construct Desired ConfigMap
  desiredCM := &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
      Name:      phare.Name + "-config",
      Namespace: phare.Namespace,
    },
    Data: phare.Spec.Config,
  }

  // Set Phare CR as the owner of this ConfigMap
  ctrl.SetControllerReference(&phare, desiredCM, r.Scheme)

  // 2. Check for Existing ConfigMap
  existingCM := &corev1.ConfigMap{}
  err := r.Get(ctx, client.ObjectKey{Name: desiredCM.Name, Namespace: desiredCM.Namespace}, existingCM)

  if err != nil {
    if errors.IsNotFound(err) {
      // ConfigMap doesn't exist, create it
      if err = r.Create(ctx, desiredCM); err != nil {
        return err
      }
      r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "CreatedResource", "Created ConfigMap %s", existingCM.Name)
    } else {
      // Another error occurred
      return err
    }
  } else {
    // 3. Check if Phare CR's spec.config has changed
    if !isDataEqual(existingCM.Data, desiredCM.Data) {
      existingCM.Data = desiredCM.Data
      r.Recorder.Eventf(&phare, corev1.EventTypeNormal, "UpdatedResource", "Updated ConfigMap %s", desiredCM.Name)
      if err = r.Update(ctx, existingCM); err != nil {
        return err
      }
    }

    // 4. If ConfigMap data itself has been changed, reconcile it
    // This is handled automatically because the desiredCM is always constructed from Phare CR's spec.config
  }

  return nil
}

// Utility function to compare map data
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
