-- name: InsertSegments :copyfrom
INSERT INTO segment (
  text, start_time, end_time, duration, conversation_id, speaker_id
) VALUES ( $1, $2, $3, $4, $5, $6 );
