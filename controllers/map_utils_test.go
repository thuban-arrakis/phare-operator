package controllers

import (
	"testing"
)

func TestMergeStringMaps(t *testing.T) {
	existing := map[string]string{"a": "1", "b": "old"}
	desired := map[string]string{"b": "new", "c": "3"}

	merged := mergeStringMaps(existing, desired)
	if merged["a"] != "1" || merged["c"] != "3" {
		t.Fatalf("expected keys from both maps in merge result")
	}
	if merged["b"] != "new" {
		t.Fatalf("expected desired map to override existing key")
	}
}

func TestCopyStringMapPreserveNil(t *testing.T) {
	if out := copyStringMapPreserveNil(nil); out != nil {
		t.Fatalf("expected nil map to stay nil, got %#v", out)
	}
	empty := map[string]string{}
	out := copyStringMapPreserveNil(empty)
	if out == nil || len(out) != 0 {
		t.Fatalf("expected empty non-nil map copy, got %#v", out)
	}
}

func TestStringMapsEqualNilEmpty(t *testing.T) {
	if !stringMapsEqualNilEmpty(nil, map[string]string{}) {
		t.Fatalf("expected nil and empty maps to be considered equal")
	}
	if stringMapsEqualNilEmpty(map[string]string{"a": "1"}, map[string]string{}) {
		t.Fatalf("expected differing maps to be considered different")
	}
}
