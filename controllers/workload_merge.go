package controllers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func mergeContainersPreservingUnknown(existing, desired []corev1.Container) []corev1.Container {
	desiredByName := make(map[string]corev1.Container, len(desired))
	for _, c := range desired {
		desiredByName[c.Name] = c
	}

	merged := make([]corev1.Container, 0, len(existing)+len(desired))
	seen := make(map[string]struct{}, len(desired))

	for _, current := range existing {
		want, ok := desiredByName[current.Name]
		if !ok {
			merged = append(merged, current)
			continue
		}
		seen[current.Name] = struct{}{}
		merged = append(merged, mergeContainerPreservingMutations(current, want))
	}

	for _, want := range desired {
		if _, ok := seen[want.Name]; ok {
			continue
		}
		merged = append(merged, want)
	}

	return merged
}

func mergeContainerPreservingMutations(existing, desired corev1.Container) corev1.Container {
	merged := *existing.DeepCopy()

	merged.Name = desired.Name
	merged.Image = desired.Image
	merged.Command = desired.Command
	merged.Args = desired.Args
	merged.Resources = desired.Resources
	merged.LivenessProbe = desired.LivenessProbe
	merged.ReadinessProbe = desired.ReadinessProbe
	merged.StartupProbe = desired.StartupProbe

	// For controller-managed containers, desired spec is authoritative.
	merged.Env = desired.Env
	merged.EnvFrom = desired.EnvFrom
	merged.VolumeMounts = desired.VolumeMounts
	merged.Ports = desired.Ports

	return merged
}

func mergeEnvPreservingUnknown(existing, desired []corev1.EnvVar) []corev1.EnvVar {
	out := make([]corev1.EnvVar, 0, len(existing)+len(desired))
	seen := make(map[string]struct{}, len(desired))

	for _, v := range desired {
		out = append(out, v)
		seen[v.Name] = struct{}{}
	}
	for _, v := range existing {
		if _, ok := seen[v.Name]; ok {
			continue
		}
		out = append(out, v)
	}
	return out
}

func mergeEnvFromPreservingUnknown(existing, desired []corev1.EnvFromSource) []corev1.EnvFromSource {
	out := make([]corev1.EnvFromSource, 0, len(existing)+len(desired))
	out = append(out, desired...)

	for _, cur := range existing {
		found := false
		for _, want := range desired {
			if envFromEqual(cur, want) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, cur)
		}
	}
	return out
}

func mergeVolumeMountsPreservingUnknown(existing, desired []corev1.VolumeMount) []corev1.VolumeMount {
	out := make([]corev1.VolumeMount, 0, len(existing)+len(desired))
	seen := make(map[string]struct{}, len(desired))

	for _, v := range desired {
		out = append(out, v)
		seen[volumeMountKey(v)] = struct{}{}
	}
	for _, v := range existing {
		if _, ok := seen[volumeMountKey(v)]; ok {
			continue
		}
		out = append(out, v)
	}
	return out
}

func mergePortsPreservingUnknown(existing, desired []corev1.ContainerPort) []corev1.ContainerPort {
	out := make([]corev1.ContainerPort, 0, len(existing)+len(desired))
	seen := make(map[string]struct{}, len(desired))

	for _, p := range desired {
		out = append(out, p)
		seen[containerPortKey(p)] = struct{}{}
	}
	for _, p := range existing {
		if _, ok := seen[containerPortKey(p)]; ok {
			continue
		}
		out = append(out, p)
	}
	return out
}

func mergeVolumesPreservingUnknown(existing, desired []corev1.Volume) []corev1.Volume {
	desiredByName := make(map[string]corev1.Volume, len(desired))
	for _, v := range desired {
		desiredByName[v.Name] = v
	}

	out := make([]corev1.Volume, 0, len(existing)+len(desired))
	seen := make(map[string]struct{}, len(desired))

	for _, v := range existing {
		if want, ok := desiredByName[v.Name]; ok {
			out = append(out, want)
			seen[v.Name] = struct{}{}
			continue
		}
		out = append(out, v)
	}
	for _, v := range desired {
		if _, ok := seen[v.Name]; ok {
			continue
		}
		out = append(out, v)
	}
	return out
}

