package gitauth_client

import (
	"context"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
)

type GithubTokenSource struct {
	ctx    context.Context
	userID int64
	cfg    *oauth2.Config
	db     *pgxpool.Pool
	q      *store.Queries
}

func (t *GithubTokenSource) Token() (*oauth2.Token, error) {
	tx, err := t.db.BeginTx(t.ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(t.ctx)
	qtx := t.q.WithTx(tx)

	connection, err := qtx.GetGithubConnectionByUserID(t.ctx, t.userID)
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		AccessToken:  connection.AccessToken,
		RefreshToken: connection.RefreshToken,
		Expiry:       connection.AccessTokenExpiresAt.Time,
	}

	if !token.Valid() {
		token, err = t.cfg.TokenSource(t.ctx, token).Token()
		if err != nil {
			return nil, err
		}
		accessTokenExpiry := CalculateAccessTokenExpiry(time.Duration(token.ExpiresIn))
		if _, err := qtx.UpsertGithubAccountTokenByUserID(t.ctx, store.UpsertGithubAccountTokenByUserIDParams{
			UserID:      t.userID,
			AccessToken: token.AccessToken,
			AccessTokenExpiresAt: pgtype.Timestamptz{
				Time: accessTokenExpiry, Valid: true,
			},
			RefreshToken: token.RefreshToken,
		}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(t.ctx); err != nil {
		return nil, err
	}

	return token, nil
}

func CalculateAccessTokenExpiry(expiresIn time.Duration) time.Time {
	return time.Now().Add(expiresIn)
}

type GithubTokenSourceDeps struct {
	Ctx     context.Context
	DB      *pgxpool.Pool
	Queries *store.Queries
	UserID  int64
	Config  *oauth2.Config
}

func NewGithubTokenSource(deps GithubTokenSourceDeps) *GithubTokenSource {
	return &GithubTokenSource{
		ctx:    deps.Ctx,
		db:     deps.DB,
		q:      deps.Queries,
		userID: deps.UserID,
		cfg:    deps.Config,
	}
}
