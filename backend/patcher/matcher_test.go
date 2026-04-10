package patcher

import (
	"strings"
	"testing"
)

// --- Happy path tests ---

func TestMatch_SimpleContext(t *testing.T) {
	fileLines := []string{"<html>", "<nav>", "</nav>", "</html>"}
	hunk := &Hunk{
		Lines: []Line{
			{Op: OpContext, Content: "<nav>"},
			{Op: OpAdd, Content: "    <a href=\"/teams\">Teams</a>"},
			{Op: OpContext, Content: "</nav>"},
		},
	}

	result, err := Match(hunk, fileLines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StartLine != 1 {
		t.Errorf("StartLine: expected 1, got %d", result.StartLine)
	}
	if result.EndLine != 3 {
		t.Errorf("EndLine: expected 3, got %d", result.EndLine)
	}
}

func TestMatch_ContextAtFileStart(t *testing.T) {
	fileLines := []string{"<?php", "namespace App;", "", "class User {"}
	hunk := &Hunk{
		Lines: []Line{
			{Op: OpContext, Content: "<?php"},
			{Op: OpContext, Content: "namespace App;"},
			{Op: OpAdd, Content: "use App\\Traits\\HasTeams;"},
		},
	}

	result, err := Match(hunk, fileLines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StartLine != 0 {
		t.Errorf("StartLine: expected 0, got %d", result.StartLine)
	}
	if result.EndLine != 2 {
		t.Errorf("EndLine: expected 2, got %d", result.EndLine)
	}
}

func TestMatch_ContextAtFileEnd(t *testing.T) {
	fileLines := []string{"line1", "line2", "line3"}
	hunk := &Hunk{
		Lines: []Line{
			{Op: OpContext, Content: "line2"},
			{Op: OpContext, Content: "line3"},
			{Op: OpAdd, Content: "line4"},
		},
	}

	result, err := Match(hunk, fileLines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StartLine != 1 {
		t.Errorf("StartLine: expected 1, got %d", result.StartLine)
	}
}

func TestMatch_WithDeletionLines(t *testing.T) {
	fileLines := []string{"a", "b", "old", "c"}
	hunk := &Hunk{
		Lines: []Line{
			{Op: OpContext, Content: "b"},
			{Op: OpDelete, Content: "old"},
			{Op: OpAdd, Content: "new"},
			{Op: OpContext, Content: "c"},
		},
	}

	result, err := Match(hunk, fileLines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Anchors are: "b", "old", "c" at indices 1,2,3
	if result.StartLine != 1 {
		t.Errorf("StartLine: expected 1, got %d", result.StartLine)
	}
	if result.EndLine != 4 {
		t.Errorf("EndLine: expected 4, got %d", result.EndLine)
	}
}

func TestMatch_DuplicateContextPicksFirst(t *testing.T) {
	fileLines := []string{"x", "marker", "y", "marker", "z"}
	hunk := &Hunk{
		Lines: []Line{
			{Op: OpContext, Content: "marker"},
			{Op: OpContext, Content: "y"},
		},
	}

	result, err := Match(hunk, fileLines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should match the first occurrence: index 1.
	if result.StartLine != 1 {
		t.Errorf("StartLine: expected 1, got %d", result.StartLine)
	}
}

func TestMatch_SingleContextLine(t *testing.T) {
	fileLines := []string{"a", "b", "c"}
	hunk := &Hunk{
		Lines: []Line{
			{Op: OpContext, Content: "b"},
			{Op: OpAdd, Content: "new"},
		},
	}

	result, err := Match(hunk, fileLines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StartLine != 1 || result.EndLine != 2 {
		t.Errorf("expected [1,2), got [%d,%d)", result.StartLine, result.EndLine)
	}
}

func TestMatch_LargeFile(t *testing.T) {
	// Context buried deep in a large file.
	var fileLines []string
	for i := 0; i < 1000; i++ {
		fileLines = append(fileLines, "filler")
	}
	fileLines = append(fileLines, "ANCHOR_START", "ANCHOR_END")
	for i := 0; i < 1000; i++ {
		fileLines = append(fileLines, "filler")
	}

	hunk := &Hunk{
		Lines: []Line{
			{Op: OpContext, Content: "ANCHOR_START"},
			{Op: OpAdd, Content: "inserted"},
			{Op: OpContext, Content: "ANCHOR_END"},
		},
	}

	result, err := Match(hunk, fileLines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StartLine != 1000 {
		t.Errorf("StartLine: expected 1000, got %d", result.StartLine)
	}
}

// --- Sad path tests ---

func TestMatch_NoContextOrDeleteLines(t *testing.T) {
	fileLines := []string{"a", "b", "c"}
	hunk := &Hunk{
		Lines: []Line{
			{Op: OpAdd, Content: "new line"},
		},
	}

	_, err := Match(hunk, fileLines)
	if err == nil {
		t.Fatal("expected error for pure addition hunk")
	}
	if !strings.Contains(err.Error(), "no context or deletion lines") {
		t.Errorf("expected 'no context' error, got: %v", err)
	}
}

func TestMatch_ContextNotFound(t *testing.T) {
	fileLines := []string{"a", "b", "c"}
	hunk := &Hunk{
		Lines: []Line{
			{Op: OpContext, Content: "x"},
			{Op: OpContext, Content: "y"},
		},
	}

	_, err := Match(hunk, fileLines)
	if err == nil {
		t.Fatal("expected error for non-matching context")
	}
	if !strings.Contains(err.Error(), "no match found") {
		t.Errorf("expected 'no match found' error, got: %v", err)
	}
}

func TestMatch_PartialMatch(t *testing.T) {
	fileLines := []string{"a", "b", "WRONG", "d"}
	hunk := &Hunk{
		Lines: []Line{
			{Op: OpContext, Content: "a"},
			{Op: OpContext, Content: "b"},
			{Op: OpContext, Content: "c"},
			{Op: OpContext, Content: "d"},
		},
	}

	_, err := Match(hunk, fileLines)
	if err == nil {
		t.Fatal("expected error for partial match")
	}
	if !strings.Contains(err.Error(), "partial match") {
		t.Errorf("expected 'partial match' diagnostic, got: %v", err)
	}
	// Should report 2/4 matched at line 1.
	if !strings.Contains(err.Error(), "2/4") {
		t.Errorf("expected '2/4 lines matched', got: %v", err)
	}
}

func TestMatch_EmptyFile(t *testing.T) {
	hunk := &Hunk{
		Lines: []Line{
			{Op: OpContext, Content: "something"},
			{Op: OpAdd, Content: "new"},
		},
	}

	_, err := Match(hunk, []string{})
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestMatch_ContextLongerThanFile(t *testing.T) {
	fileLines := []string{"a"}
	hunk := &Hunk{
		Lines: []Line{
			{Op: OpContext, Content: "a"},
			{Op: OpContext, Content: "b"},
			{Op: OpContext, Content: "c"},
		},
	}

	_, err := Match(hunk, fileLines)
	if err == nil {
		t.Fatal("expected error when context is longer than file")
	}
}

func TestMatch_SimilarButNotExactMatch(t *testing.T) {
	fileLines := []string{"  <nav>", "  </nav>"}
	hunk := &Hunk{
		Lines: []Line{
			{Op: OpContext, Content: "<nav>"},   // no leading spaces
			{Op: OpContext, Content: "</nav>"},   // no leading spaces
		},
	}

	_, err := Match(hunk, fileLines)
	if err == nil {
		t.Fatal("expected error for whitespace mismatch")
	}
}

func TestMatch_EmptyHunk(t *testing.T) {
	fileLines := []string{"a", "b"}
	hunk := &Hunk{Lines: []Line{}}

	_, err := Match(hunk, fileLines)
	if err == nil {
		t.Fatal("expected error for empty hunk")
	}
}
