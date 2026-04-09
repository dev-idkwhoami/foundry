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
				{Line: 2, From: "foo", To: "bar"},
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
				{Line: 2, From: "foo", To: "{{app_name:lower}}"},
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
				{Line: 1, From: "alpha", To: "ALPHA"},
				{Line: 3, From: "gamma", To: "GAMMA"},
			},
		},
		{
			ConfigKey: "second",
			Targets: []Target{
				{Line: 2, From: "beta", To: "BETA"},
				{Line: 4, From: "delta", To: "DELTA"},
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
				{Line: 999, From: "x", To: "y"},
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

func TestResolveMappings_TransformerChaining(t *testing.T) {
	diff := "before\nplaceholder\nafter"
	mappings := []Mapping{
		{
			ConfigKey: "name",
			Targets: []Target{
				{Line: 2, From: "placeholder", To: "{{name:snake:lower}}"},
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
