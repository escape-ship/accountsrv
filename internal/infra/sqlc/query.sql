-- name: GetUserByEmail :one
SELECT id, email, password_hash
FROM account.users
WHERE email = $1;

-- name: InsertRefreshToken :exec
INSERT INTO account.refresh_tokens (user_id, token, expires_at)
VALUES ($1, $2, $3);

-- name: InsertUser :one
INSERT INTO account.users (email, password_hash)
VALUES ($1, $2)
RETURNING id;