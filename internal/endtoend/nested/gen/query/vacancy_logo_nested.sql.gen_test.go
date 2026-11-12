package query

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sqlc-dev/sqlc-gen-go/internal/endtoend/nested/gen/entity"
)

func TestGroupListVacancyLogos(t *testing.T) {
	row := func(vacancy, company, skill byte) *ListVacancyLogosRow {
		return &ListVacancyLogosRow{
			ID:           id(vacancy),
			CompanyID:    id(company),
			Company:      entity.Company{ID: id(company), AvatarID: id(company + 1)},
			Image:        entity.Image{ID: id(company + 1)},
			AuthorAvatar: entity.Image{ID: id(vacancy + 1)},
			Skill:        entity.Skill{ID: id(skill)},
		}
	}

	t.Run("field_group_by keys the root by a non-ID field", func(t *testing.T) {
		got := GroupListVacancyLogos([]*ListVacancyLogosRow{row(1, 20, 50), row(3, 20, 51), row(5, 30, 50)})

		require.Len(t, got, 2, "two companies")
		require.Equal(t, id(1), got[0].ID, "first row of the group wins the root fields")
		require.Equal(t, id(20), got[0].Company.ID)
		require.Equal(t, id(30), got[1].Company.ID)
		require.Len(t, got[0].Skills, 2)
		require.Len(t, got[1].Skills, 1)
	})

	t.Run("entity_type aliases the second Image embed (AuthorAvatar) to the Image entity", func(t *testing.T) {
		got := GroupListVacancyLogos([]*ListVacancyLogosRow{row(1, 20, 50)})

		require.Len(t, got, 1)
		require.Equal(t, id(21), got[0].Logo.ID, "Image")
		require.Equal(t, id(2), got[0].AuthorAvatar.ID, "AuthorAvatar via entity_type")
	})
}
