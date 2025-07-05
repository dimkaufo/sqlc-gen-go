package golang

import (
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/gobuffalo/flect"
	"github.com/iancoleman/strcase"

	"github.com/sqlc-dev/sqlc-gen-go/internal/opts"
)

// NestedFieldConfig represents field mapping configuration (auto-generated from SQLC structs)
type NestedFieldConfig struct {
	Name     string `json:"name"`      // Go struct field name
	RowField string `json:"row_field"` // SQL row field name
	Type     string `json:"type"`      // Go type
	JsonTag  string `json:"json_tag"`  // JSON tag name
}

// NestedTemplateData represents the data passed to the nested template
type NestedTemplateData struct {
	Package         string
	ImportPath      string
	FunctionName    string
	QueryName       string
	RootStruct      string
	KeyType         string
	GroupField      string
	EmitJSONTags    bool
	EmitPointers    bool // Whether to emit pointer types for row parameters
	NestedStructs   []NestedStructData
	GenerateStructs []NestedStructDefinition
	NestedFields    []string
	Structs         []Struct
}

// NestedStructData represents data for a nested structure in the template
type NestedStructData struct {
	StructIn  string              // Input struct name from SQLC
	StructOut string              // Output struct name for grouping
	Field     string              // Field to group by
	IsSlice   bool                // Whether this is a slice/array field
	IsPointer bool                // Whether to use pointers
	FieldName string              // Field name in parent struct (e.g., "Books" or "Book")
	Fields    []NestedFieldConfig // Auto-extracted field mappings
	Nested    []NestedStructData  // Nested structures within this one
}

// NestedStructDefinition represents struct definition data for the template
type NestedStructDefinition struct {
	Name         string              // Struct name
	Fields       []NestedFieldConfig // Struct fields (auto-extracted)
	NestedArrays []NestedArrayField  // Arrays of nested structs
}

// NestedArrayField represents a field pointing to a nested struct (array or single)
type NestedArrayField struct {
	Name      string // Field name (e.g., "Books" or "Book")
	Type      string // Field type (e.g., "[]*BookGroup", "[]BookGroup", "*BookGroup", or "BookGroup")
	JsonTag   string // JSON tag name (e.g., "books" or "book")
	IsSlice   bool   // Whether this is a slice/array field
	IsPointer bool   // Whether to use pointers
}

// generateNestedGroupingFunctionsForSource generates nested grouping functions for a specific source file
func generateNestedGroupingFunctionsForSource(options *opts.Options, queries []Query, structs []Struct, configs []opts.NestedQueryConfig) ([]string, error) {
	var functions []string

	for _, config := range configs {
		// Find the corresponding query
		var targetQuery *Query
		for _, q := range queries {
			if q.MethodName == config.Query || q.SourceName == config.Query {
				targetQuery = &q
				break
			}
		}

		if targetQuery == nil {
			continue // Skip if query not found
		}

		// Generate the function with automatic field extraction
		function, err := generateNestedFunction(options, *targetQuery, config, structs)
		if err != nil {
			return nil, fmt.Errorf("failed to generate nested function for query %s: %w", config.Query, err)
		}

		functions = append(functions, function)
	}

	return functions, nil
}

// generateNestedFunction generates a single nested grouping function
func generateNestedFunction(options *opts.Options, query Query, config opts.NestedQueryConfig, structs []Struct) (string, error) {
	// Build template data with automatic field extraction
	data := buildNestedTemplateData(options, query, config, structs)

	// Load and execute template with custom functions
	tmpl, err := template.New("nested").Funcs(template.FuncMap{
		"lower":     strings.ToLower,
		"camelCase": strcase.ToLowerCamel,
	}).Parse(nestedTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse nested template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute nested template: %w", err)
	}

	return buf.String(), nil
}

// collectNestedFields recursively collects all nested field names from the config
func collectNestedFields(config []opts.NestedConfig) []string {
	var fields []string
	for _, nested := range config {
		fields = append(fields, nested.StructIn)
		// Recursively collect nested fields
		if len(nested.Nested) > 0 {
			fields = append(fields, collectNestedFields(nested.Nested)...)
		}
	}
	return fields
}

