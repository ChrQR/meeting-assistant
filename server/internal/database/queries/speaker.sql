-- name: GetBestMatchingSpeaker :one
-- Voice matching is scoped to the user's own speakers so embeddings never
-- match a voice from another account.
SELECT 
  s.id speaker_id,
  s.name,
  (1.0::float8 - (se.embedding <=> $1::vector))::float8 AS similarity_score
FROM
  speaker_embedding se
JOIN
  speaker s ON s.id = se.speaker_id
WHERE
  s.owner_id = sqlc.arg('owner_id')::uuid
  AND (1.0::float8 - (se.embedding <=> $1::vector)) >= sqlc.arg('min_similarity')::float8
ORDER BY
  se.embedding <=> $1::vector ASC
LIMIT 1;

-- name: CountSpeakerEmbeddings :one
SELECT COUNT(*)
FROM speaker_embedding
WHERE speaker_id = $1;

-- name: InsertAnonymousSpeaker :one
INSERT INTO speaker (
  name,
  owner_id
) VALUES ( NULL, $1 ) returning id;

-- name: InsertSpeaker :one
INSERT INTO speaker (
  name,
  email,
  company_id,
  owner_id
) VALUES ($1, $2, $3, $4) returning id;

-- name: InsertSpeakerEmbedding :one
INSERT INTO speaker_embedding (speaker_id, embedding)
VALUES ($1, $2::vector)
RETURNING id, speaker_id, created_at;

-- name: GetSpeakerEmbeddings :many
SELECT id, created_at
FROM speaker_embedding
WHERE speaker_id = $1
ORDER BY created_at ASC;

-- name: MoveEmbeddingsToSpeaker :exec
UPDATE speaker_embedding
  SET speaker_id = $2
  WHERE speaker_id = $1;

-- name: DeleteSpeaker :exec
DELETE FROM speaker
WHERE id = $1 AND owner_id = $2;

-- name: UpdateSpeakerName :exec
UPDATE speaker
  SET name = $2,
      updated_at = now()
  WHERE id = $1;

-- name: UpdateSpeaker :exec
UPDATE speaker
  SET name = CASE WHEN sqlc.narg('name')::text IS NULL THEN name ELSE sqlc.narg('name') END,
      email = COALESCE(sqlc.narg('email')::varchar, email),
      company_id = CASE WHEN sqlc.narg('clear_company')::bool THEN NULL ELSE COALESCE(sqlc.narg('company_id')::uuid, company_id) END,
      linked_user_id = CASE WHEN sqlc.narg('clear_user')::bool THEN NULL ELSE COALESCE(sqlc.narg('link_user_id')::uuid, linked_user_id) END,
      updated_at = now()
  WHERE id = $1;

-- name: GetSpeakers :many
SELECT
  sp.id,
  sp.name,
  sp.email,
  sp.company_id,
  c.name AS company_name,
  sp.linked_user_id,
  u.email AS user_email,
  COUNT(DISTINCT se.id)::int AS embedding_count,
  COUNT(DISTINCT s.id)::int AS segment_count,
  COALESCE(SUM(s.duration) FILTER (WHERE s.id IS NOT NULL), 0)::float8 AS total_speaking_time,
  COUNT(DISTINCT s.conversation_id)::int AS conversation_count,
  sp.created_at
FROM speaker sp
LEFT JOIN company c ON c.id = sp.company_id
LEFT JOIN app_user u ON u.id = sp.linked_user_id
LEFT JOIN speaker_embedding se ON se.speaker_id = sp.id
LEFT JOIN segment s ON s.speaker_id = sp.id AND NOT s.ignored
WHERE sp.owner_id = $1
GROUP BY sp.id, c.name, u.email
ORDER BY (sp.name IS NULL) ASC, sp.name ASC NULLS LAST, sp.created_at ASC;

-- name: GetSpeaker :one
SELECT
  sp.id,
  sp.name,
  sp.email,
  sp.company_id,
  c.name AS company_name,
  sp.linked_user_id,
  u.email AS user_email,
  COUNT(DISTINCT se.id)::int AS embedding_count,
  COUNT(DISTINCT s.id)::int AS segment_count,
  COALESCE(SUM(s.duration) FILTER (WHERE s.id IS NOT NULL), 0)::float8 AS total_speaking_time,
  COUNT(DISTINCT s.conversation_id)::int AS conversation_count,
  sp.created_at
FROM speaker sp
LEFT JOIN company c ON c.id = sp.company_id
LEFT JOIN app_user u ON u.id = sp.linked_user_id
LEFT JOIN speaker_embedding se ON se.speaker_id = sp.id
LEFT JOIN segment s ON s.speaker_id = sp.id AND NOT s.ignored
WHERE sp.id = $1 AND sp.owner_id = $2
GROUP BY sp.id, c.name, u.email;

-- name: PruneMostRedundantSpeakerEmbedding :exec
WITH pairwise_distances AS (
    SELECT 
        a.id AS id_a,
        b.id AS id_b,
        a.embedding <=> b.embedding AS dist,
        a.created_at AS created_a,
        b.created_at AS created_b
    FROM 
        speaker_embedding a
    JOIN 
        speaker_embedding b 
      ON a.speaker_id = b.speaker_id 
     AND a.id < b.id -- ensures each pair is compared exactly once and avoids self-comparison
    WHERE 
        a.speaker_id = $1
    ORDER BY 
        dist ASC
    LIMIT 1
)
DELETE FROM speaker_embedding
WHERE id = (
    SELECT 
        -- Drop the more recently added vector of the two
        CASE 
            WHEN created_a >= created_b THEN id_a 
            ELSE id_b 
        END
    FROM pairwise_distances
);
