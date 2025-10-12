-- name: GetBooks :many
SELECT 
    b.*,
    sqlc.embed(r),
    sqlc.embed(l)
FROM books b
LEFT JOIN authors a ON a.id = b.author_id
LEFT JOIN labels l ON l.author_id = a.id
LEFT JOIN reviews r ON r.book_id = b.id
ORDER BY b.title, r.created_at; 