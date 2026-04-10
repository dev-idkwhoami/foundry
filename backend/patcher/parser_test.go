package patcher

import (
	"strings"
	"testing"
)

// --- Happy path tests ---

func TestParse_SingleFileSingleHunk(t *testing.T) {
	input := strings.Join([]string{
		"--- a/app.blade.php",
		"+++ b/app.blade.php",
		"@@",
		" <nav>",
		"+    <a href=\"/teams\">Teams</a>",
		" </nav>",
	}, "\n")

	diff, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}
	if diff.Files[0].Path != "app.blade.php" {
		t.Errorf("path: expected %q, got %q", "app.blade.php", diff.Files[0].Path)
	}
	if len(diff.Files[0].Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(diff.Files[0].Hunks))
	}

	hunk := diff.Files[0].Hunks[0]
	if len(hunk.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(hunk.Lines))
	}
	assertLine(t, hunk.Lines[0], OpContext, "<nav>")
	assertLine(t, hunk.Lines[1], OpAdd, "    <a href=\"/teams\">Teams</a>")
	assertLine(t, hunk.Lines[2], OpContext, "</nav>")
}

func TestParse_MultipleHunksInOneFile(t *testing.T) {
	input := strings.Join([]string{
		"--- a/routes/web.php",
		"+++ b/routes/web.php",
		"@@",
		" // Auth routes",
		"+Route::get('/login', [AuthController::class, 'login']);",
		"@@",
		" // API routes",
		"+Route::get('/api/users', [UserController::class, 'index']);",
	}, "\n")

	diff, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}
	if len(diff.Files[0].Hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(diff.Files[0].Hunks))
	}

	assertLine(t, diff.Files[0].Hunks[0].Lines[1], OpAdd, "Route::get('/login', [AuthController::class, 'login']);")
	assertLine(t, diff.Files[0].Hunks[1].Lines[1], OpAdd, "Route::get('/api/users', [UserController::class, 'index']);")
}

func TestParse_MultipleFiles(t *testing.T) {
	input := strings.Join([]string{
		"--- a/file1.php",
		"+++ b/file1.php",
		"@@",
		" line1",
		"+added1",
		"--- a/file2.php",
		"+++ b/file2.php",
		"@@",
		" line2",
		"-removed2",
	}, "\n")

	diff, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diff.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(diff.Files))
	}
	if diff.Files[0].Path != "file1.php" {
		t.Errorf("file 0 path: expected %q, got %q", "file1.php", diff.Files[0].Path)
	}
	if diff.Files[1].Path != "file2.php" {
		t.Errorf("file 1 path: expected %q, got %q", "file2.php", diff.Files[1].Path)
	}

	assertLine(t, diff.Files[0].Hunks[0].Lines[1], OpAdd, "added1")
	assertLine(t, diff.Files[1].Hunks[0].Lines[1], OpDelete, "removed2")
}

func TestParse_DeletionLines(t *testing.T) {
	input := strings.Join([]string{
		"--- a/config.php",
		"+++ b/config.php",
		"@@",
		" 'driver' => 'mysql',",
		"-'host' => 'localhost',",
		"+'host' => '127.0.0.1',",
	}, "\n")

	diff, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hunk := diff.Files[0].Hunks[0]
	if len(hunk.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(hunk.Lines))
	}
	assertLine(t, hunk.Lines[0], OpContext, "'driver' => 'mysql',")
	assertLine(t, hunk.Lines[1], OpDelete, "'host' => 'localhost',")
	assertLine(t, hunk.Lines[2], OpAdd, "'host' => '127.0.0.1',")
}

func TestParse_EmptyLinesInHunk(t *testing.T) {
	input := strings.Join([]string{
		"--- a/test.php",
		"+++ b/test.php",
		"@@",
		" first",
		"",
		" third",
	}, "\n")

	diff, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hunk := diff.Files[0].Hunks[0]
	if len(hunk.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(hunk.Lines))
	}
	// Empty line inside hunk treated as context with empty content.
	assertLine(t, hunk.Lines[1], OpContext, "")
}

