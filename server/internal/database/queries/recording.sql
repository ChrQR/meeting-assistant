-- name: InsertRecording :one
INSERT INTO recording (
  path,
  extension,
  filename,
  status,
  user_id
) VALUES ( 
  $1,
  $2,
  $3,
  $4,
  $5
) returning id;

-- name: UpdateRecordingStatus :exec
UPDATE recording
  SET status = $2
  WHERE id = $1;

-- name: GetRecordings :many
SELECT * FROM recording
WHERE user_id = $1
ORDER BY filename ASC;

-- name: GetRecording :one
SELECT * FROM recording
WHERE id = $1 AND user_id = $2;

-- name: UpdateRecordingExtension :exec
UPDATE recording
  SET extension = $2
  WHERE id = $1;
