package golang

import (
	"strings"
	"testing"
)

func TestValidateCompositeFieldsAgainstEntity(t *testing.T) {
	structs := []Struct{{Name: "User", Fields: []Field{{Name: "ID"}, {Name: "Name"}}}}

	t.Run("fields present in the entity pass", func(t *testing.T) {
		err := validateCompositeFieldsAgainstEntity(structs, "UserComposite", []Field{{Name: "ID"}, {Name: "Name"}}, "User", "UserComposite", "root")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("unknown entity is skipped", func(t *testing.T) {
		err := validateCompositeFieldsAgainstEntity(structs, "XComposite", []Field{{Name: "Nope"}}, "Missing", "XComposite", "nested")
		if err != nil {
			t.Fatal(err)
		}
	})

	for _, ctx := range []string{"root", "nested"} {
		t.Run(ctx+" composite naming a field the entity lacks fails with the available fields", func(t *testing.T) {
			err := validateCompositeFieldsAgainstEntity(structs, "UserComposite", []Field{{Name: "ID"}, {Name: "Email"}}, "User", "ParentComposite", ctx)
			if err == nil {
				t.Fatal("want error")
			}
			for _, want := range []string{"UserComposite", "[Email]", "[ID, Name]"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q lacks %q", err, want)
				}
			}
		})
	}
}

func TestValidateExtractedFields(t *testing.T) {
	structs := []Struct{
		{Name: "Chat", Fields: []Field{{Name: "ID"}}},
		{Name: "ChatMember", Fields: []Field{{Name: "ID"}}},
	}

	t.Run("nested composite field missing from its embed fails", func(t *testing.T) {
		nested := []*NestedStructData{{
			StructOut:               "ChatMemberComposite",
			StructIn:                "ChatMember",
			IsComposite:             true,
			IsRowFieldExistsInQuery: true,
			Fields:                  []Field{{Name: "ID"}, {Name: "AccountID"}},
		}}
		err := validateExtractedFields([]Field{{Name: "ID"}}, nested, &Query{}, structs, "ChatGroup")
		if err == nil || !strings.Contains(err.Error(), "[AccountID]") {
			t.Fatalf("want missing AccountID, got %v", err)
		}
	})

	t.Run("composite without its embed in the row is not checked", func(t *testing.T) {
		nested := []*NestedStructData{{
			StructOut:   "ChatMemberComposite",
			StructIn:    "ChatMember",
			IsComposite: true,
			Fields:      []Field{{Name: "AccountID"}},
		}}
		if err := validateExtractedFields([]Field{{Name: "ID"}}, nested, &Query{}, structs, "ChatGroup"); err != nil {
			t.Fatal(err)
		}
	})
}
