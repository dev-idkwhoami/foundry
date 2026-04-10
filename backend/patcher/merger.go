package patcher

import (
	"fmt"
	"sort"
)

// Merge takes matched hunks for a single file, detects conflicts, and returns
// the final ordered list of hunks to apply plus any conflicts found.
//
// Rules:
//   - Hunks are sorted by StartLine.
//   - Two hunks conflict if their anchor ranges overlap AND both contain
//     deletion lines in the overlapping region.
//   - Pure-addition hunks sharing the same anchor do NOT conflict — they
//     stack in the order provided (which should be topological).
func Merge(hunks []MatchedHunk) ([]MatchedHunk, []Conflict) {
	if len(hunks) <= 1 {
		return hunks, nil
	}

	// Sort by start line, then by original slice order (stable) for topo ordering.
	sort.SliceStable(hunks, func(i, j int) bool {
		return hunks[i].MatchResult.StartLine < hunks[j].MatchResult.StartLine
	})

	var conflicts []Conflict

	for i := 0; i < len(hunks); i++ {
		for j := i + 1; j < len(hunks); j++ {
			a := hunks[i]
			b := hunks[j]

			// No overlap if b starts at or after a ends.
			if b.MatchResult.StartLine >= a.MatchResult.EndLine {
				break // sorted, so no further overlaps for a
			}

			// Ranges overlap — check if both have deletions.
			aHasDel := hasDeleteLines(a.Hunk)
			bHasDel := hasDeleteLines(b.Hunk)

			if aHasDel || bHasDel {
				conflicts = append(conflicts, Conflict{
					FeatureA: a.Hunk.FeatureID,
					FeatureB: b.Hunk.FeatureID,
					Reason: fmt.Sprintf(
						"overlapping ranges [%d-%d) and [%d-%d) with modifications",
						a.MatchResult.StartLine, a.MatchResult.EndLine,
						b.MatchResult.StartLine, b.MatchResult.EndLine,
					),
				})
			}
			// Pure additions at same anchor — no conflict, they stack.
		}
	}

	return hunks, conflicts
}

// hasDeleteLines returns true if the hunk contains any deletion lines.
func hasDeleteLines(h Hunk) bool {
	for _, l := range h.Lines {
		if l.Op == OpDelete {
			return true
		}
	}
	return false
}
