package opts

import (
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

// NestedConfig represents the configuration for nested queries with predefined composites
type NestedConfig struct {
	Composites []*NestedCompositeConfig `json:"composites,omitempty" yaml:"composites"` // Predefined composites used in queries groups
	Queries    []*NestedQueryConfig     `json:"queries,omitempty" yaml:"queries"`       // Queries to group
}

// NestedGroupConfig represents the configuration for nested grouping
type NestedGroupConfig struct {
	FieldGroupBy string               `json:"field_group_by,omitempty" yaml:"field_group_by"` // Field to group by (optional, defaults to "ID")
	FieldOut     string               `json:"field_out,omitempty" yaml:"field_out"`           // Field name in the resulted struct (optional, defaults to pluralized StructIn)
	StructIn     string               `json:"struct_in" yaml:"struct_in"`                     // Input struct name (required)
	StructOut    string               `json:"struct_out,omitempty" yaml:"struct_out"`         // Output struct name for grouping (optional, defaults to StructIn)
	IsSlice      *bool                `json:"slice,omitempty" yaml:"slice"`                   // Whether to use slice (default: true)
	IsPointer    *bool                `json:"pointer,omitempty" yaml:"pointer"`               // Whether to use pointers (default: true)
	IsComposite  *bool                `json:"composite,omitempty" yaml:"composite"`           // Whether to reuse existing composite struct that was generated in another query's struct_root (default: false)
	Group        []*NestedGroupConfig `json:"group,omitempty" yaml:"group"`                   // Nested group configuration (recursive)
	Match        []*NestedMatchConfig `json:"match,omitempty" yaml:"match"`                   // Match configuration (recursive)
}

func (n *NestedGroupConfig) GetIsSlice() bool {
	return n.IsSlice == nil || *n.IsSlice
}

func (n *NestedGroupConfig) GetIsPointer() bool {
	return n.IsPointer == nil || *n.IsPointer
}

func (n *NestedGroupConfig) GetIsComposite() bool {
	return n.IsComposite == nil || *n.IsComposite
}

// NestedMatchConfig represents the configuration for matching a struct in a nested group
type NestedMatchConfig struct {
	FromStruct *string `json:"from_struct" yaml:"from_struct"` // Struct to match from
	FromField  *string `json:"from_field" yaml:"from_field"`   // Field name in the from struct
	ToStruct   string  `json:"to_struct" yaml:"to_struct"`     // Struct to match to
	ToField    *string `json:"to_field" yaml:"to_field"`       // Field name to match by
}

// NestedCompositeConfig represents the configuration for a composite struct
type NestedCompositeConfig struct {
	Name         string               `json:"name" yaml:"name"`                     // Input struct name (required)
	StructRootIn string               `json:"struct_root_in" yaml:"struct_root_in"` // Input struct when using composite as root for query(required)
	Group        []*NestedGroupConfig `json:"group,omitempty" yaml:"group"`         // Nested group configuration (recursive)
}

// NestedQueryConfig represents nested grouping configuration for a specific query
type NestedQueryConfig struct {
	Query        string               `json:"query" yaml:"query"`                             // Query name to group
	FieldGroupBy string               `json:"field_group_by,omitempty" yaml:"field_group_by"` // Root field to group by (optional, defaults to "ID")
	StructRoot   string               `json:"struct_root" yaml:"struct_root"`                 // Root struct name
	Group        []*NestedGroupConfig `json:"group" yaml:"group"`                             // Nested group configuration
	IsComposite  *bool                `json:"composite,omitempty" yaml:"composite"`           // Is composite struct
}

type Options struct {
	EmitInterface               bool              `json:"emit_interface" yaml:"emit_interface"`
	EmitJsonTags                bool              `json:"emit_json_tags" yaml:"emit_json_tags"`
	JsonTagsIdUppercase         bool              `json:"json_tags_id_uppercase" yaml:"json_tags_id_uppercase"`
	EmitDbTags                  bool              `json:"emit_db_tags" yaml:"emit_db_tags"`
	EmitPreparedQueries         bool              `json:"emit_prepared_queries" yaml:"emit_prepared_queries"`
	EmitExactTableNames         bool              `json:"emit_exact_table_names,omitempty" yaml:"emit_exact_table_names"`
	EmitEmptySlices             bool              `json:"emit_empty_slices,omitempty" yaml:"emit_empty_slices"`
	EmitExportedQueries         bool              `json:"emit_exported_queries" yaml:"emit_exported_queries"`
	EmitResultStructPointers    bool              `json:"emit_result_struct_pointers" yaml:"emit_result_struct_pointers"`
	EmitParamsStructPointers    bool              `json:"emit_params_struct_pointers" yaml:"emit_params_struct_pointers"`
	EmitMethodsWithDbArgument   bool              `json:"emit_methods_with_db_argument,omitempty" yaml:"emit_methods_with_db_argument"`
	EmitPointersForNullTypes    bool              `json:"emit_pointers_for_null_types" yaml:"emit_pointers_for_null_types"`
	EmitEnumValidMethod         bool              `json:"emit_enum_valid_method,omitempty" yaml:"emit_enum_valid_method"`
	EmitAllEnumValues           bool              `json:"emit_all_enum_values,omitempty" yaml:"emit_all_enum_values"`
	EmitSqlAsComment            bool              `json:"emit_sql_as_comment,omitempty" yaml:"emit_sql_as_comment"`
	JsonTagsCaseStyle           string            `json:"json_tags_case_style,omitempty" yaml:"json_tags_case_style"`
	Package                     string            `json:"package" yaml:"package"`
	Out                         string            `json:"out" yaml:"out"`
	Overrides                   []Override        `json:"overrides,omitempty" yaml:"overrides"`
	Rename                      map[string]string `json:"rename,omitempty" yaml:"rename"`
	SqlPackage                  string            `json:"sql_package" yaml:"sql_package"`
	SqlDriver                   string            `json:"sql_driver" yaml:"sql_driver"`
	OutputBatchFileName         string            `json:"output_batch_file_name,omitempty" yaml:"output_batch_file_name"`
	OutputDbFileName            string            `json:"output_db_file_name,omitempty" yaml:"output_db_file_name"`
	OutputModelsFileName        string            `json:"output_models_file_name,omitempty" yaml:"output_models_file_name"`
	OutputModelsPackage         string            `json:"output_models_package,omitempty" yaml:"output_models_package"`
	ModelsPackageImportPath     string            `json:"models_package_import_path,omitempty" yaml:"models_package_import_path"`
	OutputQuerierFileName       string            `json:"output_querier_file_name,omitempty" yaml:"output_querier_file_name"`
	OutputCopyfromFileName      string            `json:"output_copyfrom_file_name,omitempty" yaml:"output_copyfrom_file_name"`
	OutputQueryFilesDirectory   string            `json:"output_query_files_directory,omitempty" yaml:"output_query_files_directory"`
	OutputNestedUtilsFileName   string            `json:"output_nested_utils_file_name,omitempty" yaml:"output_nested_utils_file_name"`
	OutputFilesSuffix           string            `json:"output_files_suffix,omitempty" yaml:"output_files_suffix"`
	InflectionExcludeTableNames []string          `json:"inflection_exclude_table_names,omitempty" yaml:"inflection_exclude_table_names"`
	QueryParameterLimit         *int32            `json:"query_parameter_limit,omitempty" yaml:"query_parameter_limit"`
	OmitSqlcVersion             bool              `json:"omit_sqlc_version,omitempty" yaml:"omit_sqlc_version"`
	OmitUnusedStructs           bool              `json:"omit_unused_structs,omitempty" yaml:"omit_unused_structs"`
	BuildTags                   string            `json:"build_tags,omitempty" yaml:"build_tags"`
	Initialisms                 []string          `json:"initialisms,omitempty" yaml:"initialisms"`
	Nested                      *NestedConfig     `json:"nested,omitempty" yaml:"nested"`

	InitialismsMap map[string]struct{} `json:"-" yaml:"-"`
}

type GlobalOptions struct {
	Overrides []Override        `json:"overrides,omitempty" yaml:"overrides"`
	Rename    map[string]string `json:"rename,omitempty" yaml:"rename"`
}

func Parse(req *plugin.GenerateRequest) (*Options, error) {
	options, err := parseOpts(req)
	if err != nil {
		return nil, err
	}
	global, err := parseGlobalOpts(req)
	if err != nil {
		return nil, err
	}
	if len(global.Overrides) > 0 {
		options.Overrides = append(global.Overrides, options.Overrides...)
	}
	if len(global.Rename) > 0 {
		if options.Rename == nil {
			options.Rename = map[string]string{}
		}
		maps.Copy(options.Rename, global.Rename)
	}
	return options, nil
}

func parseOpts(req *plugin.GenerateRequest) (*Options, error) {
	var options Options
	if len(req.PluginOptions) == 0 {
		return &options, nil
	}
	if err := json.Unmarshal(req.PluginOptions, &options); err != nil {
		return nil, fmt.Errorf("unmarshalling plugin options: %w", err)
	}

	if options.Package == "" {
		if options.Out != "" {
			options.Package = filepath.Base(options.Out)
		} else {
			return nil, fmt.Errorf("invalid options: missing package name")
		}
	}

	for i := range options.Overrides {
		if err := options.Overrides[i].parse(req); err != nil {
			return nil, err
		}
	}

	if options.SqlPackage != "" {
		if err := validatePackage(options.SqlPackage); err != nil {
			return nil, fmt.Errorf("invalid options: %s", err)
		}
	}

	if options.SqlDriver != "" {
		if err := validateDriver(options.SqlDriver); err != nil {
			return nil, fmt.Errorf("invalid options: %s", err)
		}
	}

	if options.QueryParameterLimit == nil {
		options.QueryParameterLimit = new(int32)
		*options.QueryParameterLimit = 1
	}

	if options.Initialisms == nil {
		options.Initialisms = []string{"id"}
	}

	options.InitialismsMap = map[string]struct{}{}
	for _, initial := range options.Initialisms {
		options.InitialismsMap[initial] = struct{}{}
	}

	return &options, nil
}

func parseGlobalOpts(req *plugin.GenerateRequest) (*GlobalOptions, error) {
	var options GlobalOptions
	if len(req.GlobalOptions) == 0 {
		return &options, nil
	}
	if err := json.Unmarshal(req.GlobalOptions, &options); err != nil {
		return nil, fmt.Errorf("unmarshalling global options: %w", err)
	}
	for i := range options.Overrides {
		if err := options.Overrides[i].parse(req); err != nil {
			return nil, err
		}
	}
	return &options, nil
}

func ValidateOpts(opts *Options) error {
	if opts.EmitMethodsWithDbArgument && opts.EmitPreparedQueries {
		return fmt.Errorf("invalid options: emit_methods_with_db_argument and emit_prepared_queries options are mutually exclusive")
	}
	if *opts.QueryParameterLimit < 0 {
		return fmt.Errorf("invalid options: query parameter limit must not be negative")
	}
	if opts.OutputModelsPackage != "" && opts.ModelsPackageImportPath == "" {
		return fmt.Errorf("invalid options: models_package_import_path must be set when output_models_package is used")
	}
	if opts.ModelsPackageImportPath != "" && opts.OutputModelsPackage == "" {
		return fmt.Errorf("invalid options: output_models_package must be set when models_package_import_path is used")
	}

	return nil
}
