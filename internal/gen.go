package golang

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sqlc-dev/plugin-sdk-go/metadata"
	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/sqlc-dev/plugin-sdk-go/sdk"
	"github.com/sqlc-dev/sqlc-gen-go/internal/opts"
)

type tmplCtx struct {
	Q           string
	Package     string
	SQLDriver   opts.SQLDriver
	Enums       []Enum
	Structs     []Struct
	GoQueries   []Query
	Nested      []Nested
	SqlcVersion string

	// TODO: Race conditions
	SourceName string
	FileName   string

	EmitJSONTags              bool
	JsonTagsIDUppercase       bool
	EmitDBTags                bool
	EmitPreparedQueries       bool
	EmitInterface             bool
	EmitEmptySlices           bool
	EmitMethodsWithDBArgument bool
	EmitEnumValidMethod       bool
	EmitAllEnumValues         bool
	UsesCopyFrom              bool
	UsesBatch                 bool
	OmitSqlcVersion           bool
	BuildTags                 string
	OutputModelsPackage       string
}

func (t *tmplCtx) OutputQuery(sourceName string) bool {
	return t.SourceName == sourceName
}

func (t *tmplCtx) codegenDbarg() string {
	if t.EmitMethodsWithDBArgument {
		return "db DBTX, "
	}
	return ""
}

// Called as a global method since subtemplate queryCodeStdExec does not have
// access to the toplevel tmplCtx
func (t *tmplCtx) codegenEmitPreparedQueries() bool {
	return t.EmitPreparedQueries
}

func (t *tmplCtx) codegenQueryMethod(q Query) string {
	db := "q.db"
	if t.EmitMethodsWithDBArgument {
		db = "db"
	}

	switch q.Cmd {
	case ":one":
		if t.EmitPreparedQueries {
			return "q.queryRow"
		}
		return db + ".QueryRowContext"

	case ":many":
		if t.EmitPreparedQueries {
			return "q.query"
		}
		return db + ".QueryContext"

	default:
		if t.EmitPreparedQueries {
			return "q.exec"
		}
		return db + ".ExecContext"
	}
}

func (t *tmplCtx) codegenQueryRetval(q Query) (string, error) {
	switch q.Cmd {
	case ":one":
		return "row :=", nil
	case ":many":
		return "rows, err :=", nil
	case ":exec":
		return "_, err :=", nil
	case ":execrows", ":execlastid":
		return "result, err :=", nil
	case ":execresult":
		return "return", nil
	default:
		return "", fmt.Errorf("unhandled q.Cmd case %q", q.Cmd)
	}
}

