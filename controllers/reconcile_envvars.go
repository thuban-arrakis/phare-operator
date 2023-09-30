package controllers

import (
  "context"
  "fmt"

  "github.com/localcorp/phare-controller/api/v1beta1"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/types"
  "sigs.k8s.io/controller-runtime/pkg/client"
)

// setEnvVars sets the environment variables for the pod template
// using the provided Phare spec.
func setEnvVars(phareSpec v1beta1.MicroServiceSpec, container *corev1.Container) error {
  // If no env vars are defined in the spec, return early
  if len(phareSpec.Env) == 0 {
    return nil
  }

  // Set environment variables
  container.Env = append(container.Env, phareSpec.Env...)

  return nil
}

// resolveEnvVarFromSource fetches the value of an environment variable
// from a ConfigMap or Secret based on the provided source reference.
func resolveEnvVarFromSource(ctx context.Context, client client.Client, namespace string, source *corev1.EnvVarSource) (string, error) {
  if source.ConfigMapKeyRef != nil {
    cm := &corev1.ConfigMap{}
    err := client.Get(ctx, types.NamespacedName{
      Namespace: namespace,
      Name:      source.ConfigMapKeyRef.Name,
    }, cm)
    if err != nil {
      return "", err
    }

    val, ok := cm.Data[source.ConfigMapKeyRef.Key]
    if !ok {
      return "", fmt.Errorf("key %q not found in ConfigMap %q", source.ConfigMapKeyRef.Key, source.ConfigMapKeyRef.Name)
    }

    return val, nil
  } else if source.SecretKeyRef != nil {
    secret := &corev1.Secret{}
    err := client.Get(ctx, types.NamespacedName{
      Namespace: namespace,
      Name:      source.SecretKeyRef.Name,
    }, secret)
    if err != nil {
      return "", err
    }

    val, ok := secret.Data[source.SecretKeyRef.Key]
    if !ok {
      return "", fmt.Errorf("key %q not found in Secret %q", source.SecretKeyRef.Key, source.SecretKeyRef.Name)
    }

    return string(val), nil
  }

  return "", fmt.Errorf("unsupported env var source")
}
