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

// typeMapping defines the relationship between Go types and their pgtype nullable wrappers
// This mapping is consistent with postgresql_type.go's type conversions for pgx driver
type typeMapping struct {
	pgtypeWrapper string // The pgtype wrapper type (e.g., "pgtype.Int4")
	valueField    string // The field name to extract the value (e.g., "Int32")
}

// goToPgtypeMap maps Go basic types to their pgtype nullable equivalents
// This is derived from postgresql_type.go's mappings for the pgx driver
var goToPgtypeMap = map[string]typeMapping{
	"bool":    {pgtypeWrapper: "pgtype.Bool", valueField: "Bool"},
	"int16":   {pgtypeWrapper: "pgtype.Int2", valueField: "Int16"},
	"int32":   {pgtypeWrapper: "pgtype.Int4", valueField: "Int32"},
	"int64":   {pgtypeWrapper: "pgtype.Int8", valueField: "Int64"},
	"float32": {pgtypeWrapper: "pgtype.Float4", valueField: "Float32"},
	"float64": {pgtypeWrapper: "pgtype.Float8", valueField: "Float64"},
	"string":  {pgtypeWrapper: "pgtype.Text", valueField: "String"},
}

// getNullableType returns the appropriate nullable wrapper type for a given field type
// This is used for scanning embed fields from LEFT JOIN queries where values can be NULL
// The mappings are consistent with postgresql_type.go's type conversions
func getNullableType(fieldType string, modelsPackage string) string {
	// Check if we have a mapping for basic Go types
	if mapping, ok := goToPgtypeMap[fieldType]; ok {
		return mapping.pgtypeWrapper
	}

	// Handle custom types from the models package (e.g., entity.ImageType -> entity.NullImageType)
	// This follows the pattern in postgresql_type.go for enum handling where nullable enums
	// are prefixed with "Null" (lines 582-589 in postgresql_type.go)
	if modelsPackage != "" && strings.HasPrefix(fieldType, modelsPackage+".") {
		typeName := strings.TrimPrefix(fieldType, modelsPackage+".")
		return modelsPackage + ".Null" + typeName
	}

	// For types that are already nullable (pgtype.UUID, pgtype.Timestamp, etc.), return as-is
	return fieldType
}

// getNullableValueField returns the field name to extract the actual value from a nullable wrapper
// For example: pgtype.Bool -> "Bool", pgtype.Int4 -> "Int32", entity.NullImageType -> "ImageType"
// Returns empty string if the value can be used directly (already nullable types)
func getNullableValueField(fieldType string, modelsPackage string) string {
	// Check if we have a mapping for basic Go types
	if mapping, ok := goToPgtypeMap[fieldType]; ok {
		return mapping.valueField
	}

	// Handle custom types from the models package (e.g., entity.ImageType -> "ImageType")
	// For enum types, the Null wrapper struct has a field with the same name as the non-null type
	// For example: NullImageType struct has an ImageType field
	if modelsPackage != "" && strings.HasPrefix(fieldType, modelsPackage+".") {
		return strings.TrimPrefix(fieldType, modelsPackage+".")
	}

	// For types that are already nullable or don't need extraction, return empty string
	// This will cause the template to use the value directly
	return ""
}