func Generate(ctx context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {
	options, err := opts.Parse(req)
	if err != nil {
		return nil, err
	}

	if err := opts.ValidateOpts(options); err != nil {
		return nil, err
	}

	enums := buildEnums(req, options)
	structs := buildStructs(req, options)
	queries, err := buildQueries(req, options, structs)
	if err != nil {
		return nil, err
	}

	// Populate nested config with default values to avoid checking it accross all the code
	if err := populateNestedConfigWithDefaultValues(options); err != nil {
		return nil, err
	}

	// Get nested source with configs
	nestedWithoutData, err := getNestedSourceWithConfigs(options, queries, structs)
	if err != nil {
		return nil, err
	}

	// Populate nested data items
	nestedWithData, err := populateNestedDataItems(options, queries, structs, nestedWithoutData)
	if err != nil {
		return nil, err
	}

	if options.OmitUnusedStructs {
		enums, structs = filterUnusedStructs(options, enums, structs, queries)
	}

	if err := validate(options, enums, structs, queries); err != nil {
		return nil, err
	}

	return generate(req, options, enums, structs, queries, nestedWithData)
}

func validate(options *opts.Options, enums []Enum, structs []Struct, queries []Query) error {
	enumNames := make(map[string]struct{})
	for _, enum := range enums {
		enumNames[enum.Name] = struct{}{}
		enumNames["Null"+enum.Name] = struct{}{}
	}
	structNames := make(map[string]struct{})
	for _, struckt := range structs {
		if _, ok := enumNames[struckt.Name]; ok {
			return fmt.Errorf("struct name conflicts with enum name: %s", struckt.Name)
		}
		structNames[struckt.Name] = struct{}{}
	}
	if !options.EmitExportedQueries {
		return nil
	}
	for _, query := range queries {
		if _, ok := enumNames[query.ConstantName]; ok {
			return fmt.Errorf("query constant name conflicts with enum name: %s", query.ConstantName)
		}
		if _, ok := structNames[query.ConstantName]; ok {
			return fmt.Errorf("query constant name conflicts with struct name: %s", query.ConstantName)
		}
	}
	return nil
}

func generate(
	req *plugin.GenerateRequest,
	options *opts.Options,
	enums []Enum,
	structs []Struct,
	queries []Query,
	nested []Nested,
) (*plugin.GenerateResponse, error) {
	i := &importer{
		Options: options,
		Queries: queries,
		Enums:   enums,
		Structs: structs,
	}

	tctx := tmplCtx{
		EmitInterface:             options.EmitInterface,
		EmitJSONTags:              options.EmitJsonTags,
		JsonTagsIDUppercase:       options.JsonTagsIdUppercase,
		EmitDBTags:                options.EmitDbTags,
		EmitPreparedQueries:       options.EmitPreparedQueries,
		EmitEmptySlices:           options.EmitEmptySlices,
		EmitMethodsWithDBArgument: options.EmitMethodsWithDbArgument,
		EmitEnumValidMethod:       options.EmitEnumValidMethod,
		EmitAllEnumValues:         options.EmitAllEnumValues,
		OutputModelsPackage:       options.OutputModelsPackage,
		UsesCopyFrom:              usesCopyFrom(queries),
		UsesBatch:                 usesBatch(queries),
		SQLDriver:                 parseDriver(options.SqlPackage),
		Q:                         "`",
		Package:                   options.Package,
		Enums:                     enums,
		Structs:                   structs,
		Nested:                    nested,
		SqlcVersion:               req.SqlcVersion,
		BuildTags:                 options.BuildTags,
		OmitSqlcVersion:           options.OmitSqlcVersion,
	}

	if tctx.UsesCopyFrom && !tctx.SQLDriver.IsPGX() && options.SqlDriver != opts.SQLDriverGoSQLDriverMySQL {
		return nil, errors.New(":copyfrom is only supported by pgx and github.com/go-sql-driver/mysql")
	}

	if tctx.UsesCopyFrom && options.SqlDriver == opts.SQLDriverGoSQLDriverMySQL {
		if err := checkNoTimesForMySQLCopyFrom(queries); err != nil {
			return nil, err
		}
		tctx.SQLDriver = opts.SQLDriverGoSQLDriverMySQL
	}

	if tctx.UsesBatch && !tctx.SQLDriver.IsPGX() {
		return nil, errors.New(":batch* commands are only supported by pgx")
	}

	var tmpl *template.Template
	funcMap := template.FuncMap{
		"lowerTitle": sdk.LowerTitle,
		"upperTitle": upperTitle,
		"comment":    sdk.DoubleSlashComment,
		"escape":     sdk.EscapeBacktick,
		"imports":    i.Imports,
		"hasImports": i.HasImports,
		"hasPrefix":  strings.HasPrefix,
		"trimPrefix": strings.TrimPrefix,
		"camelCase":  ToCamelCase,
		"ternary":    ternary,
		"joinTags":   joinTags,
		"list":       list,
		"add":        add,
		"dict":       dict,
		"set":        set,
		"render": func(name string, data any) string {
			var buf bytes.Buffer
			if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
				return fmt.Sprintf("ERR: %v", err)
			}
			return buf.String()
		},

		// Nullable type helpers for embed fields
		"getNullableType":       getNullableType,
		"getNullableValueField": getNullableValueField,

		// These methods are Go specific, they do not belong in the codegen package
		// (as that is language independent)
		"dbarg":               tctx.codegenDbarg,
		"emitPreparedQueries": tctx.codegenEmitPreparedQueries,
		"queryMethod":         tctx.codegenQueryMethod,
		"queryRetval":         tctx.codegenQueryRetval,
	}

	tmpl = template.Must(
		template.New("table").
			Funcs(funcMap).
			ParseFS(
				templates,
				"templates/*.tmpl",
				"templates/*/*.tmpl",
			),
	)

	output := map[string]string{}

	execute := func(fileName, packageName, templateName string) error {
		imports := i.Imports(fileName)
		replacedQueries := replaceConflictedArg(imports, queries)

		var b bytes.Buffer
		w := bufio.NewWriter(&b)
		tctx.FileName = fileName
		tctx.SourceName = fileName
		if templateName == "nestedCoreFile" {
			tctx.SourceName = extractSqlFileNameFromNestedFileName(fileName)
		}

		tctx.GoQueries = replacedQueries
		tctx.Package = packageName

		err := tmpl.ExecuteTemplate(w, templateName, &tctx)
		w.Flush()
		if err != nil {
			return err
		}
		code, err := format.Source(b.Bytes())
		if err != nil {
			// Write debug info to stderr instead of stdout to avoid corrupting protobuf
			fmt.Fprintf(os.Stderr, "Source formatting error for %s:\n%s\n", fileName, b.String())
			return fmt.Errorf("source error: %w", err)
		}

		if templateName == "queryFile" || templateName == "nestedUtilsFile" {
			if options.OutputQueryFilesDirectory != "" {
				fileName = filepath.Join(options.OutputQueryFilesDirectory, fileName)
			}
		}

		if templateName == "queryFile" {
			if options.OutputFilesSuffix != "" {
				fileName += options.OutputFilesSuffix
			}
		}

		if templateName == "nestedUtilsFile" {
			fileName = strings.TrimSuffix(fileName, ".go") + options.OutputFilesSuffix
		}

		if !strings.HasSuffix(fileName, ".go") {
			fileName += ".go"
		}
		output[fileName] = string(code)
		return nil
	}

	dbFileName := "db.go"
	if options.OutputDbFileName != "" {
		dbFileName = options.OutputDbFileName
	}
	modelsFileName := "models.go"
	if options.OutputModelsFileName != "" {
		modelsFileName = options.OutputModelsFileName
	}
	querierFileName := "querier.go"
	if options.OutputQuerierFileName != "" {
		querierFileName = options.OutputQuerierFileName
	}
	copyfromFileName := "copyfrom.go"
	if options.OutputCopyfromFileName != "" {
		copyfromFileName = options.OutputCopyfromFileName
	}

	batchFileName := "batch.go"
	if options.OutputBatchFileName != "" {
		batchFileName = options.OutputBatchFileName
	}

	nestedUtilsFileName := "nested.utils.go"
	if options.OutputNestedUtilsFileName != "" {
		nestedUtilsFileName = options.OutputNestedUtilsFileName
	}

	modelsPackageName := options.Package
	if options.OutputModelsPackage != "" {
		modelsPackageName = options.OutputModelsPackage
	}

	if err := execute(dbFileName, options.Package, "dbFile"); err != nil {
		return nil, err
	}
	if err := execute(modelsFileName, modelsPackageName, "modelsFile"); err != nil {
		return nil, err
	}
	if options.EmitInterface {
		if err := execute(querierFileName, options.Package, "interfaceFile"); err != nil {
			return nil, err
		}
	}
	if tctx.UsesCopyFrom {
		if err := execute(copyfromFileName, options.Package, "copyfromFile"); err != nil {
			return nil, err
		}
	}
	if tctx.UsesBatch {
		if err := execute(batchFileName, options.Package, "batchFile"); err != nil {
			return nil, err
		}
	}

	files := map[string]struct{}{}
	for _, gq := range queries {
		files[gq.SourceName] = struct{}{}
	}

	for source := range files {
		if err := execute(source, options.Package, "queryFile"); err != nil {
			return nil, err
		}
	}

	// Generate nested grouping functions if configured
	if len(nested) > 0 {
		// Generate _nested.sql files
		for _, nestedItem := range nested {
			nestedFileName := getNestedFileName(options, nestedItem.SourceFileName)
			if err := execute(nestedFileName, options.Package, "nestedCoreFile"); err != nil {
				return nil, err
			}
		}

		// Generate nested.gen if any nested files were generated
		if err := execute(nestedUtilsFileName, options.Package, "nestedUtilsFile"); err != nil {
			return nil, err
		}
	}

	resp := plugin.GenerateResponse{}

	for filename, code := range output {
		resp.Files = append(resp.Files, &plugin.File{
			Name:     filename,
			Contents: []byte(code),
		})
	}

	return &resp, nil
}

