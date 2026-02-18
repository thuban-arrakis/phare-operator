package gotemplates

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProcessTemplateDoesNotHTMLEscape(t *testing.T) {
	meta := metav1.ObjectMeta{
		Name: "svc<edge>&api",
	}

	out, err := ProcessTemplate("{{ .Name }}", meta)
	if err != nil {
		t.Fatalf("process template: %v", err)
	}
	if out != "svc<edge>&api" {
		t.Fatalf("expected unescaped output, got %q", out)
	}
}
