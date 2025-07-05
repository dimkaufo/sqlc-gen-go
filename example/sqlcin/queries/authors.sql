-- name: GetAuthors :many
SELECT 
    a.id,
    a.name,
    a.age,
    sqlc.embed(b),
    sqlc.embed(r),
    sqlc.embed(l)
FROM authors a
LEFT JOIN books b ON b.author_id = a.id
LEFT JOIN labels l ON l.author_id = a.id
LEFT JOIN reviews r ON r.book_id = b.id
ORDER BY a.name, b.title, r.created_at;

-- name: GetAuthor :one
SELECT 
    a.id,
    a.name,
    a.age,
    sqlc.embed(b),
    sqlc.embed(r),
    sqlc.embed(l)
FROM authors a
LEFT JOIN books b ON b.author_id = a.id
LEFT JOIN labels l ON l.author_id = a.id
LEFT JOIN reviews r ON r.book_id = b.id
WHERE a.id = $1
ORDER BY b.title, r.created_at
LIMIT 1; 