// extractFields extracts fields from a struct, excluding nested fields
func extractFields(structFields []Field, options *opts.Options, nestedFields []string, rowFieldPrefix string) []NestedFieldConfig {
	var fields []NestedFieldConfig

	// Convert SQLC fields to nested field configs, excluding configured nested fields
	for _, field := range structFields {
		// Skip fields that are configured as nested
		if slices.Contains(nestedFields, field.Name) {
			continue
		}

		// For root fields (when prefix is empty), just use the field name
		// For nested fields, use the prefix.field format
		rowField := field.Name
		if rowFieldPrefix != "" {
			rowField = fmt.Sprintf("%s.%s", rowFieldPrefix, field.Name)
		}

		nestedField := NestedFieldConfig{
			Name:     field.Name,
			RowField: rowField,
			Type:     field.Type,
			JsonTag:  extractJsonTag(field, options),
		}
		fields = append(fields, nestedField)
	}

	return fields
}

// buildNestedStructData recursively builds nested struct data
func buildNestedStructData(queryName string, config opts.NestedConfig, structs []Struct, options *opts.Options) NestedStructData {
	structOut := config.StructOut
	if structOut == "" {
		structOut = fmt.Sprintf("%s%sGroup", queryName, config.StructIn)
	}

	// Default field to "ID" if not specified
	field := config.Field
	if field == "" {
		field = "ID"
	}

	// Default slice to true if not specified
	isSlice := true
	if config.Slice != nil {
		isSlice = *config.Slice
	}

	// Default pointer to true if not specified
	isPointer := true
	if config.Pointer != nil {
		isPointer = *config.Pointer
	}

	// Generate field name based on slice setting using proper pluralization
	var fieldName string
	if isSlice {
		fieldName = pluralizeWithCase(config.StructIn)
	} else {
		fieldName = singularizeWithCase(config.StructIn)
	}

	// Collect nested fields recursively
	nestedFields := collectNestedFields(config.Nested)

	// Find the struct fields for the given StructIn
	var structFields []Field
	for _, s := range structs {
		if s.Name == config.StructIn {
			structFields = s.Fields
			break
		}
	}

	// Automatically extract fields from SQLC struct
	fields := extractFields(structFields, options, nestedFields, config.StructIn)

	// Build nested structures recursively
	var nestedStructs []NestedStructData
	for _, nested := range config.Nested {
		nestedData := buildNestedStructData(queryName, nested, structs, options)
		nestedStructs = append(nestedStructs, nestedData)
	}

	return NestedStructData{
		StructIn:  config.StructIn,
		StructOut: structOut,
		Field:     field,
		IsSlice:   isSlice,
		IsPointer: isPointer,
		FieldName: fieldName,
		Fields:    fields,
		Nested:    nestedStructs,
	}
}

// buildNestedTemplateData builds the template data for nested function generation
func buildNestedTemplateData(options *opts.Options, query Query, config opts.NestedQueryConfig, structs []Struct) NestedTemplateData {
	queryName := query.MethodName
	if queryName == "" {
		queryName = query.SourceName
	}

	// Generate struct names
	rootStruct := config.StructRoot
	if rootStruct == "" {
		rootStruct = fmt.Sprintf("%sGroup", queryName)
	}

	functionName := fmt.Sprintf("Group%s", queryName)

	// Default root field to "ID" if not specified
	rootField := config.Field
	if rootField == "" {
		rootField = "ID"
	}

	// Build nested structs data with automatic field extraction and multi-level nesting
	var nestedStructs []NestedStructData
	for _, nested := range config.Group {
		structData := buildNestedStructData(queryName, nested, structs, options)
		nestedStructs = append(nestedStructs, structData)
	}

	// Collect all nested field names recursively from the config
	nestedFields := collectNestedFields(config.Group)

	// Add root struct definition - extract only non-nested fields from root struct
	// Use the query's Row struct fields directly
	var rootRowFields []Field
	if query.Ret.Struct != nil {
		rootRowFields = query.Ret.Struct.Fields
	}
	rootFields := extractFields(rootRowFields, options, nestedFields, "")
	rootArrays := buildNestedArrayFields(config.Group)

	// Build struct definitions with automatic field extraction
	var generateStructs []NestedStructDefinition
	generateStructs = append(generateStructs, NestedStructDefinition{
		Name:         rootStruct,
		Fields:       rootFields,
		NestedArrays: rootArrays,
	})

	// Add nested struct definitions recursively
	generateStructs = append(generateStructs, buildAllNestedStructDefinitions(nestedStructs, structs, options)...)

	// Get models package import path and name
	modelsPackageImport := options.ModelsPackageImportPath
	modelsPackageName := ""
	if modelsPackageImport != "" {
		parts := strings.Split(modelsPackageImport, "/")
		modelsPackageName = parts[len(parts)-1]
	}

	// Check if any struct uses types from models package
	needsModelsPackage := false
	if modelsPackageName != "" {
		for _, s := range structs {
			for _, f := range s.Fields {
				if strings.HasPrefix(f.Type, modelsPackageName+".") {
					needsModelsPackage = true
					break
				}
			}
			if needsModelsPackage {
				break
			}
		}
	}

	return NestedTemplateData{
		Package:         options.Package,
		FunctionName:    functionName,
		QueryName:       queryName,
		RootStruct:      rootStruct,
		KeyType:         determineKeyType(rootField),
		GroupField:      rootField,
		EmitJSONTags:    options.EmitJsonTags,
		EmitPointers:    options.EmitResultStructPointers,
		NestedStructs:   nestedStructs,
		GenerateStructs: generateStructs,
	}
}

