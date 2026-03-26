-- name: AddFeatureDependency :exec
INSERT INTO feature_dependencies (feature_id, depends_on)
VALUES ($1, $2);

-- name: RemoveFeatureDependency :exec
DELETE FROM feature_dependencies
WHERE feature_id = $1 AND depends_on = $2;

-- name: ListFeatureDependencies :many
SELECT * FROM feature_dependencies WHERE feature_id = $1;

-- name: ListFeatureDependenciesForRepository :many
SELECT * FROM feature_dependencies
WHERE feature_id IN (SELECT id FROM features WHERE repository_id = $1);
