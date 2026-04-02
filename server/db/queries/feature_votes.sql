-- name: AddFeatureVote :one
INSERT INTO feature_votes (feature_id, voter_token, reason, urgency)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: RemoveFeatureVote :exec
DELETE FROM feature_votes
WHERE feature_id = $1 AND voter_token = $2;

-- name: GetFeatureVote :one
SELECT * FROM feature_votes
WHERE feature_id = $1 AND voter_token = $2;

-- name: CountFeatureVotes :one
SELECT COUNT(*) FROM feature_votes WHERE feature_id = $1;

-- name: ListFeatureVotesWithSignals :many
SELECT id, feature_id, voter_token, reason, urgency, created_at
FROM feature_votes
WHERE feature_id = $1
ORDER BY created_at DESC;
