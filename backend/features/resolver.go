package features

import (
	"fmt"
	"strings"

	"foundry/backend/transformer"
)

// ResolveMappings applies the given mappings to diffContent by performing
// string replacements and resolving any {{key:transformer}} tokens using the
// provided config values.
//
// Target modes:
//   - Line set: replace on that single line (1-indexed)
//   - Lines set: replace on each listed line (1-indexed)
//   - Neither set: replace on every line in the diff (global mode)
func ResolveMappings(diffContent string, mappings []Mapping, configValues map[string]string) (string, error) {
	lines := strings.Split(diffContent, "\n")

	for _, m := range mappings {
		for _, t := range m.Targets {
			indices := targetIndices(t, len(lines))

			for _, idx := range indices {
				lines[idx] = strings.Replace(lines[idx], t.From, t.To, 1)

				resolved, err := transformer.ResolveAll(lines[idx], configValues)
				if err != nil {
					return "", fmt.Errorf("resolving tokens on line %d: %w", idx+1, err)
				}
				lines[idx] = resolved
			}
		}
	}

	return strings.Join(lines, "\n"), nil
}

// targetIndices returns the 0-indexed line positions a target should apply to.
func targetIndices(t Target, lineCount int) []int {
	// Specific lines.
	if len(t.Lines) > 0 {
		var out []int
		for _, ln := range t.Lines {
			idx := ln - 1
			if idx >= 0 && idx < lineCount {
				out = append(out, idx)
			}
		}
		return out
	}

	// Global: all lines.
	out := make([]int, lineCount)
	for i := range out {
		out[i] = i
	}
	return out
}
