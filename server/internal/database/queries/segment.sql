-- name: InsertSegments :copyfrom
INSERT INTO segment (
  text, start_time, end_time, duration, conversation_id, speaker_id
) VALUES ( $1, $2, $3, $4, $5, $6 );

-- name: GetSegmentsBySpeaker :many
SELECT
  s.id,
  s.text,
  s.start_time,
  s.end_time,
  s.conversation_id,
  c.title AS conversation_title
FROM segment s
JOIN conversation c ON c.id = s.conversation_id
JOIN speaker sp ON sp.id = s.speaker_id
WHERE s.speaker_id = $1
  AND sp.owner_id = $2
  AND NOT s.ignored
ORDER BY COALESCE(c.meeting_at, c.created_at) DESC, s.start_time ASC;

-- name: MoveSegmentsToSpeaker :exec
-- Ignored segments move with the speaker so the source speaker can be deleted.
-- They stay ignored, so they are never surfaced as the target's speech.
UPDATE segment
  SET speaker_id = $2
  WHERE speaker_id = $1;

-- name: CountSegmentsBySpeaker :one
SELECT COUNT(*)
FROM segment s
JOIN speaker sp ON sp.id = s.speaker_id
WHERE s.speaker_id = $1
  AND sp.owner_id = $2
  AND NOT s.ignored;
