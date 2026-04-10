package patcher

import "fmt"

// Match finds where a hunk's context and deletion lines appear in fileLines
// using a sliding window search. Returns the position in the target file.
func Match(hunk *Hunk, fileLines []string) (*MatchResult, error) {
	// Extract the anchor lines: context (' ') and deletion ('-') lines
	// in order. These are the lines that must exist in the target file.
	var anchors []string
	for _, l := range hunk.Lines {
		if l.Op == OpContext || l.Op == OpDelete {
			anchors = append(anchors, l.Content)
		}
	}

	if len(anchors) == 0 {
		// Pure addition with no context — cannot anchor.
		return nil, fmt.Errorf("hunk has no context or deletion lines to match against")
	}

	// Sliding window: find anchors as a contiguous sequence in fileLines.
	bestPartial := -1
	bestPartialCount := 0

	for i := 0; i <= len(fileLines)-len(anchors); i++ {
		matched := 0
		for j, a := range anchors {
			if fileLines[i+j] == a {
				matched++
			} else {
				break
			}
		}

		if matched == len(anchors) {
			return &MatchResult{
				StartLine: i,
				EndLine:   i + len(anchors),
			}, nil
		}

		if matched > bestPartialCount {
			bestPartialCount = matched
			bestPartial = i
		}
	}

	if bestPartial >= 0 {
		return nil, fmt.Errorf(
			"no exact match found; closest partial match at line %d (%d/%d lines matched)",
			bestPartial+1, bestPartialCount, len(anchors),
		)
	}

	return nil, fmt.Errorf("no match found for %d anchor lines", len(anchors))
}
