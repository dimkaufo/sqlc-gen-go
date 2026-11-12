package query

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sqlc-dev/sqlc-gen-go/internal/endtoend/nested/gen/entity"
)

func TestGroupGetVacancyByID(t *testing.T) {
	// Author avatar is Image, company avatar is CompanyImage: each must land in its own composite.
	vacancyRow := func(skill byte) *GetVacancyByIDRow {
		return &GetVacancyByIDRow{
			ID:               id(1),
			Title:            "dev",
			RecruiterAccount: entity.RecruiterAccount{ID: id(10)},
			Account:          entity.Account{ID: id(10), UserID: id(11)},
			User:             entity.User{ID: id(11), AvatarID: id(12)},
			Image:            entity.Image{ID: id(12), Url: "author.png"},
			Company:          entity.Company{ID: id(20), AvatarID: id(21)},
			CompanyImage:     entity.Image{ID: id(21), Url: "company.png"},
			VacancySkill:     entity.VacancySkill{ID: id(skill), VacancyID: id(1), SkillID: id(skill + 100)},
			Skill:            entity.Skill{ID: id(skill + 100)},
		}
	}

	t.Run("duplicate embeds under different parents read their own row field", func(t *testing.T) {
		got := GroupGetVacancyByID([]*GetVacancyByIDRow{vacancyRow(30)})

		require.Len(t, got, 1)
		require.Equal(t, id(10), got[0].Author.ID)
		require.Equal(t, id(12), got[0].Author.Avatar.ID, "author avatar is the Image embed")
		require.Equal(t, id(20), got[0].Company.ID)
		require.Equal(t, id(21), got[0].Company.Avatar.ID, "company avatar is the CompanyImage embed")
	})

	t.Run("slice of composites dedupes repeated parents and fills each child", func(t *testing.T) {
		got := GroupGetVacancyByID([]*GetVacancyByIDRow{vacancyRow(30), vacancyRow(31), vacancyRow(30)})

		require.Len(t, got, 1)
		require.Len(t, got[0].Skills, 2)
		for i, s := range []byte{30, 31} {
			require.Equal(t, id(s), got[0].Skills[i].ID)
			require.Equal(t, id(s+100), got[0].Skills[i].Skill.ID)
		}
	})
}
