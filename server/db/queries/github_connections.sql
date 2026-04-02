-- name: GetGitHubConnectionByUserID :one
SELECT user_id, access_token_encrypted, refresh_token, token_expires_at, github_user_id, github_login, created_at, updated_at
FROM github_connections
WHERE user_id = $1;

-- name: UpsertGitHubConnection :one
INSERT INTO github_connections (user_id, access_token_encrypted, refresh_token, token_expires_at, github_user_id, github_login)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id) DO UPDATE SET
    access_token_encrypted = EXCLUDED.access_token_encrypted,
    refresh_token = EXCLUDED.refresh_token,
    token_expires_at = EXCLUDED.token_expires_at,
    github_user_id = EXCLUDED.github_user_id,
    github_login = EXCLUDED.github_login,
    updated_at = now()
RETURNING *;

-- name: DeleteGitHubConnection :exec
DELETE FROM github_connections
WHERE user_id = $1;
