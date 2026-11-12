package query

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sqlc-dev/sqlc-gen-go/internal/endtoend/nested/gen/entity"
)

func TestGroupGetChatsByConnectID(t *testing.T) {
	t.Run("nested composites in a slice are populated through their match", func(t *testing.T) {
		var rows []*GetChatsByConnectIDRow
		for _, m := range []byte{30, 31} {
			rows = append(rows, &GetChatsByConnectIDRow{
				ID:         id(6),
				ChatMember: entity.ChatMember{ID: id(m), ChatID: id(6), AccountID: id(m + 100)},
				Account:    entity.Account{ID: id(m + 100), UserID: id(m + 150)},
				User:       entity.User{ID: id(m + 150), AvatarID: id(m + 200)},
				Image:      entity.Image{ID: id(m + 200)},
			})
		}

		got := GroupGetChatsByConnectID(rows)

		require.Len(t, got, 1)
		require.Len(t, got[0].ChatMembers, 2)
		for i, m := range []byte{30, 31} {
			member := got[0].ChatMembers[i]
			require.Equal(t, id(m), member.ID)
			require.Equal(t, id(m+100), member.Account.ID)
			require.Equal(t, id(m+150), member.User.ID)
			require.Equal(t, id(m+200), member.Avatar.ID)
		}
	})

	t.Run("chats stay separate and keep only their own members", func(t *testing.T) {
		got := GroupGetChatsByConnectID([]*GetChatsByConnectIDRow{
			{ID: id(6), ChatMember: entity.ChatMember{ID: id(30), AccountID: id(130)}, Account: entity.Account{ID: id(130)}},
			{ID: id(7), ChatMember: entity.ChatMember{ID: id(31), AccountID: id(131)}, Account: entity.Account{ID: id(131)}},
		})

		require.Len(t, got, 2)
		require.Len(t, got[0].ChatMembers, 1)
		require.Equal(t, id(30), got[0].ChatMembers[0].ID)
		require.Len(t, got[1].ChatMembers, 1)
		require.Equal(t, id(31), got[1].ChatMembers[0].ID)
	})
}
