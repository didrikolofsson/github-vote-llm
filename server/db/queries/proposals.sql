-- name: ListProposals :many
SELECT * FROM proposals WHERE owner = $1 AND repo = $2 ORDER BY vote_count DESC;

-- name: GetProposal :one
SELECT * FROM proposals WHERE id = $1;

-- name: CreateProposal :one
INSERT INTO proposals (owner, repo, title, description) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: IncrementProposalVote :one
UPDATE proposals SET vote_count = vote_count + 1, updated_at = NOW()
WHERE id = $1 RETURNING *;

-- name: UpdateProposalStatus :one
UPDATE proposals SET status = $2, updated_at = NOW() WHERE id = $1 RETURNING *;
