package golang

import (
	"fmt"
	"strings"

	"github.com/sqlc-dev/sqlc-gen-go/internal/debug"
	"github.com/sqlc-dev/sqlc-gen-go/internal/opts"
)

// NestedQueryTemplateData represents the data passed to the nested template
type NestedQueryTemplateData struct {
	FunctionName   string
	Query          *Query
	RootStructName string            // Sometimes we only need name of the root struct, not the data
	RootStructData *NestedStructData // Nested structures in the root struct
	EmitPointers   bool              // Whether to emit pointer types for row parameters
	EmitJSONTags   bool              // Whether to emit JSON tags

	// Used only in wrapper function template
	CastToQueryName string // Check if we already have query to reuse
}

// MatchReferencedStruct represents a struct referenced in a match config that needs a getter method in the interface.
// This is used when a struct is referenced in a match condition (e.g., Image_2) but is marked as ignored,
// so it doesn't appear in NestedStructs but still needs a getter method for the match condition to work.
type MatchReferencedStruct struct {
	RowFieldName string // Field name in row struct (e.g., "Image_2")
	RowFieldType string // Field type in row struct (e.g., "entity.Image")
}

// ResolvedMatch holds sqlc row field names for match conditions (Get<Field>() methods).
// When the same embed appears twice (e.g. Company and Company_2), FromStruct/ToStruct use the suffixed name.
type ResolvedMatch struct {
	FromStruct string
	ToStruct   string
	FromField  string
	ToField    string
}

// NestedStructData represents data for a nested structure in the template
type NestedStructData struct {
	StructIn                string                    // Input struct name from SQLC
	StructOut               string                    // Output struct name for grouping
	FieldGroupBy            string                    // Field to group by
	FieldName               string                    // Field name in parent struct (e.g., "Books" or "Book")
	FieldType               string                    // Field type in parent struct (e.g., "[]*BookGroup", "[]BookGroup", "*BookGroup", or "BookGroup")
	RowFieldName            string                    // Field name in row struct (e.g., "Book")
	RowFieldType            string                    // Field type in row struct (e.g., "BookGroup")
	IsRowFieldExistsInQuery bool                      // Whether the corresponding row field to the StructIn exists in the query return struct
	FieldTags               map[string]string         // Field tag in parent struct (e.g., "json:books" or "json:book")
	KeyType                 string                    // Key type for the map
	IsSlice                 bool                      // Whether this is a slice/array field
	IsPointer               bool                      // Whether to use pointers
	IsComposite             bool                      // Whether this is a composite struct that was already generated
	IsEntityStruct          bool                      // Whether this is an entity struct that should be reused
	EntityType              string                    // Actual entity type when StructIn is an alias (e.g., "Image" when StructIn is "Image_2")
	IsRoot                  bool                      // Whether this is the root of the nested structs
	Match                   []*opts.NestedMatchConfig // Match configuration
	// MatchResolved is the per-query resolved match (duplicate embeds → Company_2, etc.) for template emission.
	MatchResolved []ResolvedMatch

	// MatchReferencedStructs holds structs referenced in match configs that need getter methods in the interface.
	// These are structs that appear in match conditions (from_struct) but may be marked as ignored,
	// so they don't appear in NestedStructs but still need getter methods.
	MatchReferencedStructs []*MatchReferencedStruct

	// Map indicating if this struct's StructOut appears multiple times at each tree level.
	// Key is the level (1 = immediate parent, 2 = grandparent, etc.)
	// Value is true if StructOut appears multiple times at that level.
	// Used for map naming when the same composite struct appears at different levels.
	// For example: RecruitersCompanyComposite at level 1 (direct child) and level 2 (grandchild).
	DuplicatedRelativeToParents map[int]bool

	Fields        []Field             // Non-nested fields
	NestedStructs []*NestedStructData // Nested structures data of the current struct

	// Skip struct generation if it's a composite struct
	// that was already generated or will be generated in another *_nested.sql file
	SkipStructGeneration bool

	// Should generate entity to composite function to populate composite from entity
	ShouldGenerateEntityToCompositeFunction bool
}

type NestedQueryTemplateDataBuilder struct {
	options *opts.Options
	queries []Query
	structs []Struct
	nested  []Nested
}