func TestParse_LeadingBlankLinesIgnored(t *testing.T) {
	input := strings.Join([]string{
		"",
		"--- a/file.php",
		"+++ b/file.php",
		"@@",
		" ctx",
		"+add",
	}, "\n")

	diff, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}
}

func TestParse_TrailingNewline(t *testing.T) {
	input := strings.Join([]string{
		"--- a/file.php",
		"+++ b/file.php",
		"@@",
		" ctx",
		"+add",
		"",
	}, "\n")

	diff, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The trailing empty line is outside a hunk boundary (after last hunk line),
	// but since we're still "in hunk" it becomes a context line — that's fine.
	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}
}

func TestParse_NestedPathSlashes(t *testing.T) {
	input := strings.Join([]string{
		"--- a/resources/views/layouts/app.blade.php",
		"+++ b/resources/views/layouts/app.blade.php",
		"@@",
		" <head>",
		"+<link rel=\"stylesheet\" href=\"/css/teams.css\">",
		" </head>",
	}, "\n")

	diff, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff.Files[0].Path != "resources/views/layouts/app.blade.php" {
		t.Errorf("expected nested path, got %q", diff.Files[0].Path)
	}
}

// --- Sad path tests ---

func TestParse_EmptyInput(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "no file diffs found") {
		t.Errorf("expected 'no file diffs' error, got: %v", err)
	}
}

func TestParse_NoFileHeaders(t *testing.T) {
	input := strings.Join([]string{
		"@@",
		" context",
		"+added",
	}, "\n")

	_, err := Parse(input)
	if err == nil {
		t.Fatal("expected error for @@ without file header")
	}
	if !strings.Contains(err.Error(), "without file header") {
		t.Errorf("expected 'without file header' error, got: %v", err)
	}
}

func TestParse_InvalidPrefixInHunk(t *testing.T) {
	input := strings.Join([]string{
		"--- a/file.php",
		"+++ b/file.php",
		"@@",
		" context",
		"XBAD LINE",
	}, "\n")

	_, err := Parse(input)
	if err == nil {
		t.Fatal("expected error for invalid prefix in hunk")
	}
	if !strings.Contains(err.Error(), "unexpected prefix") {
		t.Errorf("expected 'unexpected prefix' error, got: %v", err)
	}
}

func TestParse_OnlyFileHeadersNoHunks(t *testing.T) {
	input := strings.Join([]string{
		"--- a/file.php",
		"+++ b/file.php",
	}, "\n")

	diff, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// File exists but has no hunks — valid but empty.
	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}
	if len(diff.Files[0].Hunks) != 0 {
		t.Errorf("expected 0 hunks, got %d", len(diff.Files[0].Hunks))
	}
}

func TestParse_GarbageOnly(t *testing.T) {
	_, err := Parse("this is not a diff at all\njust random text\n")
	if err == nil {
		t.Fatal("expected error for garbage input")
	}
	if !strings.Contains(err.Error(), "no file diffs found") {
		t.Errorf("expected 'no file diffs' error, got: %v", err)
	}
}

func TestParse_HunkSeparatorWithExtraText(t *testing.T) {
	// @@ with trailing content should still be recognized as a hunk separator.
	input := strings.Join([]string{
		"--- a/file.php",
		"+++ b/file.php",
		"@@ some label",
		" context",
		"+added",
	}, "\n")

	diff, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(diff.Files[0].Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(diff.Files[0].Hunks))
	}
}

func TestParse_MissingPlusHeader(t *testing.T) {
	// --- without matching +++ — the @@ arrives and current is nil.
	input := strings.Join([]string{
		"--- a/file.php",
		"@@",
		" context",
	}, "\n")

	_, err := Parse(input)
	if err == nil {
		t.Fatal("expected error for @@ without +++ header")
	}
}

// --- Helpers ---

func assertLine(t *testing.T, l Line, wantOp Op, wantContent string) {
	t.Helper()
	if l.Op != wantOp {
		t.Errorf("op: expected %q, got %q", string(wantOp), string(l.Op))
	}
	if l.Content != wantContent {
		t.Errorf("content: expected %q, got %q", wantContent, l.Content)
	}
}
