package patcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Helpers ---

// setupProject creates a temp dir with the given files and returns the path.
func setupProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for path, content := range files {
		abs := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return dir
}

func readFile(t *testing.T, dir, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// --- Apply: Happy path tests ---

func TestApply_SingleAddition(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"app.blade.php": "<html>\n<nav>\n</nav>\n</html>",
	})

	cdiff := strings.Join([]string{
		"--- a/app.blade.php",
		"+++ b/app.blade.php",
		"@@",
		" <nav>",
		"+    <a href=\"/teams\">Teams</a>",
		" </nav>",
	}, "\n")

	diff, err := Parse(cdiff)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	diff.Files[0].Hunks[0].FeatureID = "teams"

	result, err := Apply(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diff}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(result.Conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %v", result.Conflicts)
	}
	if len(result.Modified) != 1 {
		t.Fatalf("expected 1 modified file, got %d", len(result.Modified))
	}

	content := readFile(t, dir, "app.blade.php")
	expected := "<html>\n<nav>\n    <a href=\"/teams\">Teams</a>\n</nav>\n</html>"
	if content != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, content)
	}
}

func TestApply_SingleDeletion(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"config.php": "a\nb\nold_line\nc",
	})

	cdiff := strings.Join([]string{
		"--- a/config.php",
		"+++ b/config.php",
		"@@",
		" b",
		"-old_line",
		" c",
	}, "\n")

	diff, err := Parse(cdiff)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	diff.Files[0].Hunks[0].FeatureID = "cleanup"

	result, err := Apply(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diff}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	content := readFile(t, dir, "config.php")
	expected := "a\nb\nc"
	if content != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, content)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("unexpected conflicts: %v", result.Conflicts)
	}
}

func TestApply_Replacement(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"config.php": "driver=mysql\nhost=localhost\nport=3306",
	})

	cdiff := strings.Join([]string{
		"--- a/config.php",
		"+++ b/config.php",
		"@@",
		" driver=mysql",
		"-host=localhost",
		"+host=127.0.0.1",
		" port=3306",
	}, "\n")

	diff, err := Parse(cdiff)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	diff.Files[0].Hunks[0].FeatureID = "config"

	result, err := Apply(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diff}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	content := readFile(t, dir, "config.php")
	expected := "driver=mysql\nhost=127.0.0.1\nport=3306"
	if content != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, content)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("unexpected conflicts: %v", result.Conflicts)
	}
}

func TestApply_MultipleHunksSameFile(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"routes.php": "// auth\nRoute::auth();\n\n// api\nRoute::api();",
	})

	cdiff := strings.Join([]string{
		"--- a/routes.php",
		"+++ b/routes.php",
		"@@",
		" // auth",
		" Route::auth();",
		"+Route::teams();",
		"@@",
		" // api",
		" Route::api();",
		"+Route::teamsApi();",
	}, "\n")

	diff, err := Parse(cdiff)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for i := range diff.Files[0].Hunks {
		diff.Files[0].Hunks[i].FeatureID = "teams"
	}

	result, err := Apply(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diff}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	content := readFile(t, dir, "routes.php")
	if !strings.Contains(content, "Route::teams();") {
		t.Error("missing Route::teams()")
	}
	if !strings.Contains(content, "Route::teamsApi();") {
		t.Error("missing Route::teamsApi()")
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("unexpected conflicts: %v", result.Conflicts)
	}
}

func TestApply_MultipleFiles(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"file1.php": "a\nb\nc",
		"file2.php": "x\ny\nz",
	})

	cdiff := strings.Join([]string{
		"--- a/file1.php",
		"+++ b/file1.php",
		"@@",
		" a",
		"+inserted1",
		" b",
		"--- a/file2.php",
		"+++ b/file2.php",
		"@@",
		" x",
		"+inserted2",
		" y",
	}, "\n")

	diff, err := Parse(cdiff)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for i := range diff.Files {
		for j := range diff.Files[i].Hunks {
			diff.Files[i].Hunks[j].FeatureID = "multi"
		}
	}

	result, err := Apply(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diff}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(result.Modified) != 2 {
		t.Fatalf("expected 2 modified files, got %d", len(result.Modified))
	}

	c1 := readFile(t, dir, "file1.php")
	if !strings.Contains(c1, "inserted1") {
		t.Error("file1 missing insertion")
	}
	c2 := readFile(t, dir, "file2.php")
	if !strings.Contains(c2, "inserted2") {
		t.Error("file2 missing insertion")
	}
}

