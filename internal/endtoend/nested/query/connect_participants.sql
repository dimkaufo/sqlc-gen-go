-- name: GetConnectParticipantsByID :one
-- The same composite twice under one parent: recruiter and interviewer are both
-- RecruiterInfoComposite, read from RecruiterAccount/User/Image/Account and their _2 twins.
-- Offers repeat every participant row.
SELECT
  c.*,
  sqlc.embed(ra),
  sqlc.embed(u),
  sqlc.embed(i),
  sqlc.embed(a),
  sqlc.embed(ia),
  sqlc.embed(iu),
  sqlc.embed(ii),
  sqlc.embed(iac),
  sqlc.embed(o)
FROM connects c
JOIN recruiter_accounts ra ON c.recruiter_account_id = ra.id
JOIN accounts a ON ra.id = a.id
JOIN users u ON a.user_id = u.id
LEFT JOIN images i ON u.avatar_id = i.id
LEFT JOIN recruiter_accounts ia ON c.interviewer_account_id = ia.id
LEFT JOIN accounts iac ON ia.id = iac.id
LEFT JOIN users iu ON iac.user_id = iu.id
LEFT JOIN images ii ON iu.avatar_id = ii.id
LEFT JOIN offers o ON c.id = o.connect_id
WHERE c.id = @id;
