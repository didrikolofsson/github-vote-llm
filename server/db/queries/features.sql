-- name: ListFeatures :many
SELECT *
FROM features
WHERE repository_id = $1
ORDER BY created_at DESC;
-- name: ListFeaturesForPortal :many
SELECT f.*,
  COUNT(fv.id) AS vote_count
FROM features f
  LEFT JOIN feature_votes fv ON fv.feature_id = f.id
WHERE f.repository_id = $1
  AND f.review_status = 'approved'
GROUP BY f.id
ORDER BY vote_count DESC,
  f.created_at DESC;
-- name: GetFeature :one
SELECT *
FROM features
WHERE id = $1;
-- name: CreateFeature :one
INSERT INTO features (
    repository_id,
    title,
    description,
    review_status,
    build_status
  )
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
-- name: PatchFeature :one
UPDATE features
SET title = COALESCE(sqlc.narg('title')::text, title),
  description = COALESCE(sqlc.narg('description')::text, description),
  review_status = COALESCE(
    sqlc.narg('review_status')::review_status_type,
    review_status
  ),
  build_status = COALESCE(
    sqlc.narg('build_status')::build_status_type,
    build_status
  ),
  area = COALESCE(sqlc.narg('area')::text, area),
  updated_at = now()
WHERE id = sqlc.arg('id')
RETURNING *;
-- name: UpdateFeaturePosition :one
UPDATE features
SET roadmap_x = $2,
  roadmap_y = $3,
  roadmap_locked = $4,
  updated_at = now()
WHERE id = $1
RETURNING *;
-- name: DeleteFeature :exec
DELETE FROM features
WHERE id = $1;