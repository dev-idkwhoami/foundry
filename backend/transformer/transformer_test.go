package transformer

import (
	"strings"
	"testing"
)

func TestIndividualTransformers(t *testing.T) {
	tests := []struct {
		name      string
		transform string
		input     string
		want      string
	}{
		// lower
		{"lower basic", "lower", "Hello World", "hello world"},

		// title
		{"title basic", "title", "hello world", "Hello World"},

		// plural
		{"plural basic", "plural", "widget", "widgets"},

		// snake
		{"snake from camel", "snake", "HelloWorld", "hello_world"},
		{"snake from spaces", "snake", "hello world", "hello_world"},

		// camel
		{"camel from snake", "camel", "hello_world", "helloWorld"},
		{"camel from spaces", "camel", "hello world", "helloWorld"},

		// kebab
		{"kebab from spaces", "kebab", "Post Fresh", "post-fresh"},
		{"kebab from camel", "kebab", "HelloWorld", "hello-world"},
		{"kebab from snake", "kebab", "hello_world", "hello-world"},

		// dot
		{"dot from snake", "dot", "hello_world", "hello.world"},
		{"dot from camel", "dot", "HelloWorld", "hello.world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := Registry[tt.transform]
			if !ok {
				t.Fatalf("transformer %q not found in Registry", tt.transform)
			}
			got := fn(tt.input)
			if got != tt.want {
				t.Errorf("Registry[%q](%q) = %q, want %q", tt.transform, tt.input, got, tt.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	values := map[string]string{
		"name": "Widget",
	}

	t.Run("single transformer", func(t *testing.T) {
		got, err := Resolve("{{name:lower}}", values)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "widget" {
			t.Errorf("got %q, want %q", got, "widget")
		}
	})

	t.Run("no transformer returns raw value", func(t *testing.T) {
		got, err := Resolve("{{name}}", values)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "Widget" {
			t.Errorf("got %q, want %q", got, "Widget")
		}
	})

	t.Run("chained transformers plural then lower", func(t *testing.T) {
		got, err := Resolve("{{name:plural:lower}}", values)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "widgets" {
			t.Errorf("got %q, want %q", got, "widgets")
		}
	})

	t.Run("chained transformers snake then dot", func(t *testing.T) {
		vals := map[string]string{"key": "HelloWorld"}
		got, err := Resolve("{{key:snake:dot}}", vals)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// snake: "hello_world", dot converts underscores to dots: "hello.world"
		if got != "hello.world" {
			t.Errorf("got %q, want %q", got, "hello.world")
		}
	})
}

func TestResolveErrors(t *testing.T) {
	values := map[string]string{
		"name": "Widget",
	}

	t.Run("missing key", func(t *testing.T) {
		_, err := Resolve("{{missing:lower}}", values)
		if err == nil {
			t.Fatal("expected an error for missing key, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error should mention 'not found', got: %v", err)
		}
	})

	t.Run("unknown transformer", func(t *testing.T) {
		_, err := Resolve("{{name:bogus}}", values)
		if err == nil {
			t.Fatal("expected an error for unknown transformer, got nil")
		}
		if !strings.Contains(err.Error(), "unknown transformer") {
			t.Errorf("error should mention 'unknown transformer', got: %v", err)
		}
	})
}

func TestResolveAll(t *testing.T) {
	values := map[string]string{
		"model": "Widget",
		"table": "user account",
	}

	t.Run("multiple tokens", func(t *testing.T) {
		input := "Model: {{model:title}}, Table: {{table:snake}}"
		got, err := ResolveAll(input, values)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "Model: Widget, Table: user_account"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("no tokens returns input unchanged", func(t *testing.T) {
		input := "no tokens here"
		got, err := ResolveAll(input, values)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != input {
			t.Errorf("got %q, want %q", got, input)
		}
	})

	t.Run("error propagates from bad token", func(t *testing.T) {
		input := "ok {{model:lower}} then {{missing:lower}}"
		_, err := ResolveAll(input, values)
		if err == nil {
			t.Fatal("expected error for missing key, got nil")
		}
	})
}
