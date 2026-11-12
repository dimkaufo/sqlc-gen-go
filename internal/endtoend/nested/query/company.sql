-- name: GetCompanyByID :one
-- CompanyComposite as a query root: its populate function is generated here, where the
-- company avatar is Image — GetVacancyByID reuses it while its company avatar is Image_2.
SELECT
  c.*,
  sqlc.embed(i)
FROM companies c
LEFT JOIN images i ON c.avatar_id = i.id
WHERE c.id = @id;
