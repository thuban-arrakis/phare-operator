package controllers

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func podTemplateCompareOptions() cmp.Options {
	return cmp.Options{
		cmpopts.IgnoreFields(corev1.Container{}, "TerminationMessagePath", "TerminationMessagePolicy", "ImagePullPolicy"),
		cmpopts.IgnoreFields(corev1.Probe{}, "TimeoutSeconds", "SuccessThreshold", "FailureThreshold", "PeriodSeconds"),
		cmpopts.IgnoreFields(corev1.HTTPGetAction{}, "Scheme"),
	}
}
