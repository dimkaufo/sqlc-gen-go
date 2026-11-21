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
-- plan: custom (no params: the exec mode is the only call argument)
SELECT
  u.*,
  sqlc.embed(i)
FROM users u
LEFT JOIN images i ON u.avatar_id = i.id
ORDER BY u.name;

-- name: SearchUsers :many
-- plan: custom: an optional filter, planned for each call's value.
SELECT *
FROM users
WHERE (sqlc.narg('name')::text IS NULL OR name = sqlc.narg('name'))
ORDER BY id
LIMIT @limit_count;
