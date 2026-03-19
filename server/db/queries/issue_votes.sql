-- name: IncrementIssueVote :one
INSERT INTO issue_votes (owner, repo, issue_number, vote_count)
VALUES ($1, $2, $3, 1)
ON CONFLICT (owner, repo, issue_number) DO UPDATE SET
    vote_count = issue_votes.vote_count + 1,
    updated_at = now()
RETURNING *;
