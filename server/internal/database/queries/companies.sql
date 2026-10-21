-- name: GetCompanies :many
SELECT
  c.id,
  c.name,
  c.created_at,
  COUNT(s.id)::int AS speaker_count
FROM company c
LEFT JOIN speaker s ON s.company_id = c.id
WHERE c.user_id = $1
GROUP BY c.id
ORDER BY c.name ASC;

-- name: GetCompany :one
SELECT
  c.id,
  c.name,
  c.created_at,
  COUNT(s.id)::int AS speaker_count
FROM company c
LEFT JOIN speaker s ON s.company_id = c.id
WHERE c.id = $1 AND c.user_id = $2
GROUP BY c.id;

-- name: InsertCompany :one
INSERT INTO company (
  name,
  user_id
) VALUES (
  $1, $2
) returning id;

-- name: UpdateCompany :exec
UPDATE company
  SET name = $2,
      updated_at = now()
  WHERE id = $1 AND user_id = $3;

-- name: DeleteCompany :exec
DELETE FROM company
  WHERE id = $1 AND user_id = $2;
