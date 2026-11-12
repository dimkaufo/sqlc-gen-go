package golang

import "testing"

// recruiterInfo is a RecruiterInfoComposite occurrence reading the given user/image/account row fields.
func recruiterInfo(field, user, image, account string) *NestedStructData {
	return &NestedStructData{
		StructOut:   "RecruiterInfoComposite",
		FieldName:   field,
		IsComposite: true,
		MatchResolved: []ResolvedMatch{{
			FromStruct: "RecruiterAccount", FromField: "ID", ToStruct: account, ToField: "ID",
		}},
		NestedStructs: []*NestedStructData{
			{FieldName: "User", RowFieldName: user},
			{FieldName: "Avatar", RowFieldName: image},
			{FieldName: "Account", RowFieldName: account},
		},
	}
}

func TestMarkInlinePopulate(t *testing.T) {
	recruiter := recruiterInfo("Recruiter", "User", "Image", "Account")
	interviewer := recruiterInfo("Interviewer", "User_2", "Image_2", "Account_2")
	interviewer.SkipStructGeneration = true // only the first occurrence emits populateRecruiterInfoComposite
	nested := []Nested{{NestedDataItems: []NestedQueryTemplateData{{
		RootStructData: &NestedStructData{
			StructOut:     "ConnectParticipantsComposite",
			IsRoot:        true,
			IsComposite:   true,
			NestedStructs: []*NestedStructData{recruiter, interviewer},
		},
	}}}}

	markInlinePopulate(nested)

	if recruiter.InlinePopulate {
		t.Error("first occurrence emits the shared populate function; it must call it")
	}
	if !interviewer.InlinePopulate {
		t.Error("occurrence reading _2 row fields must inline instead of calling the shared function")
	}
}

func TestPopulateBodySignature(t *testing.T) {
	t.Run("differs when a descendant reads another row field", func(t *testing.T) {
		a := recruiterInfo("Recruiter", "User", "Image", "Account")
		b := recruiterInfo("Recruiter", "User", "Image_2", "Account")
		if populateBodySignature(a) == populateBodySignature(b) {
			t.Error("signatures must differ")
		}
	})

	t.Run("ignores the subtree of a composite child that is never populated", func(t *testing.T) {
		company := func(image string) *NestedStructData {
			return &NestedStructData{NestedStructs: []*NestedStructData{{
				FieldName:     "Company",
				RowFieldName:  "Company",
				IsComposite:   true,
				NestedStructs: []*NestedStructData{{FieldName: "Avatar", RowFieldName: image}},
			}}}
		}
		if populateBodySignature(company("Image")) != populateBodySignature(company("Image_2")) {
			t.Error("unmatched composite child's subtree is not read; signatures must match")
		}
	})
}

func TestUpdateDuplicatesFromNodePerspectiveSiblingDuplicate(t *testing.T) {
	recruiter := &NestedStructData{StructOut: "RecruiterInfoComposite"}
	interviewer := &NestedStructData{StructOut: "RecruiterInfoComposite"}
	offer := &NestedStructData{StructOut: "Offer"}
	root := &NestedStructData{NestedStructs: []*NestedStructData{recruiter, interviewer, offer}}

	updateDuplicatesFromNodePerspective(root, nil)

	if !recruiter.SiblingDuplicate || !interviewer.SiblingDuplicate {
		t.Error("same StructOut under one parent must be flagged on both")
	}
	if offer.SiblingDuplicate {
		t.Error("unique child must not be flagged")
	}
}
