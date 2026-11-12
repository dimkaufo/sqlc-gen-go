package golang

import "strings"

// renameNumberedEmbeds replaces sqlc's numbered names for repeated embeds (Image_2,
// RecruiterAccount_2) with names taken from the nested tree that consumes them: the field_out
// of the nearest composite occurrence plus the embed type (CompanyImage, InterviewerUser).
// Without a composite above it, the embed takes its own field_out (AuthorAvatar). Embeds no
// nested node consumes, and names that would collide, keep the numbered name. Config keeps
// referring to the sqlc names (match, ignore); they are resolved before this runs.
func renameNumberedEmbeds(nested []Nested) {
	renames := make(map[string]map[string]string) // query → sqlc row field → new name
	rows := make(map[string]*Struct)              // query → row struct

	for i := range nested {
		for _, item := range nested[i].NestedDataItems {
			if item.Query == nil || item.Query.Ret.Struct == nil {
				continue
			}
			rows[item.Query.MethodName] = item.Query.Ret.Struct
			if item.RootStructData != nil {
				renames[item.Query.MethodName] = numberedEmbedNames(item.RootStructData, item.Query.Ret.Struct)
			}
		}
	}
	// A wrapper casts its row to the first query's row type: both need the same names.
	for i := range nested {
		for _, item := range nested[i].NestedDataItems {
			if item.CastToQueryName != "" && item.Query != nil {
				renames[item.Query.MethodName] = renames[item.CastToQueryName]
			}
		}
	}

	for query, m := range renames {
		if len(m) == 0 {
			continue
		}
		renameRowFields(rows[query], m)
	}
	for i := range nested {
		for _, item := range nested[i].NestedDataItems {
			if item.RootStructData != nil && item.Query != nil {
				renameNestedRowRefs(item.RootStructData, renames[item.Query.MethodName])
			}
		}
	}
}

// numberedEmbedNames maps each numbered row field a nested node reads to its tree-derived name.
func numberedEmbedNames(root *NestedStructData, row *Struct) map[string]string {
	taken := make(map[string]bool, len(row.Fields))
	for _, f := range row.Fields {
		taken[f.Name] = true
	}

	proposed := make(map[string]string)
	count := make(map[string]int)
	// twin: field_out of the outermost composite read from a numbered embed (the interviewer
	// twin of the recruiter); nearest: field_out of the closest composite above.
	var walk func(n *NestedStructData, twin, nearest string)
	walk = func(n *NestedStructData, twin, nearest string) {
		for _, ch := range n.NestedStructs {
			base, numbered := numberedBase(ch.RowFieldName)
			childTwin, childNearest := twin, nearest
			if ch.IsComposite {
				if twin == "" && numbered {
					childTwin = ch.FieldName
				}
				childNearest = ch.FieldName
			}
			if numbered {
				if _, seen := proposed[ch.RowFieldName]; !seen {
					name := embedName(ch, base, childTwin, nearest)
					proposed[ch.RowFieldName] = name
					count[name]++
				}
			}
			walk(ch, childTwin, childNearest)
		}
	}
	walk(root, "", "")

	out := make(map[string]string)
	for from, to := range proposed {
		if to == "" || taken[to] || count[to] > 1 {
			continue
		}
		out[from] = to
	}
	return out
}

// embedName: twin root + type (InterviewerUser), else nearest composite + type
// (CompanyImage), else the node's own field_out (AuthorAvatar). A prefix already ending in
// the type is used alone (PrivateChat, not PrivateChatChat).
func embedName(n *NestedStructData, base, twin, nearest string) string {
	prefix := twin
	if prefix == "" {
		prefix = nearest
	}
	if prefix == "" {
		return n.FieldName
	}
	if strings.HasSuffix(prefix, base) {
		return prefix
	}
	return prefix + base
}

// numberedBase returns Image for Image_2: sqlc's suffix for a repeated embed.
func numberedBase(name string) (string, bool) {
	i := strings.LastIndex(name, "_")
	if i <= 0 || !matchNumericSuffix(name[i+1:]) {
		return "", false
	}
	return name[:i], true
}

func renameRowFields(row *Struct, m map[string]string) {
	if row == nil {
		return
	}
	for i := range row.Fields {
		to, ok := m[row.Fields[i].Name]
		if !ok {
			continue
		}
		f := &row.Fields[i]
		f.Name = to
		if _, ok := f.Tags["db"]; ok {
			f.Tags["db"] = SetCaseStyle(to, "snake")
		}
	}
}

// renameNestedRowRefs points every row getter reference in the tree at the renamed fields.
func renameNestedRowRefs(n *NestedStructData, m map[string]string) {
	if n == nil || len(m) == 0 {
		return
	}
	rename := func(s string) string {
		if to, ok := m[s]; ok {
			return to
		}
		return s
	}
	n.RowFieldName = rename(n.RowFieldName)
	for i := range n.MatchResolved {
		n.MatchResolved[i].FromStruct = rename(n.MatchResolved[i].FromStruct)
		n.MatchResolved[i].ToStruct = rename(n.MatchResolved[i].ToStruct)
	}
	for _, ref := range n.MatchReferencedStructs {
		ref.RowFieldName = rename(ref.RowFieldName)
	}
	for _, ch := range n.NestedStructs {
		renameNestedRowRefs(ch, m)
	}
}
