-- name: InsertConversation :one
INSERT INTO conversation (
  title,
  length,
  cost,
  language,
  language_confidence,
  recording_id,
  summary,
  meeting_at
) VALUES ( 
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8
) returning id;

-- name: UpdateConversationSummary :exec
UPDATE conversation
  SET summary = $1
  WHERE id = $2;

-- name: GetConversations :many
SELECT
  c.id,
  c.title,
  c.summary,
  c.length,
  c.cost,
  c.language,
  c.created_at,
  c.meeting_at,
  c.recording_id,
  r.status AS recording_status,
  COUNT(DISTINCT s.speaker_id)::int AS speaker_count,
  -- Only count unidentified speakers that still have segments the user has not
  -- dismissed, so ignoring noise clears the warning.
  COUNT(DISTINCT s.speaker_id) FILTER (WHERE sp.name IS NULL AND NOT s.ignored)::int AS unknown_speaker_count
FROM conversation c
JOIN recording r ON r.id = c.recording_id
LEFT JOIN segment s ON s.conversation_id = c.id
LEFT JOIN speaker sp ON sp.id = s.speaker_id
WHERE r.user_id = $1
GROUP BY c.id, r.status
ORDER BY COALESCE(c.meeting_at, c.created_at) DESC;

-- name: GetConversation :one
SELECT
  c.id,
  c.title,
  c.summary,
  c.length,
  c.cost,
  c.language,
  c.created_at,
  c.meeting_at,
  c.recording_id,
  r.status AS recording_status,
  COUNT(DISTINCT s.speaker_id)::int AS speaker_count,
  COUNT(DISTINCT s.speaker_id) FILTER (WHERE sp.name IS NULL AND NOT s.ignored)::int AS unknown_speaker_count
FROM conversation c
JOIN recording r ON r.id = c.recording_id
LEFT JOIN segment s ON s.conversation_id = c.id
LEFT JOIN speaker sp ON sp.id = s.speaker_id
WHERE c.id = $1 AND r.user_id = $2
GROUP BY c.id, r.status;

-- name: GetSegmentsByConversation :many
SELECT
  s.id,
  s.text,
  s.start_time,
  s.end_time,
  s.duration,
  s.speaker_id,
  sp.name AS speaker_name,
  s.ignored
FROM segment s
LEFT JOIN speaker sp ON sp.id = s.speaker_id
JOIN conversation c ON c.id = s.conversation_id
JOIN recording r ON r.id = c.recording_id
WHERE s.conversation_id = $1 AND r.user_id = $2
ORDER BY s.start_time ASC;

-- name: GetConversationSegment :one
SELECT
  s.id,
  s.text,
  s.start_time,
  s.end_time,
  s.duration,
  s.speaker_id,
  sp.name AS speaker_name,
  s.ignored
FROM segment s
LEFT JOIN speaker sp ON sp.id = s.speaker_id
JOIN conversation c ON c.id = s.conversation_id
JOIN recording r ON r.id = c.recording_id
WHERE s.conversation_id = $1 AND s.id = $2 AND r.user_id = $3;

-- name: SetSegmentIgnored :exec
UPDATE segment
  SET ignored = $2,
      updated_at = now()
  WHERE id = $1 AND conversation_id = $3;
