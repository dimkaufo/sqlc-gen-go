package golang

import (
	"fmt"
	"strings"

	"github.com/sqlc-dev/sqlc-gen-go/internal/debug"
)

// Validation Overview:
// This file contains validation functions for nested query configurations.
//
// Key validations:
// 1. Interface compatibility: Ensures parent interfaces include all methods required by nested composites
// 2. Field existence: Validates that extracted fields exist in the model struct or nested config
// 3. Composite field mapping: Ensures all fields in a composite struct exist in the corresponding
//    entity struct from sqlc.embed. This prevents runtime errors when populating composites from entities.

// RowGetterInterfaceMethod represents a single method in a RowGetter interface
type RowGetterInterfaceMethod struct {
	MethodName string // e.g., "GetUser"
	ReturnType string // e.g., "entity.User"
	StructIn   string // e.g., "User" - the struct_in this method is for
}

// validateNestedInterfaceCompatibility validates that parent interfaces include all methods
// required by their nested composites' interfaces
func validateNestedInterfaceCompatibility(rootStruct *NestedStructData, rootStructName string) error {
	// Collect all methods required by the root interface
	rootMethods := collectRequiredMethods(rootStruct)

	debug.Printf("Validating interface compatibility for root struct: %s", rootStructName)
	debug.Printf("Root interface requires %d methods", len(rootMethods))

	// Recursively validate nested composites
	return validateNestedCompositeInterfaces(rootStruct, rootMethods, rootStructName)
}

