package features

import (
	"fmt"
	"strings"

	"foundry/backend/transformer"
)

// ResolveMappings applies the given mappings to diffContent by performing
// line-level string replacements and resolving any {{key:transformer}} tokens
// in the replacement strings using the provided config values.
func ResolveMappings(diffContent string, mappings []Mapping, configValues map[string]string) (string, error) {
	lines := strings.Split(diffContent, "\n")

	for _, m := range mappings {
		for _, t := range m.Targets {
			idx := t.Line - 1 // convert 1-indexed to 0-indexed
			if idx < 0 || idx >= len(lines) {
				continue
			}

			lines[idx] = strings.Replace(lines[idx], t.From, t.To, 1)

			resolved, err := transformer.ResolveAll(lines[idx], configValues)
			if err != nil {
				return "", fmt.Errorf("resolving tokens on line %d: %w", t.Line, err)
			}
			lines[idx] = resolved
		}
	}

	return strings.Join(lines, "\n"), nil
}
