-- name: GetUserByEmail :one
SELECT id, email, password_hash
FROM account.users
WHERE email = $1;

-- name: InsertRefreshToken :exec
INSERT INTO account.refresh_tokens (id, user_id, token, expires_at)
VALUES ($1, $2, $3, $4);

-- name: InsertUser :one
INSERT INTO account.users (id, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id;