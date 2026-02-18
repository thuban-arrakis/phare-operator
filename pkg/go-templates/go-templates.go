package gotemplates

import (
	"bytes"
	"fmt"
	"text/template"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ProcessLivenessProbeTemplate(probe *corev1.Probe, meta metav1.ObjectMeta) error {
	if probe == nil || probe.HTTPGet == nil {
		return nil
	}

	// Render template values in the liveness probe HTTP path.
	processedPath, err := ProcessTemplate(probe.HTTPGet.Path, meta)
	if err != nil {
		return fmt.Errorf("error processing template for livenessProbe path: %v", err)
	}
	probe.HTTPGet.Path = processedPath

	return nil
}

// ProcessTemplate renders a template string with object metadata values.
func ProcessTemplate(tmplStr string, meta metav1.ObjectMeta) (string, error) {
	tmpl, err := template.New("phareTemplate").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, meta); err != nil {
		return "", err
	}

	return buf.String(), nil
}
