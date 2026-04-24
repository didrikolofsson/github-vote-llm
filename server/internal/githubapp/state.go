package githubapp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const StateTTL = 10 * time.Minute

var (
	ErrInvalidState     = errors.New("githubapp: invalid or expired install state")
	ErrStateUserMismatch = errors.New("githubapp: install state user mismatch")
)

// GenerateNonce returns a cryptographically random, URL-safe token.
func GenerateNonce() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CreateInstallState persists a single-use nonce bound to userID.
func CreateInstallState(ctx context.Context, q *store.Queries, userID int64) (string, error) {
	nonce, err := GenerateNonce()
	if err != nil {
		return "", err
	}
	_, err = q.CreateInstallState(ctx, store.CreateInstallStateParams{
		Nonce:     nonce,
		UserID:    userID,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(StateTTL), Valid: true},
	})
	if err != nil {
		return "", err
	}
	return nonce, nil
}

// ConsumeInstallState atomically marks the state consumed and returns the bound userID.
// Fails if the nonce is missing, expired, or already consumed.
func ConsumeInstallState(ctx context.Context, q *store.Queries, nonce string) (int64, error) {
	if nonce == "" {
		return 0, ErrInvalidState
	}
	row, err := q.ConsumeInstallState(ctx, nonce)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrInvalidState
		}
		return 0, err
	}
	return row.UserID, nil
}
