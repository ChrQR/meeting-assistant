-- name: InsertSegments :copyfrom
INSERT INTO segment (
  text, start_time, end_time, duration, conversation_id, speaker_id
) VALUES ( $1, $2, $3, $4, $5, $6 );

-- name: GetSegmentsWithSpeakerNameByConversationID :many
SELECT 
  seg.id,
  seg.text,
  seg.start_time,
  seg.end_time,
  seg.duration,
  seg.speaker_id,
  sp.name as speaker_name,
  seg.created_at,
  seg.updated_at
FROM segment seg
JOIN speaker sp on seg.speaker_id = sp.id
WHERE seg.conversation_id = $1
ORDER BY seg.start_time ASC;