func populateNestedDataItems(
	options *opts.Options,
	queries []Query,
	structs []Struct,
	nested []Nested,
) ([]Nested, error) {
	// Build composite struct registry
	compositesBuilder := NestedCompositesDataBuilder{
		options: options,
		queries: queries,
		structs: structs,
	}
	err := compositesBuilder.buildCompositeStructRegistry()
	if err != nil {
		return nil, err
	}

	// Build data items and populate nested data items
	templateDataBuilder := NestedQueryTemplateDataBuilder{
		options: options,
		queries: queries,
		structs: structs,
		nested:  nested,
	}
	for i := range nested {
		nestedDataItem, err := templateDataBuilder.buildNestedDataItems(nested[i].Configs)
		if err != nil {
			return nil, err
		}

		nested[i].NestedDataItems = nestedDataItem
	}

	return nested, nil
}

func (b *NestedQueryTemplateDataBuilder) buildNestedDataItems(
	queryConfigs []*opts.NestedQueryConfig,
) ([]NestedQueryTemplateData, error) {
	var nestedDataItems []NestedQueryTemplateData

	// Track which struct_root has been generated to avoid duplicates
	generatedStructRoots := make(map[string]string) // struct_root -> first query that generated it

	for _, config := range queryConfigs {
		// Find the corresponding query
		var targetQuery *Query
		for _, q := range b.queries {
			if q.MethodName == config.Query || q.SourceName == config.Query {
				targetQuery = &q
				break
			}
		}

		if targetQuery == nil {
			debug.Warnf("Query '%s' not found for nested struct", config.Query)
			continue // Skip if query not found
		}

		structRoot := config.StructRoot
		if structRoot == "" {
			structRoot = config.Query + "Group"
		}

		// Check if this struct_root has already been generated
		if firstQuery, exists := generatedStructRoots[structRoot]; exists && firstQuery != config.Query {
			// Generate a wrapper function that reuses the existing Group function
			nestedDataItem, err := b.buildNestedWrapperData(targetQuery, config, firstQuery)
			if err != nil {
				return nil, fmt.Errorf("failed to generate wrapper function for query %s: %w", config.Query, err)
			}
			nestedDataItems = append(nestedDataItems, nestedDataItem)
		} else {
			// Generate the full function with struct definitions (first time)
			nestedDataItem, err := b.buildNestedData(targetQuery, config)
			if err != nil {
				return nil, fmt.Errorf("failed to generate nested function for query %s: %w", config.Query, err)
			}
			nestedDataItems = append(nestedDataItems, nestedDataItem)
			generatedStructRoots[structRoot] = config.Query
		}
	}

	return nestedDataItems, nil
}

// buildNestedWrapperData builds template data for the wrapper function
func (b *NestedQueryTemplateDataBuilder) buildNestedWrapperData(
	query *Query,
	config *opts.NestedQueryConfig,
	firstQueryName string,
) (NestedQueryTemplateData, error) {
	return NestedQueryTemplateData{
		FunctionName:    fmt.Sprintf("Group%s", query.MethodName), // Use the original function name
		Query:           query,
		RootStructName:  config.StructRoot,
		EmitJSONTags:    b.options.EmitJsonTags,
		EmitPointers:    b.options.EmitResultStructPointers,
		CastToQueryName: firstQueryName,
	}, nil
}

