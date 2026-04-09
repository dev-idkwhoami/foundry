package features

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// helper to build a registry from a slice of features, wiring up the
// dependency and incompatibility maps the same way BuildRegistry does.
func buildTestRegistry(features []*Feature) *Registry {
	r := &Registry{
		Features:      features,
		DependencyMap: make(map[string][]string),
		IncompatMap:   make(map[string][]string),
	}

	for _, f := range features {
		if len(f.Requires) > 0 {
			r.DependencyMap[f.ID] = f.Requires
		}
	}

	for _, f := range features {
		for _, inc := range f.Incompatible {
			r.IncompatMap[f.ID] = appendUnique(r.IncompatMap[f.ID], inc)
			r.IncompatMap[inc] = appendUnique(r.IncompatMap[inc], f.ID)
		}
	}

	return r
}

func TestTopologicalSort_DependencyOrder(t *testing.T) {
	// A requires B, B requires C → valid order must have C before B before A.
	r := buildTestRegistry([]*Feature{
		{ID: "A", Requires: []string{"B"}},
		{ID: "B", Requires: []string{"C"}},
		{ID: "C"},
	})

	sorted, err := r.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort returned unexpected error: %v", err)
	}

	indexOf := func(id string) int {
		for i, s := range sorted {
			if s == id {
				return i
			}
		}
		return -1
	}

	if indexOf("C") > indexOf("B") {
		t.Errorf("C must come before B, got order %v", sorted)
	}
	if indexOf("B") > indexOf("A") {
		t.Errorf("B must come before A, got order %v", sorted)
	}
}

func TestTopologicalSort_CycleDetection(t *testing.T) {
	// A requires B, B requires A → cycle.
	r := buildTestRegistry([]*Feature{
		{ID: "A", Requires: []string{"B"}},
		{ID: "B", Requires: []string{"A"}},
	})

	_, err := r.TopologicalSort()
	if err == nil {
		t.Fatal("expected error for cyclic dependencies, got nil")
	}
}

func TestIncompatibility_Bidirectional(t *testing.T) {
	// A declares incompatible: [B]. Both directions must be queryable.
	r := buildTestRegistry([]*Feature{
		{ID: "A", Incompatible: []string{"B"}},
		{ID: "B"},
	})

	aIncompat := r.GetIncompatible("A")
	if !slices.Contains(aIncompat, "B") {
		t.Errorf("GetIncompatible(\"A\") = %v, want it to contain \"B\"", aIncompat)
	}

	bIncompat := r.GetIncompatible("B")
	if !slices.Contains(bIncompat, "A") {
		t.Errorf("GetIncompatible(\"B\") = %v, want it to contain \"A\"", bIncompat)
	}
}

func TestBuildRegistry_TempDir(t *testing.T) {
	tmpDir := t.TempDir()

	featureDir := filepath.Join(tmpDir, "features", "foo")
	if err := os.MkdirAll(featureDir, 0o755); err != nil {
		t.Fatalf("creating temp feature dir: %v", err)
	}

	manifest := []byte(`id: foo
name: Foo Feature
description: A test feature
requires:
  - bar
incompatible:
  - baz
`)

	if err := os.WriteFile(filepath.Join(featureDir, "manifest.yaml"), manifest, 0o644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}

	r, err := BuildRegistry(tmpDir)
	if err != nil {
		t.Fatalf("BuildRegistry returned unexpected error: %v", err)
	}

	f := r.GetFeature("foo")
	if f == nil {
		t.Fatal("expected feature \"foo\" to be present in registry")
	}

	if f.Name != "Foo Feature" {
		t.Errorf("Name = %q, want %q", f.Name, "Foo Feature")
	}
	if f.Description != "A test feature" {
		t.Errorf("Description = %q, want %q", f.Description, "A test feature")
	}
	if !slices.Equal(f.Requires, []string{"bar"}) {
		t.Errorf("Requires = %v, want [bar]", f.Requires)
	}
	if !slices.Equal(f.Incompatible, []string{"baz"}) {
		t.Errorf("Incompatible = %v, want [baz]", f.Incompatible)
	}
}

func TestGetFeature_Lookup(t *testing.T) {
	r := buildTestRegistry([]*Feature{
		{ID: "alpha", Name: "Alpha"},
		{ID: "beta", Name: "Beta"},
	})

	tests := []struct {
		id   string
		want string // empty means expect nil
	}{
		{id: "alpha", want: "Alpha"},
		{id: "beta", want: "Beta"},
		{id: "gamma", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			f := r.GetFeature(tt.id)
			if tt.want == "" {
				if f != nil {
					t.Errorf("GetFeature(%q) = %+v, want nil", tt.id, f)
				}
				return
			}
			if f == nil {
				t.Fatalf("GetFeature(%q) = nil, want feature with Name %q", tt.id, tt.want)
			}
			if f.Name != tt.want {
				t.Errorf("GetFeature(%q).Name = %q, want %q", tt.id, f.Name, tt.want)
			}
		})
	}
}
