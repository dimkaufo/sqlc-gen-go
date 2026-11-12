-- name: GetVacancyByID :one
-- Same embed type under different parents (Image for the author, Image_2 for the company),
-- an ignored embed referenced only by a match, and a slice of composites.
SELECT
  v.*,
  sqlc.embed(ra),
  sqlc.embed(u),
  sqlc.embed(a),
  sqlc.embed(i),
  sqlc.embed(c),
  sqlc.embed(ci),
  sqlc.embed(vs),
  sqlc.embed(sk)
FROM vacancies v
JOIN recruiter_accounts ra ON v.author_id = ra.id
JOIN accounts a ON ra.id = a.id
JOIN users u ON a.user_id = u.id
LEFT JOIN images i ON u.avatar_id = i.id
JOIN companies c ON v.company_id = c.id
LEFT JOIN images ci ON c.avatar_id = ci.id
LEFT JOIN vacancy_skills vs ON v.id = vs.vacancy_id
LEFT JOIN skills sk ON vs.skill_id = sk.id
WHERE v.id = @id;
