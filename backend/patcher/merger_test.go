package patcher

import (
	"strings"
	"testing"
)

// --- Happy path tests ---

func TestMerge_SingleHunk(t *testing.T) {
	hunks := []MatchedHunk{
		{
			Hunk:        Hunk{FeatureID: "teams", Lines: []Line{{Op: OpContext, Content: "a"}, {Op: OpAdd, Content: "b"}}},
			MatchResult: MatchResult{StartLine: 5, EndLine: 6},
		},
	}

	result, conflicts := Merge(hunks)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %d", len(conflicts))
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(result))
	}
}

func TestMerge_NonOverlappingHunks(t *testing.T) {
	hunks := []MatchedHunk{
		{
			Hunk:        Hunk{FeatureID: "teams", Lines: []Line{{Op: OpContext, Content: "a"}, {Op: OpAdd, Content: "b"}}},
			MatchResult: MatchResult{StartLine: 0, EndLine: 1},
		},
		{
			Hunk:        Hunk{FeatureID: "billing", Lines: []Line{{Op: OpContext, Content: "c"}, {Op: OpAdd, Content: "d"}}},
			MatchResult: MatchResult{StartLine: 10, EndLine: 11},
		},
	}

	result, conflicts := Merge(hunks)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %d", len(conflicts))
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(result))
	}
}

func TestMerge_PureAdditionsSameAnchor(t *testing.T) {
	// Two features adding content at the same location — no conflict.
	hunks := []MatchedHunk{
		{
			Hunk: Hunk{
				FeatureID: "teams",
				Lines:     []Line{{Op: OpContext, Content: "<nav>"}, {Op: OpAdd, Content: "  <a>Teams</a>"}, {Op: OpContext, Content: "</nav>"}},
			},
			MatchResult: MatchResult{StartLine: 5, EndLine: 7},
		},
		{
			Hunk: Hunk{
				FeatureID: "billing",
				Lines:     []Line{{Op: OpContext, Content: "<nav>"}, {Op: OpAdd, Content: "  <a>Billing</a>"}, {Op: OpContext, Content: "</nav>"}},
			},
			MatchResult: MatchResult{StartLine: 5, EndLine: 7},
		},
	}

	result, conflicts := Merge(hunks)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts for pure additions, got %d: %v", len(conflicts), conflicts)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(result))
	}
	// Order preserved: teams first (topo order), then billing.
	if result[0].Hunk.FeatureID != "teams" {
		t.Errorf("expected teams first, got %s", result[0].Hunk.FeatureID)
	}
	if result[1].Hunk.FeatureID != "billing" {
		t.Errorf("expected billing second, got %s", result[1].Hunk.FeatureID)
	}
}

func TestMerge_SortsByStartLine(t *testing.T) {
	hunks := []MatchedHunk{
		{
			Hunk:        Hunk{FeatureID: "b", Lines: []Line{{Op: OpContext, Content: "x"}, {Op: OpAdd, Content: "y"}}},
			MatchResult: MatchResult{StartLine: 20, EndLine: 21},
		},
		{
			Hunk:        Hunk{FeatureID: "a", Lines: []Line{{Op: OpContext, Content: "m"}, {Op: OpAdd, Content: "n"}}},
			MatchResult: MatchResult{StartLine: 5, EndLine: 6},
		},
	}

	result, _ := Merge(hunks)
	if result[0].Hunk.FeatureID != "a" {
		t.Errorf("expected 'a' first after sort, got %s", result[0].Hunk.FeatureID)
	}
	if result[1].Hunk.FeatureID != "b" {
		t.Errorf("expected 'b' second after sort, got %s", result[1].Hunk.FeatureID)
	}
}

func TestMerge_EmptyInput(t *testing.T) {
	result, conflicts := Merge(nil)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d", len(result))
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(conflicts))
	}
}

func TestMerge_ThreeNonOverlapping(t *testing.T) {
	hunks := []MatchedHunk{
		{Hunk: Hunk{FeatureID: "a", Lines: []Line{{Op: OpContext, Content: "x"}}}, MatchResult: MatchResult{StartLine: 0, EndLine: 1}},
		{Hunk: Hunk{FeatureID: "b", Lines: []Line{{Op: OpContext, Content: "y"}}}, MatchResult: MatchResult{StartLine: 10, EndLine: 11}},
		{Hunk: Hunk{FeatureID: "c", Lines: []Line{{Op: OpContext, Content: "z"}}}, MatchResult: MatchResult{StartLine: 20, EndLine: 21}},
	}

	_, conflicts := Merge(hunks)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %d", len(conflicts))
	}
}

// --- Sad path tests ---

