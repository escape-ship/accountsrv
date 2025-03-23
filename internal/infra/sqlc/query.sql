-- name: GetUserByEmail :one
SELECT id, email, password_hash
FROM users
WHERE email = ?;

-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (user_id, token, expires_at)
VALUES (?, ?, ?);

-- name: InsertUser :exec
INSERT INTO users (email, password_hash)
VALUES (?, ?);

-- name: GetLastInsertID :one
SELECT LAST_INSERT_ID();