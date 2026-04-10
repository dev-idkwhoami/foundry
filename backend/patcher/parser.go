package patcher

import (
	"fmt"
	"strings"
)

// Parse reads a .cdiff string and returns a structured Diff.
//
// Expected format:
//
//	--- a/path/to/file
//	+++ b/path/to/file
//	@@
//	 context line
//	+added line
//	-removed line
func Parse(content string) (*Diff, error) {
	lines := strings.Split(content, "\n")
	diff := &Diff{}

	var current *FileDiff
	var hunk *Hunk
	inHunk := false

	for i, line := range lines {
		lineNum := i + 1

		// File header: --- a/path
		if strings.HasPrefix(line, "--- a/") {
			// Flush previous hunk/file.
			if hunk != nil && current != nil {
				current.Hunks = append(current.Hunks, *hunk)
				hunk = nil
			}
			if current != nil {
				diff.Files = append(diff.Files, *current)
			}
			current = nil
			inHunk = false
			continue
		}

		// File header: +++ b/path
		if strings.HasPrefix(line, "+++ b/") {
			path := strings.TrimPrefix(line, "+++ b/")
			current = &FileDiff{Path: path}
			continue
		}

		// Hunk separator.
		if line == "@@" || strings.HasPrefix(line, "@@ ") {
			if current == nil {
				return nil, fmt.Errorf("line %d: @@ without file header", lineNum)
			}
			// Flush previous hunk.
			if hunk != nil {
				current.Hunks = append(current.Hunks, *hunk)
			}
			hunk = &Hunk{}
			inHunk = true
			continue
		}

		if !inHunk {
			// Outside a hunk — skip blank lines and unknown content.
			continue
		}

		// Inside a hunk: parse prefixed lines.
		if len(line) == 0 {
			// Empty line inside a hunk is treated as a context line with empty content.
			hunk.Lines = append(hunk.Lines, Line{Op: OpContext, Content: ""})
			continue
		}

		switch line[0] {
		case ' ':
			hunk.Lines = append(hunk.Lines, Line{Op: OpContext, Content: line[1:]})
		case '+':
			hunk.Lines = append(hunk.Lines, Line{Op: OpAdd, Content: line[1:]})
		case '-':
			hunk.Lines = append(hunk.Lines, Line{Op: OpDelete, Content: line[1:]})
		default:
			return nil, fmt.Errorf("line %d: unexpected prefix %q in hunk", lineNum, string(line[0]))
		}
	}

	// Flush remaining hunk and file.
	if hunk != nil && current != nil {
		current.Hunks = append(current.Hunks, *hunk)
	}
	if current != nil {
		diff.Files = append(diff.Files, *current)
	}

	if len(diff.Files) == 0 {
		return nil, fmt.Errorf("no file diffs found in input")
	}

	return diff, nil
}
