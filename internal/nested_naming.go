package golang

import (
	"regexp"
	"strings"

	"github.com/gobuffalo/flect"
)

// namingCamelPattern regex for identifying camelCase word boundaries
var namingCamelPattern = regexp.MustCompile("[^A-Z][A-Z]+")

// commonInitialisms is a set of common initialisms that should be capitalized
// as per Go naming conventions
var commonInitialisms = map[string]bool{
	"ACL":   true,
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"LHS":   true,
	"QPS":   true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SQL":   true,
	"SSH":   true,
	"TCP":   true,
	"TLS":   true,
	"TTL":   true,
	"UDP":   true,
	"UI":    true,
	"UID":   true,
	"UUID":  true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"VM":    true,
	"XML":   true,
	"XMPP":  true,
	"XSRF":  true,
	"XSS":   true,
}

// ToPascalCaseWithInitialisms converts a string to PascalCase with proper initialism handling
func ToPascalCaseWithInitialisms(s string) string {
	if s == "" {
		return s
	}

	// Handle already PascalCase inputs
	var snakeCase string
	if strings.Contains(s, "_") {
		// Already snake_case
		snakeCase = s
	} else {
		// Convert PascalCase to snake_case using regex
		snakeCase = PascalToSnakeCase(s)
	}

	parts := strings.Split(snakeCase, "_")

	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}

		// Check if this part is a common initialism
		upperPart := strings.ToUpper(part)
		if commonInitialisms[upperPart] {
			result.WriteString(upperPart)
		} else {
			// Regular word - capitalize first letter
			result.WriteString(strings.ToUpper(part[:1]))
			result.WriteString(strings.ToLower(part[1:]))
		}
	}

	return result.String()
}

// PascalToSnakeCase converts PascalCase to snake_case using regex
func PascalToSnakeCase(s string) string {
	if s == "" {
		return s
	}

	// Insert underscore before uppercase letters that follow lowercase letters
	result := namingCamelPattern.ReplaceAllStringFunc(s, func(x string) string {
		return x[:1] + "_" + x[1:]
	})

	return strings.ToLower(result)
}

// inflectCasePreserving applies the given inflection function while preserving case
// This helper extracts common logic used by both PluralizeCasePreserving and SingularizeCasePreserving
func inflectCasePreserving(word string, inflectFunc func(string) string) string {
	if word == "" {
		return word
	}

	// For PascalCase words, handle compound words by inflecting only the last part
	if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
		// Convert to snake_case to identify word boundaries, then inflect the last part
		var snakeCase string
		if strings.Contains(word, "_") {
			snakeCase = word
		} else {
			snakeCase = PascalToSnakeCase(word)
		}
		parts := strings.Split(snakeCase, "_")

		if len(parts) > 1 {
			// Inflect only the last part for compound words
			lastPart := parts[len(parts)-1]
			parts[len(parts)-1] = inflectFunc(lastPart)
			return ToPascalCaseWithInitialisms(strings.Join(parts, "_"))
		}

		// For simple words, inflect and apply proper casing
		inflected := inflectFunc(strings.ToLower(word))
		return ToPascalCaseWithInitialisms(inflected)
	}

	// For non-PascalCase words, just inflect normally
	return inflectFunc(word)
}

// PluralizeCasePreserving properly pluralizes a word while preserving its case
func PluralizeCasePreserving(word string) string {
	return inflectCasePreserving(word, flect.Pluralize)
}

// SingularizeCasePreserving properly singularizes a word while preserving its case
func SingularizeCasePreserving(word string) string {
	return inflectCasePreserving(word, flect.Singularize)
}

// ToCamelCase converts a string to camelCase with proper initialism handling
func ToCamelCase(s string) string {
	// Handle already PascalCase inputs by just making first letter lowercase
	if len(s) > 0 && s[0] >= 'A' && s[0] <= 'Z' {
		// Check if it's already PascalCase (no underscores, starts with uppercase)
		if !strings.Contains(s, "_") {
			return strings.ToLower(s[:1]) + s[1:]
		}
	}
	// Convert to camelCase with proper initialism handling for underscore-separated names
	result := ToPascalCaseWithInitialisms(s)
	// Make first letter lowercase for camelCase
	if len(result) > 0 {
		result = strings.ToLower(result[:1]) + result[1:]
	}
	return result
}
