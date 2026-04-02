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

-- name: GetUserByEmailWithPassword :one
SELECT id,
    email,
    password,
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
    username,
    created_at,
    updated_at
FROM users
WHERE id = $1;

-- name: UpdateUserUsername :one
UPDATE users
SET username = $2, updated_at = now()
WHERE id = $1
RETURNING id, email, username, created_at, updated_at;