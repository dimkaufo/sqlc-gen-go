package golang

import "github.com/sqlc-dev/sqlc-gen-go/internal/opts"

// populateNestedConfigWithDefaultValues populates the nested config with default values
// (StructOut, IsComposite, FieldGroupBy, IsSlice, IsPointer and so on)
func populateNestedConfigWithDefaultValues(options *opts.Options) error {
	if options.Nested != nil {
		for _, config := range options.Nested.Queries {
			for _, group := range config.Group {
				populateNestedConfigItemWithDefaultValues(group)
			}
		}

		for _, config := range options.Nested.Composites {
			for _, group := range config.Group {
				populateNestedConfigItemWithDefaultValues(group)
			}
		}
	}
	return nil
}

func populateNestedConfigItemWithDefaultValues(config *opts.NestedGroupConfig) error {
	// Default the struct name StructIn if not specified
	structOut := config.StructOut
	if structOut == "" {
		structOut = config.StructIn
	}
	config.StructOut = structOut

	// Default composite to false if not specified
	// Composite structs are nested structures that were already generated in another query
	// and should be reused instead of creating new struct definitions and helper methods
	isComposite := false
	if config.IsComposite != nil {
		isComposite = *config.IsComposite
	}
	config.IsComposite = &isComposite

	// Default field to "ID" if not specified
	fieldGroupBy := config.FieldGroupBy
	if fieldGroupBy == "" {
		fieldGroupBy = "ID"
	}
	config.FieldGroupBy = fieldGroupBy

	// Default slice to true if not specified
	isSlice := true
	if config.IsSlice != nil {
		isSlice = *config.IsSlice
	}
	config.IsSlice = &isSlice

	// Default pointer to true if not specified
	isPointer := true
	if config.IsPointer != nil {
		isPointer = *config.IsPointer
	}
	config.IsPointer = &isPointer

	// Default match
	if config.Match != nil {
		for _, match := range config.Match {
			if match.FromStruct == nil {
				match.FromStruct = &config.StructIn
			}
			if match.FromField == nil {
				match.FromField = &config.FieldGroupBy
			}
			if match.ToField == nil {
				match.ToField = &config.FieldGroupBy
			}
		}
	}

	return nil
}