// buildNestedTemplateData builds the template data for nested function generation
// with control over whether composite struct definitions should be included
func (b *NestedQueryTemplateDataBuilder) buildNestedData(query *Query, config *opts.NestedQueryConfig) (NestedQueryTemplateData, error) {
	// Generate query name
	queryName := query.MethodName
	if queryName == "" {
		queryName = query.SourceName
	}

	// Generate struct names
	rootStruct := config.StructRoot
	if rootStruct == "" {
		rootStruct = fmt.Sprintf("%sGroup", queryName)
	}

	// Generate group function name
	functionName := fmt.Sprintf("Group%s", queryName)

	// Default root field to "ID" if not specified
	rootField := config.FieldGroupBy
	if rootField == "" {
		rootField = "ID"
	}

	// Add root struct definition - extract only non-nested and non-composite fields from root struct
	// Use the query's Row struct fields directly
	var structFields []Field
	if query.Ret.Struct != nil {
		structFields = query.Ret.Struct.Fields
	}

	assigner := &structInEmbedAssigner{b: b, query: query.MethodName}
	nestedStructData, err := b.buildNestedStructData(
		query.MethodName,
		&opts.NestedGroupConfig{
			Group:        config.Group,
			FieldGroupBy: rootField,
			StructIn:     rootStruct,
			StructOut:    rootStruct,
			IsComposite:  config.IsComposite,
		},
		nil,
		structFields,
		assigner,
	)
	if err != nil {
		return NestedQueryTemplateData{}, err
	}

	// // Validate interface compatibility for nested composites
	// if err := validateNestedInterfaceCompatibility(nestedStructData, rootStruct); err != nil {
	// 	return NestedQueryTemplateData{}, fmt.Errorf("validation failed for query %s: %w", queryName, err)
	// }

	return NestedQueryTemplateData{
		FunctionName:   functionName,
		Query:          query,
		RootStructName: config.StructRoot,
		RootStructData: nestedStructData,
		EmitJSONTags:   b.options.EmitJsonTags,
		EmitPointers:   b.options.EmitResultStructPointers,
	}, nil
}

// buildNestedStructData builds the data structure for a nested query configuration
func (b *NestedQueryTemplateDataBuilder) buildNestedStructData(
	queryName string,
	config *opts.NestedGroupConfig,
	parent *NestedStructData,
	structFields []Field,
	assigner *structInEmbedAssigner,
) (*NestedStructData, error) {
	query := b.getQueryByName(queryName)
	if query == nil {
		return nil, fmt.Errorf("query %s not found", queryName)
	}
	// Automatically extract fields from SQLC struct, excluding nested
	fields := b.getNonNestedStructFields(structFields, []*opts.NestedGroupConfig{config})

	// Get config of composite if we have reference to composite
	nestedConfigs := config.Group
	structIn := config.StructIn
	if config.GetIsComposite() {
		nestedConfigs = compositeStructRegistry[config.StructOut].Config.Group
		structIn = compositeStructRegistry[config.StructOut].Config.StructRootIn
	}

	// Determine if this struct should reuse an existing entity struct
	isEntity := b.shouldReuseEntityStruct(config.StructOut, config.StructIn, config)

	// Generate field name based on FieldOut if specified, otherwise use slice setting with proper pluralization
	fieldName := getFieldNameFromNestedConfig(config)

	// Generate field type
	fieldType := b.getFieldType(
		config.StructIn,
		config.StructOut,
		config.GetIsSlice(),
		config.GetIsPointer(),
		isEntity,
		config,
	)

	isRootConfig := parent == nil

	isAlreadyGenerated := b.IsCompositeStructAlreadyGenerated(config)
	willBeGeneratedInAnotherFile := b.IsCompositeStructWillBeGeneratedInAnotherFile(config)
	skipStructGeneration := !isRootConfig && (isAlreadyGenerated || willBeGeneratedInAnotherFile || parent.SkipStructGeneration)

	// Mark composite struct as already generated if it's the root of the current query or
	// parent is not skipped to render non-root compoiste in the same file as parent and it's free
	// (no one takes it to generate in another file)
	if isRootConfig || (!willBeGeneratedInAnotherFile && compositeStructRegistry[config.StructOut] != nil && !parent.SkipStructGeneration) {
		compositeStructRegistry[config.StructOut].IsStructAlreadyGenerated = true
	}

	// sqlc result struct field name: struct_in, or struct_in_<digits> for duplicate embeds
	var rowFieldName string
	if assigner != nil {
		rowFieldName = assigner.pickRowFieldName(config)
	} else {
		rowFieldName = config.StructIn
	}
	rowFieldType := fmt.Sprintf("%s.%s", b.options.OutputModelsPackage, config.StructIn)
	if rft := b.getRowFieldTypeByName(queryName, rowFieldName); rft != "" {
		rowFieldType = rft
	}

	// Create the NestedStructData
	result := &NestedStructData{
		StructIn:                structIn,
		StructOut:               config.StructOut,
		FieldGroupBy:            config.FieldGroupBy,
		IsSlice:                 config.GetIsSlice(),
		IsPointer:               config.GetIsPointer(),
		KeyType:                 determineKeyType(config.FieldGroupBy),
		FieldName:               fieldName,
		FieldType:               fieldType,
		RowFieldName:            rowFieldName,
		RowFieldType:            rowFieldType,
		FieldTags:               map[string]string{"json": JSONTagName(fieldName, b.options)},
		Fields:                  fields,
		IsEntityStruct:          isEntity,
		EntityType:              config.GetEntityType(), // Actual entity type when struct_in is an alias (e.g., "Image" for "Image_2")
		IsComposite:             config.GetIsComposite(),
		IsRowFieldExistsInQuery: b.isRowFieldExistsInQueryByName(queryName, rowFieldName),
		SkipStructGeneration:    skipStructGeneration,
		IsRoot:                  isRootConfig,
		Match:                   config.Match,
	}

	// Build nested structures recursively (do it after initialization to properly set SkipStructGeneration for children)
	// So we set SkipStructGeneration from root to leafs (to avoid cases when we skip struct generation for root, but not for children)
	nestedStructs, err := b.buildNestedStructsList(queryName, result, nestedConfigs, assigner)
	if err != nil {
		return nil, err
	}
	result.NestedStructs = nestedStructs

	// Resolve match rules to sqlc getter names using qualifying row field names (qualifyingRowFieldNamesForStructIn)
	b.resolveMatchResolvedForNode(queryName, result)

	// Match-referenced getters (e.g. Image_2 for company avatar while Image_2 is ignore:true) must appear on
	// every composite RowGetter interface that uses populate* calling r.GetImage_2(), not only the query root.
	// Root-only collection left RecruiterInfoCompositeRowGetter missing GetImage_2 on connect queries.
	if config.GetIsComposite() {
		result.MatchReferencedStructs = b.collectMatchReferencedStructs(queryName, result)
	}

	// Validate extracted fields only at the root level, since validation is recursive
	// and will check all nested structures
	if isRootConfig {
		if err := validateExtractedFields(fields, nestedStructs, query, b.structs, config.StructOut); err != nil {
			return nil, fmt.Errorf("validation failed for query %s: %w", queryName, err)
		}
	}

	// Set flags for duplicate structs (by StructOut) at each tree level from each node's perspective
	// This is used to determine when to add prefixes to map names
	updateDuplicatesFromNodePerspective(result, []*NestedStructData{})

	return result, nil
}

