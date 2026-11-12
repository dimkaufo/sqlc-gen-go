-- name: GetChatsByConnectID :many
-- Slice of nested composites, populated through a match rule.
SELECT
  ch.*,
  sqlc.embed(cm),
  sqlc.embed(a),
  sqlc.embed(u),
  sqlc.embed(i)
FROM chats ch
LEFT JOIN chat_members cm ON ch.id = cm.chat_id
LEFT JOIN accounts a ON cm.account_id = a.id
LEFT JOIN users u ON a.user_id = u.id
LEFT JOIN images i ON u.avatar_id = i.id
WHERE ch.connect_id = @connect_id
ORDER BY ch.id;
