-- name: ListFeatures :many
SELECT * FROM features
WHERE repository_id = $1
ORDER BY created_at DESC;

-- name: GetFeature :one
SELECT * FROM features WHERE id = $1;

-- name: CreateFeature :one
INSERT INTO features (repository_id, title, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: PatchFeature :one
UPDATE features
SET
  title       = COALESCE(sqlc.narg('title')::text, title),
  description = COALESCE(sqlc.narg('description')::text, description),
  status      = COALESCE(sqlc.narg('status')::feature_status, status),
  area        = COALESCE(sqlc.narg('area')::text, area),
  updated_at  = now()
WHERE id = sqlc.arg('id') RETURNING *;

-- name: UpdateFeaturePosition :one
UPDATE features
SET roadmap_x = $2, roadmap_y = $3, roadmap_locked = $4, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeleteFeature :exec
DELETE FROM features WHERE id = $1;
