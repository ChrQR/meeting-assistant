-- name: GetBestMatchingSpeaker :one
SELECT 
  s.id speaker_id,
  s.name,
  (1.0::float8 - (se.embedding <=> $1::vector))::float8 AS similarity_score
FROM
  speaker_embedding se
JOIN
  speaker s ON s.id = se.speaker_id
ORDER BY
  se.embedding <=> $1::vector ASC
LIMIT 1;

-- name: CountSpeakerEmbeddings :one
SELECT COUNT(*)
FROM speaker_embedding
WHERE speaker_id = $1;

-- name: InsertAnonymousSpeaker :one
INSERT INTO speaker (
  name
) VALUES ( NULL ) returning id;

-- name: InsertSpeakerEmbedding :one
INSERT INTO speaker_embedding (speaker_id, embedding)
VALUES ($1, $2::vector)
RETURNING id, speaker_id, created_at;

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

