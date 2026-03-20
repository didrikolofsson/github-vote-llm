-- name: CreateUser :one
INSERT INTO users (email, password)
VALUES ($1, $2)
RETURNING id,
    email,
    created_at,
    updated_at;
-- name: GetUserByEmail :one
SELECT id,
    email,
    created_at,
    updated_at
FROM users
WHERE email = $1;
-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
-- name: GetUserByID :one
SELECT id,
    email,
    created_at,
    updated_at
FROM users
WHERE id = $1;