// validateNestedCompositeInterfaces recursively validates that when a composite calls
// a nested composite's populate function, it has all required interface methods
func validateNestedCompositeInterfaces(
	parentStruct *NestedStructData,
	parentMethods []RowGetterInterfaceMethod,
	rootStructName string,
) error {
	for _, nestedStruct := range parentStruct.NestedStructs {
		// Only validate composites that will call populate functions
		if nestedStruct.IsComposite {
			debug.Printf("Validating composite: %s (nested in %s)", nestedStruct.StructOut, parentStruct.StructOut)

			// Collect methods required by this nested composite
			// For nested composites, we need to collect ALL methods they need,
			// regardless of whether they exist in the parent query
			nestedMethods := collectCompositeRequiredMethods(nestedStruct)

			debug.Printf("Nested composite %s requires %d methods", nestedStruct.StructOut, len(nestedMethods))

			// Validate that parent interface has all methods required by nested composite
			missingMethods := findMissingMethods(parentMethods, nestedMethods)

			if len(missingMethods) > 0 {
				return fmt.Errorf(
					"interface validation failed for composite '%s' in root struct '%s': "+
						"Parent interface '%sRowGetter' is missing methods required by nested composite '%sRowGetter': "+
						"Missing methods: %s. "+
						"To fix this, ensure that your SQL query for '%s' includes all necessary fields "+
						"that the nested composite '%s' requires. The query must select (sqlc.embed) all entities "+
						"needed by the nested composite structure",
					nestedStruct.StructOut,
					rootStructName,
					parentStruct.StructOut,
					nestedStruct.StructOut,
					formatMissingMethods(missingMethods),
					rootStructName,
					nestedStruct.StructOut,
				)
			}

			debug.Printf("✓ Composite %s is compatible", nestedStruct.StructOut)

			// Recursively validate nested composites of this composite
			// They inherit the parent's available methods
			if err := validateNestedCompositeInterfaces(nestedStruct, parentMethods, rootStructName); err != nil {
				return err
			}
		}

		// Recursively check non-composite nested structs as well
		if len(nestedStruct.NestedStructs) > 0 {
			if err := validateNestedCompositeInterfaces(nestedStruct, parentMethods, rootStructName); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateExtractedFields(fields []Field, nestedStructs []*NestedStructData, query *Query, structs []Struct, structOut string) error {
	// First, validate the root struct's fields if it's a composite
	// Check if this struct is defined as a composite in the registry
	if compositeData, exists := compositeStructRegistry[structOut]; exists {
		// This is a composite struct - validate its fields exist in the entity
		if err := validateCompositeFieldsAgainstEntity(
			structs,
			structOut,
			fields,
			compositeData.Config.StructRootIn,
			structOut, // parentStructName is the same as compositeStructName for root
			"root",
		); err != nil {
			return err
		}
	}

	// Then validate nested composite structs
	if err := validateCompositeFieldsExistInEntity(nestedStructs, query, structs, structOut); err != nil {
		return fmt.Errorf("validation failed for struct %s: %w", structOut, err)
	}

	return nil
}

// validateCompositeFieldsExistInEntity validates that all fields in composite structs
// exist in their corresponding entity struct from sqlc.embed
// Returns an error if any composite field doesn't exist in its entity struct
func validateCompositeFieldsExistInEntity(nestedStructs []*NestedStructData, query *Query, structs []Struct, parentStructName string) error {
	for _, nestedStruct := range nestedStructs {
		// Only validate composites that will be populated from an entity (sqlc.embed)
		if nestedStruct.IsComposite && nestedStruct.IsRowFieldExistsInQuery {
			// Use the shared validation helper
			if err := validateCompositeFieldsAgainstEntity(
				structs,
				nestedStruct.StructOut,
				nestedStruct.Fields,
				nestedStruct.StructIn,
				parentStructName,
				"nested",
			); err != nil {
				return err
			}
		}

		// Recursively validate nested structures
		if len(nestedStruct.NestedStructs) > 0 {
			if err := validateCompositeFieldsExistInEntity(nestedStruct.NestedStructs, query, structs, parentStructName); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateCompositeFieldsAgainstEntity validates that all fields in a composite struct
// exist in the corresponding entity struct. Returns an error with helpful context if validation fails.
// contextType should be either "root" or "nested"
func validateCompositeFieldsAgainstEntity(
	structs []Struct,
	compositeStructName string,
	compositeFields []Field,
	entityStructName string,
	parentStructName string, // The parent/root struct name for context in error messages
	contextType string, // "root" or "nested"
) error {
	entityStruct := getStructByName(structs, entityStructName)
	if entityStruct == nil {
		// Entity struct not found - this could be a composite referencing another composite
		// Skip validation in this case as it's handled by interface validation
		debug.Printf("Skipping validation for composite '%s' - entity struct '%s' not found", compositeStructName, entityStructName)
		return nil
	}

	debug.Printf("Validating %s composite '%s' fields against entity '%s'",
		contextType, compositeStructName, entityStructName)

	// Check that all fields in the composite exist in the entity struct
	var missingFields []string
	for _, compositeField := range compositeFields {
		if !fieldExistsInEntityFields(entityStruct.Fields, compositeField.Name) {
			missingFields = append(missingFields, compositeField.Name)
		}
	}

	// If there are missing fields, return an error
	if len(missingFields) > 0 {
		formattedFields := formatAvailableFields(entityStruct.Fields)
		if contextType == "root" {
			return fmt.Errorf(
				"root composite struct '%s' references fields %v that don't exist in entity struct '%s' from sqlc.embed. "+
					"Either remove these fields from the entity struct fields, or these fields should be defined as nested relationships in your nested config. "+
					"Available fields in entity: %s",
				compositeStructName,
				missingFields,
				entityStructName,
				formattedFields,
			)
		} else {
			return fmt.Errorf(
				"composite struct '%s' in '%s' references fields %v in nested config, "+
					"but these fields don't exist in entity struct '%s' from sqlc.embed. "+
					"Either remove these fields from the nested config for '%s', "+
					"or ensure the entity struct includes these fields. "+
					"Available fields in entity: %s",
				compositeStructName,
				parentStructName,
				missingFields,
				entityStructName,
				compositeStructName,
				formattedFields,
			)
		}
	}

	return nil
}

// fieldExistsInEntityFields checks if a field name exists in the entity fields
func fieldExistsInEntityFields(entityFields []Field, fieldName string) bool {
	for _, field := range entityFields {
		if field.Name == fieldName {
			return true
		}
	}
	return false
}

// formatAvailableFields formats the available fields for error messages
func formatAvailableFields(fields []Field) string {
	if len(fields) == 0 {
		return "none"
	}

	fieldNames := make([]string, len(fields))
	for i, field := range fields {
		fieldNames[i] = field.Name
	}
	return "[" + strings.Join(fieldNames, ", ") + "]"
}

// collectRequiredMethods collects all the getter methods required by a struct's RowGetter interface
// This collects only methods that exist in the query (what the row struct actually HAS)
func collectRequiredMethods(structData *NestedStructData) []RowGetterInterfaceMethod {
	var methods []RowGetterInterfaceMethod
	seen := make(map[string]bool) // Prevent duplicates

	// Only collect methods that actually exist in the query
	collectMethodsRecursive(structData, &methods, seen, true)

	return methods
}

// collectCompositeRequiredMethods collects ALL methods a composite needs in its RowGetter interface
// This includes all nested fields, regardless of whether they exist in a parent query
func collectCompositeRequiredMethods(structData *NestedStructData) []RowGetterInterfaceMethod {
	var methods []RowGetterInterfaceMethod
	seen := make(map[string]bool) // Prevent duplicates

	// Collect all methods the composite needs (not just what's available)
	collectMethodsRecursive(structData, &methods, seen, false)

	return methods
}

// collectMethodsRecursive recursively collects getter methods from nested structures
// When isCollectingAvailableMethods=true, only collects methods that exist in the query (parent's available methods)
// When isCollectingAvailableMethods=false, collects all methods needed by composite (child's required methods)
func collectMethodsRecursive(
	structData *NestedStructData,
	methods *[]RowGetterInterfaceMethod,
	seen map[string]bool,
	isCollectingAvailableMethods bool,
) {
	for _, nestedStruct := range structData.NestedStructs {
		// Determine if we should include this method
		var shouldIncludeMethod bool

		if isCollectingAvailableMethods {
			// When collecting available methods (what the parent HAS), only include if it exists in the query
			shouldIncludeMethod = nestedStruct.IsRowFieldExistsInQuery
		} else {
			// When collecting required methods (what a composite NEEDS), include all nested fields
			// This is for when we're checking a composite's requirements
			shouldIncludeMethod = true
		}

		if shouldIncludeMethod {
			methodName := fmt.Sprintf("Get%s", nestedStruct.StructIn)

			// Only add if not already seen
			if !seen[methodName] {
				*methods = append(*methods, RowGetterInterfaceMethod{
					MethodName: methodName,
					ReturnType: nestedStruct.RowFieldType,
					StructIn:   nestedStruct.StructIn,
				})
				seen[methodName] = true
			}
		}

		// Recursively collect from nested structures
		if len(nestedStruct.NestedStructs) > 0 {
			collectMethodsRecursive(nestedStruct, methods, seen, isCollectingAvailableMethods)
		}
	}
}

// findMissingMethods finds methods that are in 'required' but not in 'available'
func findMissingMethods(
	available []RowGetterInterfaceMethod,
	required []RowGetterInterfaceMethod,
) []RowGetterInterfaceMethod {
	var missing []RowGetterInterfaceMethod

	// Create a set of available method names for quick lookup
	availableSet := make(map[string]bool)
	for _, method := range available {
		availableSet[method.MethodName] = true
	}

	// Check each required method
	for _, requiredMethod := range required {
		if !availableSet[requiredMethod.MethodName] {
			missing = append(missing, requiredMethod)
		}
	}

	return missing
}

// formatMissingMethods formats missing methods for error messages
func formatMissingMethods(methods []RowGetterInterfaceMethod) string {
	var parts []string
	for _, method := range methods {
		parts = append(parts, fmt.Sprintf("%s() %s", method.MethodName, method.ReturnType))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
