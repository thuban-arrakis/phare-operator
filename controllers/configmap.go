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
  // Log at the start
  // r.Log.Info("Starting reconcileConfigMap")

  // Check if spec.config is set
  if phare.Spec.Config == nil {
    // r.Log.Info("phare.Spec.Config is nil, skipping")
    return nil
  }
  // Check if spec.config is set
  if phare.Spec.Config == nil {
    return nil
  }

  // Create or update the ConfigMap
  cm := &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
      Name:      phare.Name + "-config",
      Namespace: phare.Namespace,
    },
    Data: phare.Spec.Config,
  }

  // Set the Phare instance as the owner and controller
  ctrl.SetControllerReference(&phare, cm, r.Scheme)

  // Check if this ConfigMap already exists
  found := &corev1.ConfigMap{}
  err := r.Get(ctx, client.ObjectKey{Name: cm.Name, Namespace: cm.Namespace}, found)
  // Log after checking for the ConfigMap existence
  // r.Log.Info("Checking for existing ConfigMap")
  if err != nil && errors.IsNotFound(err) {
    // Create the ConfigMap
    if err = r.Create(ctx, cm); err != nil {
      return err
    }
    // Log if ConfigMap is created
    // r.Log.Info("Creating new ConfigMap")
    // r.EventRecorder.Event(&phare, corev1.EventTypeNormal, "ConfigMapCreated", "Successfully created ConfigMap")
  } else if err != nil {
    return err
  } else {
    // ConfigMap already exists, update if necessary
    found.Data = cm.Data
    if err = r.Update(ctx, found); err != nil {
      return err
    }
  }

  return nil
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