// buildAllNestedStructDefinitions recursively builds all nested struct definitions
func buildAllNestedStructDefinitions(nestedStructs []NestedStructData, structs []Struct, options *opts.Options) []NestedStructDefinition {
	var definitions []NestedStructDefinition

	for _, nested := range nestedStructs {
		// Build nested arrays for this struct
		var nestedArrays []NestedArrayField
		for _, childNested := range nested.Nested {
			var fieldName, jsonTag, fieldType string
			if childNested.IsSlice {
				fieldName = pluralizeWithCase(childNested.StructIn) // "Reviews"
				jsonTag = strings.ToLower(fieldName)                // "reviews"
				if childNested.IsPointer {
					fieldType = fmt.Sprintf("[]*%s", childNested.StructOut) // "[]*GetAuthorsBookReview"
				} else {
					fieldType = fmt.Sprintf("[]%s", childNested.StructOut) // "[]GetAuthorsBookReview"
				}
			} else {
				fieldName = singularizeWithCase(childNested.StructIn) // "Review"
				jsonTag = strings.ToLower(fieldName)                  // "review"
				if childNested.IsPointer {
					fieldType = fmt.Sprintf("*%s", childNested.StructOut) // "*GetAuthorsBookReview"
				} else {
					fieldType = childNested.StructOut // "GetAuthorsBookReview"
				}
			}

			nestedArrays = append(nestedArrays, NestedArrayField{
				Name:      fieldName,
				Type:      fieldType,
				JsonTag:   jsonTag,
				IsSlice:   childNested.IsSlice,
				IsPointer: childNested.IsPointer,
			})
		}

		// Add struct definition
		definitions = append(definitions, NestedStructDefinition{
			Name:         nested.StructOut,
			Fields:       nested.Fields,
			NestedArrays: nestedArrays,
		})

		// Recursively add child struct definitions
		definitions = append(definitions, buildAllNestedStructDefinitions(nested.Nested, structs, options)...)
	}

	return definitions
}

// extractJsonTag extracts the JSON tag from a SQLC field
func extractJsonTag(field Field, options *opts.Options) string {
	// Check if field already has a JSON tag
	if jsonTag, exists := field.Tags["json"]; exists {
		return jsonTag
	}

	// Generate JSON tag based on field name and options
	return JSONTagName(field.DBName, options)
}

