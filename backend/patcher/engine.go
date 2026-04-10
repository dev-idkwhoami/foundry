package patcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Apply parses all diffs, matches hunks against target files, merges them,
// detects conflicts, and writes the modified files. Returns the list of
// modified file paths and any conflicts found.
func Apply(req ApplyRequest) (*ApplyResult, error) {
	grouped, err := groupByFile(req)
	if err != nil {
		return nil, err
	}

	result := &ApplyResult{}

	for filePath, hunks := range grouped {
		absPath := filepath.Join(req.ProjectDir, filePath)
		content, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", filePath, err)
		}
		fileLines := strings.Split(string(content), "\n")

		matched, err := matchAll(hunks, fileLines)
		if err != nil {
			return nil, fmt.Errorf("matching hunks in %s: %w", filePath, err)
		}

		merged, conflicts := Merge(matched)
		for i := range conflicts {
			conflicts[i].File = filePath
		}
		result.Conflicts = append(result.Conflicts, conflicts...)

		if len(conflicts) > 0 {
			continue // don't write files with conflicts
		}

		newLines := applyHunks(fileLines, merged)
		if err := os.WriteFile(absPath, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", filePath, err)
		}

		result.Modified = append(result.Modified, filePath)
	}

	return result, nil
}

// Check runs the same logic as Apply but stops before writing any files.
// Returns only the conflicts found.
func Check(req ApplyRequest) ([]Conflict, error) {
	grouped, err := groupByFile(req)
	if err != nil {
		return nil, err
	}

	var allConflicts []Conflict

	for filePath, hunks := range grouped {
		absPath := filepath.Join(req.ProjectDir, filePath)
		content, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", filePath, err)
		}
		fileLines := strings.Split(string(content), "\n")

		matched, err := matchAll(hunks, fileLines)
		if err != nil {
			return nil, fmt.Errorf("matching hunks in %s: %w", filePath, err)
		}

		_, conflicts := Merge(matched)
		for i := range conflicts {
			conflicts[i].File = filePath
		}
		allConflicts = append(allConflicts, conflicts...)
	}

	return allConflicts, nil
}

// groupByFile collects all hunks from all diffs, grouped by target file path.
func groupByFile(req ApplyRequest) (map[string][]Hunk, error) {
	grouped := make(map[string][]Hunk)
	for _, diff := range req.Diffs {
		for _, fd := range diff.Files {
			for _, h := range fd.Hunks {
				if len(h.Lines) == 0 {
					continue
				}
				grouped[fd.Path] = append(grouped[fd.Path], h)
			}
		}
	}
	if len(grouped) == 0 {
		return nil, fmt.Errorf("no hunks to apply")
	}
	return grouped, nil
}

// matchAll matches each hunk against the file lines.
func matchAll(hunks []Hunk, fileLines []string) ([]MatchedHunk, error) {
	var matched []MatchedHunk
	for _, h := range hunks {
		mr, err := Match(&h, fileLines)
		if err != nil {
			return nil, fmt.Errorf("feature %q: %w", h.FeatureID, err)
		}
		matched = append(matched, MatchedHunk{Hunk: h, MatchResult: *mr})
	}
	return matched, nil
}

// applyHunks applies matched hunks to file lines in bottom-up order
// (highest StartLine first) so that line shifts from earlier hunks
// don't affect later ones. Hunks sharing the same anchor range are
// combined (their additions are stacked) before being applied.
func applyHunks(fileLines []string, hunks []MatchedHunk) []string {
	combined := combineOverlapping(hunks)

	// Sort descending by StartLine for bottom-up application.
	sort.SliceStable(combined, func(i, j int) bool {
		return combined[i].MatchResult.StartLine > combined[j].MatchResult.StartLine
	})

	result := make([]string, len(fileLines))
	copy(result, fileLines)

	for _, mh := range combined {
		result = applySingleHunk(result, mh)
	}

	return result
}

// combineOverlapping groups hunks by identical anchor range and merges
// their additions. Hunks with different ranges pass through unchanged.
func combineOverlapping(hunks []MatchedHunk) []MatchedHunk {
	type key struct{ start, end int }
	groups := make(map[key][]MatchedHunk)
	var order []key

	for _, mh := range hunks {
		k := key{mh.MatchResult.StartLine, mh.MatchResult.EndLine}
		if _, exists := groups[k]; !exists {
			order = append(order, k)
		}
		groups[k] = append(groups[k], mh)
	}

	var result []MatchedHunk
	for _, k := range order {
		group := groups[k]
		if len(group) == 1 {
			result = append(result, group[0])
			continue
		}
		result = append(result, mergeHunkGroup(group))
	}
	return result
}

// mergeHunkGroup combines multiple hunks sharing the same anchor into one.
// Context/deletion lines come from the first hunk as the skeleton; addition
// lines from all hunks are collected and inserted at the correct positions.
func mergeHunkGroup(group []MatchedHunk) MatchedHunk {
	base := group[0]

	// For each hunk, map context-line index → additions that follow it.
	// Index -1 means additions before any context line.
	type addBlock struct{ lines []Line }
	additions := make(map[int][]addBlock)

	for _, mh := range group {
		ctxIdx := -1
		var block []Line
		for _, l := range mh.Hunk.Lines {
			if l.Op == OpContext || l.Op == OpDelete {
				if len(block) > 0 {
					additions[ctxIdx] = append(additions[ctxIdx], addBlock{block})
					block = nil
				}
				ctxIdx++
			} else {
				block = append(block, l)
			}
		}
		if len(block) > 0 {
			additions[ctxIdx] = append(additions[ctxIdx], addBlock{block})
		}
	}

	// Rebuild: walk the base skeleton, emitting additions after each context line.
	var merged []Line

	// Additions before any context line (index -1).
	for _, ab := range additions[-1] {
		merged = append(merged, ab.lines...)
	}

	ctxIdx := 0
	for _, l := range base.Hunk.Lines {
		if l.Op == OpContext || l.Op == OpDelete {
			merged = append(merged, l)
			for _, ab := range additions[ctxIdx] {
				merged = append(merged, ab.lines...)
			}
			ctxIdx++
		}
		// Skip addition lines from the base — they're already in the additions map.
	}

	return MatchedHunk{
		Hunk: Hunk{
			Lines:     merged,
			FeatureID: base.Hunk.FeatureID,
		},
		MatchResult: base.MatchResult,
	}
}

// applySingleHunk replaces the matched region with the hunk's intended output.
func applySingleHunk(lines []string, mh MatchedHunk) []string {
	start := mh.MatchResult.StartLine
	end := mh.MatchResult.EndLine

	// Build the replacement: context lines stay, deletions are removed,
	// additions are inserted.
	var replacement []string
	for _, l := range mh.Hunk.Lines {
		switch l.Op {
		case OpContext:
			replacement = append(replacement, l.Content)
		case OpAdd:
			replacement = append(replacement, l.Content)
		case OpDelete:
			// Skip — line is removed.
		}
	}

	// Splice: lines[:start] + replacement + lines[end:]
	var newLines []string
	newLines = append(newLines, lines[:start]...)
	newLines = append(newLines, replacement...)
	newLines = append(newLines, lines[end:]...)

	return newLines
}