// updateDuplicatesFromNodePerspective recursively traverses the tree and for each node,
// checks if its StructOut appears multiple times at each ancestor level, looking UP the tree.
// Level 1 = immediate parent scope, Level 2 = grandparent scope, etc.
// This is used to determine when to add prefixes to map names.
func updateDuplicatesFromNodePerspective(current *NestedStructData, ancestors []*NestedStructData) {
	if current == nil {
		return
	}

	// For each nested struct (child), check duplicates from its perspective
	for _, nestedStruct := range current.NestedStructs {
		// Initialize the map in place if needed
		if nestedStruct.DuplicatedRelativeToParents == nil {
			nestedStruct.DuplicatedRelativeToParents = make(map[int]bool)
		}

		// Build the path: current node becomes an ancestor for the child
		childAncestors := append(ancestors, current)

		// Check duplicates at each ancestor level (distance from child going UP)
		for level := 1; level <= len(childAncestors); level++ {
			// Get the ancestor at this level (1-indexed: level 1 = immediate parent)
			ancestorIndex := len(childAncestors) - level
			ancestorNode := childAncestors[ancestorIndex]

			// Collect all nested structs by StructOut at this ancestor's scope
			structOutMap := make(map[string][]*NestedStructData)
			collectAllNestedStructsByStructOut(ancestorNode, structOutMap)

			// Check if this nested struct's StructOut appears multiple times at this level
			structOutStructs, structOutExists := structOutMap[nestedStruct.StructOut]
			nestedStruct.DuplicatedRelativeToParents[level] = structOutExists && len(structOutStructs) > 1
		}

		// Recursively process children of this nested struct
		updateDuplicatesFromNodePerspective(nestedStruct, childAncestors)
	}
}