// buildNestedArrayFields builds nested field definitions (arrays or single objects)
func buildNestedArrayFields(nested []opts.NestedConfig) []NestedArrayField {
	var arrays []NestedArrayField

	for _, n := range nested {
		structOut := n.StructOut
		if structOut == "" {
			structOut = fmt.Sprintf("%sGroup", n.StructIn)
		}

		// Default slice to true if not specified
		isSlice := true
		if n.Slice != nil {
			isSlice = *n.Slice
		}

		// Default pointer to true if not specified
		isPointer := true
		if n.Pointer != nil {
			isPointer = *n.Pointer
		}

		// Generate field name and JSON tag based on slice and pointer settings
		var fieldName, jsonTag, fieldType string
		if isSlice {
			fieldName = pluralizeWithCase(n.StructIn) // "Books" or "Vacancies"
			jsonTag = strings.ToLower(fieldName)      // "books" or "vacancies"
			if isPointer {
				fieldType = fmt.Sprintf("[]*%s", structOut) // "[]*GetAuthorsBook"
			} else {
				fieldType = fmt.Sprintf("[]%s", structOut) // "[]GetAuthorsBook"
			}
		} else {
			fieldName = singularizeWithCase(n.StructIn) // "Book" or "Vacancy"
			jsonTag = strings.ToLower(fieldName)        // "book" or "vacancy"
			if isPointer {
				fieldType = fmt.Sprintf("*%s", structOut) // "*GetAuthorsBook"
			} else {
				fieldType = structOut // "GetAuthorsBook"
			}
		}

		arrays = append(arrays, NestedArrayField{
			Name:      fieldName,
			Type:      fieldType,
			JsonTag:   jsonTag,
			IsSlice:   isSlice,
			IsPointer: isPointer,
		})
	}

	return arrays
}

// determineKeyType determines the key type for the map based on the field
func determineKeyType(field string) string {
	// For now, default to UUID for all ID fields
	// This could be enhanced to analyze the actual field type from SQLC catalog
	return "pgtype.UUID"
}

// pluralizeWithCase properly pluralizes a word while preserving its case
func pluralizeWithCase(word string) string {
	if word == "" {
		return word
	}

	// Convert to lowercase for flect processing
	lowercaseWord := strings.ToLower(word)
	pluralized := flect.Pluralize(lowercaseWord)

	// Preserve the original case - if first letter was uppercase, make result uppercase
	if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
		return strings.ToUpper(pluralized[:1]) + pluralized[1:]
	}

	return pluralized
}

// singularizeWithCase properly singularizes a word while preserving its case
func singularizeWithCase(word string) string {
	if word == "" {
		return word
	}

	// Convert to lowercase for flect processing
	lowercaseWord := strings.ToLower(word)
	singularized := flect.Singularize(lowercaseWord)

	// Preserve the original case - if first letter was uppercase, make result uppercase
	if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
		return strings.ToUpper(singularized[:1]) + singularized[1:]
	}

	return singularized
}

