package features

import (
	"strings"
	"testing"
)

func TestResolveMappings_BasicLineTargetedReplace(t *testing.T) {
	diff := "line one\nfoo is here\nline three"
	mappings := []Mapping{
		{
			ConfigKey: "key1",
			Targets: []Target{
				{Lines: []int{2}, From: "foo", To: "bar"},
			},
		},
	}

	result, err := ResolveMappings(diff, mappings, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line one" {
		t.Errorf("line 1: expected %q, got %q", "line one", lines[0])
	}
	if lines[1] != "bar is here" {
		t.Errorf("line 2: expected %q, got %q", "bar is here", lines[1])
	}
	if lines[2] != "line three" {
		t.Errorf("line 3: expected %q, got %q", "line three", lines[2])
	}
}

func TestResolveMappings_ReplacementWithTransformerTokens(t *testing.T) {
	diff := "header\nAPP_NAME=foo\nfooter"
	mappings := []Mapping{
		{
			ConfigKey: "app_name",
			Targets: []Target{
				{Lines: []int{2}, From: "foo", To: "{{app_name:lower}}"},
			},
		},
	}
	configValues := map[string]string{"app_name": "MyApp"}

	result, err := ResolveMappings(diff, mappings, configValues)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(result, "\n")
	if lines[1] != "APP_NAME=myapp" {
		t.Errorf("line 2: expected %q, got %q", "APP_NAME=myapp", lines[1])
	}
}

func TestResolveMappings_MultipleMappingsAndTargets(t *testing.T) {
	diff := "alpha\nbeta\ngamma\ndelta"
	mappings := []Mapping{
		{
			ConfigKey: "first",
			Targets: []Target{
				{Lines: []int{1}, From: "alpha", To: "ALPHA"},
				{Lines: []int{3}, From: "gamma", To: "GAMMA"},
			},
		},
		{
			ConfigKey: "second",
			Targets: []Target{
				{Lines: []int{2}, From: "beta", To: "BETA"},
				{Lines: []int{4}, From: "delta", To: "DELTA"},
			},
		},
	}

	result, err := ResolveMappings(diff, mappings, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(result, "\n")
	expected := []string{"ALPHA", "BETA", "GAMMA", "DELTA"}
	for i, want := range expected {
		if lines[i] != want {
			t.Errorf("line %d: expected %q, got %q", i+1, want, lines[i])
		}
	}
}

func TestResolveMappings_OutOfRangeLine(t *testing.T) {
	diff := "line one\nline two\nline three"
	mappings := []Mapping{
		{
			ConfigKey: "key",
			Targets: []Target{
				{Lines: []int{999}, From: "x", To: "y"},
			},
		},
	}

	result, err := ResolveMappings(diff, mappings, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != diff {
		t.Errorf("expected diff unchanged, got %q", result)
	}
}

func TestResolveMappings_LinesArray(t *testing.T) {
	diff := "tenant one\nno match\ntenant two\nno match\ntenant three"
	mappings := []Mapping{
		{
			ConfigKey: "noun",
			Targets: []Target{
				{Lines: []int{1, 3, 5}, From: "tenant", To: "org"},
			},
		},
	}

	result, err := ResolveMappings(diff, mappings, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(result, "\n")
	if lines[0] != "org one" {
		t.Errorf("line 1: expected %q, got %q", "org one", lines[0])
	}
	if lines[1] != "no match" {
		t.Errorf("line 2: expected %q, got %q", "no match", lines[1])
	}
	if lines[2] != "org two" {
		t.Errorf("line 3: expected %q, got %q", "org two", lines[2])
	}
	if lines[4] != "org three" {
		t.Errorf("line 5: expected %q, got %q", "org three", lines[4])
	}
}

func TestResolveMappings_GlobalMode(t *testing.T) {
	diff := "tenant here\nno tenant\ntenant there"
	mappings := []Mapping{
		{
			ConfigKey: "noun",
			Targets: []Target{
				{From: "tenant", To: "org"},
			},
		},
	}

	result, err := ResolveMappings(diff, mappings, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(result, "\n")
	if lines[0] != "org here" {
		t.Errorf("line 1: expected %q, got %q", "org here", lines[0])
	}
	if lines[1] != "no org" {
		t.Errorf("line 2: expected %q, got %q", "no org", lines[1])
	}
	if lines[2] != "org there" {
		t.Errorf("line 3: expected %q, got %q", "org there", lines[2])
	}
}

func TestResolveMappings_GlobalModeWithTransformer(t *testing.T) {
	diff := "class Team\n  table: teams\n  name: Team"
	mappings := []Mapping{
		{
			ConfigKey: "noun",
			Targets: []Target{
				{From: "Team", To: "{{noun:title}}"},
				{From: "teams", To: "{{noun:plural:lower}}"},
			},
		},
	}
	configValues := map[string]string{"noun": "organization"}

	result, err := ResolveMappings(diff, mappings, configValues)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(result, "\n")
	if lines[0] != "class Organization" {
		t.Errorf("line 1: expected %q, got %q", "class Organization", lines[0])
	}
	if lines[1] != "  table: organizations" {
		t.Errorf("line 2: expected %q, got %q", "  table: organizations", lines[1])
	}
	if lines[2] != "  name: Organization" {
		t.Errorf("line 3: expected %q, got %q", "  name: Organization", lines[2])
	}
}

func TestResolveMappings_TransformerChaining(t *testing.T) {
	diff := "before\nplaceholder\nafter"
	mappings := []Mapping{
		{
			ConfigKey: "name",
			Targets: []Target{
				{Lines: []int{2}, From: "placeholder", To: "{{name:snake:lower}}"},
			},
		},
	}
	configValues := map[string]string{"name": "MyAppName"}

	result, err := ResolveMappings(diff, mappings, configValues)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(result, "\n")
	// "MyAppName" -> snake -> "my_app_name" -> lower -> "my_app_name"
	if lines[1] != "my_app_name" {
		t.Errorf("line 2: expected %q, got %q", "my_app_name", lines[1])
	}
}
