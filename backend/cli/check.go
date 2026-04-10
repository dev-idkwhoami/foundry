package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"foundry/backend/features"
	"foundry/backend/patcher"
)

// runCheck tests whether a specific feature is compatible with a given set
// of other features.
//
// Usage: foundry-cli check <feature-id> --with feat1,feat2,...
func runCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	withFlag := fs.String("with", "", "Comma-separated list of feature IDs to check against")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: foundry-cli check <feature-id> --with feat1,feat2")
	}

	targetID := fs.Arg(0)
	if *withFlag == "" {
		return fmt.Errorf("--with is required")
	}

	withIDs := strings.Split(*withFlag, ",")
	allIDs := append(withIDs, targetID)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	fmt.Printf("Checking %s against [%s]...\n", targetID, *withFlag)

	registry, err := features.BuildRegistry(cwd)
	if err != nil {
		return fmt.Errorf("building registry: %w", err)
	}

	// Collect cdiff patches for the specified features.
	var diffs []patcher.Diff
	for _, fid := range allIDs {
		f := registry.GetFeature(fid)
		if f == nil {
			return fmt.Errorf("feature %q not found in registry", fid)
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
			diffs = append(diffs, *diff)
		}
	}

	if len(diffs) == 0 {
		fmt.Println("No cdiff patches to check.")
		return nil
	}

	conflicts, err := patcher.Check(patcher.ApplyRequest{
		ProjectDir: cwd,
		Diffs:      diffs,
	})
	if err != nil {
		return fmt.Errorf("checking: %w", err)
	}

	if len(conflicts) == 0 {
		fmt.Println("")
		fmt.Printf("%s is compatible with [%s]\n", targetID, *withFlag)
		return nil
	}

	fmt.Printf("\nFound %d conflict(s):\n", len(conflicts))
	for _, c := range conflicts {
		fmt.Printf("  %s: %s vs %s — %s\n", c.File, c.FeatureA, c.FeatureB, c.Reason)
	}
	os.Exit(1)
	return nil
}
