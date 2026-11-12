-- name: ListVacancyLogos :many
-- Query-level group (no composite root): value (pointer: false) children, entity_type for an
-- aliased embed, and a root grouped by a non-ID field.
SELECT
  v.*,
  sqlc.embed(c),
  sqlc.embed(i),
  sqlc.embed(ai),
  sqlc.embed(sk)
FROM vacancies v
JOIN companies c ON v.company_id = c.id
LEFT JOIN images i ON c.avatar_id = i.id
JOIN accounts a ON v.author_id = a.id
JOIN users u ON a.user_id = u.id
LEFT JOIN images ai ON u.avatar_id = ai.id
LEFT JOIN vacancy_skills vs ON v.id = vs.vacancy_id
LEFT JOIN skills sk ON vs.skill_id = sk.id
ORDER BY v.title;