// collectAllNestedStructsByStructOut collects all nested structs recursively from the given node.
// This creates a flattened view of all nested structs under the node, keyed by StructOut.
// Used for detecting duplicates when generating map names (since map names are based on StructOut).
func collectAllNestedStructsByStructOut(data *NestedStructData, structOutMap map[string][]*NestedStructData) {
	if data == nil {
		return
	}

	for _, nestedStruct := range data.NestedStructs {
		// Add this nested struct to the map keyed by StructOut
		structOutMap[nestedStruct.StructOut] = append(structOutMap[nestedStruct.StructOut], nestedStruct)

		// Recursively collect from deeper levels
		collectAllNestedStructsByStructOut(nestedStruct, structOutMap)
	}
}

// IsCompositeStructAlreadyGenerated checks if the composite struct was already generated
func (b *NestedQueryTemplateDataBuilder) IsCompositeStructAlreadyGenerated(config *opts.NestedGroupConfig) bool {
	return config.GetIsComposite() && compositeStructRegistry[config.StructOut].IsStructAlreadyGenerated
}

// IsCompositeStructWillBeGeneratedInAnotherFile checks if the composite struct will be generated in another _nested.sql file
func (b *NestedQueryTemplateDataBuilder) IsCompositeStructWillBeGeneratedInAnotherFile(config *opts.NestedGroupConfig) bool {
	// Check if the struct will be generated in another _nested.sql file
	willBeGeneratedInAnotherFile := false
	for _, nestedItem := range b.nested {
		for _, nestedConfig := range nestedItem.Configs {
			if nestedConfig.StructRoot == config.StructOut && config.GetIsComposite() {
				willBeGeneratedInAnotherFile = true
				break
			}
		}

		if willBeGeneratedInAnotherFile {
			break
		}
	}

	return willBeGeneratedInAnotherFile
}

// buildNestedStructsList builds a list of nested struct data from a group configuration
func (b *NestedQueryTemplateDataBuilder) buildNestedStructsList(
	queryName string,
	parent *NestedStructData,
	group []*opts.NestedGroupConfig,
	assigner *structInEmbedAssigner,
) ([]*NestedStructData, error) {
	var nestedStructs []*NestedStructData

	for _, nested := range group {
		// Skip ignored nested structs - they should not be processed at all
		if nested.GetIsIgnore() {
			continue
		}

		// Find the struct fields for the given StructIn
		var structFields []Field
		for _, s := range b.structs {
			if s.Name == nested.StructIn {
				structFields = s.Fields
				break
			}
		}

		nestedData, err := b.buildNestedStructData(queryName, nested, parent, structFields, assigner)
		if err != nil {
			return nil, err
		}
		if nestedData != nil {
			nestedStructs = append(nestedStructs, nestedData)
		}
	}

	return nestedStructs, nil
}

// extractFields extracts fields from a struct, excluding given fields
func (b *NestedQueryTemplateDataBuilder) extractFields(
	allFields []Field,
	fieldsToExclued []string,
	rowFieldPrefix string,
) []Field {
	var fields []Field

	// Convert SQLC fields to nested field configs, excluding configured nested fields and composite fields
	for _, field := range allFields {
		// Skip fields that are configured as nested
		skipField := false
		for _, fieldToExclude := range fieldsToExclued {
			if fieldToExclude == field.Name {
				skipField = true
				break
			}
		}
		if skipField {
			continue
		}

		// For root fields (when prefix is empty), just use the field name
		// For nested fields, use the prefix.field format
		rowField := field.Name
		if rowFieldPrefix != "" {
			rowField = strings.Join([]string{rowFieldPrefix, field.Name}, ".")
		}

		nestedField := Field{
			Name:   field.Name,
			DBName: rowField,
			Type:   field.Type,
			Tags:   field.Tags,
		}
		fields = append(fields, nestedField)
	}

	return fields
}

