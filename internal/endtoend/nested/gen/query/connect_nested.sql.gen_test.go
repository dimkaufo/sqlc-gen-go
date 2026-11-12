package query

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sqlc-dev/sqlc-gen-go/internal/endtoend/nested/gen/entity"
)

func TestGroupGetConnectByID(t *testing.T) {
	// chats x offers x feedbacks: every child repeats across the other collections' rows.
	t.Run("three one-to-many collections keep each element once", func(t *testing.T) {
		var rows []*GetConnectByIDRow
		for _, ch := range []byte{30, 31} {
			for _, o := range []byte{40, 41} {
				for _, f := range []byte{50, 51} {
					rows = append(rows, &GetConnectByIDRow{
						ID:               id(1),
						Vacancy:          entity.Vacancy{ID: id(2)},
						JobseekerAccount: entity.JobseekerAccount{ID: id(3)},
						RecruiterAccount: entity.RecruiterAccount{ID: id(4)},
						Chat:             entity.Chat{ID: id(ch), ConnectID: id(1)},
						ChatMember:       entity.ChatMember{ID: id(ch + 10), ChatID: id(ch), AccountID: id(ch + 20)},
						Account:          entity.Account{ID: id(ch + 20)},
						Offer:            entity.Offer{ID: id(o), ConnectID: id(1)},
						Feedback:         entity.Feedback{ID: id(f), ConnectID: id(1)},
					})
				}
			}
		}

		got := GroupGetConnectByID(rows)

		require.Len(t, got, 1)
		c := got[0]
		require.Equal(t, id(2), c.Vacancy.ID)
		require.Equal(t, id(3), c.JobseekerAccount.ID)
		require.Equal(t, id(4), c.RecruiterAccount.ID)
		require.Len(t, c.Chats, 2)
		require.Len(t, c.Offers, 2)
		require.Len(t, c.Feedbacks, 2)
		for i, ch := range []byte{30, 31} {
			require.Len(t, c.Chats[i].ChatMembers, 1)
			require.Equal(t, id(ch+20), c.Chats[i].ChatMembers[0].Account.ID)
		}
	})
}