func TestApply_TwoFeaturesStackAdditions(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"nav.blade.php": "<nav>\n    <a href=\"/\">Home</a>\n</nav>",
	})

	cdiff1 := strings.Join([]string{
		"--- a/nav.blade.php",
		"+++ b/nav.blade.php",
		"@@",
		" <nav>",
		"+    <a href=\"/teams\">Teams</a>",
		"     <a href=\"/\">Home</a>",
	}, "\n")

	cdiff2 := strings.Join([]string{
		"--- a/nav.blade.php",
		"+++ b/nav.blade.php",
		"@@",
		" <nav>",
		"+    <a href=\"/billing\">Billing</a>",
		"     <a href=\"/\">Home</a>",
	}, "\n")

	diff1, _ := Parse(cdiff1)
	diff2, _ := Parse(cdiff2)
	diff1.Files[0].Hunks[0].FeatureID = "teams"
	diff2.Files[0].Hunks[0].FeatureID = "billing"

	result, err := Apply(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diff1, *diff2}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(result.Conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %v", result.Conflicts)
	}

	content := readFile(t, dir, "nav.blade.php")
	// Both additions should be present.
	if !strings.Contains(content, "Teams") {
		t.Error("missing Teams link")
	}
	if !strings.Contains(content, "Billing") {
		t.Error("missing Billing link")
	}
}

func TestApply_MultipleDisjointDiffs(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"app.php": "line1\nline2\nline3\nline4\nline5",
	})

	// Feature A modifies near the top, Feature B near the bottom.
	cdiffA := strings.Join([]string{
		"--- a/app.php",
		"+++ b/app.php",
		"@@",
		" line1",
		"+insertedA",
		" line2",
	}, "\n")

	cdiffB := strings.Join([]string{
		"--- a/app.php",
		"+++ b/app.php",
		"@@",
		" line4",
		"+insertedB",
		" line5",
	}, "\n")

	diffA, _ := Parse(cdiffA)
	diffB, _ := Parse(cdiffB)
	diffA.Files[0].Hunks[0].FeatureID = "a"
	diffB.Files[0].Hunks[0].FeatureID = "b"

	result, err := Apply(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diffA, *diffB}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(result.Conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %v", result.Conflicts)
	}

	content := readFile(t, dir, "app.php")
	lines := strings.Split(content, "\n")
	// Should have 7 lines: original 5 + 2 insertions.
	if len(lines) != 7 {
		t.Fatalf("expected 7 lines, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(content, "insertedA") {
		t.Error("missing insertedA")
	}
	if !strings.Contains(content, "insertedB") {
		t.Error("missing insertedB")
	}
}

// --- Apply: Sad path tests ---

func TestApply_FileNotFound(t *testing.T) {
	dir := t.TempDir() // empty dir

	cdiff := strings.Join([]string{
		"--- a/missing.php",
		"+++ b/missing.php",
		"@@",
		" context",
		"+added",
	}, "\n")

	diff, _ := Parse(cdiff)
	diff.Files[0].Hunks[0].FeatureID = "test"

	_, err := Apply(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diff}})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "reading") {
		t.Errorf("expected 'reading' error, got: %v", err)
	}
}

func TestApply_ContextMismatch(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"app.php": "alpha\nbeta\ngamma",
	})

	cdiff := strings.Join([]string{
		"--- a/app.php",
		"+++ b/app.php",
		"@@",
		" nonexistent_context",
		"+added",
		" also_nonexistent",
	}, "\n")

	diff, _ := Parse(cdiff)
	diff.Files[0].Hunks[0].FeatureID = "bad"

	_, err := Apply(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diff}})
	if err == nil {
		t.Fatal("expected error for context mismatch")
	}
	if !strings.Contains(err.Error(), "matching") {
		t.Errorf("expected 'matching' error, got: %v", err)
	}
}

