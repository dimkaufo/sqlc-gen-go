package golang

import "testing"

func TestRowFieldNameQualifiesForStructIn(t *testing.T) {
	for _, tc := range []struct {
		field, structIn string
		want            bool
	}{
		{"Company", "Company", true},
		{"Company_2", "Company", true},
		{"Company_10", "Company", true},
		{"Company_", "Company", false},
		{"Company_x", "Company", false},
		{"CompanyImage", "Company", false},
		{"", "Company", false},
		{"Company", "", false},
	} {
		if got := rowFieldNameQualifiesForStructIn(tc.field, tc.structIn); got != tc.want {
			t.Errorf("rowFieldNameQualifiesForStructIn(%q, %q) = %v, want %v", tc.field, tc.structIn, got, tc.want)
		}
	}
}

func TestResolveMatchSideStructNameForMatch(t *testing.T) {
	// Second embed of RecruiterAccount: its own fields and its children are the _2 twins.
	node := &NestedStructData{
		StructIn:     "RecruiterAccount",
		RowFieldName: "RecruiterAccount_2",
		NestedStructs: []*NestedStructData{
			{StructIn: "Account", RowFieldName: "Account_2"},
			{StructIn: "Image_2", RowFieldName: "Image_2"},
		},
	}

	for _, tc := range []struct{ name, want string }{
		{"RecruiterAccount", "RecruiterAccount_2"}, // the node itself (default from_struct)
		{"Account", "Account_2"},                   // a child by struct_in
		{"Image_2", "Image_2"},                     // a child by row field name
		{"Offer", "Offer"},                         // outside the subtree: unchanged
	} {
		if got := resolveMatchSideStructNameForMatch(node, tc.name); got != tc.want {
			t.Errorf("resolve %q = %q, want %q", tc.name, got, tc.want)
		}
	}
}
