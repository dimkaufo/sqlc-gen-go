package golang

import (
	"fmt"
	"sort"
	"strings"
)

func ternary(cond bool, a, b interface{}) interface{} {
	if cond {
		return a
	}
	return b
}

func joinTags(tags map[string]string) string {
	parts := make([]string, 0, len(tags))
	for k, v := range tags {
		parts = append(parts, fmt.Sprintf(`%s:"%s"`, k, v))
	}
	sort.Strings(parts) // stable order
	return "`" + strings.Join(parts, " ") + "`"
}

// list creates a slice from the provided arguments
// This is useful for passing multiple arguments to templates
func list(args ...interface{}) []interface{} {
	return args
}

func upperTitle(s string) string {
	// capitalize first letter
	return strings.ToUpper(s[:1]) + s[1:]
}

func add(a, b int) int {
	return a + b
}

// dict creates a map from key-value pairs
// This is useful for creating maps to pass to templates for tracking state
func dict(values ...interface{}) map[interface{}]interface{} {
	if len(values)%2 != 0 {
		return nil
	}
	dict := make(map[interface{}]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		dict[values[i]] = values[i+1]
	}
	return dict
}

// set adds a key-value pair to a map and returns an empty string (for use in templates)
// Returns the previous value if the key existed, otherwise returns nil
func set(dict map[interface{}]interface{}, key interface{}, value interface{}) interface{} {
	if dict == nil {
		return nil
	}
	old := dict[key]
	dict[key] = value
	return old
}
