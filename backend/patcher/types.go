package patcher

// Op represents the operation type for a diff line.
type Op rune

const (
	OpContext Op = ' '
	OpAdd     Op = '+'
	OpDelete  Op = '-'
)

// Line is a single line in a diff hunk.
type Line struct {
	Op      Op
	Content string
}

// Hunk is a contiguous block of context, additions, and deletions.
type Hunk struct {
	Lines     []Line
	FeatureID string
}

// FileDiff groups all hunks targeting a single file.
type FileDiff struct {
	Path  string
	Hunks []Hunk
}

// Diff is the top-level result of parsing a .cdiff file.
type Diff struct {
	Files []FileDiff
}

// MatchResult records where a hunk's context was found in the target file.
type MatchResult struct {
	StartLine int // 0-indexed line in target where context begins
	EndLine   int // 0-indexed exclusive end of the context+deletion range
}

// MatchedHunk pairs a hunk with its resolved position in the target file.
type MatchedHunk struct {
	Hunk        Hunk
	MatchResult MatchResult
}

// Conflict describes an unresolvable overlap between two feature hunks.
type Conflict struct {
	File     string `json:"file"`
	FeatureA string `json:"featureA"`
	FeatureB string `json:"featureB"`
	Reason   string `json:"reason"`
}

// ApplyRequest is the input to Apply and Check.
type ApplyRequest struct {
	ProjectDir string
	Diffs      []Diff
}

// ApplyResult is the output of a successful Apply.
type ApplyResult struct {
	Modified  []string   `json:"modified"`
	Conflicts []Conflict `json:"conflicts"`
}
