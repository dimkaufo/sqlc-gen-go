package golang

import (
	"fmt"

	"github.com/sqlc-dev/sqlc-gen-go/internal/opts"
)

// compositeStructRegistry is global registry to track composite structs in composites configuration
var compositeStructRegistry = make(map[string]*CompositeStructData)

// CompositeStructData holds information about original composite structs config and computed data
// (for ex. nested fields and fields to exclude from parent structs)
type CompositeStructData struct {
	// Config of the composite struct (All the query configuration that uses this composite,
	// for ex. name, nested composites, find conditions, etc.)
	Config *opts.NestedCompositeConfig

	// TODO field_out too?
	// Nested fields that this composite struct has (e.g. Image, User) - we exclude these fields from parent structs
	DirectNestedFields []string

	// Nested composites that this composite struct has (e.g. User -> UserComposite)
	NestedFieldToCompositeNameMap map[string]string

	// All entity fields that should be excluded from parent structs when this composite is used
	// Includes both field_out names (e.g., "Avatar") and struct_in names (e.g., "Image")
	// to handle field mappings like Image -> Avatar
	EntityFieldsToExclude []string

	// Whether the struct is already generated
	IsStructAlreadyGenerated bool

	// RowGetterFieldNamesForPopulateRow lists sqlc row field names (StructIn / base embed names)
	// that populate{Composite}(..., row) and {Composite}RowGetter require in the canonical query
	// without duplicate embeds — e.g. RecruitersCompanyComposite needs Company and Image for
	// GetCompany()/GetImage(). Parent queries with duplicate Image embeds may resolve the company
	// logo to Image_2 in the nested tree while the shared RowGetter constraint still uses GetImage();
	// collectMatchReferencedStructs merges these canonical names into the parent interface.
	RowGetterFieldNamesForPopulateRow []string
}

type NestedCompositesDataBuilder struct {
	options *opts.Options
	queries []Query
	structs []Struct
}

// buildCompositeStructRegistry analyzes all configurations to pre-populate the composite struct registry
// This allows us to know what fields composite structs have before generating parent structs
func (b *NestedCompositesDataBuilder) buildCompositeStructRegistry() error {
	// Get composites config (if any)
	var compositesConfigItems []*opts.NestedCompositeConfig
	if b.options.Nested != nil && b.options.Nested.Composites != nil {
		compositesConfigItems = b.options.Nested.Composites
	}

	// Register all composite structs from nested.composites configuration
	if len(compositesConfigItems) > 0 {
		// Pass 1: Register all composite structs from nested.composites configuration and prepare computed data
		for _, composite := range compositesConfigItems {
			var nestedFields []string
			nestedFieldToCompositeNameMap := make(map[string]string)
			for _, childNestedItem := range composite.Group {
				// Always add StructIn to nestedFields - even ignored items should be excluded from struct fields
				// The ignore flag only prevents nested struct generation, not field exclusion
				nestedFields = append(nestedFields, childNestedItem.StructIn)

				// Only track composite mappings for non-ignored items
				if !childNestedItem.GetIsIgnore() {
					// Analyze what nested composites this composite struct will have
					if childNestedItem.IsComposite != nil && *childNestedItem.IsComposite {
						nestedFieldToCompositeNameMap[childNestedItem.StructIn] = childNestedItem.StructOut
					}
				}
			}

			// Register this composite struct for future references
			b.registerCompositeStructData(composite, nestedFields, nestedFieldToCompositeNameMap)
		}

		// Pass 2: Resolve and populate entity fields to exclude from parent structs (rows structs)
		for _, composite := range compositesConfigItems {
			fields, err := b.resolveAllTreeCompositeFields(composite.Name)
			if err != nil {
				return err
			}

			compositeStruct, exists := compositeStructRegistry[composite.Name]
			if !exists {
				return fmt.Errorf("composite struct '%s' not found in registry when resolving entity fields to exclude", composite.Name)
			}
			compositeStruct.EntityFieldsToExclude = fields
		}

		// Pass 3: Canonical row field names (struct_in order) for each composite's RowGetter used by
		// populate*(..., row). Independent of per-query duplicate-embed resolution (Image vs Image_2).
		for _, composite := range compositesConfigItems {
			compositeStruct, exists := compositeStructRegistry[composite.Name]
			if !exists {
				continue
			}
			compositeStruct.RowGetterFieldNamesForPopulateRow = collectRowGetterFieldNamesFromRegistryConfig(compositeStruct)
		}
	}

	return nil
}

