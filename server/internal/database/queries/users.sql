-- name: GetUser :one
SELECT * FROM app_user
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM app_user
WHERE email = $1;

-- name: InsertUser :one
INSERT INTO app_user (
  email,
  name,
  avatar_url
) VALUES (
  $1, $2, $3
) returning *;

-- name: UpdateUserProfile :exec
UPDATE app_user
  SET email = $2,
      name = $3,
      avatar_url = $4,
      updated_at = now()
  WHERE id = $1;

-- name: GetAccount :one
SELECT * FROM account
WHERE provider = $1 AND provider_account_id = $2;

-- name: InsertAccount :one
INSERT INTO account (
  user_id,
  provider,
  provider_account_id
) VALUES (
  $1, $2, $3
) returning *;

-- name: GetAccountsByUser :many
SELECT * FROM account
WHERE user_id = $1
ORDER BY created_at ASC;
