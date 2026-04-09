package features

import (
	"fmt"
	"path/filepath"
)

// Registry holds all parsed features and precomputed dependency and
// incompatibility graphs used for install planning.
type Registry struct {
	Features      []*Feature
	DependencyMap map[string][]string // feature ID → IDs it requires
	IncompatMap   map[string][]string // feature ID → IDs it's incompatible with (bidirectional)
}

// BuildRegistry walks <repoDir>/features/*/manifest.yaml, parses each
// manifest, and builds the dependency and incompatibility maps.
func BuildRegistry(repoDir string) (*Registry, error) {
	pattern := filepath.Join(repoDir, "features", "*", "manifest.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("globbing feature manifests: %w", err)
	}

	r := &Registry{
		DependencyMap: make(map[string][]string),
		IncompatMap:   make(map[string][]string),
	}

	for _, path := range matches {
		f, err := ParseManifest(path)
		if err != nil {
			return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
		}
		r.Features = append(r.Features, f)
	}

	// Build dependency map.
	for _, f := range r.Features {
		if len(f.Requires) > 0 {
			r.DependencyMap[f.ID] = f.Requires
		}
	}

	// Build bidirectional incompatibility map.
	for _, f := range r.Features {
		for _, inc := range f.Incompatible {
			r.IncompatMap[f.ID] = appendUnique(r.IncompatMap[f.ID], inc)
			r.IncompatMap[inc] = appendUnique(r.IncompatMap[inc], f.ID)
		}
	}

	return r, nil
}

// TopologicalSort returns feature IDs in install order (dependencies first)
// using Kahn's algorithm. It returns an error if the dependency graph
// contains a cycle.
func (r *Registry) TopologicalSort() ([]string, error) {
	// Build in-degree counts and adjacency list from DependencyMap.
	// An edge from A to B means "A requires B", so B must come before A.
	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // dependency → features that depend on it

	for _, f := range r.Features {
		inDegree[f.ID] = 0 // ensure every feature appears
	}

	for id, deps := range r.DependencyMap {
		inDegree[id] += len(deps)
		for _, dep := range deps {
			dependents[dep] = append(dependents[dep], id)
		}
	}

	// Seed the queue with features that have no dependencies.
	var queue []string
	for _, f := range r.Features {
		if inDegree[f.ID] == 0 {
			queue = append(queue, f.ID)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		sorted = append(sorted, id)

		for _, dep := range dependents[id] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(sorted) != len(r.Features) {
		return nil, fmt.Errorf("dependency cycle detected: sorted %d of %d features", len(sorted), len(r.Features))
	}

	return sorted, nil
}

// GetFeature returns the feature with the given ID, or nil if not found.
func (r *Registry) GetFeature(id string) *Feature {
	for _, f := range r.Features {
		if f.ID == id {
			return f
		}
	}
	return nil
}

// GetIncompatible returns the IDs of features that are incompatible with the
// given feature ID.
func (r *Registry) GetIncompatible(id string) []string {
	return r.IncompatMap[id]
}

// appendUnique appends val to slice only if it is not already present.
func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