type Nested struct {
	SourceFileName  string
	Configs         []*opts.NestedQueryConfig
	NestedDataItems []NestedQueryTemplateData
}

// getNestedSourceWithConfigs creates ordered list of source files with their configs
func getNestedSourceWithConfigs(options *opts.Options, queries []Query, structs []Struct) ([]Nested, error) {
	if options.Nested == nil || len(options.Nested.Queries) == 0 {
		return nil, nil
	}

	var sources []Nested
	seen := make(map[string]bool)

	for _, config := range options.Nested.Queries {
		// Find the source file for this query
		var sourceFile string
		for _, q := range queries {
			if q.MethodName == config.Query || q.SourceName == config.Query {
				sourceFile = q.SourceName
				break
			}
		}
		if sourceFile != "" {
			if !seen[sourceFile] {
				// First time seeing this source file, create new entry
				sources = append(sources, Nested{
					SourceFileName: sourceFile,
					Configs:        []*opts.NestedQueryConfig{config},
				})
				seen[sourceFile] = true
			} else {
				// Add config to existing entry
				for i := range sources {
					if sources[i].SourceFileName == sourceFile {
						sources[i].Configs = append(sources[i].Configs, config)
						break
					}
				}
			}
		}
	}

	return sources, nil
}

var nestedFileNameSuffix = "_nested.sql"

