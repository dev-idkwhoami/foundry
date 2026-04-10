package cli

import (
	"flag"
	"fmt"
	"os"

	"foundry/backend/features"
	"foundry/backend/patcher"
)

// runValidate builds the feature registry and tests all valid feature
// combinations for patch conflicts using the merge engine.
//
// Usage: foundry-cli validate [--verbose]
func runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	verbose := fs.Bool("verbose", false, "Show detailed output")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	fmt.Println("Building feature registry...")

	registry, err := features.BuildRegistry(cwd)
	if err != nil {
		return fmt.Errorf("building registry: %w", err)
	}

	fmt.Printf("Found %d feature(s)\n", len(registry.Features))

	// Get topo order.
	sorted, err := registry.TopologicalSort()
	if err != nil {
		return fmt.Errorf("topological sort: %w", err)
	}

	if *verbose {
		fmt.Printf("Install order: %v\n", sorted)
	}

	// Collect all cdiff patches grouped by feature.
	type featureDiff struct {
		id   string
		diff patcher.Diff
	}
	var featureDiffs []featureDiff

	for _, fid := range sorted {
		f := registry.GetFeature(fid)
		if f == nil {
			continue
		}
		for _, p := range f.Patches {
			if p.Format != "cdiff" || p.Mode == "manual" {
				continue
			}
			diffPath := fmt.Sprintf("%s/features/%s/%s", cwd, fid, p.File)
			data, err := os.ReadFile(diffPath)
			if err != nil {
				return fmt.Errorf("reading %s/%s: %w", fid, p.File, err)
			}
			diff, err := patcher.Parse(string(data))
			if err != nil {
				return fmt.Errorf("parsing %s/%s: %w", fid, p.File, err)
			}
			for i := range diff.Files {
				for j := range diff.Files[i].Hunks {
					diff.Files[i].Hunks[j].FeatureID = fid
				}
			}
			featureDiffs = append(featureDiffs, featureDiff{id: fid, diff: *diff})
		}
	}

	if len(featureDiffs) == 0 {
		fmt.Println("No cdiff patches found — nothing to validate.")
		return nil
	}

	fmt.Printf("Checking %d patch(es) for conflicts...\n", len(featureDiffs))

	// Test all features together (the maximal set, excluding incompatibles).
	var allDiffs []patcher.Diff
	for _, fd := range featureDiffs {
		allDiffs = append(allDiffs, fd.diff)
	}

	conflicts, err := patcher.Check(patcher.ApplyRequest{
		ProjectDir: cwd,
		Diffs:      allDiffs,
	})
	if err != nil {
		return fmt.Errorf("checking patches: %w", err)
	}

	if len(conflicts) == 0 {
		fmt.Println("")
		fmt.Println("All patches are compatible.")
		return nil
	}

	fmt.Printf("\nFound %d conflict(s):\n", len(conflicts))
	for _, c := range conflicts {
		fmt.Printf("  %s: %s vs %s — %s\n", c.File, c.FeatureA, c.FeatureB, c.Reason)
	}
	os.Exit(1)
	return nil
}
