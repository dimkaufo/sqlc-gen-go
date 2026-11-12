package golang

import (
	"reflect"
	"testing"
)

func TestNumberedEmbedNames(t *testing.T) {
	leaf := func(field, row string) *NestedStructData {
		return &NestedStructData{FieldName: field, RowFieldName: row}
	}
	composite := func(field, row string, children ...*NestedStructData) *NestedStructData {
		return &NestedStructData{FieldName: field, RowFieldName: row, IsComposite: true, NestedStructs: children}
	}
	row := func(names ...string) *Struct {
		s := &Struct{}
		for _, n := range names {
			s.Fields = append(s.Fields, Field{Name: n})
		}
		return s
	}

	tests := []struct {
		name string
		root *NestedStructData
		row  *Struct
		want map[string]string
	}{
		{
			name: "twin composite prefixes its whole subtree",
			root: &NestedStructData{NestedStructs: []*NestedStructData{
				composite("Recruiter", "RecruiterAccount", leaf("User", "User")),
				composite("Interviewer", "RecruiterAccount_2", leaf("User", "User_2")),
			}},
			row:  row("RecruiterAccount", "User", "RecruiterAccount_2", "User_2"),
			want: map[string]string{"RecruiterAccount_2": "InterviewerRecruiterAccount", "User_2": "InterviewerUser"},
		},
		{
			name: "nearest composite names an embed under a plain composite",
			root: &NestedStructData{NestedStructs: []*NestedStructData{
				composite("Author", "User", leaf("Avatar", "Image")),
				composite("Company", "Company", leaf("Avatar", "Image_2")),
			}},
			row:  row("User", "Image", "Company", "Image_2"),
			want: map[string]string{"Image_2": "CompanyImage"},
		},
		{
			name: "prefix ending in the type is not repeated",
			root: &NestedStructData{NestedStructs: []*NestedStructData{
				composite("PrivateChat", "Chat_2", composite("ChatMembers", "ChatMember_2")),
			}},
			row:  row("Chat", "Chat_2", "ChatMember", "ChatMember_2"),
			want: map[string]string{"Chat_2": "PrivateChat", "ChatMember_2": "PrivateChatChatMember"},
		},
		{
			name: "no composite above takes the field_out",
			root: &NestedStructData{NestedStructs: []*NestedStructData{leaf("AuthorAvatar", "Image_2")}},
			row:  row("Image", "Image_2"),
			want: map[string]string{"Image_2": "AuthorAvatar"},
		},
		{
			name: "name taken by another row field keeps the numbered name",
			root: &NestedStructData{NestedStructs: []*NestedStructData{leaf("Logo", "Image_2")}},
			row:  row("Image", "Image_2", "Logo"),
			want: map[string]string{},
		},
		{
			name: "two embeds proposing one name both keep theirs",
			root: &NestedStructData{NestedStructs: []*NestedStructData{
				composite("Company", "Company", leaf("Avatar", "Image_2"), leaf("Cover", "Image_3")),
			}},
			row:  row("Company", "Image", "Image_2", "Image_3"),
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := numberedEmbedNames(tt.root, tt.row); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenameNestedRowRefs(t *testing.T) {
	ref := &MatchReferencedStruct{RowFieldName: "Image_2"}
	child := &NestedStructData{RowFieldName: "Image_2"}
	root := &NestedStructData{
		RowFieldName:           "Company",
		MatchResolved:          []ResolvedMatch{{FromStruct: "Image_2", ToStruct: "Company"}},
		MatchReferencedStructs: []*MatchReferencedStruct{ref},
		NestedStructs:          []*NestedStructData{child},
	}
	renameNestedRowRefs(root, map[string]string{"Image_2": "CompanyImage"})

	if child.RowFieldName != "CompanyImage" || ref.RowFieldName != "CompanyImage" ||
		root.MatchResolved[0].FromStruct != "CompanyImage" || root.MatchResolved[0].ToStruct != "Company" ||
		root.RowFieldName != "Company" {
		t.Fatalf("refs not renamed: child=%s ref=%s match=%+v root=%s",
			child.RowFieldName, ref.RowFieldName, root.MatchResolved[0], root.RowFieldName)
	}
}
