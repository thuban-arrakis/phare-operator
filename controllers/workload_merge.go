package controllers

import corev1 "k8s.io/api/core/v1"

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

func mountedVolumeNames(containers, initContainers []corev1.Container) map[string]struct{} {
	names := make(map[string]struct{})
	add := func(items []corev1.Container) {
		for _, c := range items {
			for _, m := range c.VolumeMounts {
				names[m.Name] = struct{}{}
			}
			for _, d := range c.VolumeDevices {
				names[d.Name] = struct{}{}
			}
		}
	}
	add(containers)
	add(initContainers)
	return names
}
