package query

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sqlc-dev/sqlc-gen-go/internal/endtoend/nested/gen/entity"
)

func TestGroupGetConnectChatsByID(t *testing.T) {
	// general members g x private members p: every member of one chat repeats per member of the other.
	var rows []*GetConnectChatsByIDRow
	for _, g := range []byte{30, 31} {
		for _, p := range []byte{60, 61, 62} {
			rows = append(rows, &GetConnectChatsByIDRow{
				ID:                    id(1),
				Chat:                  entity.Chat{ID: id(2), ConnectID: id(1)},
				ChatMember:            entity.ChatMember{ID: id(g), ChatID: id(2), AccountID: id(g + 100)},
				Account:               entity.Account{ID: id(g + 100), UserID: id(g + 150)},
				User:                  entity.User{ID: id(g + 150), AvatarID: id(g + 200)},
				Image:                 entity.Image{ID: id(g + 200)},
				PrivateChat:           entity.Chat{ID: id(3), ConnectID: id(1), IsPrivate: true},
				PrivateChatChatMember: entity.ChatMember{ID: id(p), ChatID: id(3), AccountID: id(p + 100)},
				PrivateChatAccount:    entity.Account{ID: id(p + 100), UserID: id(p + 150)},
				PrivateChatUser:       entity.User{ID: id(p + 150), AvatarID: id(p + 50)},
				PrivateChatImage:      entity.Image{ID: id(p + 50)},
			})
		}
	}

	got := GroupGetConnectChatsByID(rows)

	require.Len(t, got, 1)
	assertChat := func(chat *ChatComposite, chatID byte, members []byte, userOff, avatarOff byte) {
		t.Helper()
		require.NotNil(t, chat)
		require.Equal(t, id(chatID), chat.ID)
		require.Len(t, chat.ChatMembers, len(members))
		for i, m := range members {
			member := chat.ChatMembers[i]
			require.Equal(t, id(m), member.ID)
			require.Equal(t, id(m+100), member.Account.ID)
			require.Equal(t, id(m+userOff), member.User.ID)
			require.Equal(t, id(m+avatarOff), member.Avatar.ID)
		}
	}

	t.Run("first occurrence keeps its members via the shared populate functions", func(t *testing.T) {
		assertChat(got[0].GeneralChat, 2, []byte{30, 31}, 150, 200)
	})

	t.Run("second occurrence inlines two levels deep and reads only the _2 twins", func(t *testing.T) {
		assertChat(got[0].PrivateChat, 3, []byte{60, 61, 62}, 150, 50)
	})
}