func TestApply_ConflictingDeletions(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"config.php": "a\nold\nb",
	})

	cdiff1 := strings.Join([]string{
		"--- a/config.php",
		"+++ b/config.php",
		"@@",
		" a",
		"-old",
		"+new1",
		" b",
	}, "\n")

	cdiff2 := strings.Join([]string{
		"--- a/config.php",
		"+++ b/config.php",
		"@@",
		" a",
		"-old",
		"+new2",
		" b",
	}, "\n")

	diff1, _ := Parse(cdiff1)
	diff2, _ := Parse(cdiff2)
	diff1.Files[0].Hunks[0].FeatureID = "feat1"
	diff2.Files[0].Hunks[0].FeatureID = "feat2"

	result, err := Apply(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diff1, *diff2}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Conflicts) == 0 {
		t.Fatal("expected conflicts for overlapping deletions")
	}
	if result.Conflicts[0].File != "config.php" {
		t.Errorf("expected conflict on config.php, got %s", result.Conflicts[0].File)
	}

	// File should NOT be modified when there are conflicts.
	content := readFile(t, dir, "config.php")
	if content != "a\nold\nb" {
		t.Errorf("file should be unchanged on conflict, got: %s", content)
	}
}

func TestApply_EmptyDiffs(t *testing.T) {
	dir := t.TempDir()
	_, err := Apply(ApplyRequest{ProjectDir: dir, Diffs: nil})
	if err == nil {
		t.Fatal("expected error for empty diffs")
	}
	if !strings.Contains(err.Error(), "no hunks") {
		t.Errorf("expected 'no hunks' error, got: %v", err)
	}
}

func TestApply_NestedSubdirectory(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"resources/views/layouts/app.blade.php": "<html>\n<head>\n</head>\n</html>",
	})

	cdiff := strings.Join([]string{
		"--- a/resources/views/layouts/app.blade.php",
		"+++ b/resources/views/layouts/app.blade.php",
		"@@",
		" <head>",
		"+<link rel=\"stylesheet\" href=\"/css/teams.css\">",
		" </head>",
	}, "\n")

	diff, _ := Parse(cdiff)
	diff.Files[0].Hunks[0].FeatureID = "teams"

	result, err := Apply(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diff}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(result.Conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %v", result.Conflicts)
	}

	content := readFile(t, dir, "resources/views/layouts/app.blade.php")
	if !strings.Contains(content, "teams.css") {
		t.Error("missing teams.css link")
	}
}

// --- Check tests ---

func TestCheck_NoConflicts(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"app.php": "a\nb\nc",
	})

	cdiff := strings.Join([]string{
		"--- a/app.php",
		"+++ b/app.php",
		"@@",
		" a",
		"+new",
		" b",
	}, "\n")

	diff, _ := Parse(cdiff)
	diff.Files[0].Hunks[0].FeatureID = "test"

	conflicts, err := Check(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diff}})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(conflicts))
	}

	// File should NOT be modified by Check.
	content := readFile(t, dir, "app.php")
	if content != "a\nb\nc" {
		t.Error("Check modified the file — it should be read-only")
	}
}

func TestCheck_DetectsConflicts(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"app.php": "a\nold\nb",
	})

	cdiff1 := strings.Join([]string{
		"--- a/app.php",
		"+++ b/app.php",
		"@@",
		" a",
		"-old",
		"+new1",
		" b",
	}, "\n")

	cdiff2 := strings.Join([]string{
		"--- a/app.php",
		"+++ b/app.php",
		"@@",
		" a",
		"-old",
		"+new2",
		" b",
	}, "\n")

	diff1, _ := Parse(cdiff1)
	diff2, _ := Parse(cdiff2)
	diff1.Files[0].Hunks[0].FeatureID = "feat1"
	diff2.Files[0].Hunks[0].FeatureID = "feat2"

	conflicts, err := Check(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diff1, *diff2}})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(conflicts) == 0 {
		t.Fatal("expected conflicts")
	}

	// File should NOT be modified by Check.
	content := readFile(t, dir, "app.php")
	if content != "a\nold\nb" {
		t.Error("Check modified the file")
	}
}

func TestCheck_FileNotFound(t *testing.T) {
	dir := t.TempDir()

	cdiff := strings.Join([]string{
		"--- a/missing.php",
		"+++ b/missing.php",
		"@@",
		" ctx",
		"+add",
	}, "\n")

	diff, _ := Parse(cdiff)
	diff.Files[0].Hunks[0].FeatureID = "test"

	_, err := Check(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diff}})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestCheck_ContextMismatch(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"app.php": "real content here",
	})

	cdiff := strings.Join([]string{
		"--- a/app.php",
		"+++ b/app.php",
		"@@",
		" wrong context",
		"+added",
	}, "\n")

	diff, _ := Parse(cdiff)
	diff.Files[0].Hunks[0].FeatureID = "test"

	_, err := Check(ApplyRequest{ProjectDir: dir, Diffs: []Diff{*diff}})
	if err == nil {
		t.Fatal("expected error for context mismatch")
	}
}