// nestedTemplate is the template for generating nested grouping functions
const nestedTemplate = `{{- $rootStruct := .RootStruct}}
{{- $queryName := .QueryName}}
{{- $keyType := .KeyType}}
{{- $groupField := .GroupField}}
{{- $generateStructs := .GenerateStructs}}

{{- range .GenerateStructs}}
// {{.Name}} represents grouped data for {{.Name}}
type {{.Name}} struct {
{{- range .Fields}}
	{{- if $.EmitJSONTags}}
	{{.Name}} {{.Type}} ` + "`json:\"{{.JsonTag}}\"`" + `
	{{- else}}
	{{.Name}} {{.Type}}
	{{- end}}
{{- end}}
{{- range .NestedArrays}}
	{{- if $.EmitJSONTags}}
	{{.Name}} {{.Type}} ` + "`json:\"{{.JsonTag}}\"`" + `
	{{- else}}
	{{.Name}} {{.Type}}
	{{- end}}
{{- end}}
}
{{- end}}

// {{.FunctionName}} groups flat {{.QueryName}} rows into nested {{.RootStruct}} structures
func {{.FunctionName}}(rows []{{if .EmitPointers}}*{{end}}{{.QueryName}}Row) []{{if .EmitPointers}}*{{end}}{{.RootStruct}} {
	{{.RootStruct | camelCase}}Map := make(map[{{.KeyType}}]*{{.RootStruct}})

	for _, r := range rows {
		{{.RootStruct | camelCase}} := getOrCreate{{.RootStruct}}({{.RootStruct | camelCase}}Map, r)

		{{- range .NestedStructs}}
		{{- $parentStruct := .}}
		// Handle {{.StructOut}} nested relationship
		if r.{{.StructIn}}.ID.Valid {
			{{- if .Nested}}
			{{.StructIn | camelCase}} := getOrCreate{{.StructOut}}({{$.RootStruct | camelCase}}, r)
			{{- range .Nested}}
			// Handle {{.StructOut}} nested within {{$parentStruct.StructIn}}
			if r.{{.StructIn}}.ID.Valid {
				getOrCreate{{.StructOut}}({{$parentStruct.StructIn | camelCase}}, r)
			}
			{{- end}}
			{{- else}}
			getOrCreate{{.StructOut}}({{$.RootStruct | camelCase}}, r)
			{{- end}}
		}
		{{- end}}
	}

	{{- if .EmitPointers}}
	var result []*{{.RootStruct}}
	for _, {{.RootStruct | camelCase}} := range {{.RootStruct | camelCase}}Map {
		result = append(result, {{.RootStruct | camelCase}})
	}
	{{- else}}
	var result []{{.RootStruct}}
	for _, {{.RootStruct | camelCase}} := range {{.RootStruct | camelCase}}Map {
		result = append(result, *{{.RootStruct | camelCase}})
	}
	{{- end}}

	return result
}

// getOrCreate{{.RootStruct}} gets or creates a {{.RootStruct}} from the map
func getOrCreate{{.RootStruct}}({{.RootStruct | camelCase}}Map map[{{.KeyType}}]*{{.RootStruct}}, r {{if .EmitPointers}}*{{end}}{{.QueryName}}Row) *{{.RootStruct}} {
	if {{.RootStruct | camelCase}}, exists := {{.RootStruct | camelCase}}Map[r.{{.GroupField}}]; exists {
		return {{.RootStruct | camelCase}}
	}

	{{.RootStruct | camelCase}} := &{{.RootStruct}}{
		{{- range .GenerateStructs}}
		{{- if eq .Name $rootStruct}}
		{{- range .Fields}}
		{{.Name}}: r.{{.RowField}},
		{{- end}}
		{{- end}}
		{{- end}}
	}
	{{.RootStruct | camelCase}}Map[r.{{.GroupField}}] = {{.RootStruct | camelCase}}
	return {{.RootStruct | camelCase}}
}

{{- range .NestedStructs}}
{{- $currentStruct := .}}
// getOrCreate{{.StructOut}} gets or creates a {{.StructOut}} within the parent structure
func getOrCreate{{.StructOut}}(parent *{{$rootStruct}}, r {{if $.EmitPointers}}*{{end}}{{$queryName}}Row) *{{.StructOut}} {
	{{- if .IsSlice}}
	// Check if {{.StructOut}} already exists in parent slice
	for i := range parent.{{.FieldName}} {
		{{- if .IsPointer}}
		if parent.{{.FieldName}}[i].ID == r.{{.StructIn}}.ID {
			return parent.{{.FieldName}}[i]
		}
		{{- else}}
		if parent.{{.FieldName}}[i].ID == r.{{.StructIn}}.ID {
			return &parent.{{.FieldName}}[i]
		}
		{{- end}}
	}

	// Create new {{.StructOut}} with auto-extracted field mapping
	{{- if .IsPointer}}
	newItem := &{{.StructOut}}{
	{{- else}}
	newItem := {{.StructOut}}{
	{{- end}}
		{{- $currentStructOut := .StructOut}}
		{{- range $generateStructs}}
		{{- if eq .Name $currentStructOut}}
		{{- range .Fields}}
		{{.Name}}: r.{{.RowField}},
		{{- end}}
		{{- end}}
		{{- end}}
	}

	parent.{{.FieldName}} = append(parent.{{.FieldName}}, newItem)
	{{- if .IsPointer}}
	return newItem
	{{- else}}
	return &parent.{{.FieldName}}[len(parent.{{.FieldName}})-1]
	{{- end}}
	{{- else}}
	// Single object case - check if already set
	{{- if .IsPointer}}
	if parent.{{.FieldName}} != nil && parent.{{.FieldName}}.ID.Valid && parent.{{.FieldName}}.ID == r.{{.StructIn}}.ID {
		return parent.{{.FieldName}}
	}

	// Create new {{.StructOut}} with auto-extracted field mapping
	parent.{{.FieldName}} = &{{.StructOut}}{
		{{- $currentStructOut := .StructOut}}
		{{- range $generateStructs}}
		{{- if eq .Name $currentStructOut}}
		{{- range .Fields}}
		{{.Name}}: r.{{.RowField}},
		{{- end}}
		{{- end}}
		{{- end}}
	}

	return parent.{{.FieldName}}
	{{- else}}
	if parent.{{.FieldName}}.ID.Valid && parent.{{.FieldName}}.ID == r.{{.StructIn}}.ID {
		return &parent.{{.FieldName}}
	}

	// Create new {{.StructOut}} with auto-extracted field mapping
	parent.{{.FieldName}} = {{.StructOut}}{
		{{- $currentStructOut := .StructOut}}
		{{- range $generateStructs}}
		{{- if eq .Name $currentStructOut}}
		{{- range .Fields}}
		{{.Name}}: r.{{.RowField}},
		{{- end}}
		{{- end}}
		{{- end}}
	}

	return &parent.{{.FieldName}}
	{{- end}}
	{{- end}}
}

{{- range .Nested}}
// getOrCreate{{.StructOut}} gets or creates a {{.StructOut}} within the {{$currentStruct.StructOut}} structure
func getOrCreate{{.StructOut}}(parent *{{$currentStruct.StructOut}}, r {{if $.EmitPointers}}*{{end}}{{$queryName}}Row) *{{.StructOut}} {
	{{- if .IsSlice}}
	// Check if {{.StructOut}} already exists in parent slice
	for i := range parent.{{.FieldName}} {
		{{- if .IsPointer}}
		if parent.{{.FieldName}}[i].ID == r.{{.StructIn}}.ID {
			return parent.{{.FieldName}}[i]
		}
		{{- else}}
		if parent.{{.FieldName}}[i].ID == r.{{.StructIn}}.ID {
			return &parent.{{.FieldName}}[i]
		}
		{{- end}}
	}

	// Create new {{.StructOut}} with auto-extracted field mapping
	{{- if .IsPointer}}
	newItem := &{{.StructOut}}{
	{{- else}}
	newItem := {{.StructOut}}{
	{{- end}}
		{{- $currentStructOut := .StructOut}}
		{{- range $generateStructs}}
		{{- if eq .Name $currentStructOut}}
		{{- range .Fields}}
		{{.Name}}: r.{{.RowField}},
		{{- end}}
		{{- end}}
		{{- end}}
	}

	parent.{{.FieldName}} = append(parent.{{.FieldName}}, newItem)
	{{- if .IsPointer}}
	return newItem
	{{- else}}
	return &parent.{{.FieldName}}[len(parent.{{.FieldName}})-1]
	{{- end}}
	{{- else}}
	// Single object case - check if already set
	{{- if .IsPointer}}
	if parent.{{.FieldName}} != nil && parent.{{.FieldName}}.ID.Valid && parent.{{.FieldName}}.ID == r.{{.StructIn}}.ID {
		return parent.{{.FieldName}}
	}

	// Create new {{.StructOut}} with auto-extracted field mapping
	parent.{{.FieldName}} = &{{.StructOut}}{
		{{- $currentStructOut := .StructOut}}
		{{- range $generateStructs}}
		{{- if eq .Name $currentStructOut}}
		{{- range .Fields}}
		{{.Name}}: r.{{.RowField}},
		{{- end}}
		{{- end}}
		{{- end}}
	}

	return parent.{{.FieldName}}
	{{- else}}
	if parent.{{.FieldName}}.ID.Valid && parent.{{.FieldName}}.ID == r.{{.StructIn}}.ID {
		return &parent.{{.FieldName}}
	}

	// Create new {{.StructOut}} with auto-extracted field mapping
	parent.{{.FieldName}} = {{.StructOut}}{
		{{- $currentStructOut := .StructOut}}
		{{- range $generateStructs}}
		{{- if eq .Name $currentStructOut}}
		{{- range .Fields}}
		{{.Name}}: r.{{.RowField}},
		{{- end}}
		{{- end}}
		{{- end}}
	}

	return &parent.{{.FieldName}}
	{{- end}}
	{{- end}}
}
{{- end}}
{{- end}}
`
