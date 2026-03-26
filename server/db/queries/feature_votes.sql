-- name: AddFeatureVote :one
INSERT INTO feature_votes (feature_id, voter_token)
VALUES ($1, $2)
RETURNING *;

-- name: RemoveFeatureVote :exec
DELETE FROM feature_votes
WHERE feature_id = $1 AND voter_token = $2;

-- name: GetFeatureVote :one
SELECT * FROM feature_votes
WHERE feature_id = $1 AND voter_token = $2;

-- name: CountFeatureVotes :one
SELECT COUNT(*) FROM feature_votes WHERE feature_id = $1;
