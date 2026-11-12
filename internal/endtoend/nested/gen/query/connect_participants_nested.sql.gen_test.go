package query

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sqlc-dev/sqlc-gen-go/internal/endtoend/nested/gen/entity"
)

func TestGroupGetConnectParticipantsByID(t *testing.T) {
	// Recruiter and interviewer are the same composite under one parent; each reads its own
	// embeds (RecruiterAccount/User/Image/Account vs their _2 twins). Offers repeat the rows.
	participantsRow := func(offer byte) *GetConnectParticipantsByIDRow {
		return &GetConnectParticipantsByIDRow{
			ID:                          id(1),
			RecruiterAccount:            entity.RecruiterAccount{ID: id(10), Title: "recruiter"},
			Account:                     entity.Account{ID: id(10), UserID: id(11)},
			User:                        entity.User{ID: id(11), AvatarID: id(12)},
			Image:                       entity.Image{ID: id(12)},
			InterviewerRecruiterAccount: entity.RecruiterAccount{ID: id(20), Title: "interviewer"},
			InterviewerAccount:          entity.Account{ID: id(20), UserID: id(21)},
			InterviewerUser:             entity.User{ID: id(21), AvatarID: id(22)},
			InterviewerImage:            entity.Image{ID: id(22)},
			Offer:                       entity.Offer{ID: id(offer), ConnectID: id(1)},
		}
	}

	t.Run("same composite twice under one parent fills each from its own embeds", func(t *testing.T) {
		got := GroupGetConnectParticipantsByID([]*GetConnectParticipantsByIDRow{participantsRow(40), participantsRow(41)})

		require.Len(t, got, 1)
		c := got[0]
		require.Len(t, c.Offers, 2)

		require.Equal(t, id(10), c.Recruiter.ID)
		require.Equal(t, "recruiter", c.Recruiter.Title)
		require.Equal(t, id(10), c.Recruiter.Account.ID)
		require.Equal(t, id(11), c.Recruiter.User.ID)
		require.Equal(t, id(12), c.Recruiter.Avatar.ID)

		require.Equal(t, id(20), c.Interviewer.ID)
		require.Equal(t, "interviewer", c.Interviewer.Title)
		require.Equal(t, id(20), c.Interviewer.Account.ID)
		require.Equal(t, id(21), c.Interviewer.User.ID)
		require.Equal(t, id(22), c.Interviewer.Avatar.ID)
	})

	t.Run("NULL second embed leaves its field nil", func(t *testing.T) {
		row := participantsRow(40)
		row.InterviewerRecruiterAccount, row.InterviewerAccount, row.InterviewerUser, row.InterviewerImage = entity.RecruiterAccount{}, entity.Account{}, entity.User{}, entity.Image{}

		got := GroupGetConnectParticipantsByID([]*GetConnectParticipantsByIDRow{row})

		require.Len(t, got, 1)
		require.Nil(t, got[0].Interviewer)
		require.Equal(t, id(12), got[0].Recruiter.Avatar.ID)
	})
}
