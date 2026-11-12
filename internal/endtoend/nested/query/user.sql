-- name: GetUserByID :one
-- Non-slice entity child renamed by field_out; :one reads every row.
SELECT
  u.*,
  sqlc.embed(i)
FROM users u
LEFT JOIN images i ON u.avatar_id = i.id
WHERE u.id = @id;

-- name: ListUsers :many
-- Second query on the same struct_root: emits a wrapper over GroupGetUserByID.
SELECT
  u.*,
  sqlc.embed(i)
FROM users u
LEFT JOIN images i ON u.avatar_id = i.id
ORDER BY u.name;
