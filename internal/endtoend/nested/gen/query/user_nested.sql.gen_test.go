package query

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sqlc-dev/sqlc-gen-go/internal/endtoend/nested/gen/entity"
)

func TestGroupGetUserByID(t *testing.T) {
	t.Run("non-slice child lands in its field_out", func(t *testing.T) {
		got := GroupGetUserByID([]*GetUserByIDRow{
			{ID: id(1), Name: "ann", AvatarID: id(2), Image: entity.Image{ID: id(2), Url: "a.png"}},
		})

		require.Len(t, got, 1)
		require.Equal(t, "ann", got[0].Name)
		require.NotNil(t, got[0].Avatar)
		require.Equal(t, id(2), got[0].Avatar.ID)
	})

	t.Run("NULL left join leaves the child nil", func(t *testing.T) {
		got := GroupGetUserByID([]*GetUserByIDRow{{ID: id(1), Name: "ann"}})

		require.Len(t, got, 1)
		require.Nil(t, got[0].Avatar)
	})
}

func TestGroupListUsers(t *testing.T) {
	t.Run("wrapper over the first query's grouper keeps row order", func(t *testing.T) {
		got := GroupListUsers([]*ListUsersRow{
			{ID: id(1), Name: "ann", Image: entity.Image{ID: id(11)}},
			{ID: id(2), Name: "bob"},
			{ID: id(3), Name: "cid", Image: entity.Image{ID: id(13)}},
		})

		require.Len(t, got, 3)
		require.Equal(t, []string{"ann", "bob", "cid"}, []string{got[0].Name, got[1].Name, got[2].Name})
		require.Equal(t, id(11), got[0].Avatar.ID)
		require.Nil(t, got[1].Avatar)
		require.Equal(t, id(13), got[2].Avatar.ID)
	})
}