// collectRowGetterFieldNamesFromRegistryConfig walks nested.composites group DFS (same shape as
// generateRowGetterInterfaceMethodsRecursive) and collects struct_in names for entity row getters.
// Composite children contribute their struct_in before recursing into the referenced composite.
func collectRowGetterFieldNamesFromRegistryConfig(reg *CompositeStructData) []string {
	if reg == nil || reg.Config == nil {
		return nil
	}
	var out []string
	var walk func([]*opts.NestedGroupConfig)
	walk = func(groups []*opts.NestedGroupConfig) {
		for _, g := range groups {
			if g.GetIsIgnore() {
				continue
			}
			if g.GetIsComposite() {
				out = append(out, g.StructIn)
				if child, ok := compositeStructRegistry[g.StructOut]; ok && child.Config != nil {
					walk(child.Config.Group)
				}
				continue
			}
			out = append(out, g.StructIn)
		}
	}
	walk(reg.Config.Group)
	return out
}

// registerCompositeStructData registers a composite struct data in the composites registry
func (b *NestedCompositesDataBuilder) registerCompositeStructData(
	config *opts.NestedCompositeConfig,
	nestedFields []string,
	nestedFieldToCompositeNameMap map[string]string,
) {
	compositeStructRegistry[config.Name] = &CompositeStructData{
		Config:                        config,
		DirectNestedFields:            nestedFields,
		NestedFieldToCompositeNameMap: nestedFieldToCompositeNameMap,

		// Will be populated afterwards when registry is complete
		EntityFieldsToExclude: []string{},
	}
}

// collectCompositeStructFields recursively collects entity fields from nested composite structs
// This identifies entity fields that would be duplicated between parent and nested composite structs
func (b *NestedCompositesDataBuilder) resolveAllTreeCompositeFields(compositeName string) ([]string, error) {
	var entityFields []string

	// Add direct nested fields from the composite struct
	compositeInfo, exists := compositeStructRegistry[compositeName]
	if !exists {
		return nil, fmt.Errorf("composite struct '%s' not found in registry while checking direct nested fields", compositeName)
	}
	entityFields = append(entityFields, compositeInfo.DirectNestedFields...)

	for _, nestedCompositeName := range compositeInfo.NestedFieldToCompositeNameMap {
		allNestedCompositeFields, err := b.resolveAllTreeCompositeFields(nestedCompositeName)
		if err != nil {
			return nil, err
		}
		entityFields = append(entityFields, allNestedCompositeFields...)
	}

	return entityFields, nil
}

// getNestedFields gets all nested and composite fields from the nested query config and composites registry
// Note: Ignored fields are still included in the result because they should be excluded from the struct fields,
// even though they don't generate nested relationships
func getNestedFields(config []*opts.NestedGroupConfig) []string {
	var fields []string
	for _, nested := range config {
		// Check if this is a composite struct that should reference existing data
		if nested.IsComposite != nil && *nested.IsComposite {
			// Try to get the composite struct data from registry
			compositeData, exists := compositeStructRegistry[nested.StructOut]
			if exists && len(compositeData.DirectNestedFields) > 0 {
				fields = append(fields, compositeData.EntityFieldsToExclude...)
			}
		} else {
			// Regular nested struct - always include in fields to exclude
			// Even ignored fields should be excluded from struct fields
			fields = append(fields, nested.StructIn)
			if len(nested.Group) > 0 {
				fields = append(fields, getNestedFields(nested.Group)...)
			}
		}
	}
	return fields
}