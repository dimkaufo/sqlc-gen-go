package golang

import (
	"sort"
	"strconv"
	"strings"

	"github.com/sqlc-dev/sqlc-gen-go/internal/opts"
)

// This file holds helpers for mapping yaml struct_in to sqlc row fields (including duplicate embeds
// like Company_2) and for resolving NestedMatchConfig into ResolvedMatch (sqlc getter names).

// rowFieldNameQualifiesForStructIn reports whether an sqlc row field is the same embed type as structIn:
// the base name, or structIn_<decimal> (sqlc's naming for repeated embeds of the same type).
func rowFieldNameQualifiesForStructIn(fieldName, structIn string) bool {
	if structIn == "" || fieldName == "" {
		return false
	}
	if fieldName == structIn {
		return true
	}
	if !strings.HasPrefix(fieldName, structIn+"_") {
		return false
	}
	suffix := fieldName[len(structIn)+1:]
	if suffix == "" {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// structInEmbedAssigner picks sqlc row field names (Company vs Company_2, etc.) for nested nodes in DFS order
// when duplicate embeds exist.
type structInEmbedAssigner struct {
	b       *NestedQueryTemplateDataBuilder
	query   string
	nextIdx map[string]int
}

func (a *structInEmbedAssigner) pickRowFieldName(config *opts.NestedGroupConfig) string {
	structIn := config.StructIn
	names := a.b.qualifyingRowFieldNamesForStructIn(a.query, structIn)
	if len(names) == 0 {
		return structIn
	}
	if len(names) == 1 {
		return names[0]
	}
	if a.nextIdx == nil {
		a.nextIdx = make(map[string]int)
	}
	i := a.nextIdx[structIn]
	a.nextIdx[structIn]++
	if i < len(names) {
		return names[i]
	}
	// More nested nodes than duplicate row fields — reuse the last suffixed field (e.g. Company_2).
	if len(names) > 0 {
		return names[len(names)-1]
	}
	return structIn
}

func (b *NestedQueryTemplateDataBuilder) getQueryByName(queryName string) *Query {
	for _, query := range b.queries {
		if query.MethodName == queryName {
			return &query
		}
	}
	return nil
}

// qualifyingRowFieldNamesForStructIn returns sqlc row field names for this embed type in declaration order (Company, Company_2, ...).
func (b *NestedQueryTemplateDataBuilder) qualifyingRowFieldNamesForStructIn(queryName, structIn string) []string {
	q := b.getQueryByName(queryName)
	if q == nil || q.Ret.Struct == nil {
		return nil
	}
	var out []string
	for _, field := range q.Ret.Struct.Fields {
		if rowFieldNameQualifiesForStructIn(field.Name, structIn) {
			out = append(out, field.Name)
		}
	}
	return out
}

// lookupRowField returns the sqlc row struct field for a query by method name and field name.
func (b *NestedQueryTemplateDataBuilder) lookupRowField(queryName, fieldName string) (Field, bool) {
	for _, query := range b.queries {
		if query.MethodName != queryName {
			continue
		}
		if query.Ret.Struct == nil {
			return Field{}, false
		}
		for _, field := range query.Ret.Struct.Fields {
			if field.Name == fieldName {
				return field, true
			}
		}
		return Field{}, false
	}
	return Field{}, false
}

// isRowFieldExistsInQueryByName checks if a row struct field exists for the query (e.g. Company_2 after duplicate resolution).
func (b *NestedQueryTemplateDataBuilder) isRowFieldExistsInQueryByName(queryName string, fieldName string) bool {
	_, ok := b.lookupRowField(queryName, fieldName)
	return ok
}

// getRowFieldTypeByName returns the type of a field by name from the query's row struct.
// Returns empty string if the field doesn't exist.
func (b *NestedQueryTemplateDataBuilder) getRowFieldTypeByName(queryName string, fieldName string) string {
	if f, ok := b.lookupRowField(queryName, fieldName); ok {
		return f.Type
	}
	return ""
}

// lookupResolvedRowFieldNameInNestedSubtree finds the sqlc row field name for a logical struct name used in match rules.
// It prefers nested children whose StructIn matches, then RowFieldName (e.g. match references Image_2 by sqlc field name).
func lookupResolvedRowFieldNameInNestedSubtree(node *NestedStructData, structName string) string {
	if node == nil || structName == "" {
		return ""
	}
	for _, ch := range node.NestedStructs {
		if ch.StructIn == structName {
			return ch.RowFieldName
		}
		if ch.RowFieldName == structName {
			return ch.RowFieldName
		}
		if got := lookupResolvedRowFieldNameInNestedSubtree(ch, structName); got != "" {
			return got
		}
	}
	return ""
}

// resolveMatchResolvedForNode fills MatchResolved for every node that has Match (duplicate-safe getter names).
func (b *NestedQueryTemplateDataBuilder) resolveMatchResolvedForNode(queryName string, node *NestedStructData) {
	if node == nil {
		return
	}
	if len(node.Match) > 0 {
		node.MatchResolved = b.buildResolvedMatches(node, node.Match)
	}
	for _, ch := range node.NestedStructs {
		b.resolveMatchResolvedForNode(queryName, ch)
	}
}

func (b *NestedQueryTemplateDataBuilder) buildResolvedMatches(
	node *NestedStructData,
	match []*opts.NestedMatchConfig,
) []ResolvedMatch {
	var out []ResolvedMatch
	for _, m := range match {
		fromName := ""
		if m.FromStruct != nil {
			fromName = *m.FromStruct
		}
		fromName = resolveMatchSideStructNameForMatch(node, fromName)
		toName := resolveMatchSideStructNameForMatch(node, m.ToStruct)
		fromField := "ID"
		if m.FromField != nil {
			fromField = *m.FromField
		}
		toField := "ID"
		if m.ToField != nil {
			toField = *m.ToField
		}
		out = append(out, ResolvedMatch{
			FromStruct: fromName,
			ToStruct:   toName,
			FromField:  fromField,
			ToField:    toField,
		})
	}
	return out
}

// resolveMatchSideStructNameForMatch maps match from_struct/to_struct to sqlc row field names for getters, using qualifying row field names (qualifyingRowFieldNamesForStructIn).
func resolveMatchSideStructNameForMatch(node *NestedStructData, name string) string {
	if resolved := lookupResolvedRowFieldNameInNestedSubtree(node, name); resolved != "" {
		return resolved
	}
	return name
}

// appendDuplicateSqlcEmbedExclusions adds suffixed embed fields (e.g. *_2) when the same embed type appears multiple times on the row
// and the base name is already excluded from the parent composite (nested validation).
func appendDuplicateSqlcEmbedExclusions(allFields []Field, excluded []string) []string {
	set := make(map[string]struct{}, len(excluded)+8)
	for _, e := range excluded {
		set[e] = struct{}{}
	}
	bases := append([]string(nil), excluded...)
	for _, f := range allFields {
		if _, ok := set[f.Name]; ok {
			continue
		}
		for _, base := range bases {
			if base == "" {
				continue
			}
			prefix := base + "_"
			if strings.HasPrefix(f.Name, prefix) {
				suffix := f.Name[len(prefix):]
				if suffix != "" && matchNumericSuffix(suffix) {
					set[f.Name] = struct{}{}
					break
				}
			}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// matchNumericSuffix reports whether s is a non-empty decimal integer (e.g. sqlc's "_2", "_10").
// strconv.ParseUint rejects non-digits; embed suffixes are small enough to fit uint64.
func matchNumericSuffix(s string) bool {
	_, err := strconv.ParseUint(s, 10, 64)
	return err == nil
}

// collectMatchReferencedStructs collects structs referenced in match configs that need getter methods.
// This traverses the entire nested struct tree and collects unique structs from match configs
// that are not already covered by direct nested row getters on the root composite RowGetter.
//
// Only direct children of root produce RowGetter methods via generateRowGetterInterfaceMethodsRecursive;
// grandchildren (e.g. Company under RecruitersCompanyComposite) do not, so we must not filter them
// out when deciding whether to add GetCompany via MatchReferencedStructs.
func (b *NestedQueryTemplateDataBuilder) collectMatchReferencedStructs(queryName string, root *NestedStructData) []*MatchReferencedStruct {
	// Direct children already get Get<RowFieldName>() on the root RowGetter interface.
	directChildRowFields := collectDirectChildRowFieldNames(root)

	matchRefs := make(map[string]*MatchReferencedStruct)
	b.collectMatchRefsRecursive(queryName, root, matchRefs)
	// When a nested composite calls populateChildComposite(..., row), the parent RowGetter must
	// expose every getter that child composite's RowGetter requires (same row type through generics).
	b.mergePopulateCalleeRowGetterRefs(queryName, root, matchRefs)

	var result []*MatchReferencedStruct
	for fieldName, ref := range matchRefs {
		if !directChildRowFields[fieldName] {
			result = append(result, ref)
		}
	}

	return result
}

// collectDirectChildRowFieldNames returns RowFieldName for each immediate NestedStructs child of data.
// These map 1:1 to methods emitted on that composite's RowGetter (before MatchReferencedStructs).
func collectDirectChildRowFieldNames(data *NestedStructData) map[string]bool {
	out := make(map[string]bool)
	if data == nil {
		return out
	}
	for _, nestedStruct := range data.NestedStructs {
		out[nestedStruct.RowFieldName] = true
	}
	return out
}

// mergePopulateCalleeRowGetterRefs walks the tree and, for each nested composite that calls
// populate{{StructOut}}(..., row) (see nestedCore.tmpl nestedGrouperRecursiveContent), adds
// Get<RowFieldName>() requirements for that child composite's subtree to matchRefs.
func (b *NestedQueryTemplateDataBuilder) mergePopulateCalleeRowGetterRefs(
	queryName string,
	root *NestedStructData,
	matchRefs map[string]*MatchReferencedStruct,
) {
	var walk func(*NestedStructData)
	walk = func(node *NestedStructData) {
		if node == nil {
			return
		}
		for _, ch := range node.NestedStructs {
			// Same condition as template: composite + MatchResolved + populate*(..., row)
			if ch.IsComposite && len(ch.MatchResolved) > 0 {
				b.mergeRowGetterRefsForPopulateSubtree(queryName, ch, matchRefs)
			}
			walk(ch)
		}
	}
	walk(root)
}

// mergeRowGetterRefsForPopulateSubtree adds getters required so populate{StructOut}(..., row) type-checks.
// Prefer canonical field names from the composite registry (struct_in for non-duplicate embeds): a
// parent query may assign the company Image to row field Image_2 in NestedStructData while
// RecruitersCompanyCompositeRowGetter still requires GetImage() from the shared generic constraint.
func (b *NestedQueryTemplateDataBuilder) mergeRowGetterRefsForPopulateSubtree(
	queryName string,
	compositeRoot *NestedStructData,
	matchRefs map[string]*MatchReferencedStruct,
) {
	if reg := compositeStructRegistry[compositeRoot.StructOut]; reg != nil && len(reg.RowGetterFieldNamesForPopulateRow) > 0 {
		for _, fieldName := range reg.RowGetterFieldNamesForPopulateRow {
			b.addMatchRefFromFieldName(queryName, matchRefs, fieldName)
		}
		return
	}
	// Fallback when registry data is unavailable: walk the nested instance tree.
	var collect func(*NestedStructData)
	collect = func(n *NestedStructData) {
		for _, sub := range n.NestedStructs {
			if sub.RowFieldName != "" {
				b.addMatchRefFromFieldName(queryName, matchRefs, sub.RowFieldName)
			}
			collect(sub)
		}
	}
	collect(compositeRoot)
}

// collectMatchRefsRecursive recursively collects structs referenced in match configs.
func (b *NestedQueryTemplateDataBuilder) collectMatchRefsRecursive(queryName string, data *NestedStructData, matchRefs map[string]*MatchReferencedStruct) {
	if data == nil {
		return
	}

	// Collect from match configs at this level
	for _, nestedStruct := range data.NestedStructs {
		if len(nestedStruct.MatchResolved) > 0 {
			for _, rm := range nestedStruct.MatchResolved {
				b.addMatchRefFromFieldName(queryName, matchRefs, rm.FromStruct)
				b.addMatchRefFromFieldName(queryName, matchRefs, rm.ToStruct)
			}
		} else {
			for _, match := range nestedStruct.Match {
				// Add from_struct if specified and exists in query
				if match.FromStruct != nil && *match.FromStruct != "" {
					fromStruct := *match.FromStruct
					b.addMatchRefFromFieldName(queryName, matchRefs, fromStruct)
				}

				// Add to_struct if it exists in query (to_struct is always required)
				toStruct := match.ToStruct
				b.addMatchRefFromFieldName(queryName, matchRefs, toStruct)
			}
		}

		// Recurse into nested structs
		b.collectMatchRefsRecursive(queryName, nestedStruct, matchRefs)
	}
}

// addMatchRefFromFieldName adds a getter reference for a field name to matchRefs.
func (b *NestedQueryTemplateDataBuilder) addMatchRefFromFieldName(
	queryName string,
	matchRefs map[string]*MatchReferencedStruct,
	fieldName string,
) {
	if fieldName == "" {
		return
	}
	if _, exists := matchRefs[fieldName]; exists {
		return
	}
	fieldType := b.getRowFieldTypeByName(queryName, fieldName)
	if fieldType == "" {
		return
	}
	matchRefs[fieldName] = &MatchReferencedStruct{
		RowFieldName: fieldName,
		RowFieldType: fieldType,
	}
}