func mergeVolumesRespectingMountedNames(existing, desired []corev1.Volume, containers, initContainers []corev1.Container) []corev1.Volume {
	desiredByName := make(map[string]corev1.Volume, len(desired))
	for _, v := range desired {
		desiredByName[v.Name] = v
	}

	mounted := mountedVolumeNames(containers, initContainers)
	out := make([]corev1.Volume, 0, len(existing)+len(desired))
	seen := make(map[string]struct{}, len(desired))

	for _, v := range existing {
		if want, ok := desiredByName[v.Name]; ok {
			out = append(out, want)
			seen[v.Name] = struct{}{}
			continue
		}
		// Preserve unknown volumes still used by preserved/injected containers.
		if _, inUse := mounted[v.Name]; inUse {
			out = append(out, v)
		}
	}

	for _, v := range desired {
		if _, ok := seen[v.Name]; ok {
			continue
		}
		out = append(out, v)
	}
	return out
}

func mergeTolerationsPreservingUnknown(existing, desired []corev1.Toleration) []corev1.Toleration {
	out := make([]corev1.Toleration, 0, len(existing)+len(desired))
	out = append(out, desired...)
	for _, cur := range existing {
		found := false
		for _, want := range desired {
			if tolerationEqual(cur, want) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, cur)
		}
	}
	return out
}

func volumeMountKey(v corev1.VolumeMount) string {
	mode := ""
	if v.MountPropagation != nil {
		mode = string(*v.MountPropagation)
	}
	return v.Name + "|" + v.MountPath + "|" + mode
}

func containerPortKey(p corev1.ContainerPort) string {
	return fmt.Sprintf("%s|%s|%d", p.Name, p.Protocol, p.ContainerPort)
}

func mountedVolumeNames(containers, initContainers []corev1.Container) map[string]struct{} {
	names := make(map[string]struct{})
	add := func(items []corev1.Container) {
		for _, c := range items {
			for _, m := range c.VolumeMounts {
				names[m.Name] = struct{}{}
			}
		}
	}
	add(containers)
	add(initContainers)
	return names
}

func tolerationEqual(a, b corev1.Toleration) bool {
	if a.Key != b.Key || a.Operator != b.Operator || a.Value != b.Value || a.Effect != b.Effect {
		return false
	}
	if a.TolerationSeconds == nil && b.TolerationSeconds == nil {
		return true
	}
	if a.TolerationSeconds == nil || b.TolerationSeconds == nil {
		return false
	}
	return *a.TolerationSeconds == *b.TolerationSeconds
}

func envFromEqual(a, b corev1.EnvFromSource) bool {
	if a.Prefix != b.Prefix {
		return false
	}
	if (a.ConfigMapRef == nil) != (b.ConfigMapRef == nil) {
		return false
	}
	if a.ConfigMapRef != nil {
		if a.ConfigMapRef.Name != b.ConfigMapRef.Name || (a.ConfigMapRef.Optional == nil) != (b.ConfigMapRef.Optional == nil) {
			return false
		}
		if a.ConfigMapRef.Optional != nil && b.ConfigMapRef.Optional != nil && *a.ConfigMapRef.Optional != *b.ConfigMapRef.Optional {
			return false
		}
	}
	if (a.SecretRef == nil) != (b.SecretRef == nil) {
		return false
	}
	if a.SecretRef != nil {
		if a.SecretRef.Name != b.SecretRef.Name || (a.SecretRef.Optional == nil) != (b.SecretRef.Optional == nil) {
			return false
		}
		if a.SecretRef.Optional != nil && b.SecretRef.Optional != nil && *a.SecretRef.Optional != *b.SecretRef.Optional {
			return false
		}
	}
	return true
}
