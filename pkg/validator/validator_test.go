package validator

import "testing"

func TestValidateYamlInvalidInputDoesNotPanic(t *testing.T) {
	valid := "spec:\n  replicas: 1\n"
	invalid := "{not-yaml"

	ok, _, _ := ValidateYaml(valid, invalid)
	if ok {
		t.Fatalf("expected invalid YAML comparison to be false")
	}
}

func TestValidateYamlEqualContent(t *testing.T) {
	desired := "spec:\n  replicas: 1\n  selector:\n    app: demo\n"
	current := "spec:\n  replicas: 1\n  selector:\n    app: demo\n"

	ok, _, _ := ValidateYaml(desired, current)
	if !ok {
		t.Fatalf("expected YAML content to match")
	}
}
