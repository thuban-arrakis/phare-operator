package controllers

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestMergeVolumesRespectingMountedNamesPreservesVolumeDevices(t *testing.T) {
	existing := []corev1.Volume{
		{
			Name: "injected-block",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "stale",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
	desired := []corev1.Volume{
		{
			Name: "desired",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
	containers := []corev1.Container{
		{
			Name: "injected-sidecar",
			VolumeDevices: []corev1.VolumeDevice{
				{
					Name:       "injected-block",
					DevicePath: "/dev/xvdb",
				},
			},
		},
	}

	got := mergeVolumesRespectingMountedNames(existing, desired, containers, nil)

	if !hasVolume(got, "desired") {
		t.Fatalf("expected desired volume to be present")
	}
	if !hasVolume(got, "injected-block") {
		t.Fatalf("expected injected block volume to be preserved")
	}
	if hasVolume(got, "stale") {
		t.Fatalf("expected stale unmounted volume to be removed")
	}
}

func hasVolume(volumes []corev1.Volume, name string) bool {
	for _, v := range volumes {
		if v.Name == name {
			return true
		}
	}
	return false
}