// shouldReuseEntityStruct determines if we should reuse an existing entity struct
// A struct should be reused if:
// 1. It exists in the schema structs (generated from SQL)
// 2. It doesn't have nested configurations (is a leaf node)
// 3. It's not marked as composite (composite structs reuse previously generated nested structures)
//
// Note: This is different from composite structs which reuse previously generated nested structures
func (b *NestedQueryTemplateDataBuilder) shouldReuseEntityStruct(structOut, structIn string, config *opts.NestedGroupConfig) bool {
	// If marked as composite, don't treat as entity struct
	if config.IsComposite != nil && *config.IsComposite {
		return false
	}

	// Determine which struct name to check in schema
	// If EntityType is specified (e.g., "Image" for an "Image_2" alias), use that instead of structIn
	structToCheck := structIn
	if config.GetEntityType() != "" {
		structToCheck = config.GetEntityType()
	}

	// Check if the struct exists in the schema
	if !b.structExistsInSchema(structToCheck) {
		return false
	}

	// If this struct has nested configurations, we need to use the generated struct
	// because it needs to contain the nested fields (like Reviews in Book)
	if len(config.Group) > 0 {
		return false
	}

	// If structIn (or its EntityType) exists in schema and has no nesting, reuse the entity struct
	return true
}

// structExistsInSchema checks if a struct name exists in the schema structs
func (b *NestedQueryTemplateDataBuilder) structExistsInSchema(structName string) bool {
	for _, s := range b.structs {
		if s.Name == structName {
			return true
		}
	}
	return false
}

// getCurrentStructFields extracts nested fields from the struct fields
func (b *NestedQueryTemplateDataBuilder) getNonNestedStructFields(fields []Field, groupConfig []*opts.NestedGroupConfig) []Field {
	excluded := getNestedFields(groupConfig)
	excluded = appendDuplicateSqlcEmbedExclusions(fields, excluded)
	return b.extractFields(
		fields,
		excluded,
		"",
	)
}

// determineKeyType determines the key type for the map based on the field
func determineKeyType(field string) string {
	// For now, default to UUID for all ID fields
	// This could be enhanced to analyze the actual field type from SQLC catalog
	return "pgtype.UUID"
}

// getFieldNameFromNestedConfig determines the field name for a nested configuration
// Uses field_out if specified, otherwise generates from struct_in based on slice configuration
func getFieldNameFromNestedConfig(nested *opts.NestedGroupConfig) string {
	fieldName := nested.FieldOut
	if fieldName == "" {
		// Use default field naming logic
		if nested.IsSlice == nil || *nested.IsSlice {
			fieldName = PluralizeCasePreserving(nested.StructIn)
		} else {
			fieldName = SingularizeCasePreserving(nested.StructIn)
		}
	}
	return fieldName
}

// getFieldType gets the field type for a nested struct based on the configuration
func (b *NestedQueryTemplateDataBuilder) getFieldType(
	structIn, structOut string,
	isSlice, isPointer, isEntityStruct bool,
	nestedConfig *opts.NestedGroupConfig, // Optional, used for entity struct detection when available
) string {
	// Determine if we should use entity prefix for the type or reuse composite struct
	useEntityPrefix := isEntityStruct
	if !useEntityPrefix && nestedConfig != nil {
		useEntityPrefix = b.shouldReuseEntityStruct(structOut, structIn, nestedConfig)
	}

	// Determine the entity type name to use
	// If EntityType is specified (e.g., "Image" for an "Image_2" alias), use that
	entityTypeName := structIn
	if nestedConfig != nil && nestedConfig.GetEntityType() != "" {
		entityTypeName = nestedConfig.GetEntityType()
	}

	structType := structOut
	if useEntityPrefix {
		structType = fmt.Sprintf("entity.%s", entityTypeName)
	}

	var fieldType string
	if isSlice {
		if isPointer {
			fieldType = fmt.Sprintf("[]*%s", structType)
		} else {
			fieldType = fmt.Sprintf("[]%s", structType)
		}
	} else {
		if isPointer {
			fieldType = fmt.Sprintf("*%s", structType)
		} else {
			fieldType = structType
		}
	}

	return fieldType
}

// isQueryRootComposite checks if any nested config's StructRoot matches the current composite struct
func (b *NestedQueryTemplateDataBuilder) isQueryRootComposite(config *opts.NestedGroupConfig) bool {
	// Only check for composite structs
	if !config.GetIsComposite() {
		return false
	}

	// Iterate through all nested configs to check if any StructRoot matches the current composite struct
	for _, nestedItem := range b.nested {
		for _, nestedConfig := range nestedItem.Configs {
			if nestedConfig.StructRoot == config.StructOut {
				return true
			}
		}
	}

	return false
}