func getNestedFileName(options *opts.Options, fileName string) string {
	baseFileName := strings.TrimSuffix(fileName, ".sql")
	nestedFileName := baseFileName + nestedFileNameSuffix + ".go"
	if options.OutputFilesSuffix != "" {
		nestedFileName = baseFileName + nestedFileNameSuffix + options.OutputFilesSuffix + ".go"
	}

	// Apply output_query_files_directory logic like query files
	if options.OutputQueryFilesDirectory != "" {
		nestedFileName = filepath.Join(options.OutputQueryFilesDirectory, nestedFileName)
	}

	return nestedFileName
}

func isNestedFileName(fileName string) bool {
	return strings.Contains(fileName, nestedFileNameSuffix) && strings.HasSuffix(fileName, ".go")
}

func extractSqlFileNameFromNestedFileName(fileName string) string {
	// Remove directory path if present and .go extension
	baseName := strings.TrimSuffix(filepath.Base(fileName), ".go")

	// Remove output files suffix if present
	nestedIndex := strings.Index(baseName, nestedFileNameSuffix)
	if nestedIndex != -1 {
		// Extract everything before "_nested.sql"
		sourceBase := baseName[:nestedIndex]
		return sourceBase + ".sql"
	}

	// Fallback: if pattern doesn't match expected format, return as-is with .sql
	return strings.TrimSuffix(baseName, nestedFileNameSuffix) + ".sql"
}

func usesCopyFrom(queries []Query) bool {
	for _, q := range queries {
		if q.Cmd == metadata.CmdCopyFrom {
			return true
		}
	}
	return false
}

func usesBatch(queries []Query) bool {
	for _, q := range queries {
		for _, cmd := range []string{metadata.CmdBatchExec, metadata.CmdBatchMany, metadata.CmdBatchOne} {
			if q.Cmd == cmd {
				return true
			}
		}
	}
	return false
}

func checkNoTimesForMySQLCopyFrom(queries []Query) error {
	for _, q := range queries {
		if q.Cmd != metadata.CmdCopyFrom {
			continue
		}
		for _, f := range q.Arg.CopyFromMySQLFields() {
			if f.Type == "time.Time" {
				return fmt.Errorf("values with a timezone are not yet supported")
			}
		}
	}
	return nil
}

func filterUnusedStructs(options *opts.Options, enums []Enum, structs []Struct, queries []Query) ([]Enum, []Struct) {
	keepTypes := make(map[string]struct{})

	for _, query := range queries {
		if !query.Arg.isEmpty() {
			keepTypes[query.Arg.Type()] = struct{}{}
			if query.Arg.IsStruct() {
				for _, field := range query.Arg.Struct.Fields {
					keepTypes[field.Type] = struct{}{}
				}
			}
		}
		if query.hasRetType() {
			keepTypes[query.Ret.Type()] = struct{}{}
			if query.Ret.IsStruct() {
				for _, field := range query.Ret.Struct.Fields {
					keepTypes[field.Type] = struct{}{}
					for _, embedField := range field.EmbedFields {
						keepTypes[embedField.Type] = struct{}{}
					}
				}
			}
		}
	}

	keepEnums := make([]Enum, 0, len(enums))
	for _, enum := range enums {
		var enumType string
		if options.ModelsPackageImportPath != "" {
			enumType = options.OutputModelsPackage + "." + enum.Name
		} else {
			enumType = enum.Name
		}

		_, keep := keepTypes[enumType]
		_, keepNull := keepTypes["Null"+enumType]
		if keep || keepNull {
			keepEnums = append(keepEnums, enum)
		}
	}

	keepStructs := make([]Struct, 0, len(structs))
	for _, st := range structs {
		if _, ok := keepTypes[st.Type()]; ok {
			keepStructs = append(keepStructs, st)
		}
	}

	return keepEnums, keepStructs
}
