-- name: InsertRecording :one
INSERT INTO recording (
  path,
  extension,
  filename,
  status
) VALUES ( 
  $1,
  $2,
  $3,
  $4
) returning id;

-- name: UpdateRecordingStatus :exec
UPDATE recording
  SET status = $2
  WHERE id = $1;

-- name: GetRecordings :many
SELECT * FROM recording;
