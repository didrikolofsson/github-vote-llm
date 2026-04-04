-- name: CreateRun :one
INSERT INTO feature_runs (
        prompt,
        feature_id,
        status,
        created_by_user_id
    )
VALUES ($1, $2, $3, $4)
RETURNING *;