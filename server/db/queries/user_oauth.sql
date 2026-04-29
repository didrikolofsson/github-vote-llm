-- name: CreateAuthCode :one
INSERT INTO
    user_authorization_codes (
        code,
        user_id,
        code_challenge,
        redirect_uri,
        expires_at
    )
VALUES ($1, $2, $3, $4, $5)
RETURNING
    id,
    code,
    user_id,
    code_challenge,
    redirect_uri,
    used,
    expires_at,
    created_at;

-- name: GetAuthCode :one
SELECT
    id,
    code,
    user_id,
    code_challenge,
    redirect_uri,
    used,
    expires_at,
    created_at
FROM user_authorization_codes
WHERE
    code = $1;

-- name: UseAuthCode :exec
UPDATE user_authorization_codes SET used = TRUE WHERE id = $1;

-- name: CreateRefreshToken :one
INSERT INTO
    user_refresh_tokens (
        token_hash,
        user_id,
        expires_at
    )
VALUES ($1, $2, $3)
RETURNING
    id,
    token_hash,
    user_id,
    expires_at,
    created_at;

-- name: GetRefreshToken :one
SELECT
    id,
    token_hash,
    user_id,
    expires_at,
    created_at
FROM user_refresh_tokens
WHERE
    token_hash = $1;

-- name: DeleteRefreshToken :exec
DELETE FROM user_refresh_tokens WHERE token_hash = $1;

-- name: UpsertGithubAccountTokenByUserID :one
INSERT INTO
    github_connections (
        user_id,
        access_token,
        access_token_expires_at,
        refresh_token
    )
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO
UPDATE
SET
    access_token = EXCLUDED.access_token,
    access_token_expires_at = EXCLUDED.access_token_expires_at,
    refresh_token = EXCLUDED.refresh_token,
    updated_at = now()
RETURNING
    user_id,
    access_token,
    access_token_expires_at,
    refresh_token,
    updated_at;

-- name: GetGithubConnectionByUserID :one
SELECT * FROM github_connections WHERE user_id = $1;