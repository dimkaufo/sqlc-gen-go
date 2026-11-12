-- name: GetConnectByID :one
-- Three one-to-many collections: rows are chats x offers x feedbacks.
SELECT
  c.*,
  sqlc.embed(v),
  sqlc.embed(ja),
  sqlc.embed(ra),
  sqlc.embed(ch),
  sqlc.embed(cm),
  sqlc.embed(a),
  sqlc.embed(u),
  sqlc.embed(i),
  sqlc.embed(o),
  sqlc.embed(f)
FROM connects c
JOIN vacancies v ON c.vacancy_id = v.id
JOIN jobseeker_accounts ja ON c.jobseeker_account_id = ja.id
JOIN recruiter_accounts ra ON c.recruiter_account_id = ra.id
LEFT JOIN chats ch ON c.id = ch.connect_id
LEFT JOIN chat_members cm ON ch.id = cm.chat_id
LEFT JOIN accounts a ON cm.account_id = a.id
LEFT JOIN users u ON a.user_id = u.id
LEFT JOIN images i ON u.avatar_id = i.id
LEFT JOIN offers o ON c.id = o.connect_id
LEFT JOIN feedbacks f ON c.id = f.connect_id
WHERE c.id = @id;
