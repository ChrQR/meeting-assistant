-- name: InsertConversation :one
INSERT INTO conversation (
  title,
  length,
  cost,
  language,
  language_confidence,
  recording_id
) VALUES ( 
  $1,
  $2,
  $3,
  $4,
  $5,
  $6
) returning id;

-- name: UpdateConversationSummary :exec
UPDATE conversation
  SET summary = $1
  WHERE id = $2;
