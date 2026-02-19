package controllers

import (
	"testing"

	pharev1beta1 "github.com/localcorp/phare-controller/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGenerateConfigMapDoesNotMutateSpecConfig(t *testing.T) {
	scheme := testScheme(t)

	r := &PhareReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme: scheme,
	}

	phare := pharev1beta1.Phare{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "default",
		},
		Spec: pharev1beta1.PhareSpec{
			ToolChain: &pharev1beta1.ToolChainSpec{
				Config: pharev1beta1.ConfigSpec{
					"raw":       "plain",
					"templated": "/{{ .Name }}/ready",
				},
			},
		},
	}

	cm := r.generateConfigMap(phare)
	if cm.Data["templated"] != "/app/ready" {
		t.Fatalf("expected templated value to be rendered, got %q", cm.Data["templated"])
	}
	if phare.Spec.ToolChain.Config["templated"] != "/{{ .Name }}/ready" {
		t.Fatalf("expected source spec config to remain unchanged, got %q", phare.Spec.ToolChain.Config["templated"])
	}
}
