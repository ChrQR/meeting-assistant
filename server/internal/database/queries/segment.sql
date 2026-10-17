-- name: InsertSegments :copyfrom
INSERT INTO segment (
  text, start_time, end_time, duration, conversation_id, speaker_label
) VALUES ( $1, $2, $3, $4, $5, $6 );

-- name: UpdateSegmentsSpeaker :exec
UPDATE segment
  SET speaker_id = $1
  WHERE speaker_label = $2 and conversation_id = $3;
