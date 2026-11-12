-- name: GetConnectChatsByID :one
-- The same composite twice under one parent, each with a matched composite slice inside:
-- the private chat inlines two levels deep on the _2 twins. Members repeat across both chats' rows.
SELECT
  c.*,
  sqlc.embed(gc),
  sqlc.embed(gcm),
  sqlc.embed(ga),
  sqlc.embed(gu),
  sqlc.embed(gi),
  sqlc.embed(pc),
  sqlc.embed(pcm),
  sqlc.embed(pa),
  sqlc.embed(pu),
  sqlc.embed(pi)
FROM connects c
LEFT JOIN chats gc ON c.id = gc.connect_id AND NOT gc.is_private
LEFT JOIN chat_members gcm ON gc.id = gcm.chat_id
LEFT JOIN accounts ga ON gcm.account_id = ga.id
LEFT JOIN users gu ON ga.user_id = gu.id
LEFT JOIN images gi ON gu.avatar_id = gi.id
LEFT JOIN chats pc ON c.id = pc.connect_id AND pc.is_private
LEFT JOIN chat_members pcm ON pc.id = pcm.chat_id
LEFT JOIN accounts pa ON pcm.account_id = pa.id
LEFT JOIN users pu ON pa.user_id = pu.id
LEFT JOIN images pi ON pu.avatar_id = pi.id
WHERE c.id = @id;
