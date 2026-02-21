-- name: CreateUser :exec
INSERT INTO users (email, password_hash)
VALUES ($1, $2);

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1
    LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1
    LIMIT 1;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash = $2,
    updated_at    = now()
WHERE id = $1
    RETURNING *;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token, expires_at)
VALUES ($1, $2, $3)
    RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1
    LIMIT 1;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens
WHERE token = $1;

-- name: DeleteUserRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE user_id = $1;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at < now();