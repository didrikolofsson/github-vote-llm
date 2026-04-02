-- name: ListFeatureComments :many
SELECT * FROM feature_comments
WHERE feature_id = $1
ORDER BY created_at ASC;

-- name: CreateFeatureComment :one
INSERT INTO feature_comments (feature_id, body, author_name)
VALUES ($1, $2, $3)
RETURNING *;