func TestMerge_OverlappingWithDeletions(t *testing.T) {
	hunks := []MatchedHunk{
		{
			Hunk: Hunk{
				FeatureID: "teams",
				Lines:     []Line{{Op: OpContext, Content: "a"}, {Op: OpDelete, Content: "old"}, {Op: OpAdd, Content: "new1"}},
			},
			MatchResult: MatchResult{StartLine: 5, EndLine: 7},
		},
		{
			Hunk: Hunk{
				FeatureID: "billing",
				Lines:     []Line{{Op: OpContext, Content: "a"}, {Op: OpDelete, Content: "old"}, {Op: OpAdd, Content: "new2"}},
			},
			MatchResult: MatchResult{StartLine: 5, EndLine: 7},
		},
	}

	_, conflicts := Merge(hunks)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].FeatureA != "teams" || conflicts[0].FeatureB != "billing" {
		t.Errorf("wrong features in conflict: %v", conflicts[0])
	}
	if !strings.Contains(conflicts[0].Reason, "overlapping") {
		t.Errorf("expected 'overlapping' in reason, got: %s", conflicts[0].Reason)
	}
}

func TestMerge_OneSideDeletion(t *testing.T) {
	// One hunk deletes, the other is pure addition — still a conflict
	// because the deletion modifies lines the other hunk's context depends on.
	hunks := []MatchedHunk{
		{
			Hunk: Hunk{
				FeatureID: "teams",
				Lines:     []Line{{Op: OpContext, Content: "a"}, {Op: OpDelete, Content: "b"}, {Op: OpAdd, Content: "c"}},
			},
			MatchResult: MatchResult{StartLine: 0, EndLine: 2},
		},
		{
			Hunk: Hunk{
				FeatureID: "billing",
				Lines:     []Line{{Op: OpContext, Content: "a"}, {Op: OpAdd, Content: "d"}, {Op: OpContext, Content: "b"}},
			},
			MatchResult: MatchResult{StartLine: 0, EndLine: 2},
		},
	}

	_, conflicts := Merge(hunks)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
}

func TestMerge_MultipleConflicts(t *testing.T) {
	hunks := []MatchedHunk{
		{
			Hunk:        Hunk{FeatureID: "a", Lines: []Line{{Op: OpDelete, Content: "x"}}},
			MatchResult: MatchResult{StartLine: 0, EndLine: 1},
		},
		{
			Hunk:        Hunk{FeatureID: "b", Lines: []Line{{Op: OpDelete, Content: "x"}}},
			MatchResult: MatchResult{StartLine: 0, EndLine: 1},
		},
		{
			Hunk:        Hunk{FeatureID: "c", Lines: []Line{{Op: OpDelete, Content: "y"}}},
			MatchResult: MatchResult{StartLine: 0, EndLine: 1},
		},
	}

	_, conflicts := Merge(hunks)
	// a vs b, a vs c, b vs c = 3 conflicts
	if len(conflicts) != 3 {
		t.Fatalf("expected 3 conflicts, got %d", len(conflicts))
	}
}

func TestMerge_PartialOverlapWithDeletion(t *testing.T) {
	// Hunk A spans lines 5-8, hunk B spans 7-10, both have deletions.
	hunks := []MatchedHunk{
		{
			Hunk:        Hunk{FeatureID: "a", Lines: []Line{{Op: OpContext, Content: "x"}, {Op: OpDelete, Content: "y"}, {Op: OpContext, Content: "z"}}},
			MatchResult: MatchResult{StartLine: 5, EndLine: 8},
		},
		{
			Hunk:        Hunk{FeatureID: "b", Lines: []Line{{Op: OpContext, Content: "z"}, {Op: OpDelete, Content: "w"}, {Op: OpContext, Content: "q"}}},
			MatchResult: MatchResult{StartLine: 7, EndLine: 10},
		},
	}

	_, conflicts := Merge(hunks)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict for partial overlap, got %d", len(conflicts))
	}
}

func TestMerge_AdjacentRangesNoConflict(t *testing.T) {
	// Hunk A ends at 5, hunk B starts at 5 — no overlap.
	hunks := []MatchedHunk{
		{
			Hunk:        Hunk{FeatureID: "a", Lines: []Line{{Op: OpDelete, Content: "x"}}},
			MatchResult: MatchResult{StartLine: 3, EndLine: 5},
		},
		{
			Hunk:        Hunk{FeatureID: "b", Lines: []Line{{Op: OpDelete, Content: "y"}}},
			MatchResult: MatchResult{StartLine: 5, EndLine: 7},
		},
	}

	_, conflicts := Merge(hunks)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts for adjacent ranges, got %d", len(conflicts))
	}
}
