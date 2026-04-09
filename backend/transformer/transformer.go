// Package transformer provides token-based string transformation utilities.
// It supports a template syntax of {{key:transform1:transform2}} where transforms
// are applied left-to-right to values looked up by key.
package transformer

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/jinzhu/inflection"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Registry maps transformer names to their corresponding functions.
var Registry = map[string]func(string) string{
	"lower":  strings.ToLower,
	"title":  toTitle,
	"plural": inflection.Plural,
	"snake":  toSnake,
	"camel":  toCamel,
	"kebab":  toKebab,
	"dot":    toDot,
}

// tokenPattern matches {{key:t1:t2}} tokens in a string.
var tokenPattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

func toTitle(s string) string {
	return cases.Title(language.English).String(s)
}

// splitWords breaks a string into words by splitting on underscores, dots,
// spaces, and camelCase boundaries.
func splitWords(s string) []string {
	// First replace underscores, dots, and hyphens with spaces.
	r := strings.NewReplacer("_", " ", ".", " ", "-", " ")
	s = r.Replace(s)

	// Insert a space before each uppercase letter that follows a lowercase letter
	// or before an uppercase letter followed by a lowercase letter in a run of uppercase.
	var result []rune
	runes := []rune(s)
	for i, ch := range runes {
		if i > 0 && unicode.IsUpper(ch) {
			prev := runes[i-1]
			if unicode.IsLower(prev) {
				result = append(result, ' ')
			} else if unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
				result = append(result, ' ')
			}
		}
		result = append(result, ch)
	}

	parts := strings.Fields(string(result))
	return parts
}

func toSnake(s string) string {
	words := splitWords(s)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, "_")
}

func toCamel(s string) string {
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}
	words[0] = strings.ToLower(words[0])
	for i := 1; i < len(words); i++ {
		words[i] = cases.Title(language.English).String(strings.ToLower(words[i]))
	}
	return strings.Join(words, "")
}

func toKebab(s string) string {
	words := splitWords(s)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, "-")
}

func toDot(s string) string {
	words := splitWords(s)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, ".")
}

// Resolve parses a single {{key:t1:t2}} token (with or without the surrounding
// braces), looks up the key in values, and applies the named transformers
// left-to-right. It returns an error if the key is missing from values or a
// transformer name is unrecognized.
func Resolve(token string, values map[string]string) (string, error) {
	// Strip surrounding {{ and }} if present.
	inner := strings.TrimPrefix(token, "{{")
	inner = strings.TrimSuffix(inner, "}}")
	inner = strings.TrimSpace(inner)

	parts := strings.Split(inner, ":")
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("transformer: empty token")
	}

	key := parts[0]
	val, ok := values[key]
	if !ok {
		return "", fmt.Errorf("transformer: key %q not found in values", key)
	}

	for _, name := range parts[1:] {
		fn, ok := Registry[name]
		if !ok {
			return "", fmt.Errorf("transformer: unknown transformer %q", name)
		}
		val = fn(val)
	}

	return val, nil
}

// ResolveAll finds every {{...}} token in input, resolves each one using the
// provided values map, and returns the resulting string with all tokens replaced.
func ResolveAll(input string, values map[string]string) (string, error) {
	var resolveErr error

	result := tokenPattern.ReplaceAllStringFunc(input, func(match string) string {
		if resolveErr != nil {
			return match
		}
		resolved, err := Resolve(match, values)
		if err != nil {
			resolveErr = err
			return match
		}
		return resolved
	})

	if resolveErr != nil {
		return "", resolveErr
	}
	return result, nil
}
