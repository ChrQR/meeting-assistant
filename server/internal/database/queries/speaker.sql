-- name: GetSimilarSpeaker :one
SELECT 
  id,
  voice_embedding,
  (1.0::float8 - (voice_embedding <=> $1::vector))::float8 AS similarity_score
FROM
  speaker
ORDER BY
  voice_embedding <=> $1::vector ASC
LIMIT 1;

-- name: InsertAnonymousSpeaker :one
INSERT INTO speaker (
  voice_embedding
) VALUES ( $1 ) returning id;
